package db

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
)

func TestOpenGormWithDialector_Success(t *testing.T) {
	sqlDB, mock, err := sqlmock.New() // fake *sql.DB
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer sqlDB.Close()

	// Expect a Ping from our code
	mock.ExpectPing()

	// Build a mysql dialector that uses our mocked *sql.DB
	dial := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true, // don't query @@version
	})

	gdb, err := OpenGormWithDialector(dial)
	if err != nil {
		t.Fatalf("OpenGormWithDialector error: %v", err)
	}
	if gdb == nil {
		t.Fatalf("got nil gorm.DB")
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestOpenGormWithDialector_PingFails(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer sqlDB.Close()

	mock.ExpectPing().WillReturnError(errors.New("no ping"))

	dial := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})

	gdb, err := OpenGormWithDialector(dial)
	if err == nil {
		t.Fatalf("expected error, got nil (gdb=%v)", gdb)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
