package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	OpenAIAPIKey string
}

func LoadEnv() (Env, error) {
	// load env
	err := godotenv.Load()
	if err != nil {
		return Env{}, fmt.Errorf("load env: %w", err)
	}

	// get openai api key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return Env{}, fmt.Errorf("OPENAI_API_KEY is required")
	}

	return Env{OpenAIAPIKey: apiKey}, nil
}
