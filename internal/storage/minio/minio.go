package minio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type FileStorage struct {
	cl *minio.Client
}

var (
	ErrBucketNotFound = errors.New("bucket not found")
)

func New(host string, port int, user string, password string, useSSL bool) (*FileStorage, error) {
	endpoint := net.JoinHostPort(host, strconv.Itoa(port))

	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(user, password, ""),
		Secure: useSSL,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect storage: %w", err)
	}

	return &FileStorage{cl: client}, nil
}

func (s *FileStorage) SaveFile(ctx context.Context, bucketName string, objectName string, data []byte, contentType string) (string, error) {
	exists, err := s.cl.BucketExists(ctx, bucketName)
	if err != nil {
		return "", fmt.Errorf("error check exists bucket: %w", err)
	}

	if !exists {
		err = s.cl.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return "", fmt.Errorf("error create bucket: %w", err)
		}
	}

	reader := bytes.NewReader(data)
	size := int64(len(data))

	info, err := s.cl.PutObject(ctx, bucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("error put object: %w", err)
	}

	return info.Key, nil
}

func (s *FileStorage) GetFile(ctx context.Context, bucketName string, objectName string) ([]byte, error) {
	exists, err := s.cl.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("error check exists bucket: %w", err)
	}

	if !exists {
		return nil, ErrBucketNotFound
	}

	obj, err := s.cl.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("error get object: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("error read file: %w", err)
	}

	return data, nil
}

func (s *FileStorage) DeleteFile(ctx context.Context, bucketName string, objectName string) error {
	exists, err := s.cl.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("error check exists bucket: %w", err)
	}

	if !exists {
		return ErrBucketNotFound
	}

	return s.cl.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
}

func (fs *FileStorage) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2 * time.Second)
	defer cancel()
	_, err := fs.cl.ListBuckets(ctx)
	return err
}

func (fs *FileStorage) Name() string {
	return "minio"
}
