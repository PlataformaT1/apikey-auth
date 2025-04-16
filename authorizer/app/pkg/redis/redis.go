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
			Addr:         redisHost,
			Password:     "",            // Si tienes contraseña
			DB:           0,             // DB por defecto
			TLSConfig:    &tls.Config{}, // Habilitar TLS
			DialTimeout:  15 * time.Second,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
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

// CheckAndIncrementRateLimitWithBlocking implementa un rate limiter que bloquea temporalmente
// a los clientes que exceden el límite repetidamente
func CheckAndIncrementRateLimitWithBlocking(ctx context.Context, clientID string, ipAddress string, maxRequestsPerSecond int) (bool, string, error) {
	rdb := GetClient()
	now := time.Now().Unix()

	// Clave para el contador por segundo
	rateLimitKey := fmt.Sprintf("ratelimit:%s:%d", clientID, now)

	// Clave para el bloqueo temporal
	blockKey := fmt.Sprintf("ratelimit:blocked:%s", clientID)

	// Clave para contar excesos de límite
	exceedCountKey := fmt.Sprintf("ratelimit:exceed:%s", clientID)

	// Clave para limitación por IP
	ipKey := fmt.Sprintf("ratelimit:ip:%s:%d", ipAddress, now/60) // Por minuto

	// Script Lua para implementar rate limiting con bloqueo
	script := `
	-- Comprobar si el cliente está bloqueado
	local isBlocked = redis.call('EXISTS', KEYS[2])
	if isBlocked == 1 then
	local ttl = redis.call('TTL', KEYS[2])
	return {0, "BLOCKED", ttl}
	end

	-- Incrementar contador de IP (por minuto)
	local ipCount = redis.call('INCR', KEYS[4])
	if ipCount == 1 then
	redis.call('EXPIRE', KEYS[4], 60)
	end

	-- Bloquear inmediatamente si hay demasiadas solicitudes desde la misma IP
	local ipLimit = tonumber(ARGV[2]) * 10 -- Multiplicador para IP
	if ipCount > ipLimit then
	redis.call('SET', KEYS[2], 1)
	redis.call('EXPIRE', KEYS[2], 300) -- Bloquear por 5 minutos
	return {0, "IP_RATE_EXCEEDED", 300}
	end

	-- Incrementar contador normal
	local current = redis.call('INCR', KEYS[1])
	if current == 1 then
	redis.call('EXPIRE', KEYS[1], 1)
	end

	-- Comprobar si excede el límite
	if current > tonumber(ARGV[1]) then
	-- Incrementar contador de excesos
	local exceedCount = redis.call('INCR', KEYS[3])
	if exceedCount == 1 then
	redis.call('EXPIRE', KEYS[3], 60) -- Expirar después de 1 minuto
	end

	-- Si ha excedido el límite muchas veces, bloquearlo temporalmente
	if exceedCount > 5 then
	local blockTime = 30 -- 30 segundos por defecto
	redis.call('SET', KEYS[2], 1)
	redis.call('EXPIRE', KEYS[2], blockTime)
	return {0, "RATE_EXCEEDED_BLOCKED", blockTime}
	end

	return {0, "RATE_EXCEEDED", 0}
	end

	return {1, "OK", current}
	`

	// Ejecutar el script
	result, err := rdb.Eval(
		ctx,
		script,
		[]string{rateLimitKey, blockKey, exceedCountKey, ipKey},
		maxRequestsPerSecond, maxRequestsPerSecond,
	).Result()

	if err != nil {
		if logger != nil {
			logger.WithError(err).Error("Error al ejecutar script de rate limit")
		}
		return true, "ERROR", err // Permitir en caso de error
	}

	// Analizar resultado
	results, ok := result.([]interface{})
	if !ok || len(results) < 2 {
		return true, "INVALID_RESULT", fmt.Errorf("formato de resultado inesperado")
	}

	// Extraer el resultado principal (permitido o no)
	allowed, _ := results[0].(int64)
	reason, _ := results[1].(string)

	// Registrar información
	if logger != nil {
		logFields := logrus.Fields{
			"clientID":    clientID,
			"ipAddress":   ipAddress,
			"maxRequests": maxRequestsPerSecond,
			"allowed":     allowed == 1,
			"reason":      reason,
		}

		if len(results) > 2 {
			if ttl, ok := results[2].(int64); ok {
				logFields["ttl"] = ttl
			}
		}

		if allowed == 1 {
			logger.WithFields(logFields).Debug("Rate limit check passed")
		} else {
			logger.WithFields(logFields).Warn("Rate limit exceeded")
		}
	}

	return allowed == 1, reason, nil
}
