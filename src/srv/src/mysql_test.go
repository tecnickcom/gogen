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
		t.Error(fmt.Errorf("An error was expected while initializing Mysql with bad DSN"))
	}
}

func TestInitMysqlNoPing(t *testing.T) {
	cfg := &MysqlData{
		DSN: "user:pwd@tcp(localhost:12345)/table",
	}
	err := initMysql(cfg)
	if err == nil {
		t.Error(fmt.Errorf("An error was expected while initializing Mysql with no ping"))
	}
}

func TestInitMysql(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Error(fmt.Errorf("An error '%s' was not expected when opening a stub database connection", err))
	}
	defer func() {
		_ = db.Close()
	}()
	cfg := &MysqlData{
		DSN: "user:pwd@tcp(localhost:12345)/table",
		db:  db,
	}
	err = initMysql(cfg)
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected while initializing Mysql: %v", err))
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
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected when opening a stub database connection: %v", err))
	}
	defer func() {
		err = db.Close()
		if err == nil {
			t.Error(fmt.Errorf("An error was expected when closing the db stub"))
		}
	}()
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewErrorResult(nil))
	appParams.mysql.db = db
	err = isMysqlAlive()
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected when checking if DB is alive: %v", err))
	}
}

func TestStmtCloseNil(t *testing.T) {
	stmtClose(nil)
}

func TestStmtClose(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(fmt.Errorf("An error '%s' was not expected when opening a stub database connection", err))
	}
	defer func() {
		_ = db.Close()
	}()
	mock.ExpectBegin()
	mock.ExpectPrepare("SELECT").WillReturnCloseError(fmt.Errorf("stmt close error"))
	txn, err := db.Begin()
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected while opening transaction: %v", err))
	}
	stmt, err := txn.Prepare("SELECT")
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected when preparing sql statement: %v", err))
	}
	stmtClose(stmt)
}
