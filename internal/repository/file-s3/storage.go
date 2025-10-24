package files3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type Storage struct {
	tmp     string
	bucket  string
	session *session.Session
}

type StorageConfig struct {
	TempDir      string
	Endpoint     string
	Bucket       string
	AccessKey    string
	AccessSecret string
}

func NewStorage(c StorageConfig) (*Storage, error) {
	config := aws.NewConfig().
		WithEndpoint(c.Endpoint).
		WithCredentials(credentials.NewStaticCredentials(c.AccessKey, c.AccessSecret, "")).
		WithRegion("ru-central1")

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &Storage{
		tmp:     c.TempDir,
		bucket:  c.Bucket,
		session: sess,
	}, nil
}

func (s *Storage) Blocks(ctx context.Context) ([]string, error) {
	lister := s3.New(s.session)

	objects, err := lister.ListObjects(&s3.ListObjectsInput{
		Bucket: &s.bucket,
	})
	if err != nil {
		return nil, fmt.Errorf("list objects: %w", err)
	}

	var blocksMap = make(map[string]struct{}, len(objects.Contents))
	for _, content := range objects.Contents {
		if content.Key == nil {
			continue
		}
		p := strings.SplitN(*content.Key, "/", 2)
		blocksMap[p[0]] = struct{}{}
	}

	var blocks = make([]string, 0, len(objects.Contents))
	for b := range blocksMap {
		blocks = append(blocks, b)
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

	downloader := s3manager.NewDownloader(s.session)

	r, w := io.Pipe()
	go func() {
		var r *string
		if size != 0 {
			v := "bytes=" + strconv.Itoa(offset) + "-" + strconv.Itoa(offset+size)
			r = &v
		}

		_, err := downloader.DownloadWithContext(ctx, newWriteAt(w), &s3.GetObjectInput{
			Bucket: &s.bucket,
			Key:    aws.String(path.Join(block, name)),
			Range:  r,
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

	uploader := s3manager.NewUploader(s.session)

	pr, pw := io.Pipe()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		_, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
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
