// Package storage wraps an S3-compatible object store (MinIO) for certificates,
// manuals and condition photos. Keys are app-controlled; reads use presigned URLs.
package storage

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/ncl/mooring-api/internal/config"
)

// FileStore is a thin wrapper over a MinIO client bound to a single bucket.
//
// Two clients are kept: mc talks to the in-cluster endpoint (server-to-server
// puts/reads), while presign signs URLs against the public endpoint that a
// browser can actually reach. They differ in Docker (minio:9000 internally vs
// localhost:9100 from the host); the host is part of the signature so a single
// client cannot serve both.
type FileStore struct {
	mc      *minio.Client
	presign *minio.Client
	bucket  string
	region  string
}

func newClient(endpoint, accessKey, secretKey, region string, useSSL bool) (*minio.Client, error) {
	ep := strings.TrimPrefix(endpoint, "https://")
	ep = strings.TrimPrefix(ep, "http://")
	secure := useSSL || strings.HasPrefix(endpoint, "https://")
	return minio.New(ep, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
		Region: region,
	})
}

// New builds a FileStore from config.
func New(cfg *config.Config) (*FileStore, error) {
	mc, err := newClient(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Region, cfg.S3UseSSL)
	if err != nil {
		return nil, err
	}
	public := cfg.S3PublicEndpoint
	if public == "" {
		public = cfg.S3Endpoint
	}
	presign, err := newClient(public, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Region, cfg.S3UseSSL)
	if err != nil {
		return nil, err
	}
	return &FileStore{mc: mc, presign: presign, bucket: cfg.S3Bucket, region: cfg.S3Region}, nil
}

// EnsureBucket creates the bucket if it does not yet exist.
func (f *FileStore) EnsureBucket(ctx context.Context) error {
	exists, err := f.mc.BucketExists(ctx, f.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return f.mc.MakeBucket(ctx, f.bucket, minio.MakeBucketOptions{Region: f.region})
}

// Put stores an object with the given content type.
func (f *FileStore) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := f.mc.PutObject(ctx, f.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	return err
}

// PresignGet returns a time-limited GET URL for the object.
func (f *FileStore) PresignGet(ctx context.Context, key string) (string, error) {
	u, err := f.presign.PresignedGetObject(ctx, f.bucket, key, 24*time.Hour, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// Remove deletes an object (best-effort).
func (f *FileStore) Remove(ctx context.Context, key string) error {
	return f.mc.RemoveObject(ctx, f.bucket, key, minio.RemoveObjectOptions{})
}
