package database_test

import (
	"testing"

	"github.com/Kash4299/todo-chat-app/pkg/database"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func TestNewTxManagerProvidesInterfaceForFx(t *testing.T) {
	app := fx.New(
		fx.NopLogger,
		fx.Provide(func() *gorm.DB { return &gorm.DB{} }),
		fx.Provide(database.NewTxManager),
		fx.Invoke(func(database.ITxManager) {}),
	)

	if err := app.Err(); err != nil {
		t.Fatalf("expected NewTxManager to provide database.ITxManager, got %v", err)
	}
}
