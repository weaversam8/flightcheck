package fc

import (
	"encoding/json"
	"errors"
	"os"
)

// Config holds every tunable. Values in ~/.flightcheck/config.json override
// these defaults field by field; anything omitted keeps its default.
type Config struct {
	RateLimit  RateLimitConfig  `json:"rate_limit"`
	SRS        SRSConfig        `json:"srs"`
	Rep        RepConfig        `json:"rep"`
	Classifier ClassifierConfig `json:"classifier"`
}

type RateLimitConfig struct {
	// A rep becomes eligible after a random amount of *active* time in
	// [MinActiveMinutes, MaxActiveMinutes] since the previous rep.
	MinActiveMinutes float64 `json:"min_active_minutes"`
	MaxActiveMinutes float64 `json:"max_active_minutes"`
	// Gaps between hook events longer than this count only this much, so
	// lunch or a meeting doesn't count as active time.
	IdleCapMinutes float64 `json:"idle_cap_minutes"`
	// Minimum user turns (messages) since the previous rep.
	MinTurns int `json:"min_turns"`
	// Hard ceiling on reps per local calendar day.
	DailyCap int `json:"daily_cap"`
}

type SRSConfig struct {
	InitialEase     float64            `json:"initial_ease"`
	MinEase         float64            `json:"min_ease"`
	MaxIntervalDays float64            `json:"max_interval_days"`
	FailEasePenalty float64            `json:"fail_ease_penalty"`
	FailInterval    float64            `json:"fail_interval_days"`
	PlacementDays   map[string]float64 `json:"placement_days"`
}

type RepConfig struct {
	// Edits that hit a due block but are longer than this are refused with a
	// request to split them, so a rep stays small.
	MaxLines int `json:"max_lines"`
}

type ClassifierConfig struct {
	// "auto" (API if ANTHROPIC_API_KEY is set, else the claude CLI), "cli", or "api".
	Backend        string   `json:"backend"`
	CLICommand     string   `json:"cli_command"`
	CLIArgs        []string `json:"cli_args"`
	APIModel       string   `json:"api_model"`
	APIURL         string   `json:"api_url"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	MaxSnippetChar int      `json:"max_snippet_chars"`
}

func DefaultConfig() Config {
	return Config{
		RateLimit: RateLimitConfig{
			MinActiveMinutes: 15,
			MaxActiveMinutes: 30,
			IdleCapMinutes:   5,
			MinTurns:         8,
			DailyCap:         8,
		},
		SRS: SRSConfig{
			InitialEase:     2.5,
			MinEase:         1.3,
			MaxIntervalDays: 90,
			FailEasePenalty: 0.20,
			FailInterval:    1,
			PlacementDays: map[string]float64{
				"easy": 21, "good": 7, "hard": 2, "fail": 1,
			},
		},
		Rep: RepConfig{MaxLines: 30},
		Classifier: ClassifierConfig{
			Backend:        "auto",
			CLICommand:     "claude",
			CLIArgs:        []string{"-p", "--model", "haiku", "--output-format", "json"},
			APIModel:       "claude-haiku-5-5",
			APIURL:         "https://api.anthropic.com/v1/messages",
			TimeoutSeconds: 25,
			MaxSnippetChar: 12000,
		},
	}
}

// LoadConfig reads config.json over the defaults. A missing file is fine.
func LoadConfig() (Config, error) {
	cfg := DefaultConfig()
	b, err := os.ReadFile(ConfigPath())
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	// Unmarshal onto the defaults so omitted fields keep default values.
	if err := json.Unmarshal(b, &cfg); err != nil {
		return DefaultConfig(), err
	}
	return cfg, nil
}
