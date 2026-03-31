package storage

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/ix64/s3-go/s3"
	"github.com/ix64/s3-go/s3down"
	"github.com/ix64/s3-go/s3up"
	"github.com/minio/minio-go/v7"
)

var ErrDisabled = errors.New("object storage is disabled")

// Service is the interface implemented by both the live and disabled storage backends.
type Service interface {
	Enabled() bool
	Bucket() string
	Prefix() string
	ResolveExpire(expireSeconds int) time.Duration
	GenerateUpload(ctx context.Context, params *s3up.GenerateParams, expireSeconds int) (*s3up.GenerateResult, time.Duration, error)
	GenerateDownload(ctx context.Context, params *s3down.GenerateParams, expireSeconds int) (*url.URL, time.Duration, error)
	Stat(ctx context.Context, remotePath string) (minio.ObjectInfo, error)
}

// storageService is the live implementation backed by a real S3 client.
type storageService struct {
	client *s3.Client
	cfg    Config
}

// disabledService is returned when object storage is disabled in config.
type disabledService struct {
	cfg Config
}

func NewService(client *s3.Client, cfg Config) Service {
	return &storageService{client: client, cfg: cfg}
}

func NewDisabledService(cfg Config) Service {
	return &disabledService{cfg: cfg}
}

// --- storageService ---

func (s *storageService) Enabled() bool { return true }
func (s *storageService) Bucket() string { return s.cfg.Bucket }
func (s *storageService) Prefix() string { return s.cfg.Prefix }

func (s *storageService) ResolveExpire(expireSeconds int) time.Duration {
	if expireSeconds > 0 {
		return time.Duration(expireSeconds) * time.Second
	}
	if s.cfg.PresignExpireSeconds > 0 {
		return time.Duration(s.cfg.PresignExpireSeconds) * time.Second
	}
	return 15 * time.Minute
}

func (s *storageService) GenerateUpload(ctx context.Context, params *s3up.GenerateParams, expireSeconds int) (*s3up.GenerateResult, time.Duration, error) {
	expireIn := s.ResolveExpire(expireSeconds)
	params.ExpireIn = expireIn

	ret, err := s.client.GenerateUpload(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	return ret, expireIn, nil
}

func (s *storageService) GenerateDownload(ctx context.Context, params *s3down.GenerateParams, expireSeconds int) (*url.URL, time.Duration, error) {
	expireIn := s.ResolveExpire(expireSeconds)
	params.ExpireIn = expireIn

	ret, err := s.client.GenerateDownload(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	return ret, expireIn, nil
}

func (s *storageService) Stat(ctx context.Context, remotePath string) (minio.ObjectInfo, error) {
	return s.client.Stat(ctx, remotePath)
}

// --- disabledService ---

func (s *disabledService) Enabled() bool { return false }
func (s *disabledService) Bucket() string { return s.cfg.Bucket }
func (s *disabledService) Prefix() string { return s.cfg.Prefix }

func (s *disabledService) ResolveExpire(expireSeconds int) time.Duration {
	if expireSeconds > 0 {
		return time.Duration(expireSeconds) * time.Second
	}
	if s.cfg.PresignExpireSeconds > 0 {
		return time.Duration(s.cfg.PresignExpireSeconds) * time.Second
	}
	return 15 * time.Minute
}

func (s *disabledService) GenerateUpload(_ context.Context, _ *s3up.GenerateParams, _ int) (*s3up.GenerateResult, time.Duration, error) {
	return nil, 0, ErrDisabled
}

func (s *disabledService) GenerateDownload(_ context.Context, _ *s3down.GenerateParams, _ int) (*url.URL, time.Duration, error) {
	return nil, 0, ErrDisabled
}

func (s *disabledService) Stat(_ context.Context, _ string) (minio.ObjectInfo, error) {
	return minio.ObjectInfo{}, ErrDisabled
}

// NormalizeRemotePath validates and normalizes a remote object path.
func NormalizeRemotePath(remotePath string) (string, error) {
	normalized := strings.TrimSpace(remotePath)
	normalized = strings.TrimPrefix(normalized, "/")
	if normalized == "" {
		return "", errors.New("remote_path is required")
	}

	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("remote_path contains invalid path segment")
		}
	}

	return normalized, nil
}
