package sql

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"time"

	"ariga.io/atlas-go-sdk/atlasexec"
	"github.com/XSAM/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	"go.uber.org/fx"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	DSN          string
	MaxLifetime  int
	MaxOpen      int
	MaxIdle      int
	MigrationDSN string
	Migrate      bool
}

func NewDB(lc fx.Lifecycle, logger *zap.Logger, cfg Config) (*sql.DB, error) {
	logger = logger.Named("init:postgres")

	db, err := otelsql.Open("pgx", cfg.DSN, otelsql.WithAttributes(semconv.DBSystemNamePostgreSQL))
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime))
	db.SetMaxOpenConns(cfg.MaxOpen)
	db.SetMaxIdleConns(cfg.MaxIdle)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := db.PingContext(ctx); err != nil {
				return fmt.Errorf("ping db failed: %w", err)
			}
			logger.Info("postgres connected")
			return nil
		},
		OnStop: func(context.Context) error {
			return db.Close()
		},
	})

	return db, nil
}

func ApplyMigrations(ctx context.Context, logger *zap.Logger, dsn string, migrations fs.FS) error {
	wd, err := atlasexec.NewWorkingDir(atlasexec.WithMigrations(migrations))
	if err != nil {
		return fmt.Errorf("failed to load migrations directory: %w", err)
	}
	defer wd.Close()

	c, err := atlasexec.NewClient(wd.Path(), "atlas")
	if err != nil {
		return fmt.Errorf("failed to create atlas client: %w", err)
	}

	ret, err := c.MigrateApply(ctx, &atlasexec.MigrateApplyParams{URL: dsn})
	if err != nil {
		return fmt.Errorf("failed to run migration: %w", err)
	}

	for _, v := range ret.Applied {
		logger.Info("migration applied", zap.String("version", v.Version), zap.String("description", v.Description))
	}
	return nil
}

func AssertMigrations(ctx context.Context, logger *zap.Logger, dsn string, migrations fs.FS) (bool, error) {
	wd, err := atlasexec.NewWorkingDir(atlasexec.WithMigrations(migrations))
	if err != nil {
		return false, fmt.Errorf("failed to load migrations directory: %w", err)
	}
	defer wd.Close()

	c, err := atlasexec.NewClient(wd.Path(), "atlas")
	if err != nil {
		return false, fmt.Errorf("failed to create atlas client: %w", err)
	}

	ret, err := c.MigrateStatus(ctx, &atlasexec.MigrateStatusParams{URL: dsn})
	if err != nil {
		return false, fmt.Errorf("failed to get migration status: %w", err)
	}

	for _, v := range ret.Pending {
		logger.Info("missing migration", zap.String("version", v.Version), zap.String("description", v.Description))
	}
	return len(ret.Pending) == 0, nil
}

func RunMigrations(lc fx.Lifecycle, cfg Config, logger *zap.Logger, migrations fs.FS) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			dsn := cfg.MigrationDSN
			if dsn == "" {
				dsn = cfg.DSN
			}

			if cfg.Migrate {
				if err := ApplyMigrations(ctx, logger, dsn, migrations); err != nil {
					return fmt.Errorf("apply migration failed: %w", err)
				}
			} else {
				if ok, err := AssertMigrations(ctx, logger, dsn, migrations); err != nil {
					return fmt.Errorf("check migration failed: %w", err)
				} else if !ok {
					return fmt.Errorf("manually version migration required")
				}
			}
			logger.Info("version migration up to date")
			return nil
		},
	})
}
