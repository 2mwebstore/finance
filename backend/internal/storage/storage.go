// Package storage keeps uploaded files (transaction attachments).
//
// Cloudflare R2 is used when R2_BUCKET, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY and
// R2_ACCOUNT_ID (or R2_ENDPOINT) are set; otherwise files go to UPLOAD_DIR on local disk,
// which is fine for development but not for hosts with ephemeral disks (Railway).
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ErrNotFound is returned when an object does not exist.
var ErrNotFound = errors.New("file not found")

type Store interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	// Kind is "r2" or "local", for logs.
	Kind() string
}

// NewFromEnv picks R2 when it is configured, local disk otherwise.
func NewFromEnv() (Store, error) {
	bucket := os.Getenv("R2_BUCKET")
	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	endpoint := os.Getenv("R2_ENDPOINT")
	if endpoint == "" && os.Getenv("R2_ACCOUNT_ID") != "" {
		endpoint = os.Getenv("R2_ACCOUNT_ID") + ".r2.cloudflarestorage.com"
	}
	if bucket != "" || accessKey != "" || secretKey != "" {
		if bucket == "" || accessKey == "" || secretKey == "" || endpoint == "" {
			return nil, errors.New("R2 is partly configured: set R2_BUCKET, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY and R2_ACCOUNT_ID (or R2_ENDPOINT)")
		}
		return newR2(endpoint, bucket, accessKey, secretKey)
	}
	dir := os.Getenv("UPLOAD_DIR")
	if dir == "" {
		dir = "uploads"
	}
	return newLocal(dir)
}

// --- Cloudflare R2 (S3-compatible) ---

type r2Store struct {
	client *minio.Client
	bucket string
}

func newR2(endpoint, bucket, accessKey, secretKey string) (*r2Store, error) {
	endpoint = strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://"), "/")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: true,
		Region: "auto",
	})
	if err != nil {
		return nil, fmt.Errorf("r2 client: %w", err)
	}
	return &r2Store{client: client, bucket: bucket}, nil
}

func (s *r2Store) Kind() string { return "r2" }

func (s *r2Store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *r2Store) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	// GetObject is lazy; Stat surfaces "missing" before we start streaming a response.
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return obj, nil
}

func (s *r2Store) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// --- Local disk (development fallback) ---

type localStore struct {
	root string
}

func newLocal(dir string) (*localStore, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("upload dir: %w", err)
	}
	return &localStore{root: root}, nil
}

func (s *localStore) Kind() string { return "local" }

func (s *localStore) path(key string) (string, error) {
	p := filepath.Join(s.root, filepath.FromSlash(key))
	if !strings.HasPrefix(p, s.root+string(os.PathSeparator)) {
		return "", errors.New("invalid storage key")
	}
	return p, nil
}

func (s *localStore) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(p)
		return err
	}
	return f.Close()
}

func (s *localStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

func (s *localStore) Delete(_ context.Context, key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
