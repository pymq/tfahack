package models

import (
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
