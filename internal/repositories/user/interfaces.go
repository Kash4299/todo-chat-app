package user

import (
	"context"
	"todo/internal/model"
)

// IUserRepository defines the contract for user data access
type IUserRepository interface {
	CreateUser(ctx context.Context, in *model.User) error
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
}
