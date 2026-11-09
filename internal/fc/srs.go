package fc

import (
	"fmt"
	"math"
	"time"
)

// Grade is the result of a rep, judged by the agent.
type Grade string

const (
	Easy Grade = "easy" // correct and idiomatic; the agent changed nothing
	Good Grade = "good" // correct with minor touch-ups
	Hard Grade = "hard" // needed hints or real corrections
	Fail Grade = "fail" // incomplete, wrong, or mostly rewritten
)

func ParseGrade(s string) (Grade, error) {
	switch g := Grade(s); g {
	case Easy, Good, Hard, Fail:
		return g, nil
	}
	return "", fmt.Errorf("unknown grade %q (want easy, good, hard, or fail)", s)
}

// quality maps a grade onto SM-2's 0-5 response quality scale.
func (g Grade) quality() int {
	switch g {
	case Easy:
		return 5
	case Good:
		return 4
	case Hard:
		return 3
	}
	return 2
}

// Review applies one graded rep to a card using SM-2 (Wozniak, 1987),
// adapted in two ways:
//
//  1. Placement: a block's first rep gets an interval set by the grade
//     (placement_hours), from a few hours up to two days, instead of SM-2's
//     fixed 1-day, 6-day ladder.
//  2. Lapses: a failed rep resets the interval to placement_hours["fail"]
//     and cuts ease by fail_ease_penalty; the next pass grows from there.
//
// Ease follows the SM-2 update EF' = EF + (0.1 - (5-q)(0.08 + 0.02(5-q))),
// i.e. +0.10 easy, 0 good, -0.14 hard, floored at min_ease. Intervals are
// capped at max_interval_days.
func Review(c *Card, g Grade, cfg SRSConfig, t time.Time) *Card {
	if c == nil {
		c = &Card{Ease: cfg.InitialEase}
	}
	out := *c
	q := float64(g.quality())

	if g == Fail {
		out.Ease -= cfg.FailEasePenalty
		out.IntervalDays = cfg.PlacementHours[string(Fail)] / 24
		out.Lapses++
	} else {
		out.Ease += 0.1 - (5-q)*(0.08+(5-q)*0.02)
		if c.Reviews == 0 {
			out.IntervalDays = cfg.PlacementHours[string(g)] / 24
		} else {
			// Grow from the old interval using the pre-update ease, as SM-2 does.
			out.IntervalDays = c.IntervalDays * c.Ease
		}
	}
	out.Ease = math.Max(cfg.MinEase, math.Round(out.Ease*100)/100)
	out.IntervalDays = math.Min(cfg.MaxIntervalDays, math.Round(out.IntervalDays*100)/100)
	if out.IntervalDays <= 0 {
		out.IntervalDays = 1
	}
	out.Reviews++
	out.LastReview = t
	out.Due = t.Add(time.Duration(out.IntervalDays * 24 * float64(time.Hour)))
	return &out
}
