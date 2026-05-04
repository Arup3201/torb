package core

import "gorm.io/gorm"

type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

func (m *TxManager) WithTx(fn func(tx *gorm.DB) error) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}
