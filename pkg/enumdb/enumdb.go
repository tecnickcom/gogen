/*
Package enumdb allows loading enumeration sets
(github.com/Vonage/gosrvlib/pkg/enumcache) from multiple database tables.

Each enumeration has a numerical ID ("id" on the database table as the primary
key) and a string name ("name" on the database table as a unique string).

Example of a MySQL database table that can be used with this package:

	CREATE TABLE IF NOT EXISTS `example` (
	  `id` SMALLINT UNSIGNED NOT NULL,
	  `name` VARCHAR(50) NOT NULL,
	  `disabled` TINYINT NOT NULL DEFAULT 0,
	  PRIMARY KEY (`id`),
	  UNIQUE INDEX `id_UNIQUE` (`id` ASC),
	  UNIQUE INDEX `name_UNIQUE` (`name` ASC))
	ENGINE = InnoDB
	COMMENT = 'Example enumeration table';
*/
package enumdb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/enumcache"
	"github.com/Vonage/gosrvlib/pkg/logging"
)

// EnumDB maps each enumeration table name with the corresponding enumeration cache.
type EnumDB map[string]*enumcache.EnumCache

// EnumTableQuery maps each enumeration table name with the SQL query string used to read the data.
type EnumTableQuery map[string]string

// New returns a new enumeration cache for all the tables listed in the queries map.
func New(ctx context.Context, db *sql.DB, queries EnumTableQuery) (EnumDB, error) {
	enum := make(EnumDB, len(queries))

	for table, query := range queries {
		cache, err := loadTableEnumCache(ctx, db, query)
		if err != nil {
			return nil, fmt.Errorf("failed to load the enumeration table '%s': %w", table, err)
		}

		enum[table] = cache
	}

	return enum, nil
}

// loadTableEnumCache load the cache using the specified SQL query.
//
//nolint:interfacer
func loadTableEnumCache(ctx context.Context, db *sql.DB, query string) (*enumcache.EnumCache, error) {
	stmt, err := db.PrepareContext(ctx, query) //nolint:sqlclosecheck
	if err != nil {
		return nil, fmt.Errorf("failed preparing statement: %w", err)
	}

	defer logging.Close(ctx, stmt, "error closing statement")

	rows, err := stmt.QueryContext(ctx) //nolint:sqlclosecheck
	if err != nil {
		return nil, fmt.Errorf("failed executing query: %w", err)
	}

	defer logging.Close(ctx, rows, "error closing query")

	cache := enumcache.New()

	var (
		id   int
		name string
	)

	for rows.Next() {
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}

		cache.Set(id, name)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed reading the rows: %w", err)
	}

	return cache, nil
}
