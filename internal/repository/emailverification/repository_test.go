package emailverification

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}

	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		t.Fatalf("create extension: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.EmailVerificationToken{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE email_verification_tokens, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	return db
}

func createUser(t *testing.T, db *gorm.DB, email string) model.User {
	t.Helper()

	u := model.User{
		ID:            uuid.New(),
		Email:         email,
		DisplayName:   "test",
		EmailVerified: false,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func TestEmailVerificationRepository_CreateAndFindByHash(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmailVerificationRepository(db)
	user := createUser(t, db, "repo-find@example.com")

	token := &model.EmailVerificationToken{
		UserID:    user.ID,
		TokenHash: "hash-find",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := repo.Create(token); err != nil {
		t.Fatalf("create token: %v", err)
	}

	got, err := repo.FindByHash("hash-find")
	if err != nil {
		t.Fatalf("find token: %v", err)
	}
	if got.UserID != user.ID {
		t.Fatalf("expected user_id %s, got %s", user.ID, got.UserID)
	}
}

func TestEmailVerificationRepository_DeleteByHash(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmailVerificationRepository(db)
	user := createUser(t, db, "repo-delete-hash@example.com")

	token := &model.EmailVerificationToken{
		UserID:    user.ID,
		TokenHash: "hash-delete",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := repo.Create(token); err != nil {
		t.Fatalf("create token: %v", err)
	}

	if err := repo.DeleteByHash("hash-delete"); err != nil {
		t.Fatalf("delete by hash: %v", err)
	}

	if _, err := repo.FindByHash("hash-delete"); err == nil {
		t.Fatal("expected token to be deleted")
	}
}

func TestEmailVerificationRepository_DeleteByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmailVerificationRepository(db)
	user := createUser(t, db, "repo-delete-user@example.com")

	token := &model.EmailVerificationToken{
		UserID:    user.ID,
		TokenHash: "hash-user-delete",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := repo.Create(token); err != nil {
		t.Fatalf("create token: %v", err)
	}

	if err := repo.DeleteByUserID(user.ID); err != nil {
		t.Fatalf("delete by user id: %v", err)
	}

	if _, err := repo.FindByHash("hash-user-delete"); err == nil {
		t.Fatal("expected token to be deleted")
	}
}

func TestEmailVerificationRepository_FindLatestByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmailVerificationRepository(db)
	user := createUser(t, db, "repo-latest-user@example.com")

	oldToken := &model.EmailVerificationToken{
		UserID:    user.ID,
		TokenHash: "hash-old",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := repo.Create(oldToken); err != nil {
		t.Fatalf("create old token: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	newToken := &model.EmailVerificationToken{
		UserID:    user.ID,
		TokenHash: "hash-new",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := repo.Create(newToken); err != nil {
		t.Fatalf("create new token: %v", err)
	}

	got, err := repo.FindLatestByUserID(user.ID)
	if err != nil {
		t.Fatalf("find latest by user id: %v", err)
	}
	if got.TokenHash != "hash-new" {
		t.Fatalf("expected latest token hash-new, got %s", got.TokenHash)
	}
}

func TestEmailVerificationRepository_ConsumeValidByHashDeletesToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmailVerificationRepository(db)
	user := createUser(t, db, "repo-consume@example.com")

	token := &model.EmailVerificationToken{
		UserID:    user.ID,
		TokenHash: "hash-consume",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := repo.Create(token); err != nil {
		t.Fatalf("create token: %v", err)
	}

	userID, err := repo.ConsumeValidByHash("hash-consume", time.Now())
	if err != nil {
		t.Fatalf("consume valid token: %v", err)
	}
	if userID != user.ID {
		t.Fatalf("expected user_id %s, got %s", user.ID, userID)
	}

	_, err = repo.ConsumeValidByHash("hash-consume", time.Now())
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected consumed token to be gone, got %v", err)
	}
}
