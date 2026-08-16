// Package mysqlstore 实现 Guidance 业务包声明的 MySQL 持久化端口。
package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"hospital/service/guidance/rpc/internal/rules/precedence"
)

type Store struct {
	db *sql.DB
}

func New(dataSource string) (*Store, error) {
	if dataSource == "" {
		return nil, errors.New("guidance mysql data source is required")
	}
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		return nil, fmt.Errorf("open guidance mysql: %w", err)
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping guidance mysql: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) ListAll(ctx context.Context) ([]precedence.Rule, error) {
	return listRules(ctx, s.db)
}

func (s *Store) WithinWriteTransaction(ctx context.Context, fn func(precedence.TxStore) error) error {
	if fn == nil {
		return errors.New("precedence transaction callback is required")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin precedence transaction: %w", err)
	}
	if err = fn(&txStore{tx: tx}); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback precedence transaction: %w", rollbackErr))
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit precedence transaction: %w", err)
	}
	return nil
}

var _ precedence.Store = (*Store)(nil)
