package postrgesql

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct {}

type TransactionManager struct {
	*Storage
}

func NewTransactionManager(st *Storage) *TransactionManager {
	return &TransactionManager{Storage: st}
}

func (tm *TransactionManager) WithTransaction(ctx context.Context, tFunc func(ctx context.Context) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return tFunc(txCtx)
	})
}
