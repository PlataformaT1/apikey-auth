package env

import (
	"fmt"
	"os"
)

func Validate(variables []string) error {
	for _, variable := range variables {
		value, exists := os.LookupEnv(variable)
		if !exists || value == "" {
			return fmt.Errorf("environment variable %s is mandatory", variable)
		}
	}
	return nil
}

func GetEnvs() []string {
	return []string{
		"USER_VAR_LOG_CHAN",
		"USER_VAR_LOG_LEVEL",
		"USER_VAR_DB_MONGO_URI",
	}
}
