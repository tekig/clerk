package awss3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type Storage struct {
	bucket  string
	session *session.Session
}

type StorageConfig struct {
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

func (s *Storage) SaveBlock(ctx context.Context, dir string) error {
	uploader := s3manager.NewUploader(s.session)
	blockName := filepath.Base(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		f, err := os.Open(path.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("open %s: %w", entry.Name(), err)
		}
		defer f.Close()

		bucket := s.bucket
		key := path.Join(blockName, entry.Name())

		if _, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
			Body:   f,
			Bucket: &bucket,
			Key:    &key,
		}); err != nil {
			return fmt.Errorf("upload: %w", err)
		}
	}

	return nil
}

func (s *Storage) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	return s.ReadRange(ctx, name, 0, 0)
}

func (s *Storage) ReadRange(ctx context.Context, name string, offset, size int) (io.ReadCloser, error) {
	downloader := s3manager.NewDownloader(s.session)

	r, w := io.Pipe()
	go func() {
		var r *string
		if size != 0 {
			v := "bytes=" + strconv.Itoa(offset) + "-" + strconv.Itoa(offset+size)
			r = &v
		}

		_, err := downloader.DownloadWithContext(ctx, &WriteAt{w}, &s3.GetObjectInput{
			Bucket: &s.bucket,
			Key:    &name,
			Range:  r,
		})
		if err != nil {
			err = fmt.Errorf("downloader: %w", err)
		}

		w.CloseWithError(err)
	}()

	return r, nil
}
