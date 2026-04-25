package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort string

	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime int

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	KafkaBrokers string

	Auth0Domain   string
	Auth0Audience string
	WebhookSecret string

	JWTSecret           string
	JWTAccessExpiryMin  int
	JWTRefreshExpiryDay int

	AllowedOrigins string
}

func NewConfig() *Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),

		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USERNAME", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", "postgres"),
		DBName:            getEnv("DB_DATABASE", "todo_chat_app"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		DBMaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 50),
		DBMaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME_SECONDS", 1800),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,

		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),

		Auth0Domain:   getEnv("AUTH0_DOMAIN", ""),
		Auth0Audience: getEnv("AUTH0_AUDIENCE", ""),
		WebhookSecret: getEnv("WEBHOOK_SECRET", ""),

		JWTSecret:           getEnv("JWT_SECRET", ""),
		JWTAccessExpiryMin:  getEnvAsInt("JWT_ACCESS_EXPIRY_MIN", 15),
		JWTRefreshExpiryDay: getEnvAsInt("JWT_REFRESH_EXPIRY_DAY", 7),

		AllowedOrigins: getEnv("ALLOWED_ORIGINS", ""),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}
