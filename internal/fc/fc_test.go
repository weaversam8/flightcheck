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
	if c.IntervalDays != 2 || c.Ease != 2.6 {
		t.Fatalf("easy placement: got %v days, ease %v", c.IntervalDays, c.Ease)
	}
	c = Review(c, Good, cfg, t0.Add(48*time.Hour))
	if c.IntervalDays != 5.2 || c.Ease != 2.6 { // 2 * 2.6
		t.Fatalf("good after easy: got %v days, ease %v", c.IntervalDays, c.Ease)
	}
	long := *c
	long.IntervalDays = 80
	if capped := Review(&long, Good, cfg, t0); capped.IntervalDays != 90 {
		t.Fatalf("cap: got %v", capped.IntervalDays)
	}
	c = Review(c, Fail, cfg, t0)
	if c.IntervalDays != 0.17 || c.Ease != 2.4 || c.Lapses != 1 {
		t.Fatalf("fail: got %v days, ease %v, lapses %d", c.IntervalDays, c.Ease, c.Lapses)
	}
	c = Review(c, Hard, cfg, t0)
	if c.IntervalDays != 0.41 || c.Ease != 2.26 { // 0.17 * 2.4, no 1-day floor
		t.Fatalf("hard after fail: got %v days, ease %v", c.IntervalDays, c.Ease)
	}
	for i := 0; i < 20; i++ {
		c = Review(c, Hard, cfg, t0)
	}
	if c.Ease != cfg.MinEase {
		t.Fatalf("ease floor: got %v", c.Ease)
	}
	h := Review(nil, Hard, cfg, t0)
	if h.IntervalDays != 0.33 {
		t.Fatalf("hard placement: got %v", h.IntervalDays)
	}
	if g := Review(h, Good, cfg, t0); g.IntervalDays != 0.78 { // 0.33 * 2.36
		t.Fatalf("sub-day growth: got %v", g.IntervalDays)
	}
	if g := Review(nil, Good, cfg, t0); g.IntervalDays != 1 {
		t.Fatalf("good placement: got %v", g.IntervalDays)
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
		var beginner, intermediate int
		for _, bl := range blocks {
			if bl.Level == Beginner {
				beginner++
			} else {
				intermediate++
			}
		}
		if intermediate != want[name] {
			t.Fatalf("want %d intermediate %s blocks, got %d", want[name], name, intermediate)
		}
		if beginner < 10 || beginner > 14 {
			t.Fatalf("%s: want 10-14 beginner blocks, got %d", name, beginner)
		}
		if v := blocksSchema(string(b)); v != blocksSchemaVersion {
			t.Fatalf("%s: schema %d", name, v)
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
	if !strings.HasSuffix(stub, "\n  # body\n") {
		t.Fatalf("existing code should stay below the TODO: %q", stub)
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
	if !strings.Contains(out.String(), "with_chains: next due in 1 day") || !strings.Contains(out.String(), "genserver: not exercised") {
		t.Fatalf("unexpected grade output:\n%s", out.String())
	}

	led, _ := LoadLedger()
	if c := led.Card("elixir", "with_chains"); c == nil || c.IntervalDays != 1 {
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

func TestMidLineEditPutsTodoOnOwnLine(t *testing.T) {
	t.Setenv("FLIGHTCHECK_HOME", t.TempDir())
	t.Setenv("FLIGHTCHECK_FAKE_CLASSIFY", "with_chains")
	defer func(f func() float64) { randFloat = f }(randFloat)
	randFloat = func() float64 { return 0 }
	if _, err := seedDefaults(false); err != nil {
		t.Fatal(err)
	}
	s := &session{t: t, id: "sess-1", clk: time.Date(2026, 3, 2, 10, 0, 0, 0, time.Local)}
	now = func() time.Time { return s.clk }
	defer func() { now = time.Now }()

	file := filepath.Join(t.TempDir(), "worker.ex")
	os.WriteFile(file, []byte("def f do\n  x = foo(a)\nend\n"), 0o644)
	s.hook("pre-tool-use", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": "true"}})
	for i := 0; i < 8; i++ {
		s.advance(2 * time.Minute)
		s.hook("user-prompt-submit", map[string]any{"prompt": "keep going"})
	}
	s.advance(time.Minute)

	d, upd := decision(s.hook("pre-tool-use", map[string]any{
		"tool_name": "Edit",
		"tool_input": map[string]any{
			"file_path":  file,
			"old_string": "foo(a)",
			"new_string": "with {:ok, y} <- foo(a), do: y",
		},
	}))
	if d != "allow" || upd == nil {
		t.Fatalf("expected allow with updatedInput, got %q", d)
	}
	if upd["old_string"] != "  x = foo(a)" {
		t.Fatalf("old_string not widened to line start: %q", upd["old_string"])
	}
	got := upd["new_string"].(string)
	if !strings.HasPrefix(got, "  # TODO(flightcheck ") || !strings.HasSuffix(got, "\n  x = foo(a)") {
		t.Fatalf("bad new_string: %q", got)
	}
	st, _ := LoadState()
	if r := st.Pending["sess-1"].Restore; r != "  x = with {:ok, y} <- foo(a), do: y" {
		t.Fatalf("bad restore: %q", r)
	}
}

func TestLineLead(t *testing.T) {
	file := filepath.Join(t.TempDir(), "a.go")
	os.WriteFile(file, []byte("func f() {\n\treturn foo(a)\n}\ndup dup\n"), 0o644)
	if lead, ok := lineLead(file, "foo(a)"); !ok || lead != "\treturn " {
		t.Fatalf("mid-line: %q %v", lead, ok)
	}
	if lead, ok := lineLead(file, "func f"); !ok || lead != "" {
		t.Fatalf("first line: %q %v", lead, ok)
	}
	if _, ok := lineLead(file, "dup"); ok {
		t.Fatal("ambiguous old_string should not match")
	}
	if _, ok := lineLead(file, "missing"); ok {
		t.Fatal("missing old_string should not match")
	}
}

func TestDeferAppliesEditOverKeptCode(t *testing.T) {
	t.Setenv("FLIGHTCHECK_HOME", t.TempDir())
	EnsureDirs()
	file := filepath.Join(t.TempDir(), "a.ex")
	stub := "  # TODO(flightcheck x2): stub\n  old code\n"
	os.WriteFile(file, []byte("before\n"+stub+"after\n"), 0o644)
	st := &State{}
	st.init()
	st.Pending["s"] = &Rep{ID: "x2", SessionID: "s", File: file, Stub: stub, Restore: "  new code\n", Original: "  new code\n"}
	SaveState(st)
	var out bytes.Buffer
	if err := DeferRep(&out, "x2"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(file)
	if string(b) != "before\n  new code\nafter\n" {
		t.Fatalf("not restored: %q", b)
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

func TestLoadBlocksLevels(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) []Block {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte(body), 0o644)
		bs, err := LoadBlocks(p)
		if err != nil {
			t.Fatal(err)
		}
		return bs
	}
	bs := write("a.md", "---\nschema_version: 2\n---\n# Title\n\n# Beginner\n## a: A\nfirst\n\n# intermediate\n## b: B\nsecond\n")
	if len(bs) != 2 || bs[0].Level != Beginner || bs[1].Level != Intermediate || bs[0].Description != "first" {
		t.Fatalf("got %+v", bs)
	}
	bs = write("old.md", "# Title\n\n## a: A\nold\n")
	if len(bs) != 1 || bs[0].Level != Intermediate {
		t.Fatalf("schema 1 should be intermediate: %+v", bs)
	}
	for in, want := range map[string]int{
		"# T\n":                         1,
		"---\nother: 1\n---\n":          1,
		"---\nschema_version: 2\n---\n": 2,
		"---\nschema_version: 9\n---\n": 9,
		"---\nschema_version: x\n---\n": blocksSchemaVersion,
	} {
		if got := blocksSchema(in); got != want {
			t.Fatalf("blocksSchema(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestMigrateBlocks(t *testing.T) {
	t.Setenv("FLIGHTCHECK_HOME", t.TempDir())
	if err := EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	goPath := filepath.Join(BlocksDir(), "go.md")
	old := "# My Go\n\nMy notes.\n\n## channels: Channels\nMy edited description.\n\n## variables: Mine\nI already had this id.\n"
	os.WriteFile(goPath, []byte(old), 0o644)
	custom := filepath.Join(BlocksDir(), "python.md")
	customBody := "## comps: Comprehensions\nx\n"
	os.WriteFile(custom, []byte(customBody), 0o644)
	newer := filepath.Join(BlocksDir(), "rust.md")
	newerBody := "---\nschema_version: 99\n---\n## x: X\ny\n"
	os.WriteFile(newer, []byte(newerBody), 0o644)

	wrote, err := seedDefaults(false)
	if err != nil {
		t.Fatal(err)
	}
	report := strings.Join(wrote, "\n")
	if !strings.Contains(report, "go.md (added beginner blocks") || !strings.Contains(report, "already use: variables") {
		t.Fatalf("report: %s", report)
	}
	b, _ := os.ReadFile(goPath)
	got := string(b)
	if !strings.HasSuffix(got, "# Intermediate\n\n## channels: Channels\nMy edited description.\n\n## variables: Mine\nI already had this id.\n") {
		t.Fatalf("user's blocks not kept verbatim:\n%s", got)
	}
	if !strings.Contains(got, "My notes.") || blocksSchema(got) != blocksSchemaVersion {
		t.Fatalf("bad migrated file:\n%s", got)
	}
	bak, _ := os.ReadFile(goPath + ".flightcheck.bak")
	if string(bak) != old {
		t.Fatalf("backup: %q", bak)
	}
	bs, err := LoadBlocks("go.md")
	if err != nil {
		t.Fatal(err)
	}
	levels := map[string]Level{}
	for _, bl := range bs {
		if _, dup := levels[bl.ID]; dup {
			t.Fatalf("duplicate id %s", bl.ID)
		}
		levels[bl.ID] = bl.Level
	}
	if levels["if_else"] != Beginner || levels["channels"] != Intermediate || levels["variables"] != Intermediate {
		t.Fatalf("levels: %v", levels)
	}

	// Custom and newer-schema files are left alone.
	if b, _ := os.ReadFile(custom); string(b) != customBody {
		t.Fatalf("custom file changed: %q", b)
	}
	if b, _ := os.ReadFile(newer); string(b) != newerBody {
		t.Fatalf("newer file changed: %q", b)
	}

	// A second run does nothing.
	wrote, err = seedDefaults(false)
	if err != nil || len(wrote) != 0 {
		t.Fatalf("second seed: %v %v", wrote, err)
	}
	if after, _ := os.ReadFile(goPath); string(after) != got {
		t.Fatal("second seed changed go.md")
	}
}

func TestPromptLevels(t *testing.T) {
	t.Setenv("FLIGHTCHECK_HOME", t.TempDir())
	if _, err := seedDefaults(false); err != nil {
		t.Fatal(err)
	}
	// A custom language with no beginner blocks isn't asked about. It sorts
	// before every built-in, so it would otherwise take the first answer.
	os.WriteFile(filepath.Join(BlocksDir(), "aaa.md"), []byte("## x: X\ny\n"), 0o644)
	raw, _ := os.ReadFile(LanguagesPath())
	kvs, _ := parseObject(raw)
	kvs = append(kvs, kv{"aaa", json.RawMessage(`{"extensions":[".aaa"],"blocks_file":"aaa.md"}`)})
	out, _ := encodeObject(kvs)
	os.WriteFile(LanguagesPath(), out, 0o644)

	var w bytes.Buffer
	// aaa skipped; ansible invalid then b; elixir Enter; erlang "intermediate";
	// go b; then EOF keeps rust and shell.
	if err := promptLevels(&w, strings.NewReader("nope\nb\n\nintermediate\nbeginner\n")); err != nil {
		t.Fatal(err)
	}
	ls, _ := LoadLevels()
	if ls.For("ansible") != Beginner || ls.For("go") != Beginner || ls.For("elixir") != Intermediate ||
		ls.For("erlang") != Intermediate || ls.For("rust") != Intermediate || ls.For("aaa") != Intermediate {
		t.Fatalf("levels: %v", ls)
	}
	if !strings.Contains(w.String(), "aaa: no beginner blocks") || !strings.Contains(w.String(), "Please answer b or i.") {
		t.Fatalf("output:\n%s", w.String())
	}

	// Enter keeps the current level, shown as the default.
	w.Reset()
	if err := promptLevels(&w, strings.NewReader("\n\n\n\n\n\n")); err != nil {
		t.Fatal(err)
	}
	if ls, _ := LoadLevels(); ls.For("go") != Beginner {
		t.Fatalf("Enter changed go: %v", ls)
	}
	if !strings.Contains(w.String(), "go - (b)eginner or (i)ntermediate? [beginner]:") {
		t.Fatalf("output:\n%s", w.String())
	}

	// Non-interactive install leaves levels alone and never reads stdin.
	settings := filepath.Join(t.TempDir(), "settings.json")
	w.Reset()
	if err := Install(&w, strings.NewReader("i\ni\ni\ni\ni\ni\n"), false, settings, false); err != nil {
		t.Fatal(err)
	}
	if ls, _ := LoadLevels(); ls.For("go") != Beginner {
		t.Fatalf("non-interactive install changed levels: %v", ls)
	}
	if !strings.Contains(w.String(), "No terminal") {
		t.Fatalf("output:\n%s", w.String())
	}
}

func TestLoadConfigLegacySRS(t *testing.T) {
	t.Setenv("FLIGHTCHECK_HOME", t.TempDir())
	if err := EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	load := func(srs string) map[string]float64 {
		t.Helper()
		os.WriteFile(ConfigPath(), []byte(`{"srs":`+srs+`}`), 0o644)
		cfg, err := LoadConfig()
		if err != nil {
			t.Fatal(err)
		}
		return cfg.SRS.PlacementHours
	}
	stock := `{"fail_interval_days":1,"placement_days":{"easy":21,"good":7,"hard":2,"fail":1}}`
	if h := load(stock); h["easy"] != 48 || h["fail"] != 4 {
		t.Fatalf("stock values should give new defaults: %v", h)
	}
	if h := load(`{"placement_days":{"easy":30,"good":7,"hard":2,"fail":1}}`); h["easy"] != 720 || h["good"] != 168 || h["fail"] != 24 {
		t.Fatalf("custom days should convert: %v", h)
	}
	if h := load(`{"fail_interval_days":0.5}`); h["fail"] != 12 || h["easy"] != 48 {
		t.Fatalf("custom fail interval should convert: %v", h)
	}
	if h := load(`{"placement_days":{"easy":30},"fail_interval_days":3,"placement_hours":{"easy":5}}`); h["easy"] != 5 || h["fail"] != 4 {
		t.Fatalf("placement_hours should win: %v", h)
	}
}

func TestEndToEndBeginnerRep(t *testing.T) {
	t.Setenv("FLIGHTCHECK_HOME", t.TempDir())
	// "channels" is intermediate, so it must never reach a beginner's rep.
	t.Setenv("FLIGHTCHECK_FAKE_CLASSIFY", "variables,channels")
	defer func(f func() float64) { randFloat = f }(randFloat)
	randFloat = func() float64 { return 0 }
	if _, err := seedDefaults(false); err != nil {
		t.Fatal(err)
	}
	if err := SaveLevels(Levels{"go": Beginner}); err != nil {
		t.Fatal(err)
	}
	s := &session{t: t, id: "sess-b", clk: time.Date(2026, 3, 2, 10, 0, 0, 0, time.Local)}
	now = func() time.Time { return s.clk }
	defer func() { now = time.Now }()
	for i := 0; i < 8; i++ {
		s.advance(2 * time.Minute)
		s.hook("user-prompt-submit", map[string]any{"prompt": "go on"})
	}
	s.advance(time.Minute)

	file := filepath.Join(t.TempDir(), "main.go")
	os.WriteFile(file, []byte("package main\n\n// body\n"), 0o644)
	edit := func(code string) map[string]any {
		return map[string]any{"tool_name": "Edit", "tool_input": map[string]any{
			"file_path": file, "old_string": "// body\n", "new_string": code,
		}}
	}
	if d, _ := decision(s.hook("pre-tool-use", edit(strings.Repeat("x := 1\n", 11)))); d != "deny" {
		t.Fatalf("11 lines should exceed the beginner limit, got %q", d)
	}
	d, upd := decision(s.hook("pre-tool-use", edit("x := 1\n")))
	if d != "allow" || upd == nil {
		t.Fatalf("expected a rep, got %q", d)
	}
	st, _ := LoadState()
	rep := st.Pending["sess-b"]
	if rep == nil || len(rep.Blocks) != 1 || rep.Blocks[0] != "variables" {
		t.Fatalf("rep: %+v", rep)
	}
	os.WriteFile(file, []byte("package main\n\nx := 1\n"), 0o644)
	var out bytes.Buffer
	if err := GradeRep(&out, rep.ID, Hard, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "variables: next due in 8 hours") {
		t.Fatalf("grade output:\n%s", out.String())
	}
	hist, _ := os.ReadFile(HistoryPath())
	if !strings.Contains(string(hist), `"level":"beginner"`) {
		t.Fatalf("history: %s", hist)
	}
	out.Reset()
	if err := Status(&out, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "flightcheck 1.2.3\n") {
		t.Fatalf("status should start with the version:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "go, beginner (.go): 13 blocks") || !strings.Contains(out.String(), "rust, intermediate") {
		t.Fatalf("status:\n%s", out.String())
	}
}
