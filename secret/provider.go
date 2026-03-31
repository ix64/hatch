package secret

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewStoreProvider(lc fx.Lifecycle, logger *zap.Logger, cfg Config) (Store, error) {
	logger = logger.Named("init:secret_store")

	store, err := NewOpenBaoStore(OpenBaoOptions{
		Address:       cfg.Address,
		Token:         cfg.Token,
		Namespace:     cfg.Namespace,
		KVMount:       cfg.KVMount,
		Insecure:      cfg.Insecure,
		CACertFile:    cfg.CACertFile,
		CAPath:        cfg.CAPath,
		TLSServerName: cfg.TLSServerName,
		Timeout:       cfg.Timeout,
	})
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := store.Ping(checkCtx); err != nil {
				return fmt.Errorf("connect openbao failed: %w", err)
			}

			logger.Info("secret store connected",
				zap.String("provider", "openbao"),
				zap.String("address", cfg.Address),
				zap.String("kv_mount", cfg.KVMount),
			)
			return nil
		},
	})

	return store, nil
}
