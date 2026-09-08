package utils

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type EnvironmentVariables struct {
	DatabaseURL     string
	TestDatabaseURL string
}

func LoadEnvironmentVariables() EnvironmentVariables {
	error := godotenv.Load()
	if error != nil {
		fmt.Printf("Warning: could not load .env file: %v\n", error)
	}

	return EnvironmentVariables{
		DatabaseURL:     getEnv("DATABASE_URL"),
		TestDatabaseURL: getEnv("TEST_DATABASE_URL"),
	}
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}
