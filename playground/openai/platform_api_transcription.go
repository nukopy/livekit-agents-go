package main

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
)

const (
	transcriptionModel          = "gpt-4o-mini-transcribe-2025-12-15"
	transcriptionInputPath      = "playground/openai/audio/transcription.input.mp3"
	transcriptionLanguage       = "en"
)

// TranscribeAudioFile calls POST /v1/audio/transcriptions (see Audio API — create transcription).
func TranscribeAudioFile(ctx context.Context, env Env) (string, error) {
	client := newClient(env)
	f, err := os.Open(transcriptionInputPath)
	if err != nil {
		return "", fmt.Errorf("open audio file: %w", err)
	}
	defer f.Close()

	res, err := client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		File:           f,
		Model:          transcriptionModel,
		Language:       openai.String(transcriptionLanguage),
		ResponseFormat: openai.AudioResponseFormatJSON,
	})
	if err != nil {
		return "", err
	}
	tr := res.AsTranscription()
	return tr.Text, nil
}
