package fc

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// now is swappable in tests.
var now = time.Now

// State is the per-user runtime state shared by every Claude Code session.
type State struct {
	LastEventAt    time.Time `json:"last_event_at"`
	ActiveSinceRep float64   `json:"active_minutes_since_rep"`
	TargetActive   float64   `json:"target_active_minutes"`
	TurnsSinceRep  int       `json:"turns_since_rep"`
	LastRepAt      time.Time `json:"last_rep_at"`
	RepsDay        string    `json:"reps_day"`
	RepsToday      int       `json:"reps_today"`

	// Pending reps keyed by Claude Code session id (one per session).
	Pending map[string]*Rep `json:"pending"`
	// Most recent graded rep per session, used to measure the agent's
	// fix-up edits until the user's next message.
	RecentGrade map[string]*RecentGrade `json:"recent_grade"`
}

type Rep struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Language  string    `json:"language"`
	File      string    `json:"file"`
	Tool      string    `json:"tool"`
	Blocks    []string  `json:"blocks"`
	OldString string    `json:"old_string,omitempty"`
	Original  string    `json:"original"`
	Stub      string    `json:"stub"`
	Created   time.Time `json:"created"`
	Announced bool      `json:"announced"`
	// User turns since the agent was told to coach; edits to *other* files
	// are allowed again once this is >= 1 (the agent has actually stopped).
	TurnsSinceAnnounce int `json:"turns_since_announce"`
}

type RecentGrade struct {
	RepID string `json:"rep_id"`
	File  string `json:"file"`
}

func (s *State) init() {
	if s.Pending == nil {
		s.Pending = map[string]*Rep{}
	}
	if s.RecentGrade == nil {
		s.RecentGrade = map[string]*RecentGrade{}
	}
}

// FindRep looks a pending rep up by id across sessions.
func (s *State) FindRep(id string) *Rep {
	for _, r := range s.Pending {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// Card is the SM-2 scheduling record for one language block.
type Card struct {
	Ease         float64   `json:"ease"`
	IntervalDays float64   `json:"interval_days"`
	Reviews      int       `json:"reviews"`
	Lapses       int       `json:"lapses"`
	LastReview   time.Time `json:"last_review"`
	Due          time.Time `json:"due"`
}

// Ledger maps "language/block" to its card. Blocks with no card are due.
type Ledger struct {
	Cards map[string]*Card `json:"cards"`
}

func cardKey(lang, block string) string { return lang + "/" + block }

func (l *Ledger) Card(lang, block string) *Card {
	if l.Cards == nil {
		l.Cards = map[string]*Card{}
	}
	return l.Cards[cardKey(lang, block)]
}

// IsDue reports whether a block should be practiced the next time it shows up.
func (l *Ledger) IsDue(lang, block string, t time.Time) bool {
	c := l.Card(lang, block)
	return c == nil || !t.Before(c.Due)
}

// Lock takes an exclusive advisory lock on the data directory. Several
// Claude Code sessions can run hooks at once, so every read-modify-write of
// state or ledger happens under it.
func Lock() (func(), error) {
	if err := EnsureDirs(); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(lockPath(), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

// writeJSON writes atomically (temp file + rename) so a crash never leaves
// a half-written ledger.
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func LoadState() (*State, error) {
	s := &State{}
	if err := readJSON(StatePath(), s); err != nil {
		return nil, err
	}
	s.init()
	return s, nil
}

func SaveState(s *State) error { return writeJSON(StatePath(), s) }

func LoadLedger() (*Ledger, error) {
	l := &Ledger{}
	if err := readJSON(LedgerPath(), l); err != nil {
		return nil, err
	}
	if l.Cards == nil {
		l.Cards = map[string]*Card{}
	}
	return l, nil
}

func SaveLedger(l *Ledger) error { return writeJSON(LedgerPath(), l) }

// appendJSONL appends one JSON object per line. Used for the review history
// (kept so FSRS parameters can be fitted later) and the classifier log.
func appendJSONL(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}
