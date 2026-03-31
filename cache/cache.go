package cache

import (
	"context"
	"fmt"

	"github.com/valkey-io/valkey-go"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	URL string
}

func NewValkeyClient(lc fx.Lifecycle, logger *zap.Logger, cfg Config) (valkey.Client, error) {
	logger = logger.Named("init:valkey")

	opt, err := valkey.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse valkey url failed: %w", err)
	}

	client, err := valkey.NewClient(opt)
	if err != nil {
		return nil, fmt.Errorf("create valkey client failed: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
				client.Close()
				return fmt.Errorf("ping valkey failed: %w", err)
			}

			logger.Info("valkey connected", zap.Strings("address", opt.InitAddress))
			return nil
		},
		OnStop: func(context.Context) error {
			client.Close()
			return nil
		},
	})

	return client, nil
}
