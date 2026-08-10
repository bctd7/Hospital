package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Store owns the shared Identity database connection pool. Domain-specific
// Store implementations are grouped in mysql_<domain>_store.go files.
type Store struct {
	db *sql.DB
}

func New(dataSource string) (*Store, error) {
	if dataSource == "" {
		return nil, errors.New("identity mysql data source is required")
	}
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		return nil, fmt.Errorf("open identity mysql: %w", err)
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping identity mysql: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
