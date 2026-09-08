package utils

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv(filePath string) error {
	return godotenv.Load(filePath)
}

func GetEnv(key string) string {
	return os.Getenv(key)
}

func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}
