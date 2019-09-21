package main

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestInitMysqlNil(t *testing.T) {
	cfg := &MysqlData{}
	err := initMysql(cfg)
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected while initializing Mysql with empty DSN: %v", err))
	}
}

func TestInitMysqlBadDSN(t *testing.T) {
	cfg := &MysqlData{
		DSN: ":",
	}
	err := initMysql(cfg)
	if err == nil {
		t.Error(fmt.Errorf("An error  not expected while initializing Mysql with bad DSN"))
	}
}

func TestMysqlDataCloseNil(t *testing.T) {
	cfg := &MysqlData{}
	cfg.Close()
}

func TestMysqlDataCloseErr(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Error(fmt.Errorf("An error '%s' was not expected when opening a stub database connection", err))
	}
	defer func() {
		err = db.Close()
		if err != nil {
			t.Error(fmt.Errorf("An error was not expected when closing the db stub: %v", err))
		}
	}()
	cfg := &MysqlData{
		DSN: "test",
		db:  db,
	}
	cfg.Close()
}

func TestIsMysqlAliveNil(t *testing.T) {
	err := isMysqlAlive()
	if err == nil {
		t.Error(fmt.Errorf("An error was expected while checking if DB is alive"))
	}
}

func TestIsMysqlAlive(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected when opening a stub database connection: %v", err))
	}
	defer func() {
		err = db.Close()
		if err == nil {
			t.Error(fmt.Errorf("An error was expected when closing the db stub"))
		}
	}()
	appParams.mysql.db = db
	err = isMysqlAlive()
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected when checking if DB is alive: %v", err))
	}
}

func TestStmtCloseNil(t *testing.T) {
	stmtClose(nil)
}
