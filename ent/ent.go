package ent

import (
	"database/sql"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/sony/sonyflake/v2"
)

type Config struct {
	Debug bool
}

func OpenDB(db *sql.DB) dialect.Driver {
	return entsql.OpenDB(dialect.Postgres, db)
}

func NewIDGenerator(settings sonyflake.Settings) (*sonyflake.Sonyflake, error) {
	return sonyflake.New(settings)
}
