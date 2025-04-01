package resp

import (
	"github.com/aws/aws-lambda-go/events"
)

func Respond(context map[string]interface{}, effect, clientID, resource string) (events.APIGatewayCustomAuthorizerResponse, error) {
	return events.APIGatewayCustomAuthorizerResponse{
			PrincipalID: clientID,
			PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
				Version: "2012-10-17",
				Statement: []events.IAMPolicyStatement{
					{
						Action:   []string{"execute-api:Invoke"},
						Effect:   effect,
						Resource: []string{resource},
					},
				},
			},
			Context: context, // Add additional context here
		},
		nil
}
