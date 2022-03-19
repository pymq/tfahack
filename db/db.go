package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
)

//go:embed init_struct.sql
var initStructQuery string

type DB struct {
	db bun.DB
}

func NewDB() {
	sqldb, err := sql.Open(sqliteshim.ShimName, "./sqlite.db")
	if err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())

	// log
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))

	_, err = db.Exec(initStructQuery)
	if err != nil {
		panic(fmt.Sprintf("db struct init error: %v", err))
	}
}
