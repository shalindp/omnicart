package utils

import (
	"fmt"
	"os"
	"strconv"

	"onion.api/infrastrucre"
	"onion.api/persistence"
	"onion.api/presentation"

	"github.com/joho/godotenv"
)

type EnvironmentVariables struct {
	PersistenceSettings    persistence.PersistenceSettings
	InfrastructureSettings infrastrucre.InfrastructureSettings
	PresentationSettings   presentation.PresentationSettings
}

func LoadEnvironmentVariables() EnvironmentVariables {
	error := godotenv.Load()
	if error != nil {
		fmt.Printf("Warning: could not load .env file: %v\n", error)
	}

	return EnvironmentVariables{
		PersistenceSettings: persistence.PersistenceSettings{
			ConnectionString: getEnv("DATABASE_URL"),
		},
		InfrastructureSettings: infrastrucre.InfrastructureSettings{
			PakNSave: infrastrucre.RetailerSettings{
				ReferenceStore:      getEnv("PAKNSAVE_REFERENCE_STORE"),
				DegreeOfParallelism: getEnvInt("PAKNSAVE_DEGREE_OF_PARALLELISM", 12),
				NumOfRetries:        getEnvInt("PAKNSAVE_NUM_OF_RETRIES", 3),
				DelayInMs:           getEnvInt("PAKNSAVE_DELAY_IN_MS", 400),
				DelayMaxInMs:        getEnvInt("PAKNSAVE_DELAY_MAX_IN_MS", 600),
				TimeoutInMs:         getEnvInt("PAKNSAVE_TIMEOUT_IN_MS", 30000),
			},
			Woolworths: infrastrucre.RetailerSettings{
				ReferenceStore:      getEnv("WOOLWORTHS_REFERENCE_STORE"),
				DegreeOfParallelism: getEnvInt("WOOLWORTHS_DEGREE_OF_PARALLELISM", 6),
				NumOfRetries:        getEnvInt("WOOLWORTHS_NUM_OF_RETRIES", 3),
				DelayInMs:           getEnvInt("WOOLWORTHS_DELAY_IN_MS", 400),
				DelayMaxInMs:        getEnvInt("WOOLWORTHS_DELAY_MAX_IN_MS", 600),
				TimeoutInMs:         getEnvInt("WOOLWORTHS_TIMEOUT_IN_MS", 30000),
			},
		},
		PresentationSettings: presentation.PresentationSettings{
			Port: getEnvWithDefault("PORT", ":8080"),
		},
	}
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func getEnvWithDefault(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultValue
	}
	value, error := strconv.Atoi(raw)
	if error != nil {
		fmt.Printf("Warning: invalid value for %s, using default %d\n", key, defaultValue)
		return defaultValue
	}
	return value
}

func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}
