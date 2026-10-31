package fc

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
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

// Block is one language building block from a blocks markdown file.
type Block struct {
	ID          string
	Title       string
	Description string
}

// LoadBlocks parses a blocks file. Format:
//
//	## block_id: Human title
//	Plain-English description of what counts as using this block.
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
	var blocks []Block
	var cur *Block
	flush := func() {
		if cur != nil {
			cur.Description = strings.TrimSpace(cur.Description)
			blocks = append(blocks, *cur)
		}
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "## ") {
			flush()
			head := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			id, title, _ := strings.Cut(head, ":")
			cur = &Block{ID: strings.TrimSpace(id), Title: strings.TrimSpace(title)}
			continue
		}
		if cur != nil {
			cur.Description += line + "\n"
		}
	}
	flush()
	return blocks, sc.Err()
}

// DueBlocks returns the blocks of a language that would become a rep if an
// edit used them right now.
func DueBlocks(lang string, l Language, ledger *Ledger) ([]Block, error) {
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
		if !skip[b.ID] && ledger.IsDue(lang, b.ID, t) {
			due = append(due, b)
		}
	}
	return due, nil
}
