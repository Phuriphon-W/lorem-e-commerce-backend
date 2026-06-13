package database

import (
	"context"

	"gorm.io/gorm"
)

type txCtxKey struct{}

// WithTransaction returns a new context with the provided gorm.DB transaction.
func WithTransaction(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// GetDB returns the transaction if present in context, or the default DB instance.
func GetDB(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txCtxKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return defaultDB.WithContext(ctx)
}
