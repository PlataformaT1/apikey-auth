package main

import (
	"apikey/internal/service"
	"apikey/pkg/redis"
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

// Inicializa los servicios necesarios
var apiKeyService service.ServiceApiKey
var logger *logrus.Logger

func init() {
	// Configurar logger
	logger = logrus.New()

	// Configurar formato JSON para CloudWatch Logs (opcional)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Configurar nivel de log basado en variable de entorno
	logLevel := os.Getenv("USER_VAR_LOG_LEVEL")
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel // Nivel por defecto
	}
	logger.SetLevel(level)

	logger.Info("Inicializando Lambda Authorizer")

	// Inicialización y logs
	logger.Info("Inicializando Lambda Authorizer")

	redisHost := os.Getenv("USER_VAR_REDIS_HOST")
	logger.WithField("redisHost", redisHost).Info("Configuración de Redis cargada")

	// Inicializa redis
	_ = redis.GetClient()

	// Inicializa los servicios (esto dependerá de tu implementación)
	// apiKeyService = service.NewApiKeyService(...)
}

// HandleRequest es la función Lambda que se ejecuta para autorizar solicitudes
func HandleRequest(ctx context.Context, request events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
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
		return generatePolicy("user", "Deny", request.MethodArn, map[string]interface{}{
			"error": "API key is inactive",
		}), nil
	}
	reqLogger.Printf("API Key válida para ClientID: %s, Plataforma: %s", data.ClientID, data.Platform)

	// Verificar que el API key no haya expirado
	currentTime := time.Now()
	if currentTime.After(data.ExpiredAt) {
		return generatePolicy("user", "Deny", request.MethodArn, map[string]interface{}{
			"error": "API key has expired",
		}), nil
	}

	// Verificar tamaño de payload si hay header Content-Length
	if contentLength, ok := request.Headers["content-length"]; ok && contentLength != "" {
		payloadSizeBytes, err := strconv.ParseInt(contentLength, 10, 64)
		if err == nil {
			// Convertir MB a bytes (1 MB = 1,048,576 bytes)
			maxPayloadSizeBytes := int64(data.UsageLimits.MaxPayloadSize) * 1024 * 1024

			if payloadSizeBytes > maxPayloadSizeBytes {
				return generatePolicyWithHeaders("user", "Deny", request.MethodArn, map[string]interface{}{
					"error": "Payload size exceeds maximum allowed size",
				}, map[string]string{
					"X-Max-Payload-Size": fmt.Sprintf("%d", data.UsageLimits.MaxPayloadSize),
				}), nil
			}
		} else {
			reqLogger.Printf("Could not parse content-length header: %v", err)
		}
	}

	// Verificar rate limit
	allowed, err := redis.CheckAndIncrementRateLimit(ctx, data.ClientID, data.UsageLimits.RequestsPerSecond)
	if err != nil {
		// En caso de error de Redis, permitimos la solicitud pero registramos el error
		reqLogger.Printf("Error checking rate limit: %v", err)
	} else if !allowed {
		reqLogger.Printf("Rate limit excedido para ClientID: %s (Límite: %d reqs/seg)", data.ClientID, data.UsageLimits.RequestsPerSecond)
		// Rate limit excedido
		resetTime := time.Now().Add(1 * time.Second).Unix()
		return generatePolicyWithHeaders("user", "Deny", request.MethodArn, map[string]interface{}{
			"error": "Rate limit exceeded",
		}, map[string]string{
			"X-RateLimit-Limit":     fmt.Sprintf("%d", data.UsageLimits.RequestsPerSecond),
			"X-RateLimit-Remaining": "0",
			"X-RateLimit-Reset":     fmt.Sprintf("%d", resetTime),
		}), nil
	}

	reqLogger.Printf("Decisión de autorización: %s", "Allow")
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
