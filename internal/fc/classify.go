package fc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// InternalEnv is set on the nested `claude -p` classifier process so that
// flightcheck's own hooks no-op inside it.
const InternalEnv = "FLIGHTCHECK_INTERNAL"

// fakeEnv lets tests (and dry runs) bypass the model: a comma-separated
// list of block ids to report, or "none".
const fakeEnv = "FLIGHTCHECK_FAKE_CLASSIFY"

// Classify asks a small model which of the candidate (due) blocks the code
// actually uses. It only ever sees the code being written and the block
// definitions, never the agent's commentary, so the agent can't talk it out
// of a tag. Returned ids are always a subset of the candidates.
func Classify(cfg ClassifierConfig, lang, file, code string, candidates []Block) ([]string, error) {
	if v, ok := os.LookupEnv(fakeEnv); ok {
		return filterIDs(strings.Split(v, ","), candidates), nil
	}
	if len(code) > cfg.MaxSnippetChar && cfg.MaxSnippetChar > 0 {
		code = code[:cfg.MaxSnippetChar]
	}
	prompt := buildPrompt(lang, file, code, candidates)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	backend := cfg.Backend
	if backend == "" || backend == "auto" {
		backend = "cli"
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			backend = "api"
		}
	}
	var text string
	var err error
	switch backend {
	case "cli":
		text, err = classifyCLI(ctx, cfg, prompt)
	case "api":
		text, err = classifyAPI(ctx, cfg, prompt)
	default:
		return nil, fmt.Errorf("unknown classifier backend %q", backend)
	}
	if err != nil {
		return nil, err
	}
	ids, err := parseBlocks(text)
	if err != nil {
		return nil, err
	}
	return filterIDs(ids, candidates), nil
}

func buildPrompt(lang, file, code string, candidates []Block) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You label code with %s language building blocks. Do not use any tools.\n\n", lang)
	if hasBeginnerBlock(candidates) {
		b.WriteString("These are beginner skills. A block counts if someone writing the code below by hand would have to write that syntax themselves, even in one small, ordinary use. Incidental mentions, or code merely near the construct, do not count. When unsure, leave it out.\n\n")
	} else {
		b.WriteString("A block counts only if the code below substantively exercises it: someone writing this code by hand would need that skill. Incidental mentions, a single trivial call, or code merely near the construct do not count. When unsure, leave it out.\n\n")
	}
	b.WriteString("Candidate blocks:\n")
	for _, bl := range candidates {
		fmt.Fprintf(&b, "- %s (%s): %s\n", bl.ID, bl.Title, oneLine(bl.Description))
	}
	fmt.Fprintf(&b, "\nFile: %s\n<code>\n%s\n</code>\n\n", file, code)
	b.WriteString(`Reply with only a JSON object, no prose: {"blocks": ["id", ...]}. Use [] if none apply.`)
	return b.String()
}

func hasBeginnerBlock(blocks []Block) bool {
	for _, b := range blocks {
		if b.Level == Beginner {
			return true
		}
	}
	return false
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func classifyCLI(ctx context.Context, cfg ClassifierConfig, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, cfg.CLICommand, cfg.CLIArgs...)
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Env = append(os.Environ(), InternalEnv+"=1")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w: %s", cfg.CLICommand, err, strings.TrimSpace(errb.String()))
	}
	// With --output-format json the CLI wraps the reply: {"result": "..."}.
	var wrapped struct {
		Result  string `json:"result"`
		IsError bool   `json:"is_error"`
	}
	if json.Unmarshal(out.Bytes(), &wrapped) == nil && wrapped.Result != "" {
		if wrapped.IsError {
			return "", errors.New("claude CLI returned an error: " + wrapped.Result)
		}
		return wrapped.Result, nil
	}
	return out.String(), nil
}

func classifyAPI(ctx context.Context, cfg ClassifierConfig, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":      cfg.APIModel,
		"max_tokens": 300,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	})
	req, err := http.NewRequestWithContext(ctx, "POST", cfg.APIURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", os.Getenv("ANTHROPIC_API_KEY"))
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("anthropic API %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var msg struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, c := range msg.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return sb.String(), nil
}

// parseBlocks pulls the first {...} object out of the model's reply.
func parseBlocks(text string) ([]string, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("no JSON in classifier reply: %q", truncate(text, 200))
	}
	var v struct {
		Blocks []string `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &v); err != nil {
		return nil, fmt.Errorf("bad classifier JSON: %w", err)
	}
	return v.Blocks, nil
}

func filterIDs(ids []string, candidates []Block) []string {
	ok := map[string]bool{}
	for _, c := range candidates {
		ok[c.ID] = true
	}
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if ok[id] && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
