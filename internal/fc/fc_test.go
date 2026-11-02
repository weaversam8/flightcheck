package fc

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReviewPlacementAndGrowth(t *testing.T) {
	cfg := DefaultConfig().SRS
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	c := Review(nil, Easy, cfg, t0)
	if c.IntervalDays != 21 || c.Ease != 2.6 {
		t.Fatalf("easy placement: got %v days, ease %v", c.IntervalDays, c.Ease)
	}
	c = Review(c, Good, cfg, t0.Add(21*24*time.Hour))
	if c.IntervalDays != 54.6 || c.Ease != 2.6 { // 21 * 2.6
		t.Fatalf("good after easy: got %v days, ease %v", c.IntervalDays, c.Ease)
	}
	c = Review(c, Good, cfg, t0)
	if c.IntervalDays != 90 {
		t.Fatalf("cap: got %v", c.IntervalDays)
	}
	c = Review(c, Fail, cfg, t0)
	if c.IntervalDays != 1 || c.Ease != 2.4 || c.Lapses != 1 {
		t.Fatalf("fail: got %v days, ease %v, lapses %d", c.IntervalDays, c.Ease, c.Lapses)
	}
	c = Review(c, Hard, cfg, t0)
	if c.IntervalDays != 2.4 || c.Ease != 2.26 {
		t.Fatalf("hard after fail: got %v days, ease %v", c.IntervalDays, c.Ease)
	}
	for i := 0; i < 20; i++ {
		c = Review(c, Hard, cfg, t0)
	}
	if c.Ease != cfg.MinEase {
		t.Fatalf("ease floor: got %v", c.Ease)
	}
	if h := Review(nil, Hard, cfg, t0); h.IntervalDays != 2 {
		t.Fatalf("hard placement: got %v", h.IntervalDays)
	}
}

func TestRateLimitIdleCapAndTurns(t *testing.T) {
	cfg := DefaultConfig().RateLimit
	defer func(f func() float64) { randFloat = f }(randFloat)
	randFloat = func() float64 { return 0.5 } // target = 22.5 minutes
	s := &State{}
	s.init()
	t0 := time.Date(2026, 1, 1, 9, 0, 0, 0, time.Local)
	s.Touch(t0, cfg)
	if s.TargetActive != 22.5 {
		t.Fatalf("target: %v", s.TargetActive)
	}
	// A 2-hour lunch counts as 5 minutes.
	s.Touch(t0.Add(2*time.Hour), cfg)
	if s.ActiveSinceRep != 5 {
		t.Fatalf("idle cap: %v", s.ActiveSinceRep)
	}
	// 20 more minutes of steady work in 2-minute steps.
	tt := t0.Add(2 * time.Hour)
	for i := 0; i < 10; i++ {
		tt = tt.Add(2 * time.Minute)
		s.Touch(tt, cfg)
	}
	if open, why := s.GateOpen(tt, cfg); open || !strings.Contains(why, "turns") {
		t.Fatalf("should be closed on turns, got %v %q", open, why)
	}
	s.TurnsSinceRep = 8
	if open, why := s.GateOpen(tt, cfg); !open {
		t.Fatalf("should be open: %s", why)
	}
	s.StartRep(tt, cfg)
	if open, _ := s.GateOpen(tt, cfg); open {
		t.Fatal("should close after a rep starts")
	}
	s.RepsToday = cfg.DailyCap
	s.ActiveSinceRep, s.TurnsSinceRep = 100, 100
	if open, why := s.GateOpen(tt, cfg); open || !strings.Contains(why, "daily") {
		t.Fatalf("daily cap: %v %q", open, why)
	}
}

func TestSettingsMergePreservesOrderAndOtherHooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	orig := `{
  "model": "opus",
  "hooks": {
    "PostToolUse": [
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "prettier --write"}]}
    ]
  },
  "permissions": {"allow": ["Bash(npm test)"]}
}`
	os.WriteFile(path, []byte(orig), 0o644)
	if err := editHooks(path, true); err != nil {
		t.Fatal(err)
	}
	if err := editHooks(path, true); err != nil { // idempotent
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	s := string(b)
	if strings.Index(s, `"model"`) > strings.Index(s, `"hooks"`) || strings.Index(s, `"hooks"`) > strings.Index(s, `"permissions"`) {
		t.Fatalf("key order changed:\n%s", s)
	}
	if strings.Count(s, "hook pre-tool-use") != 1 || strings.Count(s, "hook user-prompt-submit") != 1 {
		t.Fatalf("expected exactly one of each hook:\n%s", s)
	}
	if !strings.Contains(s, "prettier --write") {
		t.Fatal("dropped the user's own hook")
	}
	if err := editHooks(path, false); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(path)
	s = string(b)
	if strings.Contains(s, "flightcheck hook") || strings.Contains(s, "PreToolUse") {
		t.Fatalf("uninstall left hooks behind:\n%s", s)
	}
	if !strings.Contains(s, "prettier --write") {
		t.Fatal("uninstall removed the user's own hook")
	}
}

func TestBlocksFilesParse(t *testing.T) {
	want := map[string]int{"ansible": 30, "elixir": 37, "erlang": 33, "go": 29, "rust": 31, "shell": 25}
	b, err := defaultsFS.ReadFile("defaults/languages.json")
	if err != nil {
		t.Fatal(err)
	}
	var langs Languages
	if err := json.Unmarshal(b, &langs); err != nil {
		t.Fatal(err)
	}
	if len(langs) != len(want) {
		t.Fatalf("want %d languages, got %d", len(want), len(langs))
	}
	for name, l := range langs {
		b, err := defaultsFS.ReadFile("defaults/blocks/" + l.BlocksFile)
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(t.TempDir(), l.BlocksFile)
		os.WriteFile(p, b, 0o644)
		blocks, err := LoadBlocks(p)
		if err != nil {
			t.Fatal(err)
		}
		if len(blocks) != want[name] {
			t.Fatalf("want %d %s blocks, got %d", want[name], name, len(blocks))
		}
		seen := map[string]bool{}
		for _, bl := range blocks {
			if bl.ID == "" || bl.Title == "" || bl.Description == "" || strings.Contains(bl.ID, " ") {
				t.Fatalf("%s: bad block %+v", name, bl)
			}
			if seen[bl.ID] {
				t.Fatalf("%s: duplicate id %s", name, bl.ID)
			}
			seen[bl.ID] = true
		}
		for _, id := range l.SkipBlocks {
			if !seen[id] {
				t.Fatalf("%s: skip_blocks names unknown block %s", name, id)
			}
		}
	}
}

func TestSeedMergesMissingLanguages(t *testing.T) {
	t.Setenv("FLIGHTCHECK_HOME", t.TempDir())
	if err := EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	custom := `{"elixir": {"extensions": [".ex"], "blocks_file": "mine.md"}}`
	os.WriteFile(LanguagesPath(), []byte(custom), 0o644)

	wrote, err := seedDefaults(false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(wrote, "\n"), "(added ansible, erlang, go, rust, shell)") {
		t.Fatalf("expected the other languages to be reported as added, got %v", wrote)
	}
	langs, err := LoadLanguages()
	if err != nil {
		t.Fatal(err)
	}
	if langs["elixir"].BlocksFile != "mine.md" || len(langs["elixir"].Extensions) != 1 {
		t.Fatalf("merge overwrote the user's elixir entry: %+v", langs["elixir"])
	}
	if langs["go"].BlocksFile != "go.md" || langs["shell"].BlocksFile != "shell.md" {
		t.Fatalf("missing merged languages: %+v", langs)
	}

	// Nothing left to add: the file isn't touched again.
	before, _ := os.ReadFile(LanguagesPath())
	wrote, err = seedDefaults(false)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(LanguagesPath())
	if len(wrote) != 0 || !bytes.Equal(before, after) {
		t.Fatalf("second seed changed things: %v", wrote)
	}
}

func TestSplitCommon(t *testing.T) {
	p, m, s := splitCommon("end\n", "  def parse(<<n::16>>), do: n\nend\n")
	if p != "" || m != "  def parse(<<n::16>>), do: n\n" || s != "end\n" {
		t.Fatalf("got %q | %q | %q", p, m, s)
	}
	p, m, s = splitCommon("defmodule A do\nend\n", "defmodule A do\n  @x 1\nend\n")
	if p != "defmodule A do\n" || m != "  @x 1\n" || s != "end\n" {
		t.Fatalf("got %q | %q | %q", p, m, s)
	}
	if _, m, _ := splitCommon("a\nb\n", "a\nb\n"); m != "" {
		t.Fatalf("identical: middle %q", m)
	}
}

func TestBashWriteDetection(t *testing.T) {
	f := "/repo/lib/parser.ex"
	cases := map[string]bool{
		"cat > /repo/lib/parser.ex <<EOF":            true,
		"sed -i 's/a/b/' lib/parser.ex":              true,
		"mix test":                                   false,
		"cat lib/parser.ex":                          false,
		"flightcheck grade 123 good --blocks x":      false,
		"flightcheck grade 1 good; echo > parser.ex": true,
	}
	for cmd, want := range cases {
		if got := bashWritesFile(cmd, f); got != want {
			t.Errorf("%q: got %v want %v", cmd, got, want)
		}
	}
}

func TestAutoModeEditsCountAsTurns(t *testing.T) {
	t.Setenv("FLIGHTCHECK_HOME", t.TempDir())
	if _, err := seedDefaults(false); err != nil {
		t.Fatal(err)
	}
	s := &session{t: t, id: "sess-auto"}
	edit := func(mode string) {
		s.hook("post-tool-use", map[string]any{
			"tool_name": "Edit", "permission_mode": mode,
			"tool_input": map[string]any{"file_path": "/tmp/x.go", "new_string": "x"},
		})
	}
	turns := func() int { st, _ := LoadState(); return st.TurnsSinceRep }

	for i := 0; i < 8; i++ {
		edit("default")
	}
	if n := turns(); n != 0 {
		t.Fatalf("edits outside auto mode shouldn't count, got %d turns", n)
	}
	for i := 0; i < 9; i++ {
		edit("auto")
	}
	if n := turns(); n != 2 {
		t.Fatalf("9 auto edits at 4 per turn should be 2 turns, got %d", n)
	}
	s.hook("user-prompt-submit", map[string]any{"prompt": "go", "permission_mode": "auto"})
	if n := turns(); n != 3 {
		t.Fatalf("user messages still count in auto mode, got %d turns", n)
	}

	os.WriteFile(ConfigPath(), []byte(`{"rate_limit": {"auto_edits_per_turn": 0}}`), 0o644)
	for i := 0; i < 8; i++ {
		edit("auto")
	}
	if n := turns(); n != 3 {
		t.Fatalf("auto_edits_per_turn 0 should disable counting, got %d turns", n)
	}
}

// ---- end-to-end: a simulated Claude Code session ----

type session struct {
	t   *testing.T
	id  string
	clk time.Time
}

func (s *session) hook(event string, in map[string]any) map[string]any {
	s.t.Helper()
	in["session_id"] = s.id
	b, _ := json.Marshal(in)
	var out bytes.Buffer
	RunHook(event, bytes.NewReader(b), &out)
	if out.Len() == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		s.t.Fatalf("%s output not JSON: %s", event, out.String())
	}
	return m
}

func (s *session) advance(d time.Duration) { s.clk = s.clk.Add(d) }

func decision(m map[string]any) (string, map[string]any) {
	if m == nil {
		return "", nil
	}
	h, _ := m["hookSpecificOutput"].(map[string]any)
	d, _ := h["permissionDecision"].(string)
	u, _ := h["updatedInput"].(map[string]any)
	return d, u
}

func TestEndToEndRep(t *testing.T) {
	home := t.TempDir()
	t.Setenv("FLIGHTCHECK_HOME", home)
	t.Setenv("FLIGHTCHECK_FAKE_CLASSIFY", "with_chains,genserver")
	defer func(f func() float64) { randFloat = f }(randFloat)
	randFloat = func() float64 { return 0 } // 15-minute target
	if _, err := seedDefaults(false); err != nil {
		t.Fatal(err)
	}

	s := &session{t: t, id: "sess-1", clk: time.Date(2026, 3, 2, 10, 0, 0, 0, time.Local)}
	now = func() time.Time { return s.clk }
	defer func() { now = time.Now }()

	repo := t.TempDir()
	file := filepath.Join(repo, "lib", "worker.ex")
	os.MkdirAll(filepath.Dir(file), 0o755)
	os.WriteFile(file, []byte("defmodule Worker do\n  # body\nend\n"), 0o644)
	edit := map[string]any{
		"tool_name": "Edit",
		"tool_input": map[string]any{
			"file_path":  file,
			"old_string": "  # body\n",
			"new_string": "  def run(x) do\n    with {:ok, y} <- parse(x) do\n      y\n    else\n      _ -> :error\n    end\n  end\n",
		},
	}

	// Fresh install: gate closed, edits pass through untouched.
	if d, _ := decision(s.hook("pre-tool-use", edit)); d != "" {
		t.Fatalf("expected passthrough before gate opens, got %q", d)
	}

	// Work for a while: 8 turns, 2 minutes apart.
	for i := 0; i < 8; i++ {
		s.advance(2 * time.Minute)
		s.hook("user-prompt-submit", map[string]any{"prompt": "keep going"})
	}
	s.advance(time.Minute)

	// Gate open: the edit becomes a stub.
	d, upd := decision(s.hook("pre-tool-use", edit))
	if d != "allow" || upd == nil {
		t.Fatalf("expected allow with updatedInput, got %q", d)
	}
	stub := upd["new_string"].(string)
	if !strings.Contains(stub, "TODO(flightcheck ") || !strings.HasPrefix(stub, "  # ") {
		t.Fatalf("bad stub: %q", stub)
	}
	// Simulate Claude Code applying the modified edit.
	content, _ := os.ReadFile(file)
	os.WriteFile(file, []byte(strings.Replace(string(content), "  # body\n", stub, 1)), 0o644)

	post := s.hook("post-tool-use", map[string]any{"tool_name": "Edit", "tool_input": upd})
	if post["decision"] != "block" || !strings.Contains(post["reason"].(string), "Do not write the code") {
		t.Fatalf("expected coaching block, got %v", post)
	}

	st, _ := LoadState()
	rep := st.Pending["sess-1"]
	if rep == nil || len(rep.Blocks) != 2 {
		t.Fatalf("pending rep missing: %+v", rep)
	}

	// Same turn: further edits are refused until the user replies.
	other := map[string]any{"tool_name": "Write", "tool_input": map[string]any{"file_path": filepath.Join(repo, "README.md"), "content": "x"}}
	if d, _ := decision(s.hook("pre-tool-use", other)); d != "deny" {
		t.Fatalf("expected deny for other file before user replies, got %q", d)
	}

	// User replies; other files are fine, the rep file is not.
	s.advance(3 * time.Minute)
	ctx := s.hook("user-prompt-submit", map[string]any{"prompt": "what does with/else do?"})
	if !strings.Contains(ctx["hookSpecificOutput"].(map[string]any)["additionalContext"].(string), "grade "+rep.ID) {
		t.Fatal("expected grading instructions in context")
	}
	if d, _ := decision(s.hook("pre-tool-use", other)); d != "" {
		t.Fatalf("other file should pass after user reply, got %q", d)
	}
	if d, _ := decision(s.hook("pre-tool-use", edit)); d != "deny" {
		t.Fatalf("rep file must stay blocked, got %q", d)
	}

	// Grading with the marker still present is refused.
	var out bytes.Buffer
	if err := GradeRep(&out, rep.ID, Good, nil); err == nil {
		t.Fatal("grade should refuse while the marker is present")
	}

	// User writes the code; agent grades only one block.
	content, _ = os.ReadFile(file)
	userCode := "  def run(x) do\n    with {:ok, y} <- parse(x), do: y, else: (_ -> :error)\n  end\n"
	os.WriteFile(file, []byte(strings.Replace(string(content), stub, userCode, 1)), 0o644)
	out.Reset()
	if err := GradeRep(&out, rep.ID, Good, []string{"with_chains"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "with_chains: next due in 7 days") || !strings.Contains(out.String(), "genserver: not exercised") {
		t.Fatalf("unexpected grade output:\n%s", out.String())
	}

	led, _ := LoadLedger()
	if c := led.Card("elixir", "with_chains"); c == nil || c.IntervalDays != 7 {
		t.Fatalf("with_chains card: %+v", c)
	}
	if led.Card("elixir", "genserver") != nil {
		t.Fatal("genserver should stay due")
	}

	// Fix-up edit right after grading is allowed and doesn't start a new rep.
	fix := map[string]any{"tool_name": "Edit", "tool_input": map[string]any{
		"file_path": file, "old_string": userCode, "new_string": "  def run(x) do\n    with {:ok, y} <- parse(x) do\n      y\n    else\n      _ -> :error\n    end\n  end\n",
	}}
	if d, _ := decision(s.hook("pre-tool-use", fix)); d != "" {
		t.Fatalf("fix-up should pass through, got %q", d)
	}
	s.hook("post-tool-use", fix)
	hist, _ := os.ReadFile(HistoryPath())
	if !strings.Contains(string(hist), `"type":"fixup"`) {
		t.Fatal("expected fixup logged in history")
	}
	if fileExists(pendingFile(rep.ID)) {
		t.Fatal("pending file should be cleaned up")
	}
}

func TestDeferRestoresOriginal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("FLIGHTCHECK_HOME", home)
	EnsureDirs()
	file := filepath.Join(t.TempDir(), "a.ex")
	os.WriteFile(file, []byte("before\n  # TODO(flightcheck x1): stub\nafter\n"), 0o644)
	st := &State{}
	st.init()
	st.Pending["s"] = &Rep{ID: "x1", SessionID: "s", File: file, Stub: "  # TODO(flightcheck x1): stub\n", Original: "  real code\n"}
	SaveState(st)
	var out bytes.Buffer
	if err := DeferRep(&out, "x1"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(file)
	if string(b) != "before\n  real code\nafter\n" {
		t.Fatalf("not restored: %q", b)
	}
	st, _ = LoadState()
	if len(st.Pending) != 0 {
		t.Fatal("pending not cleared")
	}
}
