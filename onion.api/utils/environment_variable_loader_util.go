package utils

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type RetailerEnvironmentVariables struct {
	ReferenceStore      string
	DegreeOfParallelism int
	NumOfRetries        int
	DelayInMs           int
	DelayMaxInMs        int
	TimeoutInMs         int
}

type EnvironmentVariables struct {
	DatabaseURL   string
	TestDatabaseURL string
	PakNSave    RetailerEnvironmentVariables
	Woolworths  RetailerEnvironmentVariables
}

func LoadEnvironmentVariables() EnvironmentVariables {
	error := godotenv.Load()
	if error != nil {
		fmt.Printf("Warning: could not load .env file: %v\n", error)
	}

	return EnvironmentVariables{
		DatabaseURL:       getEnv("DATABASE_URL"),
		TestDatabaseURL:   getEnv("TEST_DATABASE_URL"),
		PakNSave: RetailerEnvironmentVariables{
			ReferenceStore:      getEnv("PAKNSAVE_REFERENCE_STORE"),
			DegreeOfParallelism: getEnvInt("PAKNSAVE_DEGREE_OF_PARALLELISM", 12),
			NumOfRetries:        getEnvInt("PAKNSAVE_NUM_OF_RETRIES", 3),
			DelayInMs:           getEnvInt("PAKNSAVE_DELAY_IN_MS", 400),
			DelayMaxInMs:        getEnvInt("PAKNSAVE_DELAY_MAX_IN_MS", 600),
			TimeoutInMs:         getEnvInt("PAKNSAVE_TIMEOUT_IN_MS", 30000),
		},
		Woolworths: RetailerEnvironmentVariables{
			ReferenceStore:      getEnv("WOOLWORTHS_REFERENCE_STORE"),
			DegreeOfParallelism: getEnvInt("WOOLWORTHS_DEGREE_OF_PARALLELISM", 6),
			NumOfRetries:        getEnvInt("WOOLWORTHS_NUM_OF_RETRIES", 3),
			DelayInMs:           getEnvInt("WOOLWORTHS_DELAY_IN_MS", 400),
			DelayMaxInMs:        getEnvInt("WOOLWORTHS_DELAY_MAX_IN_MS", 600),
			TimeoutInMs:         getEnvInt("WOOLWORTHS_TIMEOUT_IN_MS", 30000),
		},
	}
}

func getEnv(key string) string {
	return os.Getenv(key)
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
