package main

import (
	"fmt"
	"path/filepath"
	"runtime"

	"onion.api/utils"
)

func loadEnvironmentVariables() {
	_, currentFile, _, _ := runtime.Caller(0)
	presentationDirectory := filepath.Dir(currentFile)
	envFilePath := filepath.Join(presentationDirectory, ".env")

	error := utils.LoadEnv(envFilePath)
	if error != nil {
		fmt.Printf("Warning: could not load .env file: %v\n", error)
	}
}
