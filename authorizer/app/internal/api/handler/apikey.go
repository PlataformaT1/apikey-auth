package handler

import (
	"apikey/internal/api/resp"
	"apikey/internal/service"
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/sirupsen/logrus"
)

const INVALID_COMMERCE = "invalid commerce id: %v"

type ApiKeyHandler interface {
	HandleValidateApiKey(ctx context.Context, request events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error)
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

func (c *apikeyHandler) HandleValidateApiKey(ctx context.Context, request events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error) {

	status := "Allow"

	apikey := request.AuthorizationToken

	data, err := c.apikeyService.ValidateApiKey(ctx, apikey)

	if err != nil {
		errorMessage := fmt.Sprintf(" Error %v ", err)
		status = "Deny"
		data.ClientID = ""
		data.PlatformData = map[string]interface{}{
			"error": errorMessage,
		}

	}
	return resp.Respond(data.PlatformData, status, data.ClientID, request.MethodArn)
}
