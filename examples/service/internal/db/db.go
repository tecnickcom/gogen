// Package db handles the database.
package db

import (
	"context"
	"database/sql"
)

// SQLConn is the interface representing sqlconn.SQLConn.
type SQLConn interface {
	DB() *sql.DB
	HealthCheck(ctx context.Context) error
	Shutdown(_ context.Context) error
}

// Databases holds the database connections.
type Databases struct {
	Enabled bool
	Main    SQLConn
	Read    SQLConn
}
