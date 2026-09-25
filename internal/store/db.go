package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"

	"notifications/internal/config"
	"notifications/migrations"
)

const (
	driverName = "pgx"
	dialect    = "postgres"
)

type DB struct {
	db   *sqlx.DB
	pool *pgxpool.Pool
}

func New(ctx context.Context, cfg config.Database) (*DB, error) {
	poolCfg, err := poolConfig(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}

	return &DB{db: sqlx.NewDb(stdlib.OpenDBFromPool(pool), driverName), pool: pool}, nil
}

func (s *DB) DB() *sqlx.DB {
	return s.db
}

func (s *DB) Close() {
	_ = s.db.Close()
	s.pool.Close()
}

func poolConfig(cfg config.Database) (*pgxpool.Config, error) {
	poolCfg, err := pgxpool.ParseConfig("sslmode=" + cfg.SSLMode)
	if err != nil {
		return nil, fmt.Errorf("DB_SSLMODE=%q is not a valid sslmode: %w", cfg.SSLMode, err)
	}

	poolCfg.ConnConfig.Host = cfg.Host
	poolCfg.ConnConfig.Port = cfg.Port
	poolCfg.ConnConfig.User = cfg.User
	poolCfg.ConnConfig.Password = cfg.Password
	poolCfg.ConnConfig.Database = cfg.Name
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	return poolCfg, nil
}

func (s *DB) Migrate(ctx context.Context) error {
	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.UpContext(ctx, s.db.DB, "."); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
