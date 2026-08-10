package mysqlstore

import (
	"errors"

	mysql "github.com/go-sql-driver/mysql"
)

func isDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
