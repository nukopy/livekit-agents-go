package main

import (
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const openAIAPIBaseURL = "https://api.openai.com/v1"

func newClient(env Env) openai.Client {
	return openai.NewClient(
		option.WithAPIKey(env.OpenAIAPIKey),
		option.WithBaseURL(openAIAPIBaseURL),
	)
}
