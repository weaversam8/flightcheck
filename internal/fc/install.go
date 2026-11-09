package fc

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

//go:embed defaults
var defaultsFS embed.FS

// hookSpec is one hook flightcheck registers.
type hookSpec struct {
	event   string
	matcher string
	arg     string
	timeout int
}

var hookSpecs = []hookSpec{
	// Bash is included so shell writes to a pending rep's file are blocked.
	{"PreToolUse", "Edit|Write|MultiEdit|Bash", "pre-tool-use", 60},
	{"PostToolUse", "Edit|Write|MultiEdit", "post-tool-use", 15},
	{"UserPromptSubmit", "", "user-prompt-submit", 15},
}

// ours matches commands flightcheck installed, under either binary name.
var ours = regexp.MustCompile(`(^|/|')(flightcheck|flchk|fc)[^/ ']*'? hook (pre-tool-use|post-tool-use|user-prompt-submit)\s*$`)

// SettingsPath resolves which Claude Code settings file to edit.
func SettingsPath(scope, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	switch scope {
	case "user", "":
		dir := os.Getenv("CLAUDE_CONFIG_DIR")
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			dir = filepath.Join(home, ".claude")
		}
		return filepath.Join(dir, "settings.json"), nil
	case "project":
		return filepath.Join(".claude", "settings.json"), nil
	case "local":
		return filepath.Join(".claude", "settings.local.json"), nil
	}
	return "", fmt.Errorf("unknown scope %q (want user, project, or local)", scope)
}

// Install seeds ~/.flightcheck, asks for a level per language when
// interactive, and registers the hooks in a settings file.
func Install(w io.Writer, in io.Reader, interactive bool, settingsPath string, force bool) error {
	if err := EnsureDirs(); err != nil {
		return err
	}
	seeded, err := seedDefaults(force)
	if err != nil {
		return err
	}
	for _, s := range seeded {
		fmt.Fprintf(w, "wrote %s\n", s)
	}
	if interactive {
		if err := promptLevels(w, in); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(w, "No terminal, so language levels were left as they are (intermediate unless set). Run `flightcheck install` in a terminal to change them.\n")
	}
	if err := editHooks(settingsPath, true); err != nil {
		return err
	}
	fmt.Fprintf(w, "added flightcheck hooks to %s\n", settingsPath)
	fmt.Fprintf(w, "\nflightcheck is on. Restart any open Claude Code sessions (or run /hooks) to pick it up.\n")
	fmt.Fprintf(w, "Check on it anytime with: flightcheck status\n")
	return nil
}

// Uninstall removes the hooks; data in ~/.flightcheck is kept unless purge.
func Uninstall(w io.Writer, settingsPath string, purge bool) error {
	if err := editHooks(settingsPath, false); err != nil {
		return err
	}
	fmt.Fprintf(w, "removed flightcheck hooks from %s\n", settingsPath)
	if purge {
		if err := os.RemoveAll(Home()); err != nil {
			return err
		}
		fmt.Fprintf(w, "deleted %s\n", Home())
	} else {
		fmt.Fprintf(w, "kept your ledger and history in %s (use --purge to delete)\n", Home())
	}
	return nil
}

// promptLevels asks for a level for each language that has beginner blocks.
// Enter keeps the current level, and end of input keeps all remaining ones.
func promptLevels(w io.Writer, in io.Reader) error {
	langs, err := LoadLanguages()
	if err != nil {
		return err
	}
	levels, err := LoadLevels()
	if err != nil {
		return err
	}
	names := make([]string, 0, len(langs))
	for n := range langs {
		names = append(names, n)
	}
	sort.Strings(names)

	fmt.Fprintln(w, "\nHow comfortable are you writing each language by hand? Beginner practice is basic syntax in small edits; intermediate is larger multi-line changes. Re-run install to change these.")
	r := bufio.NewReader(in)
	changed, eof := false, false
	for _, n := range names {
		blocks, err := LoadBlocks(langs[n].BlocksFile)
		if err != nil {
			fmt.Fprintf(w, "%s: can't load blocks (%v), skipping\n", n, err)
			continue
		}
		if !hasBeginnerBlock(blocks) {
			fmt.Fprintf(w, "%s: no beginner blocks, using intermediate\n", n)
			if levels[n] == Beginner {
				delete(levels, n)
				changed = true
			}
			continue
		}
		for !eof {
			current := levels.For(n)
			fmt.Fprintf(w, "%s - (b)eginner or (i)ntermediate? [%s]: ", n, current)
			line, err := r.ReadString('\n')
			if err != nil {
				if !errors.Is(err, io.EOF) {
					return err
				}
				eof = true
				fmt.Fprintln(w)
			}
			answer := strings.ToLower(strings.TrimSpace(line))
			var picked Level
			switch answer {
			case "":
				picked = current
			case "b", "beginner":
				picked = Beginner
			case "i", "intermediate":
				picked = Intermediate
			default:
				fmt.Fprintf(w, "Please answer b or i.\n")
				continue
			}
			if picked != current {
				levels[n] = picked
				changed = true
			}
			break
		}
	}
	if changed {
		return SaveLevels(levels)
	}
	return nil
}

func seedDefaults(force bool) ([]string, error) {
	if err := EnsureDirs(); err != nil {
		return nil, err
	}
	var wrote []string
	write := func(path string, data []byte) error {
		if !force && fileExists(path) {
			return nil
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		wrote = append(wrote, path)
		return nil
	}
	cfg, _ := json.MarshalIndent(DefaultConfig(), "", "  ")
	if err := write(ConfigPath(), append(cfg, '\n')); err != nil {
		return nil, err
	}
	langs, err := defaultsFS.ReadFile("defaults/languages.json")
	if err != nil {
		return nil, err
	}
	if !force && fileExists(LanguagesPath()) {
		added, err := mergeLanguages(LanguagesPath(), langs)
		if err != nil {
			return nil, err
		}
		if len(added) > 0 {
			wrote = append(wrote, fmt.Sprintf("%s (added %s)", LanguagesPath(), strings.Join(added, ", ")))
		}
	} else if err := write(LanguagesPath(), langs); err != nil {
		return nil, err
	}
	err = fs.WalkDir(defaultsFS, "defaults/blocks", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := defaultsFS.ReadFile(p)
		if err != nil {
			return err
		}
		dest := filepath.Join(BlocksDir(), filepath.Base(p))
		if !force && fileExists(dest) {
			note, err := migrateBlocksFile(dest, data)
			if note != "" {
				wrote = append(wrote, note)
			}
			return err
		}
		return write(dest, data)
	})
	return wrote, err
}

// migrateBlocksFile upgrades an installed copy of a built-in blocks file to
// the current schema. The user's blocks are kept verbatim; everything in a
// schema 1 file is intermediate, and the default beginner blocks are added
// above them. It returns a note describing the change, or "" if none.
func migrateBlocksFile(path string, defaults []byte) (string, error) {
	orig, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if blocksSchema(string(orig)) >= blocksSchemaVersion {
		return "", nil
	}
	userBlocks, err := parseBlockFile(bytes.NewReader(orig))
	if err != nil {
		return "", err
	}
	have := map[string]bool{}
	for _, b := range userBlocks {
		have[b.ID] = true
	}
	beginner, dropped := beginnerSection(string(defaults), have)
	preamble, body := splitPreamble(string(orig))

	var out strings.Builder
	fmt.Fprintf(&out, "---\nschema_version: %d\n---\n", blocksSchemaVersion)
	out.WriteString(strings.TrimRight(preamble, "\n"))
	out.WriteString("\n\n# Beginner\n\n")
	out.WriteString(beginner)
	out.WriteString("# Intermediate\n\n")
	out.WriteString(body)

	backup := path + ".flightcheck.bak"
	if err := os.WriteFile(backup, orig, 0o644); err != nil {
		return "", err
	}
	if err := writeFileAtomic(path, []byte(out.String()), 0o644); err != nil {
		return "", err
	}
	note := fmt.Sprintf("%s (added beginner blocks; backup at %s)", path, backup)
	if len(dropped) > 0 {
		note += fmt.Sprintf("; skipped beginner blocks whose ids you already use: %s", strings.Join(dropped, ", "))
	}
	return note, nil
}

// splitPreamble splits a blocks file at its first "## " heading.
func splitPreamble(s string) (preamble, body string) {
	if strings.HasPrefix(s, "## ") {
		return "", s
	}
	if i := strings.Index(s, "\n## "); i >= 0 {
		return s[:i+1], s[i+1:]
	}
	return s, ""
}

// beginnerSection returns the text of the "# Beginner" blocks in a default
// file, leaving out any whose id is in skip, and the ids it left out.
func beginnerSection(defaults string, skip map[string]bool) (string, []string) {
	var out strings.Builder
	var dropped []string
	inSection, keep := false, false
	for _, line := range strings.SplitAfter(defaults, "\n") {
		if l, ok := levelHeading(strings.TrimRight(line, "\n")); ok {
			inSection = l == Beginner
			keep = false
			continue
		}
		if !inSection {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			id, _, _ := strings.Cut(strings.TrimPrefix(line, "## "), ":")
			id = strings.TrimSpace(id)
			keep = !skip[id]
			if !keep {
				dropped = append(dropped, id)
			}
		}
		if keep {
			out.WriteString(line)
		}
	}
	return out.String(), dropped
}

// mergeLanguages appends default languages missing from an existing
// languages.json, leaving the user's entries alone. It returns the names it
// added, and doesn't rewrite the file if there were none.
func mergeLanguages(path string, defaults []byte) ([]string, error) {
	orig, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	have, err := parseObject(orig)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	want, err := parseObject(defaults)
	if err != nil {
		return nil, err
	}
	var added []string
	for _, p := range want {
		if getKey(have, p.key) == nil {
			have = append(have, p)
			added = append(added, p.key)
		}
	}
	if len(added) == 0 {
		return nil, nil
	}
	out, err := encodeObject(have)
	if err != nil {
		return nil, err
	}
	return added, os.WriteFile(path, out, 0o644)
}

// ---- settings.json editing ----
//
// Settings files are hand-edited, so this keeps the existing key order and
// only touches flightcheck's own hook entries.

type kv struct {
	key string
	val json.RawMessage
}

func parseObject(b []byte) ([]kv, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("expected a JSON object")
	}
	var out []kv
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, errors.New("expected an object key")
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		out = append(out, kv{key, raw})
	}
	return out, nil
}

func encodeObject(kvs []kv) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, p := range kvs {
		if i > 0 {
			buf.WriteByte(',')
		}
		k, _ := json.Marshal(p.key)
		buf.Write(k)
		buf.WriteByte(':')
		buf.Write(p.val)
	}
	buf.WriteByte('}')
	var out bytes.Buffer
	if err := json.Indent(&out, buf.Bytes(), "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

func setKey(kvs []kv, key string, val json.RawMessage) []kv {
	for i := range kvs {
		if kvs[i].key == key {
			kvs[i].val = val
			return kvs
		}
	}
	return append(kvs, kv{key, val})
}

func getKey(kvs []kv, key string) json.RawMessage {
	for _, p := range kvs {
		if p.key == key {
			return p.val
		}
	}
	return nil
}

func editHooks(path string, add bool) error {
	orig, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(bytes.TrimSpace(orig)) == 0 {
		if !add {
			return nil
		}
		orig = []byte("{}")
	}
	top, err := parseObject(orig)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	hooksRaw := getKey(top, "hooks")
	if hooksRaw == nil {
		hooksRaw = json.RawMessage("{}")
	}
	events, err := parseObject(hooksRaw)
	if err != nil {
		return fmt.Errorf("parse hooks in %s: %w", path, err)
	}

	cmdPath := SelfCommand()
	for _, spec := range hookSpecs {
		var groups []map[string]any
		if raw := getKey(events, spec.event); raw != nil {
			if err := json.Unmarshal(raw, &groups); err != nil {
				return fmt.Errorf("parse hooks.%s: %w", spec.event, err)
			}
		}
		groups = removeOurs(groups)
		if add {
			g := map[string]any{"hooks": []any{map[string]any{
				"type": "command", "command": cmdPath + " hook " + spec.arg, "timeout": spec.timeout,
			}}}
			if spec.matcher != "" {
				g["matcher"] = spec.matcher
			}
			groups = append(groups, g)
		}
		if len(groups) == 0 {
			events = removeKey(events, spec.event)
			continue
		}
		b, err := json.Marshal(groups)
		if err != nil {
			return err
		}
		events = setKey(events, spec.event, b)
	}
	hb, err := encodeObject(events)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		top = removeKey(top, "hooks")
	} else {
		top = setKey(top, "hooks", json.RawMessage(bytes.TrimSpace(hb)))
	}
	out, err := encodeObject(top)
	if err != nil {
		return err
	}
	if bytes.Equal(out, orig) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if len(orig) > 0 && fileExists(path) {
		if err := os.WriteFile(path+".flightcheck.bak", orig, 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(path, out, 0o644)
}

func removeOurs(groups []map[string]any) []map[string]any {
	var kept []map[string]any
	for _, g := range groups {
		hs, _ := g["hooks"].([]any)
		var keep []any
		for _, h := range hs {
			m, _ := h.(map[string]any)
			cmd, _ := m["command"].(string)
			if !ours.MatchString(strings.TrimSpace(cmd)) {
				keep = append(keep, h)
			}
		}
		if len(keep) == 0 && len(hs) > 0 {
			continue
		}
		g["hooks"] = keep
		kept = append(kept, g)
	}
	return kept
}

func removeKey(kvs []kv, key string) []kv {
	out := kvs[:0]
	for _, p := range kvs {
		if p.key != key {
			out = append(out, p)
		}
	}
	return out
}
