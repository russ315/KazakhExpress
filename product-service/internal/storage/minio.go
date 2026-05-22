package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIO struct {
	client   *minio.Client
	bucket   string
	endpoint string
	public   string
	useSSL   bool
}

func NewMinIO(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicEndpoint ...string) (*MinIO, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	public := endpoint
	if len(publicEndpoint) > 0 && publicEndpoint[0] != "" {
		public = publicEndpoint[0]
	}
	return &MinIO{client: client, bucket: bucket, endpoint: endpoint, public: public, useSSL: useSSL}, nil
}

func (m *MinIO) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}
	if err := m.client.SetBucketPolicy(ctx, m.bucket, fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": "*",
      "Action": ["s3:GetObject"],
      "Resource": ["arn:aws:s3:::%s/*"]
    }
  ]
}`, m.bucket)); err != nil {
		return err
	}
	return nil
}

func (m *MinIO) Save(ctx context.Context, objectName, contentType string, content []byte) (string, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := m.client.PutObject(ctx, m.bucket, objectName, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}
	scheme := "http"
	if m.useSSL {
		scheme = "https"
	}
	return (&url.URL{Scheme: scheme, Host: m.public, Path: "/" + m.bucket + "/" + objectName}).String(), nil
}
