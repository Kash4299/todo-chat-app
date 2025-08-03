package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"todo/internal/kafka"
	"todo/internal/redis"
)

// Todo represents a todo item for the service layer
type Todo struct {
	ID          uint   `json:"id"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

type TodoService struct {
	hub         *Hub
	redisClient *redis.Client
	kafkaClient *kafka.Client
}

func NewTodoService(h *Hub, redisClient *redis.Client, kafkaClient *kafka.Client) *TodoService {
	return &TodoService{
		hub:         h,
		redisClient: redisClient,
		kafkaClient: kafkaClient,
	}
}

// GetAllTodos retrieves all todos with Redis caching
func (s *TodoService) GetAllTodos() ([]Todo, error) {
	ctx := context.Background()

	// Try to get from cache first
	if s.redisClient.IsEnabled() {
		cached, err := s.redisClient.Get(ctx, "todos:all")
		if err == nil && cached != "" {
			var todos []Todo
			if err := json.Unmarshal([]byte(cached), &todos); err == nil {
				return todos, nil
			}
		}
	}

	// TODO: Implement actual database call through repository
	// For now, return mock data
	todos := []Todo{
		{ID: 1, Title: "Learn Go", Description: "Study Go programming language", Completed: false},
		{ID: 2, Title: "Build API", Description: "Create REST API with Gin", Completed: true},
	}

	// Cache the result
	if s.redisClient.IsEnabled() {
		if data, err := json.Marshal(todos); err == nil {
			s.redisClient.Set(ctx, "todos:all", string(data), 5*time.Minute)
		}
	}

	// Publish event to Kafka
	if s.kafkaClient.IsEnabled() {
		event := map[string]interface{}{
			"event": "todos_retrieved",
			"count": len(todos),
			"time":  time.Now().UTC(),
		}
		if eventData, err := json.Marshal(event); err == nil {
			s.kafkaClient.SendMessage(ctx, "todos_retrieved", string(eventData))
		}
	}

	return todos, nil
}

// GetTodoByID retrieves a todo by ID with Redis caching
func (s *TodoService) GetTodoByID(id uint) (*Todo, error) {
	ctx := context.Background()

	if id == 0 {
		return nil, errors.New("invalid todo ID")
	}

	// Try to get from cache first
	cacheKey := fmt.Sprintf("todo:%d", id)
	if s.redisClient.IsEnabled() {
		cached, err := s.redisClient.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			var todo Todo
			if err := json.Unmarshal([]byte(cached), &todo); err == nil {
				return &todo, nil
			}
		}
	}

	// TODO: Implement actual database call through repository
	// For now, return mock data
	todo := &Todo{
		ID:          id,
		Title:       "Sample Todo",
		Description: "This is a sample todo item",
		Completed:   false,
	}

	// Cache the result
	if s.redisClient.IsEnabled() {
		if data, err := json.Marshal(todo); err == nil {
			s.redisClient.Set(ctx, cacheKey, string(data), 10*time.Minute)
		}
	}

	// Publish event to Kafka
	if s.kafkaClient.IsEnabled() {
		event := map[string]interface{}{
			"event":   "todo_retrieved",
			"todo_id": id,
			"time":    time.Now().UTC(),
		}
		if eventData, err := json.Marshal(event); err == nil {
			s.kafkaClient.SendMessage(ctx, "todo_retrieved", string(eventData))
		}
	}

	return todo, nil
}

// CreateTodo creates a new todo
func (s *TodoService) CreateTodo(todo *Todo) error {
	ctx := context.Background()

	// TODO: Implement actual database call through repository
	// For now, just assign a mock ID
	todo.ID = 1

	// Invalidate cache
	if s.redisClient.IsEnabled() {
		s.redisClient.Del(ctx, "todos:all")
	}

	// Publish event to Kafka
	if s.kafkaClient.IsEnabled() {
		event := map[string]interface{}{
			"event": "todo_created",
			"todo":  todo,
			"time":  time.Now().UTC(),
		}
		if eventData, err := json.Marshal(event); err == nil {
			s.kafkaClient.SendMessage(ctx, "todo_created", string(eventData))
		}
	}

	return nil
}

// UpdateTodo updates an existing todo
func (s *TodoService) UpdateTodo(todo *Todo) error {
	ctx := context.Background()

	if todo.ID == 0 {
		return errors.New("invalid todo ID")
	}

	// TODO: Implement actual database call through repository

	// Invalidate cache
	if s.redisClient.IsEnabled() {
		s.redisClient.Del(ctx, "todos:all", fmt.Sprintf("todo:%d", todo.ID))
	}

	// Publish event to Kafka
	if s.kafkaClient.IsEnabled() {
		event := map[string]interface{}{
			"event": "todo_updated",
			"todo":  todo,
			"time":  time.Now().UTC(),
		}
		if eventData, err := json.Marshal(event); err == nil {
			s.kafkaClient.SendMessage(ctx, "todo_updated", string(eventData))
		}
	}

	return nil
}

// DeleteTodo deletes a todo by ID
func (s *TodoService) DeleteTodo(id uint) error {
	ctx := context.Background()

	if id == 0 {
		return errors.New("invalid todo ID")
	}

	// TODO: Implement actual database call through repository

	// Invalidate cache
	if s.redisClient.IsEnabled() {
		s.redisClient.Del(ctx, "todos:all", fmt.Sprintf("todo:%d", id))
	}

	// Publish event to Kafka
	if s.kafkaClient.IsEnabled() {
		event := map[string]interface{}{
			"event":   "todo_deleted",
			"todo_id": id,
			"time":    time.Now().UTC(),
		}
		if eventData, err := json.Marshal(event); err == nil {
			s.kafkaClient.SendMessage(ctx, "todo_deleted", string(eventData))
		}
	}

	return nil
}
