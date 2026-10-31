package fc

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// randFloat is swappable in tests.
var randFloat = rand.Float64

func dayKey(t time.Time) string { return t.Local().Format("2006-01-02") }

// Touch records a hook event and accrues active time. Each gap since the
// previous event counts for at most IdleCapMinutes, so a long break adds
// only a few minutes of "activity".
func (s *State) Touch(t time.Time, cfg RateLimitConfig) {
	if !s.LastEventAt.IsZero() {
		gap := t.Sub(s.LastEventAt).Minutes()
		if gap < 0 {
			gap = 0
		}
		if gap > cfg.IdleCapMinutes {
			gap = cfg.IdleCapMinutes
		}
		s.ActiveSinceRep += gap
	}
	s.LastEventAt = t
	if s.TargetActive == 0 {
		s.TargetActive = pickTarget(cfg)
	}
	if s.RepsDay != dayKey(t) {
		s.RepsDay = dayKey(t)
		s.RepsToday = 0
	}
}

func pickTarget(cfg RateLimitConfig) float64 {
	lo, hi := cfg.MinActiveMinutes, cfg.MaxActiveMinutes
	if hi < lo {
		hi = lo
	}
	return lo + randFloat()*(hi-lo)
}

// GateOpen reports whether a new rep may start now, and why not if closed.
func (s *State) GateOpen(t time.Time, cfg RateLimitConfig) (bool, string) {
	if s.RepsDay == dayKey(t) && s.RepsToday >= cfg.DailyCap {
		return false, fmt.Sprintf("daily cap reached (%d/%d)", s.RepsToday, cfg.DailyCap)
	}
	if s.ActiveSinceRep < s.TargetActive {
		return false, fmt.Sprintf("%.0f of %.0f active minutes", s.ActiveSinceRep, s.TargetActive)
	}
	if s.TurnsSinceRep < cfg.MinTurns {
		return false, fmt.Sprintf("%d of %d turns", s.TurnsSinceRep, cfg.MinTurns)
	}
	return true, "open"
}

// StartRep resets the gates when a rep is created (not when it's graded),
// so the spacing is measured from when you were last interrupted.
func (s *State) StartRep(t time.Time, cfg RateLimitConfig) {
	s.ActiveSinceRep = 0
	s.TurnsSinceRep = 0
	s.TargetActive = pickTarget(cfg)
	s.LastRepAt = t
	if s.RepsDay != dayKey(t) {
		s.RepsDay = dayKey(t)
		s.RepsToday = 0
	}
	s.RepsToday++
}
