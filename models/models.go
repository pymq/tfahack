package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Recipient struct {
	bun.BaseModel `bun:"table:Recipients,alias:recipient"`

	RecipientId     int64  `bun:"RecipientId,pk,autoincrement,unique"`
	RecipientName   string `bun:"RecipientName,notnull"`
	RecipientTGName string `bun:"RecipientTGName,notnull,unique"`
	RecipientTGId   int64  `bun:"RecipientTGId,notnull,unique"`
}

type MailingList struct {
	bun.BaseModel `bun:"table:MailingList,alias:mailingList"`

	ListId     int64  `bun:"ListId,pk,autoincrement,unique"`
	SenderTGId int64  `bun:"SenderTGId,notnull"`
	ListName   string `bun:"ListName,notnull"`
}

type MailingListRelations struct {
	bun.BaseModel `bun:"table:MailingListRelations,alias:mailingListRelations"`

	ListId      int64 `bun:"ListId"`
	RecipientId int64 `bun:"RecipientId"`
}

type Topic struct {
	bun.BaseModel `bun:"table:Topics,alias:topic"`

	TopicId    int64  `bun:"TopicId,pk,autoincrement,unique"`
	SenderTGId int64  `bun:"SenderTGId,notnull"`
	Topic      string `bun:"Topic,notnull"`
}

type Message struct {
	bun.BaseModel `bun:"table:Messages,alias:message"`

	MessageId          int64     `bun:"MessageId,pk,autoincrement,unique"`
	MessageTGId        int64     `bun:"MessageTGId,notnull"`
	SenderTGId         int64     `bun:"SenderTGId,notnull"`
	RecipientId        int64     `bun:"RecipientId,notnull"`
	TopicId            int64     `bun:"TopicId,notnull"`
	ListId             int64     `bun:"ListId,notnull"`
	SendDateTime       time.Time `bun:"SendDateTime,notnull"`
	Message            string    `bun:"Message,notnull"`
	React              string    `bun:"React"`
	Read               int64     `bun:"Read,notnull"`
	IsRecipientMessage int64     `bun:"IsRecipientMessage,notnull"`
}
