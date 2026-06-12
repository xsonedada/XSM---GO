package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Security SecurityConfig
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

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

type SecurityConfig struct {
	EncryptionKey      string
	RSAKeySize         int
	CORSAllowedOrigins []string
	RateLimitRequests  int
	RateLimitDuration  time.Duration
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	jwtExpiration, _ := time.ParseDuration(getEnv("JWT_EXPIRATION", "24h"))
	rsaKeySize, _ := strconv.Atoi(getEnv("RSA_KEY_SIZE", "4096"))
	rateLimitRequests, _ := strconv.Atoi(getEnv("RATE_LIMIT_REQUESTS", "100"))
	rateLimitDuration, _ := strconv.Atoi(getEnv("RATE_LIMIT_DURATION", "60"))

	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "1111"),
			DBName:   getEnv("DB_NAME", "xsm_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", ""),
			Expiration: jwtExpiration,
		},
		Security: SecurityConfig{
			EncryptionKey:     getEnv("ENCRYPTION_KEY", ""),
			RSAKeySize:        rsaKeySize,
			RateLimitRequests: rateLimitRequests,
			RateLimitDuration: time.Duration(rateLimitDuration) * time.Second,
		},
	}, nil
}

func (c *DatabaseConfig) DSN() string {
	return "host=" + c.Host +
		" port=" + c.Port +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.DBName +
		" sslmode=" + c.SSLMode
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
