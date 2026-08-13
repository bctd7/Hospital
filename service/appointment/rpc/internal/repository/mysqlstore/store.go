package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"hospital/service/appointment/rpc/internal/catalog"
)

// Store owns the Appointment database connection pool and will provide the
// MySQL adapter for Appointment domain stores.
type Store struct {
	db *sql.DB
}

func New(dataSource string) (*Store, error) {
	if dataSource == "" {
		return nil, errors.New("appointment mysql data source is required")
	}
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		return nil, fmt.Errorf("open appointment mysql: %w", err)
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping appointment mysql: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

const examinationItemSelect = `
SELECT id, owner_department_id, name, description, status, version, created_at, updated_at
FROM appointment_examination_items`

func (s *Store) GetItem(ctx context.Context, itemID string) (catalog.ExaminationItem, error) {
	item, err := scanExaminationItem(s.db.QueryRowContext(ctx, examinationItemSelect+" WHERE id = ?", itemID))
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.ExaminationItem{}, catalog.ErrNotFound
	}
	if err != nil {
		return catalog.ExaminationItem{}, fmt.Errorf("get examination item: %w", err)
	}
	return item, nil
}

func (s *Store) ListItems(ctx context.Context, filter catalog.ListFilter) ([]catalog.ExaminationItem, int64, error) {
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if filter.OwnerDepartmentID != "" {
		conditions = append(conditions, "owner_department_id = ?")
		args = append(args, filter.OwnerDepartmentID)
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM appointment_examination_items"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count examination items: %w", err)
	}

	queryArgs := append(append([]any(nil), args...), filter.Limit, filter.Offset)
	rows, err := s.db.QueryContext(ctx,
		examinationItemSelect+where+" ORDER BY name, id LIMIT ? OFFSET ?",
		queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list examination items: %w", err)
	}
	defer rows.Close()

	items := make([]catalog.ExaminationItem, 0)
	for rows.Next() {
		item, err := scanExaminationItem(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan examination item list: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate examination item list: %w", err)
	}
	return items, total, nil
}

func (s *Store) WithinCatalogTransaction(ctx context.Context, fn func(catalog.TxStore) error) error {
	if fn == nil {
		return errors.New("catalog transaction callback is required")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin catalog transaction: %w", err)
	}
	txStore := &catalogTxStore{tx: tx}
	if err := fn(txStore); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback catalog transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog transaction: %w", err)
	}
	return nil
}

var _ catalog.Store = (*Store)(nil)

type examinationItemScanner interface {
	Scan(dest ...any) error
}

func scanExaminationItem(scanner examinationItemScanner) (catalog.ExaminationItem, error) {
	var item catalog.ExaminationItem
	err := scanner.Scan(
		&item.ItemID,
		&item.OwnerDepartmentID,
		&item.Name,
		&item.Description,
		&item.Status,
		&item.Version,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}
