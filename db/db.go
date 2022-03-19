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

func (db *DB) GetRecipientsByIds(tgIds []int64) ([]models.Recipient, error) {
	recipients := make([]models.Recipient, 1)
	err := db.db.NewSelect().Model(&recipients).Where("recipient.RecipientTGId in (?)", bun.In(tgIds)).Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return recipients, nil
}

func (db *DB) GetRecipientsByTGNames(tgNames []string) ([]models.Recipient, error) {
	recipients := make([]models.Recipient, 1)
	err := db.db.NewSelect().Model(&recipients).Where("recipient.RecipientTGName in (?)", bun.In(tgNames)).Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return recipients, nil
}

func (db *DB) AddMailingList(mList models.MailingList, recipientsIds []int64) error {
	_, err := db.db.NewInsert().Model(&mList).Exec(context.Background())
	if err != nil {
		return err
	}

	mailingListRelations := make([]models.MailingListRelations, len(recipientsIds))
	for id, recipientsId := range recipientsIds {
		mailingListRelations[id].ListId = mList.ListId
		mailingListRelations[id].RecipientId = recipientsId
	}
	_, err = db.db.NewInsert().Model(&mailingListRelations).Exec(context.Background())
	return err
}

func (db *DB) GetMailingListBySender(senderTGId int64) ([]models.MailingList, error) {
	mList := make([]models.MailingList, 1)
	err := db.db.NewSelect().
		Model(&mList).
		Where("MailingList.SenderTGId = (?)", senderTGId).
		Scan(context.Background())
	return mList, err
}

func (db *DB) GetMailingListRecipientsById(listId int64) ([]models.Recipient, error) {
	recipients := make([]models.Recipient, 0)
	err := db.db.NewSelect().
		Model(&recipients).
		Join("RIGHT JOIN MailingListRelations ON mailingList.ListId = MailingListRelations.ListId").
		Join("RIGHT JOIN MailingList ON mailingList.ListId = MailingListRelations.ListId").
		Where("MailingList.ListId = (?)", listId).
		Scan(context.Background())

	return recipients, err
}

func (db *DB) AddTopic(topic models.Topic) error {
	_, err := db.db.NewInsert().Model(&topic).Exec(context.Background())
	return err
}

func (db *DB) GetUserTopicsBySender(senderTGId int64) ([]models.Topic, error) {
	topics := make([]models.Topic, 1)
	err := db.db.NewSelect().
		Model(&topics).
		Where("topic.SenderTGId = (?)", senderTGId).
		Scan(context.Background())
	return topics, err
}

func (db *DB) GetUserTopicById(topicId int64) (models.Topic, error) {
	topic := models.Topic{}
	err := db.db.NewSelect().
		Model(&topic).
		Where("topic.TopicId = (?)", topicId).
		Scan(context.Background())
	return topic, err
}
