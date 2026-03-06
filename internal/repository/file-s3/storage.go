package files3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Storage struct {
	tmp      string
	bucket   string
	client   *s3.Client
	transfer *transfermanager.Client
}

type StorageConfig struct {
	TempDir      string
	Endpoint     string
	Bucket       string
	AccessKey    string
	AccessSecret string
	Region       string
	Upload       UploadConfig
}

type UploadConfig struct {
	PartSize    int
	Concurrency int
}

func NewStorage(c StorageConfig) (*Storage, error) {
	if c.Endpoint != "" {
		// Backward compatibility
		// Previously, aws-sdk-go accepted Endpoints without specifying a schema.
		uri, err := url.Parse(c.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("parse endpoint: %w", err)
		}

		if uri.Scheme == "" {
			c.Endpoint = "https://" + c.Endpoint
		}
	}

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(c.Region),
		config.WithBaseEndpoint(c.Endpoint),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.AccessKey, c.AccessSecret, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	client := s3.NewFromConfig(cfg)

	transfer := transfermanager.New(client, func(o *transfermanager.Options) {
		o.Concurrency = c.Upload.Concurrency
		o.PartSizeBytes = int64(c.Upload.PartSize)
	})

	return &Storage{
		tmp:      c.TempDir,
		bucket:   c.Bucket,
		client:   client,
		transfer: transfer,
	}, nil
}

func (s *Storage) Blocks(ctx context.Context) ([]string, error) {
	var continuationToken *string
	var blocks []string
	for {
		objects, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            &s.bucket,
			Delimiter:         aws.String("/"),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}

		for _, cp := range objects.CommonPrefixes {
			if cp.Prefix == nil {
				continue
			}
			block := strings.TrimSuffix(*cp.Prefix, "/")
			blocks = append(blocks, block)
		}

		if objects.NextContinuationToken == nil {
			break
		}

		continuationToken = objects.NextContinuationToken
	}

	return blocks, nil
}

func (s *Storage) Read(ctx context.Context, block, name string) (io.ReadCloser, error) {
	return s.ReadRange(ctx, block, name, 0, -1)
}

func (s *Storage) ReadRange(ctx context.Context, block, name string, offset, size int) (io.ReadCloser, error) {
	f, err := os.Open(path.Join(s.tmp, block, name))
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("open: %w", err)
	}
	if f != nil {
		if _, err := f.Seek(int64(offset), 1); err != nil {
			return nil, fmt.Errorf("seek: %w", err)
		}

		var r io.ReadCloser = f
		if size != -1 {
			r = newLimitedReader(f, size)
		}

		return r, nil
	}

	r, w := io.Pipe()
	go func() {
		var r *string
		if size != 0 {
			v := "bytes=" + strconv.Itoa(offset) + "-" + strconv.Itoa(offset+size)
			r = &v
		}

		_, err := s.transfer.DownloadObject(ctx, &transfermanager.DownloadObjectInput{
			Bucket:   &s.bucket,
			Key:      aws.String(path.Join(block, name)),
			Range:    r, // rename to Range https://github.com/aws/aws-sdk-go-v2/issues/3322
			WriterAt: newWriteAt(w),
		})
		if err != nil {
			err = fmt.Errorf("downloader: %w", err)
		}

		w.CloseWithError(err)
	}()

	return r, nil
}

func (s *Storage) Write(ctx context.Context, block, name string) (io.WriteCloser, error) {
	if err := os.MkdirAll(path.Join(s.tmp, block), 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.Create(path.Join(s.tmp, block, name))
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}

	pr, pw := io.Pipe()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		_, err := s.transfer.UploadObject(ctx, &transfermanager.UploadObjectInput{
			Body:   pr,
			Bucket: &s.bucket,
			Key:    aws.String(path.Join(block, name)),
		})
		if err != nil {
			err = fmt.Errorf("upload: %w", err)
		}
		pr.CloseWithError(err)
	}()

	return newMultiWriter(&closeWriter{
		w: f,
		after: func() error {
			err1 := os.Remove(path.Join(s.tmp, block, name))

			// Automatically delete directory when last file is deleted.
			// The error is not critical if the directory is not empty.
			// The problem is that the error returned is different for each platform.
			// To support more platforms, the error is simply ignored.
			err2 := os.Remove(path.Join(s.tmp, block))
			err2 = nil

			return errors.Join(err1, err2)
		},
	}, &closeWriter{
		w: pw,
		after: func() error {
			wg.Wait()
			return nil
		},
	}), nil
}
