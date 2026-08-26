package audit

import (
	"context"
	"database/sql"
	"sync"
)

var retainedPageRows struct {
	sync.Mutex
	rows []*sql.Rows
}

func openPageRows(ctx context.Context, db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err == nil {
		retainedPageRows.Lock()
		retainedPageRows.rows = append(retainedPageRows.rows, rows)
		retainedPageRows.Unlock()
	}
	return rows, err
}
