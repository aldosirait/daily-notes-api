package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	ServerPort     string
	JWTSecret      string
	JWTExpiryHours int

	// Rate limiting configuration
	AuthRateLimit     int           // requests per window
	AuthRateWindow    time.Duration // time window
	AuthRateCleanup   time.Duration // cleanup interval
	GeneralRateLimit  int           // general API rate limit
	GeneralRateWindow time.Duration // general API time window

	// SMTP Email configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	AppName      string
	AppURL       string

	// Redis Cache configuration
	RedisHost       string
	RedisPort       string
	RedisPassword   string
	RedisDB         int
	CacheEnabled    bool
	CacheTTLMinutes int
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	jwtExpiryHours, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))

	// Rate limiting configuration
	authRateLimit, _ := strconv.Atoi(getEnv("AUTH_RATE_LIMIT", "5"))
	authRateWindowMinutes, _ := strconv.Atoi(getEnv("AUTH_RATE_WINDOW_MINUTES", "15"))
	authRateCleanupMinutes, _ := strconv.Atoi(getEnv("AUTH_RATE_CLEANUP_MINUTES", "30"))

	generalRateLimit, _ := strconv.Atoi(getEnv("GENERAL_RATE_LIMIT", "100"))
	generalRateWindowMinutes, _ := strconv.Atoi(getEnv("GENERAL_RATE_WINDOW_MINUTES", "1"))

	// SMTP configuration
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))

	// Redis configuration
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	cacheEnabled, _ := strconv.ParseBool(getEnv("CACHE_ENABLED", "true"))
	cacheTTLMinutes, _ := strconv.Atoi(getEnv("CACHE_TTL_MINUTES", "30"))

	return &Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "3306"),
		DBUser:         getEnv("DB_USER", "root"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", "daily_notes"),
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		JWTSecret:      getEnv("JWT_SECRET", "default-secret-change-this"),
		JWTExpiryHours: jwtExpiryHours,

		// Rate limiting
		AuthRateLimit:     authRateLimit,
		AuthRateWindow:    time.Duration(authRateWindowMinutes) * time.Minute,
		AuthRateCleanup:   time.Duration(authRateCleanupMinutes) * time.Minute,
		GeneralRateLimit:  generalRateLimit,
		GeneralRateWindow: time.Duration(generalRateWindowMinutes) * time.Minute,

		// SMTP Email settings
		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     smtpPort,
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", ""),
		AppName:      getEnv("APP_NAME", "Daily Notes"),
		AppURL:       getEnv("APP_URL", "http://localhost:3000"),

		// Redis Cache settings
		RedisHost:       getEnv("REDIS_HOST", "localhost"),
		RedisPort:       getEnv("REDIS_PORT", "6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:         redisDB,
		CacheEnabled:    cacheEnabled,
		CacheTTLMinutes: cacheTTLMinutes,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
