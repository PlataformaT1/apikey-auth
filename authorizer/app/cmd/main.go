package main

import (
	"apikey/internal/service"
	"apikey/pkg/redis"
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Inicializa los servicios necesarios
var apiKeyService service.ServiceApiKey

func init() {
	// Inicializa redis
	_ = redis.GetClient()

	// Inicializa los servicios (esto dependerá de tu implementación)
	// apiKeyService = service.NewApiKeyService(...)
}

// HandleRequest es la función Lambda que se ejecuta para autorizar solicitudes
func HandleRequest(ctx context.Context, request events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	// Extrae el API key del header
	log.Printf("inicia el autorizador")
	log.Printf(request.Headers["x-api-key"])
	apiKey := request.Headers["x-api-key"]
	if apiKey == "" {
		return generatePolicy("user", "Deny", request.MethodArn, map[string]interface{}{
			"error": "Missing API key",
		}), nil
	}

	// Validar el API key con tu servicio existente
	data, err := apiKeyService.ValidateApiKey(ctx, apiKey)
	if err != nil {
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
			log.Printf("Could not parse content-length header: %v", err)
		}
	}

	// Verificar rate limit
	allowed, err := redis.CheckAndIncrementRateLimit(ctx, data.ClientID, data.UsageLimits.RequestsPerSecond)
	if err != nil {
		// En caso de error de Redis, permitimos la solicitud pero registramos el error
		log.Printf("Error checking rate limit: %v", err)
	} else if !allowed {
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
