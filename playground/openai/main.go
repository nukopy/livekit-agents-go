package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"
)

type stepTiming struct {
	name string
	d    time.Duration
}

func main() {
	var timings []stepTiming

	// load env
	t0 := time.Now()
	env, err := LoadEnv()
	if err != nil {
		log.Fatal(err)
	}
	noteStep(&timings, "LoadEnv", t0)

	// prepare
	ctx := context.Background()
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	// run response API
	fmt.Println("=== RunResponseAPI ===")

	t0 = time.Now()
	r1, byPhase, err := RunResponseAPI(env)
	if err != nil {
		log.Fatal(err)
	}
	noteStep(&timings, "RunResponseAPI", t0)
	printAssistantTextByPhase(byPhase)
	if err := enc.Encode(r1); err != nil {
		log.Fatal(err)
	}
	fmt.Println()

	// run response API streaming
	fmt.Println("=== RunResponseAPIStreaming ===")
	t0 = time.Now()
	r2, err := RunResponseAPIStreaming(env)
	if err != nil {
		log.Fatal(err)
	}
	noteStep(&timings, "RunResponseAPIStreaming", t0)
	if err := enc.Encode(r2); err != nil {
		log.Fatal(err)
	}
	fmt.Println()

	// transcribe audio file
	fmt.Println("=== Transcribe (audio/transcriptions) ===")
	t0 = time.Now()
	text, err := TranscribeAudioFile(ctx, env)
	if err != nil {
		log.Fatal(err)
	}
	noteStep(&timings, "TranscribeAudioFile", t0)
	fmt.Println(text)
	fmt.Println()

	// synthesize speech to file
	fmt.Println("=== Speech (audio/speech) ===")
	t0 = time.Now()
	if err := SynthesizeSpeechToFile(ctx, env, text); err != nil {
		log.Fatal(err)
	}
	noteStep(&timings, "SynthesizeSpeechToFile", t0)
	fmt.Println("wrote:", speechOutputPath)

	fmt.Println("=== Play speech.output.mp3 ===")
	t0 = time.Now()
	if err := PlaySpeechOutputFile(speechOutputPath); err != nil {
		log.Printf("playback: %v", err)
	} else {
		noteStep(&timings, "PlaySpeechOutputFile", t0)
	}
	fmt.Println()

	// print timing summary
	printTimingSummary(timings)
}

func printAssistantTextByPhase(p AssistantTextByPhase) {
	fmt.Println("--- phase: commentary ---")
	fmt.Println(p.Commentary)
	fmt.Println("--- phase: final_answer ---")
	fmt.Println(p.FinalAnswer)
	fmt.Println("--- phase: unphased ---")
	fmt.Println(p.Unphased)
}

// noteStep records duration since start and prints one line for this step.
func noteStep(timings *[]stepTiming, name string, start time.Time) {
	d := time.Since(start)
	*timings = append(*timings, stepTiming{name, d})
	fmt.Printf("%s: %s\n", name, formatDuration(d))
}

// printTimingSummary prints all step durations in a table plus total.
func printTimingSummary(timings []stepTiming) {
	fmt.Println("=== Timings ===")
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "step\tduration")

	var total time.Duration
	for _, s := range timings {
		total += s.d
		fmt.Fprintf(tw, "%s\t%s\n", s.name, formatDuration(s.d))
	}
	fmt.Fprintf(tw, "%s\t%s\n", "total", formatDuration(total))
	tw.Flush()
}

// formatDuration returns a short human-readable string for durations.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return d.Round(time.Microsecond).String()
	}
	return d.Round(time.Millisecond).String()
}
