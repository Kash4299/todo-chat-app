package user

import (
	"context"
	"todo/internal/database"
	"todo/internal/model"
)

type UserRepository struct {
	db *database.Database
}

func NewUserRepository(db *database.Database) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, in *model.User) error {
	dbContext := r.db.DB.WithContext(ctx)
	//TODO : update clause on conflict do nothing to avoid create same account
	return dbContext.Model(model.User{}).Create(in).Error
}
