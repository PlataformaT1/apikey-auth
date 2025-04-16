package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var (
	redisClient     *redis.Client
	redisClientOnce sync.Once
	logger          *logrus.Logger
)

// InitLogger inicializa el logger para el paquete Redis
func InitLogger(l *logrus.Logger) {
	if l != nil {
		logger = l
	} else {
		// Fallback a un logger básico si no se proporciona uno
		logger = logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{})
	}
}

// GetClient returns a singleton Redis client
func GetClient() *redis.Client {
	redisClientOnce.Do(func() {

		// Si no se ha inicializado el logger y no estamos usando el logger global
		if logger == nil {
			// Usamos el logger global o creamos uno básico
			defaultLogger := logrus.New()
			defaultLogger.SetFormatter(&logrus.JSONFormatter{})
			logger = defaultLogger
		}

		// Get Redis connection details from environment variables
		redisHost := os.Getenv("USER_VAR_REDIS_HOST")
		if redisHost == "" {
			redisHost = "localhost:6379" // Default for local development
			if logger != nil {
				logger.WithField("redisHost", redisHost).Warn("No se ha configurado USER_VAR_REDIS_HOST, usando valor por defecto")
			} else {
				logger.Printf("No se ha configurado USER_VAR_REDIS_HOST, usando valor por defecto: %s", redisHost)
			}
		}

		// Configurar opciones de Redis con TLS
		opts := &redis.Options{
			Addr:      redisHost,
			Password:  "",            // Si tienes contraseña
			DB:        0,             // DB por defecto
			TLSConfig: &tls.Config{}, // Habilitar TLS
		}

		// Create Redis client
		redisClient = redis.NewClient(opts)

		// Test the connection
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		_, err := redisClient.Ping(ctx).Result()
		if err != nil {
			if logger != nil {
				logger.WithFields(logrus.Fields{
					"redisHost": redisHost,
					"error":     err,
				}).Error("Error al conectar con Redis")
			} else {
				logger.Printf("Failed to connect to Redis: %v", err)
			}
		} else {
			if logger != nil {
				logger.WithField("redisHost", redisHost).Info("Conexión a Redis establecida correctamente")
			} else {
				logger.Printf("Successfully connected to Redis at %s", redisHost)
			}
		}
	})

	return redisClient
}

// CheckAndIncrementRateLimit checks if the seller has exceeded their rate limit
// and increments the counter if not
func CheckAndIncrementRateLimit(ctx context.Context, sellerID string, maxRequestsPerSecond int) (bool, error) {
	rdb := GetClient()

	// Create a timestamp for the current second
	now := time.Now().Unix()
	key := fmt.Sprintf("ratelimit:%s:%d", sellerID, now)

	// Use a Lua script to ensure atomicity
	script := `
	local current = redis.call('INCR', KEYS[1])
	if current == 1 then
		redis.call('EXPIRE', KEYS[1], 1)
	end
	return current
	`

	// Run the script
	result, err := rdb.Eval(ctx, script, []string{key}).Int64()
	if err != nil {
		logger.Printf("Redis error: %v", err)
		// In case of Redis error, we'll allow the request to proceed
		return true, err
	}

	// Check if rate limit is exceeded
	return int(result) <= maxRequestsPerSecond, nil
}
