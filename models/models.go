package models

import (
	"github.com/uptrace/bun"
)

type Recipient struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	RecipientId     int64 `bun:",pk,autoincrement,unique"`
	RecipientName   string
	RecipientTGName string
	RecipientTGId   string
	RecipientRoom   string
}
