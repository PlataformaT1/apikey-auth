package redis

import (
	"context"
	"crypto/tls"
	"errors"
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
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
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
}import (
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
			logger.WithField("redisHost", redisHost).Warn("No se ha configurado USER_VAR_REDIS_HOST, usando valor por defecto")
		}

		// Configurar opciones de Redis con TLS
		opts := &redis.Options{
			Addr:         redisHost,
			Password:     "",            // Si tienes contraseña
			DB:           0,             // DB por defecto
			TLSConfig:    &tls.Config{}, // Habilitar TLS
			DialTimeout:  5 * time.Second,   // Reducido para fallar más rápido
			ReadTimeout:  3 * time.Second,   // Reducido para fallar más rápido
			WriteTimeout: 3 * time.Second,   // Reducido para fallar más rápido
			PoolSize:     10,                // Limitar el tamaño del pool
			MinIdleConns: 2,                 // Mantener algunas conexiones idle
		}

		// Create Redis client
		redisClient = redis.NewClient(opts)

		// Test the connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := redisClient.Ping(ctx).Result()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"redisHost": redisHost,
				"error":     err,
			}).Error("Error al conectar con Redis")
		} else {
			logger.WithField("redisHost", redisHost).Info("Conexión a Redis establecida correctamente")
		}
	})

	return redisClient
}

// CheckAndIncrementRateLimitWithBlocking implementa un rate limiter que bloquea temporalmente
// a los clientes que exceden el límite repetidamente.
// Retorna:
// - allowed: true si está permitido, false si excede el límite
// - reason: explicación del resultado ("OK", "RATE_EXCEEDED", "BLOCKED", etc.)
// - error: cualquier error técnico ocurrido
func CheckAndIncrementRateLimitWithBlocking(ctx context.Context, clientID string, ipAddress string, maxRequestsPerSecond int) (bool, string, error) {
	// Validar parámetros básicos
	if clientID == "" {
		return false, "INVALID_CLIENT_ID", errors.New("clientID no puede estar vacío")
	}
	
	if maxRequestsPerSecond <= 0 {
		// Si no hay límite configurado o es inválido, permitimos la solicitud
		return true, "NO_LIMIT", nil
	}

	rdb := GetClient()
	if rdb == nil {
		// Si no hay cliente Redis, permitimos pero registramos
		if logger != nil {
			logger.Error("Cliente Redis no inicializado")
		}
		return true, "REDIS_ERROR", errors.New("cliente Redis no inicializado")
	}

	now := time.Now().Unix()

	// Clave para el contador por segundo (granularidad de segundo)
	rateLimitKey := fmt.Sprintf("ratelimit:%s:%d", clientID, now)
	
	// Clave para el bloqueo temporal del cliente
	blockKey := fmt.Sprintf("ratelimit:blocked:%s", clientID)
	
	// Clave para contar cuántas veces se excede el límite
	exceedCountKey := fmt.Sprintf("ratelimit:exceed:%s", clientID)
	
	// Clave para limitación por IP (granularidad de minuto)
	ipKey := fmt.Sprintf("ratelimit:ip:%s:%d", ipAddress, now/60)
	
	// Script Lua simplificado y más robusto para implementar rate limiting
	script := `
	-- Verificar si el cliente está bloqueado
	local isBlocked = redis.call('EXISTS', KEYS[2])
	if isBlocked == 1 then
		local ttl = redis.call('TTL', KEYS[2])
		return {0, "BLOCKED", ttl}
	end

	-- Verificar límite por IP (protección contra abusos)
	local ipCount = redis.call('INCR', KEYS[4])
	if ipCount == 1 then
		redis.call('EXPIRE', KEYS[4], 60) -- Expira en 1 minuto
	end

	-- Umbral de IP: 10 veces el límite por segundo * 60 segundos
	local ipLimit = tonumber(ARGV[1]) * 10
	if ipCount > ipLimit then
		-- Bloquear temporalmente por exceso de IP
		redis.call('SETEX', KEYS[2], 60, 1) -- Bloquear por 1 minuto
		return {0, "IP_RATE_EXCEEDED", 60}
	end

	-- Incrementar contador normal (por segundo)
	local current = redis.call('INCR', KEYS[1])
	if current == 1 then
		redis.call('EXPIRE', KEYS[1], 2) -- 2 segundos para mayor seguridad
	end

	-- Verificar si excede el límite por segundo
	if current > tonumber(ARGV[1]) then
		-- Incrementar contador de excesos
		local exceedCount = redis.call('INCR', KEYS[3])
		if exceedCount == 1 then
			redis.call('EXPIRE', KEYS[3], 60) -- Contar excesos durante 1 minuto
		end

		-- Si excede más de 5 veces en un minuto, bloquear brevemente
		if exceedCount > 5 then
			local blockTime = 10 -- 10 segundos
			redis.call('SETEX', KEYS[2], blockTime, 1)
			return {0, "RATE_EXCEEDED_BLOCKED", blockTime}
		end

		-- Excede pero no está bloqueado aún
		return {0, "RATE_EXCEEDED", 0}
	end

	-- Todo bien, dentro del límite
	return {1, "OK", current}
	`

	// Ejecutar el script con retry en caso de error
	var result interface{}
	var err error
	
	// Intentar hasta 2 veces con un pequeño delay entre intentos
	for attempts := 0; attempts < 2; attempts++ {
		result, err = rdb.Eval(
			ctx,
			script,
			[]string{rateLimitKey, blockKey, exceedCountKey, ipKey},
			maxRequestsPerSecond,
		).Result()
		
		if err == nil {
			break // Si no hay error, salimos del bucle
		}
		
		// Si hay error y es el primer intento, esperamos un poco
		if attempts == 0 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Si después de los intentos aún hay error
	if err != nil {
		logger.WithFields(logrus.Fields{
			"clientID": clientID,
			"error":    err,
		}).Error("Error al ejecutar script de rate limit")
		return true, "REDIS_ERROR", err // Permitir en caso de error técnico
	}

	// Analizar resultado del script Lua
	results, ok := result.([]interface{})
	if !ok || len(results) < 2 {
		logger.WithField("result", result).Error("Formato de resultado inesperado")
		return true, "INVALID_RESULT", fmt.Errorf("formato de resultado inesperado: %v", result)
	}

	// Extraer el resultado principal y la razón
	allowed, ok := results[0].(int64)
	if !ok {
		logger.Error("No se pudo convertir el resultado a int64")
		return true, "CONVERSION_ERROR", fmt.Errorf("error de conversión en resultado: %v", results[0])
	}
	
	reason, ok := results[1].(string)
	if !ok {
		logger.Error("No se pudo convertir la razón a string")
		reason = "UNKNOWN"
	}

	// Extraer TTL o contador si está disponible (3er valor)
	var extraValue int64
	if len(results) > 2 {
		extraValue, _ = results[2].(int64)
	}

	// Registrar información detallada
	logFields := logrus.Fields{
		"clientID":    clientID,
		"ipAddress":   ipAddress,
		"maxRequests": maxRequestsPerSecond,
		"allowed":     allowed == 1,
		"reason":      reason,
	}
	
	if extraValue > 0 {
		if reason == "BLOCKED" || reason == "RATE_EXCEEDED_BLOCKED" || reason == "IP_RATE_EXCEEDED" {
			logFields["blockSeconds"] = extraValue
		} else {
			logFields["currentCount"] = extraValue
		}
	}

	if allowed == 1 {
		logger.WithFields(logFields).Debug("Rate limit check: permitido")
		return true, reason, nil
	} else {
		logger.WithFields(logFields).Info("Rate limit check: denegado")
		return false, reason, ErrRateLimitExceeded
	}
}

// CheckRedisHealth verifica si Redis está disponible y funcionando correctamente
func CheckRedisHealth(ctx context.Context) error {
	client := GetClient()
	if client == nil {
		return errors.New("cliente Redis no inicializado")
	}
	
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	
	_, err := client.Ping(timeoutCtx).Result()
	return err
}
