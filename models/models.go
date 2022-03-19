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
