package handler

/*import (
	"apikey/internal/api/resp"
	"apikey/internal/service"
	"apikey/pkg/redis"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/sirupsen/logrus"
)

const INVALID_COMMERCE = "invalid commerce id: %v"

type ApiKeyHandler interface {
	HandleValidateApiKey(ctx context.Context, request events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error)
}

type apikeyHandler struct {
	logger        *logrus.Logger
	apikeyService service.ServiceApiKey
}

func NewApiKeyHandler(logger *logrus.Logger, apikeyService service.ServiceApiKey) ApiKeyHandler {
	return &apikeyHandler{
		logger:        logger,
		apikeyService: apikeyService,
	}
}

func (c *apikeyHandler) HandleValidateApiKey(ctx context.Context, request events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	c.logger.Info("Processing authorization request")

	status := "Allow"

	// Obtener el API key del header
	apikey := request.Headers["x-api-key"]
	if apikey == "" {
		c.logger.Warn("Missing x-api-key header")
		status = "Deny"
		return resp.RespondWithError("Missing API key", status, "", request.MethodArn), nil
	}

	// Validar el API key
	data, err := c.apikeyService.ValidateApiKey(ctx, apikey)
	if err != nil {
		errorMessage := fmt.Sprintf("Error validating API key: %v", err)
		c.logger.Warn(errorMessage)
		status = "Deny"
		return resp.RespondWithError(errorMessage, status, "", request.MethodArn), nil
	}

	// Verificar que el API key esté activo
	if !data.IsActive {
		c.logger.Warnf("API key %s is inactive", apikey)
		status = "Deny"
		return resp.RespondWithError("API key is inactive", status, data.ClientID, request.MethodArn), nil
	}

	// Verificar que el API key no haya expirado
	currentTime := time.Now()
	if currentTime.After(data.ExpiredAt) {
		c.logger.Warnf("API key %s has expired", apikey)
		status = "Deny"
		return resp.RespondWithError("API key has expired", status, data.ClientID, request.MethodArn), nil
	}

	// Verificar tamaño de payload
	if contentLength, ok := request.Headers["content-length"]; ok && contentLength != "" {
		payloadSizeBytes, err := strconv.ParseInt(contentLength, 10, 64)
		if err == nil {
			// Convertir MB a bytes (1 MB = 1,048,576 bytes)
			maxPayloadSizeBytes := int64(data.UsageLimits.MaxPayloadSize) * 1024 * 1024

			if payloadSizeBytes > maxPayloadSizeBytes {
				c.logger.Warnf("Payload size exceeds maximum allowed size. Size: %d, Max: %d",
					payloadSizeBytes, maxPayloadSizeBytes)

				status = "Deny"
				return resp.RespondWithErrorAndHeaders(
					"Payload size exceeds maximum allowed size",
					status,
					data.ClientID,
					request.MethodArn,
					map[string]string{
						"X-Max-Payload-Size": fmt.Sprintf("%d", data.UsageLimits.MaxPayloadSize),
					}), nil
			}
		} else {
			c.logger.Warnf("Could not parse content-length header: %v", err)
		}
	}

	// Verificar rate limit
	allowed, err := redis.CheckAndIncrementRateLimit(ctx, data.ClientID, data.UsageLimits.RequestsPerSecond)
	if err != nil {
		// En caso de error de Redis, permitimos la solicitud pero registramos el error
		c.logger.Warnf("Error checking rate limit: %v", err)
	} else if !allowed {
		// Rate limit excedido
		resetTime := time.Now().Add(1 * time.Second).Unix()
		c.logger.Warnf("Rate limit exceeded for client %s", data.ClientID)

		status = "Deny"
		return resp.RespondWithErrorAndHeaders(
			"Rate limit exceeded",
			status,
			data.ClientID,
			request.MethodArn,
			map[string]string{
				"X-RateLimit-Limit":     fmt.Sprintf("%d", data.UsageLimits.RequestsPerSecond),
				"X-RateLimit-Remaining": "0",
				"X-RateLimit-Reset":     fmt.Sprintf("%d", resetTime),
			}), nil
	}

	// Si todo está bien, continuamos con la respuesta Allow
	c.logger.Infof("Request authorized for client %s", data.ClientID)
	return resp.Respond(data.PlatformData, status, data.ClientID, request.MethodArn), nil
}*/
