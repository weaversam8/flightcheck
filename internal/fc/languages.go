package fc

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Language describes one tracked language in languages.json.
type Language struct {
	Extensions []string `json:"extensions"`
	BlocksFile string   `json:"blocks_file"`
	// Comment syntax for the TODO stub, as [prefix, suffix]. "default"
	// applies to every extension not listed explicitly.
	Comments map[string][2]string `json:"comments"`
	// Blocks that appear in nearly every change. They are never turned into
	// reps because they'd fire constantly and stay current anyway.
	SkipBlocks []string `json:"skip_blocks"`
}

type Languages map[string]Language

func LoadLanguages() (Languages, error) {
	langs := Languages{}
	if err := readJSON(LanguagesPath(), &langs); err != nil {
		return nil, err
	}
	return langs, nil
}

// ForFile maps a file path to a tracked language by extension.
func (ls Languages) ForFile(path string) (string, Language, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "", Language{}, false
	}
	names := make([]string, 0, len(ls))
	for n := range ls {
		names = append(names, n)
	}
	sort.Strings(names) // deterministic if two languages claim an extension
	for _, n := range names {
		for _, e := range ls[n].Extensions {
			if strings.ToLower(e) == ext {
				return n, ls[n], true
			}
		}
	}
	return "", Language{}, false
}

// CommentFor returns the stub comment delimiters for a file.
func (l Language) CommentFor(path string) (string, string) {
	ext := strings.ToLower(filepath.Ext(path))
	if c, ok := l.Comments[ext]; ok {
		return c[0], c[1]
	}
	if c, ok := l.Comments["default"]; ok {
		return c[0], c[1]
	}
	return "//", ""
}

// Level is how far along a user is in a language. It selects which blocks
// can become reps.
type Level string

const (
	Beginner     Level = "beginner"
	Intermediate Level = "intermediate"
)

// Levels maps a language to the user's level, as stored in levels.json.
type Levels map[string]Level

// For returns a language's level; anything unset is intermediate.
func (ls Levels) For(lang string) Level {
	if ls[lang] == Beginner {
		return Beginner
	}
	return Intermediate
}

func LoadLevels() (Levels, error) {
	ls := Levels{}
	if err := readJSON(LevelsPath(), &ls); err != nil {
		return nil, err
	}
	return ls, nil
}

func SaveLevels(ls Levels) error { return writeJSON(LevelsPath(), ls) }

// Block is one language building block from a blocks markdown file.
type Block struct {
	ID          string
	Title       string
	Description string
	Level       Level
}

// blocksSchemaVersion is the current blocks file format. Version 1 had no
// version field and no level sections; version 2 added both.
const blocksSchemaVersion = 2

// blocksSchema reads schema_version from a blocks file's frontmatter. A file
// with none is version 1. An unreadable value counts as current, so a
// migration never rewrites a file it doesn't understand.
func blocksSchema(data string) int {
	lines := strings.Split(data, "\n")
	if strings.TrimSpace(lines[0]) != "---" {
		return 1
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "schema_version:"); ok {
			n, err := strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return blocksSchemaVersion
			}
			return n
		}
	}
	return 1
}

// levelHeading reports whether line is a "# Beginner" or "# Intermediate"
// section heading. Other top-level headings, like a file's title, are not.
func levelHeading(line string) (Level, bool) {
	rest, ok := strings.CutPrefix(line, "# ")
	if !ok {
		return "", false
	}
	switch strings.ToLower(strings.TrimSpace(rest)) {
	case "beginner":
		return Beginner, true
	case "intermediate":
		return Intermediate, true
	}
	return "", false
}

// LoadBlocks parses a blocks file. Format:
//
//	---
//	schema_version: 2
//	---
//	# Beginner
//	## block_id: Human title
//	Plain-English description of what counts as using this block.
//
//	# Intermediate
//	## other_id: Another title
//	...
//
// A level heading applies to the blocks after it. Blocks before any level
// heading are intermediate, so files without sections stay valid.
func LoadBlocks(file string) ([]Block, error) {
	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(BlocksDir(), file)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseBlockFile(f)
}

func parseBlockFile(r io.Reader) ([]Block, error) {
	var blocks []Block
	var cur *Block
	level := Intermediate
	flush := func() {
		if cur != nil {
			cur.Description = strings.TrimSpace(cur.Description)
			blocks = append(blocks, *cur)
			cur = nil
		}
	}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if l, ok := levelHeading(line); ok {
			flush()
			level = l
			continue
		}
		if strings.HasPrefix(line, "## ") {
			flush()
			head := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			id, title, _ := strings.Cut(head, ":")
			cur = &Block{ID: strings.TrimSpace(id), Title: strings.TrimSpace(title), Level: level}
			continue
		}
		if cur != nil {
			cur.Description += line + "\n"
		}
	}
	flush()
	return blocks, sc.Err()
}

// DueBlocks returns the blocks of a language, at the user's level, that
// would become a rep if an edit used them right now.
func DueBlocks(lang string, l Language, level Level, ledger *Ledger) ([]Block, error) {
	all, err := LoadBlocks(l.BlocksFile)
	if err != nil {
		return nil, err
	}
	skip := map[string]bool{}
	for _, s := range l.SkipBlocks {
		skip[s] = true
	}
	t := now()
	var due []Block
	for _, b := range all {
		if b.Level == level && !skip[b.ID] && ledger.IsDue(lang, b.ID, t) {
			due = append(due, b)
		}
	}
	return due, nil
}
