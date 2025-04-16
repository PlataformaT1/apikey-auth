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
	redisClient          *redis.Client
	redisClientOnce      sync.Once
	logger               *logrus.Logger
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
			Password:     "",              // Si tienes contraseña
			DB:           0,               // DB por defecto
			TLSConfig:    &tls.Config{},   // Habilitar TLS
			DialTimeout:  5 * time.Second, // Reducido para fallar más rápido
			ReadTimeout:  3 * time.Second, // Reducido para fallar más rápido
			WriteTimeout: 3 * time.Second, // Reducido para fallar más rápido
			PoolSize:     10,              // Limitar el tamaño del pool
			MinIdleConns: 2,               // Mantener algunas conexiones idle
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

// CheckAndIncrementRateLimitWithBlocking verifica e incrementa el contador de rate limit
// Esta versión corregida funciona con Redis Cluster
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

	// IMPORTANTE: En Redis Cluster, necesitamos asegurar que todas las claves usadas
	// en una operación pertenezcan al mismo slot de hash.
	// Para hacer esto, usamos el formato: {hash_tag}:resto:de:la:clave
	// Donde todo lo que está entre {} se usa para calcular el slot.

	// Usamos clientID como hash tag para asegurar que todas las claves estén en el mismo slot
	prefix := fmt.Sprintf("{%s}", clientID)

	// Clave para el contador por segundo
	rateLimitKey := fmt.Sprintf("%s:rate:%d", prefix, now)

	// Clave para el bloqueo temporal
	blockKey := fmt.Sprintf("%s:block", prefix)

	// Clave para contar excesos
	exceedCountKey := fmt.Sprintf("%s:exceed", prefix)

	// Para IP también usamos el mismo prefijo para asegurar que esté en el mismo slot
	ipKey := fmt.Sprintf("%s:ip:%s:%d", prefix, ipAddress, now/60)

	// Verificar primero si está bloqueado (operación simple)
	blocked, err := rdb.Exists(ctx, blockKey).Result()
	if err != nil {
		logger.WithError(err).Error("Error al verificar estado de bloqueo")
		return true, "REDIS_ERROR", err
	}

	if blocked == 1 {
		// Cliente bloqueado, obtener TTL
		_, err := rdb.TTL(ctx, blockKey).Result()
		if err != nil {
			logger.WithError(err).Error("Error al obtener TTL del bloqueo")
			return false, "BLOCKED", nil
		}
		return false, "BLOCKED", nil
	}

	// Verificar e incrementar contador de IP
	ipCount, err := rdb.Incr(ctx, ipKey).Result()
	if err != nil {
		logger.WithError(err).Error("Error al incrementar contador de IP")
		return true, "REDIS_ERROR", err
	}

	// Establecer expiración si es la primera incrementación
	if ipCount == 1 {
		rdb.Expire(ctx, ipKey, 60*time.Second)
	}

	// Umbral de IP: 10 veces el límite por segundo
	ipLimit := int64(maxRequestsPerSecond * 10)
	if ipCount > ipLimit {
		// Bloquear temporalmente por exceso de IP
		err = rdb.SetEx(ctx, blockKey, "1", 60*time.Second).Err()
		if err != nil {
			logger.WithError(err).Error("Error al establecer bloqueo por IP")
		}
		return false, "IP_RATE_EXCEEDED", nil
	}

	// Incrementar contador principal de rate limit
	current, err := rdb.Incr(ctx, rateLimitKey).Result()
	if err != nil {
		logger.WithError(err).Error("Error al incrementar contador de rate limit")
		return true, "REDIS_ERROR", err
	}

	// Establecer expiración si es la primera incrementación
	if current == 1 {
		rdb.Expire(ctx, rateLimitKey, 2*time.Second)
	}

	// Verificar límite
	if current > int64(maxRequestsPerSecond) {
		// Incrementar contador de excesos
		exceedCount, err := rdb.Incr(ctx, exceedCountKey).Result()
		if err != nil {
			logger.WithError(err).Error("Error al incrementar contador de excesos")
			return false, "RATE_EXCEEDED", nil
		}

		// Establecer expiración si es la primera incrementación
		if exceedCount == 1 {
			rdb.Expire(ctx, exceedCountKey, 60*time.Second)
		}

		// Si excede más de 5 veces en un minuto, bloquear
		if exceedCount > 5 {
			err = rdb.SetEx(ctx, blockKey, "1", 10*time.Second).Err()
			if err != nil {
				logger.WithError(err).Error("Error al establecer bloqueo por excesos")
			}
			return false, "RATE_EXCEEDED_BLOCKED", nil
		}

		return false, "RATE_EXCEEDED", nil
	}

	// Dentro del límite
	return true, "OK", nil
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
