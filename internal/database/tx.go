package database

import (
	"context"

	"gorm.io/gorm"
)

// WithTx runs fn within a transaction, committing on nil error, rolling back otherwise.
func WithTx(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}
