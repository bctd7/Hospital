package mysqlstore

import "testing"

func TestNewRequiresDataSource(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Fatal("expected empty data source to be rejected")
	}
}
