package database

import "gorm.io/gorm"

type ITxManager interface {
	RunInTx(fn func(tx *gorm.DB) error) error
}

type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) ITxManager {
	return &TxManager{db: db}
}

func (m *TxManager) RunInTx(fn func(tx *gorm.DB) error) error {
	return m.db.Transaction(fn)
}
