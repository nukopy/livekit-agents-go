package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

const responseAPIModel = "gpt-5-nano-2025-08-07"

// NBAPlayer is one Lakers player row from the Responses API (structured JSON).
type NBAPlayer struct {
	Name      string `json:"name"`
	BeginDate string `json:"begin_date"`
	EndDate   string `json:"end_date"`
	IsActive  bool   `json:"is_active"`
}

// NBAPlayersResponse is the root object the model must return.
type NBAPlayersResponse struct {
	Players []NBAPlayer `json:"players"`
}

func nbaPlayersJSONSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"players": map[string]any{
				"type":     "array",
				"minItems": 5,
				"maxItems": 5,
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"name": map[string]any{
							"type":        "string",
							"description": "Player name (Lakers only)",
						},
						"begin_date": map[string]any{
							"type":        "string",
							"description": "First Lakers season start YYYY-MM-DD",
						},
						"end_date": map[string]any{
							"type":        "string",
							"description": "Last date as a Laker YYYY-MM-DD; empty if still on Lakers",
						},
						"is_active": map[string]any{
							"type":        "boolean",
							"description": "On Lakers roster today",
						},
					},
					"required": []string{"name", "begin_date", "end_date", "is_active"},
				},
			},
		},
		"required": []string{"players"},
	}
}

const nbaPlayersPrompt = `Exactly 5 Los Angeles Lakers players (mix eras).
begin_date / end_date: YYYY-MM-DD as a Laker;
end_date "" if still on the team;
is_active = on roster now.`

func nbaPlayersResponseParams() responses.ResponseNewParams {
	// gpt-5-nano rejects temperature (see API error: unsupported with this model).
	// Reasoning models use reasoning.effort instead: https://platform.openai.com/docs/guides/reasoning
	return responses.ResponseNewParams{
		Model: responseAPIModel,
		Reasoning: shared.ReasoningParam{
			Effort: shared.ReasoningEffortLow,
		},
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(nbaPlayersPrompt),
		},
		Text: responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigParamOfJSONSchema(
				"nba_players",
				nbaPlayersJSONSchema(),
			),
		},
	}
}

// AssistantTextByPhase is assistant output_text grouped by [responses.ResponseOutputMessage.Phase].
// Some models omit phase (unphased); treat that like a final answer for decoding priority.
type AssistantTextByPhase struct {
	Commentary  string
	FinalAnswer string
	Unphased    string
}

func assistantTextByPhase(resp *responses.Response) AssistantTextByPhase {
	var out AssistantTextByPhase
	if resp == nil {
		return out
	}
	var c, f, unph strings.Builder
	for i := range resp.Output {
		item := &resp.Output[i]
		if item.Type != "message" {
			continue
		}
		msg := item.AsMessage()
		var part strings.Builder
		for _, co := range msg.Content {
			if co.Type == "output_text" {
				part.WriteString(co.Text)
			}
		}
		s := part.String()
		if s == "" {
			continue
		}
		switch msg.Phase {
		case responses.ResponseOutputMessagePhaseCommentary:
			c.WriteString(s)
		case responses.ResponseOutputMessagePhaseFinalAnswer:
			f.WriteString(s)
		default:
			unph.WriteString(s)
		}
	}
	out.Commentary = c.String()
	out.FinalAnswer = f.String()
	out.Unphased = unph.String()
	return out
}

func (p AssistantTextByPhase) textForJSONDecode(resp *responses.Response) string {
	if p.FinalAnswer != "" {
		return p.FinalAnswer
	}
	if p.Unphased != "" {
		return p.Unphased
	}
	if p.Commentary != "" {
		return p.Commentary
	}
	if resp == nil {
		return ""
	}
	return resp.OutputText()
}

// RunResponseAPI calls the Responses API with structured JSON (json_schema) and returns 5 Lakers players
// plus assistant text split by phase (for logging / debugging).
func RunResponseAPI(env Env) (*NBAPlayersResponse, AssistantTextByPhase, error) {
	client := newClient(env)

	ctx := context.Background()
	resp, err := client.Responses.New(ctx, nbaPlayersResponseParams())
	if err != nil {
		return nil, AssistantTextByPhase{}, err
	}

	byPhase := assistantTextByPhase(resp)
	var out NBAPlayersResponse
	text := byPhase.textForJSONDecode(resp)
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, byPhase, fmt.Errorf("decode model JSON: %w", err)
	}
	return &out, byPhase, nil
}

// RunResponseAPIStreaming is like [RunResponseAPI] but streams; text deltas go to stdout, then JSON is parsed.
func RunResponseAPIStreaming(env Env) (*NBAPlayersResponse, error) {
	client := newClient(env)

	ctx := context.Background()
	stream := client.Responses.NewStreaming(ctx, nbaPlayersResponseParams())
	defer stream.Close()

	fmt.Println("--- streaming ---")

	var text strings.Builder
	for stream.Next() {
		ev := stream.Current()
		switch e := ev.AsAny().(type) {
		case responses.ResponseTextDeltaEvent:
			text.WriteString(e.Delta)
			fmt.Print(e.Delta)
		case responses.ResponseErrorEvent:
			return nil, fmt.Errorf("stream error: %s (%s)", e.Message, e.Code)
		case responses.ResponseFailedEvent:
			return nil, fmt.Errorf("response failed: status=%s %s: %s", e.Response.Status, e.Response.Error.Code, e.Response.Error.Message)
		}
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	fmt.Println()
	fmt.Println("--- final output ---")

	var out NBAPlayersResponse
	if err := json.Unmarshal([]byte(text.String()), &out); err != nil {
		return nil, fmt.Errorf("decode model JSON: %w", err)
	}
	return &out, nil
}
