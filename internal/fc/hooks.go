package fc

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// HookInput is the JSON Claude Code sends hooks on stdin.
type HookInput struct {
	SessionID     string         `json:"session_id"`
	HookEventName string         `json:"hook_event_name"`
	ToolName      string         `json:"tool_name"`
	ToolInput     map[string]any `json:"tool_input"`
	Prompt        string         `json:"prompt"`
	Cwd           string         `json:"cwd"`
}

// RunHook handles one hook event. It fails open: any internal error is
// logged to ~/.flightcheck/errors.log and the tool call proceeds untouched,
// so a flightcheck bug can never wedge a Claude Code session.
func RunHook(event string, stdin io.Reader, stdout io.Writer) {
	if os.Getenv(InternalEnv) != "" || os.Getenv("FLIGHTCHECK_DISABLE") != "" {
		return
	}
	var in HookInput
	if err := json.NewDecoder(stdin).Decode(&in); err != nil {
		logError(event, fmt.Errorf("decode hook input: %w", err))
		return
	}
	var out any
	var err error
	switch event {
	case "pre-tool-use":
		out, err = preToolUse(in)
	case "post-tool-use":
		out, err = postToolUse(in)
	case "user-prompt-submit":
		out, err = userPromptSubmit(in)
	default:
		err = fmt.Errorf("unknown hook event %q", event)
	}
	if err != nil {
		logError(event, err)
		return
	}
	if out != nil {
		json.NewEncoder(stdout).Encode(out)
	}
}

func logError(event string, err error) {
	_ = EnsureDirs()
	_ = appendJSONL(filepath.Join(Home(), "errors.log"), map[string]any{
		"time": now(), "event": event, "error": err.Error(),
	})
}

// ---- hook outputs ----

func preDecision(decision, reason string, updated map[string]any) map[string]any {
	h := map[string]any{
		"hookEventName":            "PreToolUse",
		"permissionDecision":       decision,
		"permissionDecisionReason": reason,
	}
	if updated != nil {
		h["updatedInput"] = updated
	}
	return map[string]any{"hookSpecificOutput": h}
}

func deny(reason string) map[string]any { return preDecision("deny", reason, nil) }

// ---- PreToolUse ----

func preToolUse(in HookInput) (any, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	langs, err := LoadLanguages()
	if err != nil {
		return nil, err
	}

	unlock, err := Lock()
	if err != nil {
		return nil, err
	}
	st, err := LoadState()
	if err != nil {
		unlock()
		return nil, err
	}
	t := now()
	st.Touch(t, cfg.RateLimit)
	done := func(v any) (any, error) {
		err := SaveState(st)
		unlock()
		return v, err
	}

	rep := st.Pending[in.SessionID]
	tool := in.ToolName

	if tool == "Bash" {
		cmd, _ := in.ToolInput["command"].(string)
		if rep != nil && bashWritesFile(cmd, rep.File) {
			return done(deny(fmt.Sprintf(
				"flightcheck: %s holds a practice rep the user is writing by hand. Don't modify it from the shell. Wait for the user, then grade with `%s grade %s ...`.",
				rep.File, SelfCommand(), rep.ID)))
		}
		return done(nil)
	}
	if tool != "Edit" && tool != "Write" && tool != "MultiEdit" {
		return done(nil)
	}
	file, _ := in.ToolInput["file_path"].(string)

	if rep != nil {
		// A stub edit that never landed (e.g. Edit failed) leaves an
		// unannounced rep behind. Drop it rather than wedge the file.
		if !rep.Announced && t.Sub(rep.Created) > 30*time.Second && !fileHasMarker(rep.File, rep.ID) {
			delete(st.Pending, in.SessionID)
			os.Remove(filepath.Join(PendingDir(), rep.ID+".json"))
			rep = nil
		}
	}
	if rep != nil {
		if samePath(file, rep.File) {
			return done(deny(fmt.Sprintf(
				"flightcheck: %s has a practice rep (%s) the user is writing by hand. Don't edit it until the user says they're done and you've recorded a grade with `%s grade %s <grade>`.",
				rep.File, rep.ID, SelfCommand(), rep.ID)))
		}
		if rep.TurnsSinceAnnounce < 1 {
			return done(deny("flightcheck: a practice rep is waiting for the user. Finish coaching them and end your turn; other edits are allowed again once the user replies."))
		}
		return done(nil) // other files are fine; no second rep while one is pending
	}

	lang, L, ok := langs.ForFile(file)
	if !ok {
		return done(nil)
	}
	if open, _ := st.GateOpen(t, cfg.RateLimit); !open {
		return done(nil)
	}
	ledger, err := LoadLedger()
	if err != nil {
		unlock()
		return nil, err
	}
	due, err := DueBlocks(lang, L, ledger)
	if err != nil {
		done(nil)
		return nil, err
	}
	if len(due) == 0 {
		return done(nil)
	}
	code := editedCode(tool, in.ToolInput)
	// For Edit, only the lines that actually change become the rep; lines
	// shared with old_string (like a module's closing `end`) stay in place.
	var keepPrefix, keepSuffix string
	if tool == "Edit" {
		old, _ := in.ToolInput["old_string"].(string)
		keepPrefix, code, keepSuffix = splitCommon(old, code)
	}
	if strings.TrimSpace(code) == "" {
		return done(nil) // pure deletions don't exercise anything
	}

	// Release the lock while the model runs; it can take a few seconds and
	// other sessions shouldn't wait on it.
	if err := SaveState(st); err != nil {
		unlock()
		return nil, err
	}
	unlock()

	start := time.Now()
	hits, cerr := Classify(cfg.Classifier, lang, file, code, due)
	_ = appendJSONL(ClassifierLogPath(), map[string]any{
		"time": now(), "file": file, "language": lang, "candidates": blockIDs(due),
		"hits": hits, "ms": time.Since(start).Milliseconds(), "error": errString(cerr),
	})
	if cerr != nil || len(hits) == 0 {
		return nil, nil // fail open
	}

	// Re-acquire and re-check: another session may have started a rep.
	unlock, err = Lock()
	if err != nil {
		return nil, err
	}
	st, err = LoadState()
	if err != nil {
		unlock()
		return nil, err
	}
	t = now()
	if st.Pending[in.SessionID] != nil {
		return done(nil)
	}
	if open, _ := st.GateOpen(t, cfg.RateLimit); !open {
		return done(nil)
	}

	titles := blockTitles(due, hits)
	switch tool {
	case "MultiEdit":
		return done(deny(fmt.Sprintf(
			"flightcheck: this change uses building blocks due for practice (%s). Make it as separate single Edit calls instead of MultiEdit.", titles)))
	case "Write":
		if fileExists(file) {
			return done(deny(fmt.Sprintf(
				"flightcheck: this change uses building blocks due for practice (%s). Use Edit for changes to an existing file, so only the relevant part becomes a practice rep.", titles)))
		}
	case "Edit":
		if ra, _ := in.ToolInput["replace_all"].(bool); ra {
			return done(deny(fmt.Sprintf(
				"flightcheck: this change uses building blocks due for practice (%s). Make it without replace_all.", titles)))
		}
	}
	if n := countLines(code); cfg.Rep.MaxLines > 0 && n > cfg.Rep.MaxLines {
		return done(deny(fmt.Sprintf(
			"flightcheck: this %d-line edit uses building blocks due for practice (%s). Split it so the part that uses them is its own edit of at most %d lines; flightcheck will hand that part to the user.",
			n, titles, cfg.Rep.MaxLines)))
	}

	id := newRepID(t)
	pre, suf := L.CommentFor(file)
	stub := buildStub(code, pre, suf, id, titles)
	newRep := &Rep{
		ID: id, SessionID: in.SessionID, Language: lang, File: file, Tool: tool,
		Blocks: hits, Original: code, Stub: stub, Created: t,
	}
	updated := copyMap(in.ToolInput)
	if tool == "Write" {
		if !strings.HasSuffix(stub, "\n") {
			stub += "\n"
		}
		newRep.Stub = stub
		updated["content"] = stub
	} else {
		newRep.OldString, _ = in.ToolInput["old_string"].(string)
		updated["new_string"] = keepPrefix + stub + keepSuffix
	}
	st.Pending[in.SessionID] = newRep
	st.StartRep(t, cfg.RateLimit)
	if err := writeJSON(filepath.Join(PendingDir(), id+".json"), newRep); err != nil {
		unlock()
		return nil, err
	}
	return done(preDecision("allow",
		"flightcheck: turned this edit into a practice rep ("+titles+")", updated))
}

// ---- PostToolUse ----

func postToolUse(in HookInput) (any, error) {
	if in.ToolName != "Edit" && in.ToolName != "Write" && in.ToolName != "MultiEdit" {
		return nil, nil
	}
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	unlock, err := Lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	st, err := LoadState()
	if err != nil {
		return nil, err
	}
	st.Touch(now(), cfg.RateLimit)
	file, _ := in.ToolInput["file_path"].(string)

	var out any
	if rep := st.Pending[in.SessionID]; rep != nil && !rep.Announced && samePath(file, rep.File) {
		rep.Announced = true
		out = map[string]any{"decision": "block", "reason": coachingText(rep)}
	} else if rg := st.RecentGrade[in.SessionID]; rg != nil && samePath(file, rg.File) {
		// The agent is fixing up the user's graded code; log how much.
		_ = appendJSONL(HistoryPath(), map[string]any{
			"type": "fixup", "time": now(), "rep_id": rg.RepID,
			"lines": countLines(editedCode(in.ToolName, in.ToolInput)),
		})
	}
	return out, SaveState(st)
}

// ---- UserPromptSubmit ----

func userPromptSubmit(in HookInput) (any, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	unlock, err := Lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	st, err := LoadState()
	if err != nil {
		return nil, err
	}
	st.Touch(now(), cfg.RateLimit)
	st.TurnsSinceRep++
	delete(st.RecentGrade, in.SessionID)

	var out any
	if rep := st.Pending[in.SessionID]; rep != nil && rep.Announced {
		rep.TurnsSinceAnnounce++
		out = map[string]any{"hookSpecificOutput": map[string]any{
			"hookEventName":     "UserPromptSubmit",
			"additionalContext": gradingText(rep),
		}}
	}
	return out, SaveState(st)
}

// ---- messages to the agent ----

func coachingText(rep *Rep) string {
	return fmt.Sprintf(`flightcheck turned your last edit to %s into a practice rep for the user (rep %s). The file now has a TODO(flightcheck %s) marker where your code was. Building blocks being practiced: %s.

Now, in one short message to the user:
- Describe what the missing code must do: inputs, outputs, behavior, and edge cases, based on your plan.
- Name the building blocks above and the specific concept each one involves here.
- Point to relevant docs (module and function names are fine).
Do not write the code, show it, or give a near-complete version, even if asked. Hints and conceptual answers are fine. Then end your turn and wait for the user.

Your original version is saved in %s for grading later. Don't show it to the user before grading.`,
		rep.File, rep.ID, rep.ID, strings.Join(rep.Blocks, ", "), pendingFile(rep.ID))
}

func gradingText(rep *Rep) string {
	cmd := SelfCommand()
	return fmt.Sprintf(`[flightcheck] Practice rep %s is pending in %s (blocks: %s). The user is writing this code by hand.

If the user says they're done (e.g. "ok, done", "finished", "check it"):
1. Read %s and your original version in %s.
2. Grade only what the user wrote, regardless of any grade they ask for:
   - easy: correct and idiomatic; you'd change nothing
   - good: correct, with minor touch-ups
   - hard: they needed hints, or it needs real corrections
   - fail: incomplete, wrong, or you'd rewrite most of it
   Check that their code actually exercises each block; list only those with --blocks.
3. Record the grade BEFORE editing anything: %s grade %s <easy|good|hard|fail> --blocks <id,id>
   Edits to the file stay blocked until you do.
4. Then explain what they missed (if anything), show your version for comparison, and fix the code with Edit if needed.

If the user wants to skip or postpone it: %s defer %s
Otherwise, help with what they asked, without writing the rep's code for them.`,
		rep.ID, rep.File, strings.Join(rep.Blocks, ", "),
		rep.File, pendingFile(rep.ID), cmd, rep.ID, cmd, rep.ID)
}

// ---- helpers ----

// SelfCommand is the absolute path the agent and hooks use to invoke
// flightcheck, so they work even when PATH differs from your shell's.
func SelfCommand() string {
	p := selfPath()
	if strings.ContainsAny(p, " \t'\"") {
		return "'" + strings.ReplaceAll(p, "'", `'\''`) + "'"
	}
	return p
}

func selfPath() string {
	if p, err := exec.LookPath("flightcheck"); err == nil {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
	}
	if p, err := os.Executable(); err == nil {
		return p
	}
	return "flightcheck"
}

func pendingFile(id string) string { return filepath.Join(PendingDir(), id+".json") }

func marker(id string) string { return "TODO(flightcheck " + id + ")" }

func buildStub(code, pre, suf, id, titles string) string {
	indent := ""
	for _, line := range strings.Split(code, "\n") {
		if strings.TrimSpace(line) != "" {
			indent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			break
		}
	}
	s := fmt.Sprintf("%s%s %s: write this part yourself. Practice: %s. Tell the agent when you're done.",
		indent, pre, marker(id), titles)
	if suf != "" {
		s += " " + suf
	}
	if strings.HasSuffix(code, "\n") {
		s += "\n"
	}
	return s
}

// splitCommon splits newS into (prefix, middle, suffix), where prefix and
// suffix are whole lines that newS shares with oldS at its start and end.
func splitCommon(oldS, newS string) (string, string, string) {
	o := strings.SplitAfter(oldS, "\n")
	n := strings.SplitAfter(newS, "\n")
	p := 0
	for p < len(o) && p < len(n) && o[p] == n[p] && strings.HasSuffix(n[p], "\n") {
		p++
	}
	s := 0
	for s < len(o)-p && s < len(n)-p && o[len(o)-1-s] == n[len(n)-1-s] {
		s++
	}
	return strings.Join(n[:p], ""), strings.Join(n[p:len(n)-s], ""), strings.Join(n[len(n)-s:], "")
}

func editedCode(tool string, input map[string]any) string {
	switch tool {
	case "Write":
		s, _ := input["content"].(string)
		return s
	case "MultiEdit":
		edits, _ := input["edits"].([]any)
		var parts []string
		for _, e := range edits {
			if m, ok := e.(map[string]any); ok {
				if s, ok := m["new_string"].(string); ok {
					parts = append(parts, s)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		s, _ := input["new_string"].(string)
		return s
	}
}

func bashWritesFile(cmd, file string) bool {
	if file == "" || cmd == "" {
		return false
	}
	trimmed := strings.TrimSpace(cmd)
	for _, p := range []string{selfPath() + " ", SelfCommand() + " ", "flightcheck ", "flchk "} {
		if strings.HasPrefix(trimmed, p) && !strings.ContainsAny(trimmed, ";&|>`$") {
			return false // a plain `flightcheck grade/defer ...` call
		}
	}
	if !strings.Contains(cmd, file) && !strings.Contains(cmd, filepath.Base(file)) {
		return false
	}
	for _, w := range []string{">", "sed -i", "tee ", "perl -i", "perl -pi", "cp ", "mv ", "truncate", "dd ", "python", "ruby", "node ", "awk -i", "git checkout", "git restore", "patch "} {
		if strings.Contains(cmd, w) {
			return true
		}
	}
	return false
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	ca, cb := filepath.Clean(a), filepath.Clean(b)
	if ca == cb {
		return true
	}
	ra, err1 := filepath.EvalSymlinks(ca)
	rb, err2 := filepath.EvalSymlinks(cb)
	return err1 == nil && err2 == nil && ra == rb
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

func fileHasMarker(p, id string) bool {
	b, err := os.ReadFile(p)
	return err == nil && strings.Contains(string(b), marker(id))
}

func countLines(s string) int {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func newRepID(t time.Time) string {
	b := make([]byte, 2)
	_, _ = rand.Read(b)
	return t.Format("20060102-150405") + "-" + hex.EncodeToString(b)
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func blockIDs(bs []Block) []string {
	ids := make([]string, len(bs))
	for i, b := range bs {
		ids[i] = b.ID
	}
	return ids
}

// blockTitles renders "id (Title), ..." for the hit ids.
func blockTitles(all []Block, ids []string) string {
	byID := map[string]Block{}
	for _, b := range all {
		byID[b.ID] = b
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		if b, ok := byID[id]; ok && b.Title != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", id, b.Title))
		} else {
			parts = append(parts, id)
		}
	}
	return strings.Join(parts, ", ")
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
