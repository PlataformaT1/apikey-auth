package resp

/*import (
	"github.com/aws/aws-lambda-go/events"
)

// Respond creates an APIGatewayCustomAuthorizerResponse
func Respond(context map[string]interface{}, effect, principalId, resource string) events.APIGatewayCustomAuthorizerResponse {
	authResponse := events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: principalId,
		Context:     make(map[string]interface{}),
	}

	// Add all context key-values to the response context
	for k, v := range context {
		authResponse.Context[k] = v
	}

	// Add policy document if effect and resource are provided
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

// RespondWithError creates a response with an error message
func RespondWithError(errorMessage, effect, principalId, resource string) events.APIGatewayCustomAuthorizerResponse {
	return Respond(map[string]interface{}{
		"error": errorMessage,
	}, effect, principalId, resource)
}

// RespondWithErrorAndHeaders creates a response with an error message and custom headers
func RespondWithErrorAndHeaders(errorMessage, effect, principalId, resource string, headers map[string]string) events.APIGatewayCustomAuthorizerResponse {
	resp := Respond(map[string]interface{}{
		"error": errorMessage,
	}, effect, principalId, resource)

	// Add headers to context
	if resp.Context == nil {
		resp.Context = make(map[string]interface{})
	}
	resp.Context["responseHeaders"] = headers

	return resp
}*/
