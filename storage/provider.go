package storage

import (
	"context"
	"fmt"

	"github.com/ix64/s3-go/s3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewServiceProvider(lc fx.Lifecycle, logger *zap.Logger, cfg Config) (*Service, error) {
	logger = logger.Named("init:object_storage")

	if !cfg.Enabled {
		lc.Append(fx.Hook{
			OnStart: func(context.Context) error {
				logger.Info("object storage disabled")
				return nil
			},
		})
		return NewDisabledService(cfg), nil
	}

	client, err := s3.NewClient(&s3.Config{
		Endpoint:     cfg.Endpoint,
		Bucket:       cfg.Bucket,
		BucketLookup: cfg.BucketLookup,
		Prefix:       cfg.Prefix,
		Region:       cfg.Region,
		AccessKey:    cfg.AccessKey,
		SecretKey:    cfg.SecretKey,
	})
	if err != nil {
		return nil, fmt.Errorf("create object storage client failed: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			logger.Info("object storage connected",
				zap.String("endpoint", cfg.Endpoint),
				zap.String("bucket", cfg.Bucket),
				zap.String("prefix", cfg.Prefix),
			)
			return nil
		},
	})

	return NewService(client, cfg), nil
}
