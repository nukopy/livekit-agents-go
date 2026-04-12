package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/openai/openai-go/v3"
)

const (
	ttsModel         = "gpt-4o-mini-tts-2025-12-15"
	ttsVoiceID       = "echo"
	speechOutputPath = "playground/openai/audio/speech.output.mp3"
)

// SynthesizeSpeechToFile calls POST /v1/audio/speech (see Audio API — create speech) and writes audio to speechOutputPath.
func SynthesizeSpeechToFile(ctx context.Context, env Env, text string) error {
	client := newClient(env)
	resp, err := client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Model: ttsModel,
		Voice: openai.AudioSpeechNewParamsVoiceUnion{
			OfString: openai.String(ttsVoiceID),
		},
		Input:          text,
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(speechOutputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("write audio: %w", err)
	}
	return nil
}

// PlaySpeechOutputFile plays an audio file after synthesis. On macOS it uses /usr/bin/afplay.
func PlaySpeechOutputFile(path string) error {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("afplay", path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("PlaySpeechOutputFile: no built-in player on GOOS=%s (open %q manually)", runtime.GOOS, path)
	}
}
