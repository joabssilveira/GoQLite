package goqlitegorm

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// DB TYPES

// JSONB is a generic wrapper for saving any struct/slice/map as jsonb in Postgres.
type JSONB[T any] struct {
	Data T
}

//
// =======================
// DATABASE (GORM / SQL)
// =======================
//

// Value converts to JSON before saving to the database
func (j JSONB[T]) Value() (driver.Value, error) {
	return json.Marshal(j.Data)
}

func (JSONB[T]) GormDataType() string {
	return "jsonb"
}

func (JSONB[T]) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return "jsonb"
}

// Scan converts JSON from the database back to the Go type.
func (j *JSONB[T]) Scan(value interface{}) error {
	if value == nil {
		var empty T
		j.Data = empty
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("dbtypes.JSONB: invalid type Scan")
	}

	return json.Unmarshal(bytes, &j.Data)
}

//
// =======================
// API (JSON HTTP)
// =======================
//

// MarshalJSON makes the API return only the content, without "Data".
func (j JSONB[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Data)
}

// UnmarshalJSON allows you to receive JSON directly in the field.
func (j *JSONB[T]) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &j.Data)
}
