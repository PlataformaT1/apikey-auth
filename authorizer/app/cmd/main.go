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
	/// Crear un logger con contexto de la solicitud
	reqLogger := logger.WithFields(logrus.Fields{
		"method": request.HTTPMethod,
		"path":   request.Path,
		"source": request.RequestContext.Identity.SourceIP,
	})

	reqLogger.Info("Procesando solicitud de autorización")

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

	// Verificar tamaño de payload si hay header Content-Length
	/*if contentLength, ok := request.Headers["content-length"]; ok && contentLength != "" {
		payloadSizeBytes, err := strconv.ParseInt(contentLength, 10, 64)
		if err == nil {
			// Convertir MB a bytes (1 MB = 1,048,576 bytes)
			maxPayloadSizeBytes := int64(data.UsageLimits.MaxPayloadSize) * 1024 * 1024

			if payloadSizeBytes > maxPayloadSizeBytes {
				reqLogger.WithFields(logrus.Fields{
					"clientID":         data.ClientID,
					"payloadSize":      payloadSizeBytes,
					"maxPayloadSize":   data.UsageLimits.MaxPayloadSize,
					"maxPayloadBytes":  maxPayloadSizeBytes,
				}).Warn("Tamaño de payload excede el máximo permitido")

				return generatePolicyWithHeaders("user", "Deny", request.MethodArn, map[string]interface{}{
					"error": "Payload size exceeds maximum allowed size",
				}, map[string]string{
					"X-Max-Payload-Size": fmt.Sprintf("%d", data.UsageLimits.MaxPayloadSize),
				}), nil
			}
		} else {
			reqLogger.WithError(err).Warn("No se pudo parsear el header content-length")
		}
	}*/

	// Verificar rate limit con el sistema mejorado
	ipAddress := request.RequestContext.Identity.SourceIP
	allowed, reason, err := redis.CheckAndIncrementRateLimitWithBlocking(
		ctx,
		data.ClientID,
		ipAddress,
		data.UsageLimits.RequestsPerSecond,
	)

	if err != nil {
		// En caso de error, permitimos la solicitud pero registramos el error
		reqLogger.WithError(err).Error("Error al verificar rate limit en Redis")
	} else if !allowed {
		reqLogger.WithFields(logrus.Fields{
			"clientID":          data.ClientID,
			"ipAddress":         ipAddress,
			"requestsPerSecond": data.UsageLimits.RequestsPerSecond,
			"reason":            reason,
		}).Warn("Rate limit excedido")

		// Rate limit excedido
		resetTime := time.Now().Add(1 * time.Second).Unix()

		// Mensajes personalizados según el motivo
		errorMsg := "Rate limit exceeded"
		if reason == "BLOCKED" || reason == "RATE_EXCEEDED_BLOCKED" {
			errorMsg = "Too many requests. You have been temporarily blocked."
		} else if reason == "IP_RATE_EXCEEDED" {
			errorMsg = "Too many requests from your IP address."
		}

		return generatePolicyWithHeaders("user", "Deny", request.MethodArn, map[string]interface{}{
			"error":      errorMsg,
			"statusCode": 429, // Too Many Requests
			"reason":     reason,
		}, map[string]string{
			"X-RateLimit-Limit":     fmt.Sprintf("%d", data.UsageLimits.RequestsPerSecond),
			"X-RateLimit-Remaining": "0",
			"X-RateLimit-Reset":     fmt.Sprintf("%d", resetTime),
		}), nil
	}

	reqLogger.WithFields(logrus.Fields{
		"clientID": data.ClientID,
		"decision": "Allow",
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

// generatePolicyWithHeaders crea una política con headers personalizados
func generatePolicyWithHeaders(principalID string, effect string, resource string, context map[string]interface{}, headers map[string]string) events.APIGatewayCustomAuthorizerResponse {
	// Crear la política base
	policy := generatePolicy(principalID, effect, resource, context)

	// Añadir headers a la respuesta
	if policy.Context == nil {
		policy.Context = make(map[string]interface{})
	}
	policy.Context["responseHeaders"] = headers

	return policy
}

func main() {
	lambda.Start(HandleRequest)
}
