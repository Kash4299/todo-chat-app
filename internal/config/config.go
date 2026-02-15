package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	GRPC     GRPCConfig
}

type GRPCConfig struct {
	Enabled bool
	Port    string
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Enabled  bool
	Host     string
	Port     string
	Password string
	DB       int
}

type KafkaConfig struct {
	Enabled bool
	Brokers []string
	Topic   string
}

func NewConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Host: getEnv("SERVER_HOST", "localhost"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USERNAME", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			DBName:   getEnv("DB_NAME", "postgres"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Enabled:  getEnvBool("ENABLE_REDIS", false),
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Kafka: KafkaConfig{
			Enabled: getEnvBool("ENABLE_KAFKA", false),
			Brokers: getEnvSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			Topic:   getEnv("KAFKA_TOPIC", "todo-events"),
		},
		GRPC: GRPCConfig{
			Enabled: getEnvBool("ENABLE_GRPC", true),
			Port:    getEnv("GRPC_PORT", "50051"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated values
		// For more complex parsing, you might want to use a library
		return []string{value}
	}
	return defaultValue
}
