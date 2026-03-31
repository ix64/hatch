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

type Service struct {
	client *s3.Client
	cfg    Config
}

func NewService(client *s3.Client, cfg Config) *Service {
	return &Service{client: client, cfg: cfg}
}

func NewDisabledService(cfg Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) Enabled() bool {
	return s.cfg.Enabled && s.client != nil
}

func (s *Service) Bucket() string {
	return s.cfg.Bucket
}

func (s *Service) Prefix() string {
	return s.cfg.Prefix
}

func (s *Service) ResolveExpire(expireSeconds int) time.Duration {
	if expireSeconds > 0 {
		return time.Duration(expireSeconds) * time.Second
	}
	if s.cfg.PresignExpireSeconds > 0 {
		return time.Duration(s.cfg.PresignExpireSeconds) * time.Second
	}
	return 15 * time.Minute
}

func (s *Service) GenerateUpload(ctx context.Context, params *s3up.GenerateParams, expireSeconds int) (*s3up.GenerateResult, time.Duration, error) {
	if !s.Enabled() {
		return nil, 0, ErrDisabled
	}

	expireIn := s.ResolveExpire(expireSeconds)
	params.ExpireIn = expireIn

	ret, err := s.client.GenerateUpload(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	return ret, expireIn, nil
}

func (s *Service) GenerateDownload(ctx context.Context, params *s3down.GenerateParams, expireSeconds int) (*url.URL, time.Duration, error) {
	if !s.Enabled() {
		return nil, 0, ErrDisabled
	}

	expireIn := s.ResolveExpire(expireSeconds)
	params.ExpireIn = expireIn

	ret, err := s.client.GenerateDownload(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	return ret, expireIn, nil
}

func (s *Service) Stat(ctx context.Context, remotePath string) (minio.ObjectInfo, error) {
	if !s.Enabled() {
		return minio.ObjectInfo{}, ErrDisabled
	}
	return s.client.Stat(ctx, remotePath)
}

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
