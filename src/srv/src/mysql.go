package main

import (
	"database/sql"
	"errors"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

// MysqlData store a single Mysql configuration
type MysqlData struct {
	DSN string  `json:"DSN"` // Mysql DSN "username:password@protocol(address)/dbname?param=value"
	db  *sql.DB // Mysql database handle
}

// initMysql initialize a new Mysql handle
func initMysql(cfg *MysqlData) (err error) {
	if cfg.DSN == "" {
		cfg.db = nil
		return nil
	}
	if cfg.db == nil {
		cfg.db, err = sql.Open("mysql", cfg.DSN)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("unable to open the database connection")
			return err
		}
	}
	// Open doesn't open a connection. Validate DSN data:
	err = cfg.db.Ping()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to connect to the specified database")
		return err
	}
	return nil
}

// Close the MySQL connection
func (md *MysqlData) Close() {
	if md.db == nil {
		return
	}
	err := md.db.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to Close the database connection")
	}
}

// isMysqlAlive returns the status of Mysql
func isMysqlAlive() error {
	if appParams.mysql.db == nil {
		return errors.New("the database is not available")
	}
	_, err := appParams.mysql.db.Exec("SELECT 1")
	return err
}

// stmtClose closes the prepared database statement and report errors if any
func stmtClose(stmt *sql.Stmt) {
	if stmt == nil {
		return
	}
	err := stmt.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to close the database statement")
	}
}
