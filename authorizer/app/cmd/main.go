package main

import (
	"apikey/internal/service"
	"apikey/pkg/redis"
	"context"
	"fmt"
	"os"
	"time"

	mongodb "apikey/internal/repository"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Inicializa los servicios necesarios
var apiKeyService service.ServiceApiKey
var logger *logrus.Logger

func init() {
	// Configurar logger
	logger = logrus.New()

	// Configurar formato JSON para CloudWatch Logs
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Configurar nivel de log basado en variable de entorno
	logLevel := os.Getenv("USER_VAR_LOG_LEVEL")
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel // Nivel por defecto
		logger.WithFields(logrus.Fields{
			"error": err,
			"level": level,
		}).Info("No se pudo parsear LOG_LEVEL, usando nivel por defecto")
	}
	logger.SetLevel(level)

	logger.Info("Inicializando Lambda Authorizer")

	// Inicializar el logger de Redis
	redis.InitLogger(logger)

	// Cargar configuración de Redis
	redisHost := os.Getenv("USER_VAR_REDIS_HOST")
	logger.WithField("redisHost", redisHost).Info("Configuración de Redis cargada")

	// Inicializa redis
	redisClient := redis.GetClient()

	// Verificar conexión a Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		logger.WithError(err).Error("Error al conectar con Redis")
	} else {
		logger.Info("Conexión a Redis establecida correctamente")
	}

	// Configurar conexión MongoDB
	mongoURI := os.Getenv("USER_VAR_DB_MONGO_URI")
	if mongoURI == "" {
		logger.Error("La variable de entorno USER_VAR_DB_MONGO_URI es requerida")
	}

	mongoDBName := os.Getenv("USER_VAR_MONGO_DB_NAME")
	if mongoDBName == "" {
		mongoDBName = "apikey_db" // Valor predeterminado
		logger.WithField("dbName", mongoDBName).Warn("USER_VAR_MONGO_DB_NAME no configurada, usando valor predeterminado")
	}

	mongoCollection := os.Getenv("USER_VAR_MONGO_COLLECTION")
	if mongoCollection == "" {
		mongoCollection = "apikeys" // Valor predeterminado
		logger.WithField("collection", mongoCollection).Warn("USER_VAR_MONGO_COLLECTION no configurada, usando valor predeterminado")
	}

	// Configurar cliente MongoDB
	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mongoCancel()

	mongoClient, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		logger.WithError(err).Error("Error al conectar con MongoDB")
	}

	// Verificar conexión a MongoDB
	err = mongoClient.Ping(mongoCtx, nil)
	if err != nil {
		logger.WithError(err).Error("Error al verificar conexión con MongoDB")
	}
	logger.Info("Conexión a MongoDB establecida correctamente")

	// Inicializa el repositorio
	apiKeyRepo := mongodb.NewApiKeyRepository(logger, mongoClient, mongoDBName, mongoCollection)

	// Inicializa el servicio
	apiKeyService = service.NewApiKeyService(logger, apiKeyRepo)

	logger.Info("Servicios inicializados correctamente")
}

// HandleRequest es la función Lambda que se ejecuta para autorizar solicitudes
func HandleRequest(ctx context.Context, request events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	// Recuperar de pánico para evitar que la Lambda falle completamente
	defer func() {
		if r := recover(); r != nil {
			logger.WithField("panic", r).Error("Panic recuperado en el autorizador")
		}
	}()

	// Crear un logger con contexto de la solicitud
	reqLogger := logger.WithFields(logrus.Fields{
		"method": request.HTTPMethod,
		"path":   request.Path,
		"source": request.RequestContext.Identity.SourceIP,
	})

	reqLogger.Info("Procesando solicitud de autorización")
	logger.Info(request)
	// Verificar que el API key esté presente
	apiKey := request.Headers["x-api-key"]
	if apiKey == "" {
		reqLogger.Warn("Error: API key no encontrada en headers")
		return generatePolicy("user", "Deny", request.MethodArn, map[string]interface{}{
			"error": "Missing API key",
		}), nil
	}

	// Ocultar parte del API key por seguridad
	maskedKey := apiKey
	if len(apiKey) > 8 {
		maskedKey = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
	}
	reqLogger = reqLogger.WithField("apiKey", maskedKey)

	// Validar el API key con tu servicio existente
	data, err := apiKeyService.ValidateApiKey(ctx, apiKey)
	if err != nil {
		reqLogger.WithError(err).Error("Error validando API key")
		return generatePolicy("user", "Deny", request.MethodArn, map[string]interface{}{
			"error": fmt.Sprintf("Error validating API key: %v", err),
		}), nil
	}

	// Verificar que el API key esté activo
	if !data.IsActive {
		reqLogger.WithField("clientID", data.ClientID).Warn("API key inactiva")
		return generatePolicy("user", "Deny", request.MethodArn, map[string]interface{}{
			"error": "API key is inactive",
		}), nil
	}

	reqLogger.WithFields(logrus.Fields{
		"clientID": data.ClientID,
		"platform": data.Platform,
	}).Info("API Key válida")

	// Verificar que el API key no haya expirado
	currentTime := time.Now()
	if currentTime.After(data.ExpiredAt) {
		reqLogger.WithFields(logrus.Fields{
			"clientID":  data.ClientID,
			"expiredAt": data.ExpiredAt,
		}).Warn("API key expirada")

		return generatePolicy("user", "Deny", request.MethodArn, map[string]interface{}{
			"error": "API key has expired",
		}), nil
	}

	// Verificar rate limit con timeout específico para evitar bloqueos largos
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	ipAddress := request.RequestContext.Identity.SourceIP

	// Verificamos el límite de velocidad sin usar script Lua para compatibilidad con Redis Cluster
	allowed, reason, err := redis.CheckAndIncrementRateLimitWithBlocking(
		timeoutCtx,
		data.ClientID,
		ipAddress,
		data.UsageLimits.RequestsPerSecond,
	)

	// Log detallado del resultado de la verificación del rate limit
	rateLimitFields := logrus.Fields{
		"clientID":          data.ClientID,
		"ipAddress":         ipAddress,
		"requestsPerSecond": data.UsageLimits.RequestsPerSecond,
		"allowed":           allowed,
		"reason":            reason,
	}

	if err != nil {
		// Error al verificar rate limit - registramos el error
		reqLogger.WithFields(rateLimitFields).WithError(err).Error("Error al verificar rate limit en Redis")

		// Al encontrar un error en Redis, podemos decidir uno de estos enfoques:
		// 1. Permitir la petición (más permisivo, pero puede sobrecargar el sistema)
		// 2. Denegar la petición (más seguro, pero puede bloquear peticiones legítimas)

		// Opción 1: Permitir con advertencia (comentada)
		// return generatePolicy("user", "Allow", request.MethodArn, data.PlatformData), nil

		// Opción 2: Denegar con información clara (opción más segura)
		return generatePolicy("user", "Deny", request.MethodArn, map[string]interface{}{
			"error":          "Error interno al verificar límites de uso",
			"rateLimitError": true,
		}), nil
	}

	if !allowed {
		// Rate limit excedido - denegar con información clara
		reqLogger.WithFields(rateLimitFields).Warn("Rate limit excedido")

		// Crear respuesta para API Gateway
		// API Gateway convertirá "Deny" en un HTTP 403
		return generatePolicy("user", "Deny", request.MethodArn, map[string]interface{}{
			"rateLimitExceeded": true,
			"reason":            reason,
			"resetAt":           time.Now().Add(1 * time.Second).Unix(),
			"limit":             data.UsageLimits.RequestsPerSecond,
			// Estos campos serán visibles en el contexto de la respuesta
		}), nil
	}

	// Log final antes de autorizar la solicitud
	reqLogger.WithFields(logrus.Fields{
		"clientID":  data.ClientID,
		"decision":  "Allow",
		"rateCheck": rateLimitFields,
	}).Info("Solicitud autorizada")

	// Si todo está bien, autorizamos la solicitud
	return generatePolicy("user", "Allow", request.MethodArn, data.PlatformData), nil
}

// generatePolicy crea una política de IAM para responder al API Gateway
func generatePolicy(principalID string, effect string, resource string, context map[string]interface{}) events.APIGatewayCustomAuthorizerResponse {
	authResponse := events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: principalID,
		Context:     make(map[string]interface{}),
	}

	// Añadir todos los valores de contexto a la respuesta
	for k, v := range context {
		authResponse.Context[k] = v
	}

	// Añadir documento de política si se proporcionan efecto y recurso
	if effect != "" && resource != "" {
		authResponse.PolicyDocument = events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		}
	}

	return authResponse
}

func main() {
	lambda.Start(HandleRequest)
}
