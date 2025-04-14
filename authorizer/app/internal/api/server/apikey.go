package server

/*import (
	"apikey/internal/api/handler"
	"context"

	"github.com/aws/aws-lambda-go/events"
)

// ApiKeyServer maneja las operaciones de autorización de API key
type ApiKeyServer struct {
	apikeyHandler handler.ApiKeyHandler
	router        *Router // Asumiendo que tienes esta estructura
}

// NewApiKeyServer crea una nueva instancia del servidor de API key
func NewApiKeyServer(apikeyHandler handler.ApiKeyHandler, router *Router) *ApiKeyServer {
	return &ApiKeyServer{
		apikeyHandler: apikeyHandler,
		router:        router,
	}
}

// RegisterAuthorizer registra el autorizador para todas las rutas
func (s *ApiKeyServer) RegisterAuthorizer() {
	// Asumiendo que tienes un método para configurar el autorizador global
	s.router.SetAuthorizer(func(ctx context.Context, request events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
		// Crear un nuevo objeto del tipo REQUEST con los campos necesarios
		requestTypeRequest := events.APIGatewayCustomAuthorizerRequestTypeRequest{
			Type:                  request.Type,
			MethodArn:             request.MethodArn,
			Headers:               map[string]string{},
			PathParameters:        map[string]string{},
			QueryStringParameters: map[string]string{},
			StageVariables:        map[string]string{},
		}

		// Trasladar el token de autorización a un header
		if request.AuthorizationToken != "" {
			requestTypeRequest.Headers["x-api-key"] = request.AuthorizationToken
		}

		// Llamar al handler con el objeto convertido
		return s.apikeyHandler.HandleValidateApiKey(ctx, requestTypeRequest)
	})
}*/
