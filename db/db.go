package db

import (
	"context"
	"database/sql"
	_ "embed"

	"github.com/pymq/tfahack/models"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
)

//go:embed init_struct.sql
var initStructQuery string

type DB struct {
	db *bun.DB
}

func NewDB() (*DB, error) {
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
		return nil, err
	}

	return &DB{db: db}, nil
}

func (db *DB) AddRecipient(recipient models.Recipient) error {
	_, err := db.db.NewInsert().Model(&recipient).Exec(context.Background())
	return err
}

func (db *DB) GetRecipients(tgIds []int64) ([]models.Recipient, error) {
	recipients := make([]models.Recipient, 1)
	err := db.db.NewSelect().Model(&recipients).Where("recipient.RecipientTGId in (?)", bun.In(tgIds)).Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return recipients, nil
}
