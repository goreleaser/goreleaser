package blob

import (
	"fmt"
	"os"
	"strings"
)

// Check required ENV variables based on Blob Provider
func checkProvider(provider string) error {

	switch provider {
	case "azblob":
		return checkEnv("AZURE_STORAGE_ACCOUNT", "AZURE_STORAGE_KEY")
	case "gs":
		return checkEnv("GOOGLE_APPLICATION_CREDENTIALS")
	case "s3":
		return checkEnv("AWS_ACCESS_KEY", "AWS_SECRET_KEY", "AWS_REGION")
	default:
		return fmt.Errorf("unknown provider [%v],currently supported providers: [azblob, gs, s3]", provider)
	}

}

func checkEnv(envs ...string) error {

	var missingEnv []string

	for _, env := range envs {
		s := os.Getenv(env)
		if s == "" {
			missingEnv = append(missingEnv, env)
		}
	}

	if len(missingEnv) != 0 {
		return fmt.Errorf("missing %v", strings.Join(missingEnv, ","))
	}
	return nil
}

// Check if error contains specific string
func errorContains(err error, subs ...string) bool {

	for _, sub := range subs {
		if strings.Contains(err.Error(), sub) {
			return true
		}
	}
	return false
}
