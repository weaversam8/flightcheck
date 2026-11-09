package fc

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// GradeRep records the agent's grade for a pending rep. blocks narrows the
// grade to the blocks the user's code actually exercised; the rest stay due.
func GradeRep(w io.Writer, id string, g Grade, blocks []string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	unlock, err := Lock()
	if err != nil {
		return err
	}
	defer unlock()
	st, err := LoadState()
	if err != nil {
		return err
	}
	rep := st.FindRep(id)
	if rep == nil {
		return fmt.Errorf("no pending rep %q (run `%s status` to list them)", id, SelfCommand())
	}
	content, err := os.ReadFile(rep.File)
	if err != nil {
		return fmt.Errorf("can't read %s: %w (if the file is gone, use `%s defer %s`)", rep.File, err, SelfCommand(), id)
	}
	if strings.Contains(string(content), marker(id)) {
		return fmt.Errorf("the %s marker is still in %s; the user needs to write their code and delete that comment before you grade (or use `%s defer %s`)",
			marker(id), rep.File, SelfCommand(), id)
	}

	inRep := map[string]bool{}
	for _, b := range rep.Blocks {
		inRep[b] = true
	}
	if len(blocks) == 0 {
		blocks = rep.Blocks
	}
	graded := map[string]bool{}
	for _, b := range blocks {
		if !inRep[b] {
			return fmt.Errorf("block %q isn't part of rep %s (its blocks: %s)", b, id, strings.Join(rep.Blocks, ", "))
		}
		graded[b] = true
	}

	ledger, err := LoadLedger()
	if err != nil {
		return err
	}
	levels, err := LoadLevels()
	if err != nil {
		return err
	}
	t := now()
	fmt.Fprintf(w, "Recorded %s for rep %s:\n", g, id)
	for _, b := range rep.Blocks {
		if !graded[b] {
			fmt.Fprintf(w, "  - %s: not exercised, stays due\n", b)
			_ = appendJSONL(HistoryPath(), map[string]any{
				"type": "not_exercised", "time": t, "rep_id": id, "language": rep.Language, "block": b,
			})
			continue
		}
		before := ledger.Card(rep.Language, b)
		after := Review(before, g, cfg.SRS, t)
		ledger.Cards[cardKey(rep.Language, b)] = after
		entry := map[string]any{
			"type": "review", "time": t, "rep_id": id, "language": rep.Language, "block": b,
			"grade": g, "level": levels.For(rep.Language), "interval_after": after.IntervalDays, "ease_after": after.Ease,
		}
		if before != nil {
			entry["interval_before"] = before.IntervalDays
			entry["ease_before"] = before.Ease
			entry["days_since_review"] = t.Sub(before.LastReview).Hours() / 24
		}
		_ = appendJSONL(HistoryPath(), entry)
		fmt.Fprintf(w, "  - %s: next due in %s (%s)\n", b, humanDays(after.IntervalDays), dueLabel(after.Due, t))
	}
	if err := SaveLedger(ledger); err != nil {
		return err
	}

	snapshot := string(content)
	if len(snapshot) > 256*1024 {
		snapshot = snapshot[:256*1024]
	}
	_ = writeJSON(filepath.Join(RepsDir(), id+".json"), map[string]any{
		"rep": rep, "grade": g, "graded_blocks": blocks, "graded_at": t, "user_file": snapshot,
	})
	os.Remove(pendingFile(id))
	delete(st.Pending, rep.SessionID)
	st.RecentGrade[rep.SessionID] = &RecentGrade{RepID: id, File: rep.File}
	if err := SaveState(st); err != nil {
		return err
	}
	fmt.Fprintf(w, "\nEdits to %s are unblocked. Now tell the user what they missed (if anything), show your version for comparison, and fix the code with Edit if needed.\n", rep.File)
	return nil
}

// DeferRep cancels a pending rep without grading it. If the TODO stub is
// untouched it puts the agent's original code back; the blocks stay due.
func DeferRep(w io.Writer, id string) error {
	unlock, err := Lock()
	if err != nil {
		return err
	}
	defer unlock()
	st, err := LoadState()
	if err != nil {
		return err
	}
	rep := st.FindRep(id)
	if rep == nil {
		return fmt.Errorf("no pending rep %q", id)
	}
	restored := false
	if b, err := os.ReadFile(rep.File); err == nil {
		s := string(b)
		if strings.Count(s, rep.Stub) == 1 {
			info, _ := os.Stat(rep.File)
			mode := os.FileMode(0o644)
			if info != nil {
				mode = info.Mode().Perm()
			}
			restore := rep.Restore
			if restore == "" {
				restore = rep.Original
			}
			if err := os.WriteFile(rep.File, []byte(strings.Replace(s, rep.Stub, restore, 1)), mode); err == nil {
				restored = true
			}
		}
	}
	_ = appendJSONL(HistoryPath(), map[string]any{
		"type": "defer", "time": now(), "rep_id": id, "language": rep.Language, "blocks": rep.Blocks,
	})
	os.Remove(pendingFile(id))
	delete(st.Pending, rep.SessionID)
	if err := SaveState(st); err != nil {
		return err
	}
	if restored {
		fmt.Fprintf(w, "Deferred rep %s and restored the original code in %s. Re-read the file before editing it again. Its blocks stay due and will come back later.\n", id, rep.File)
	} else {
		fmt.Fprintf(w, "Deferred rep %s. The TODO stub in %s was already changed, so nothing was restored. The original code was:\n\n%s\n\nApply it with Edit if the user wants it.\n", id, rep.File, rep.Original)
	}
	return nil
}

// Status prints gates, pending reps, and what's due.
func Status(w io.Writer) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	st, err := LoadState()
	if err != nil {
		return err
	}
	ledger, err := LoadLedger()
	if err != nil {
		return err
	}
	langs, err := LoadLanguages()
	if err != nil {
		return err
	}
	levels, err := LoadLevels()
	if err != nil {
		return err
	}
	t := now()
	fmt.Fprintf(w, "flightcheck data: %s\n\n", Home())

	open, why := st.GateOpen(t, cfg.RateLimit)
	gate := "closed (" + why + ")"
	if open {
		gate = "open"
	}
	fmt.Fprintf(w, "Next rep gate: %s\n", gate)
	if st.TargetActive == 0 {
		fmt.Fprintf(w, "  active minutes since last rep: %.0f / %.0f-%.0f (target picked on first use)\n",
			st.ActiveSinceRep, cfg.RateLimit.MinActiveMinutes, cfg.RateLimit.MaxActiveMinutes)
	} else {
		fmt.Fprintf(w, "  active minutes since last rep: %.0f / %.0f\n", st.ActiveSinceRep, st.TargetActive)
	}
	fmt.Fprintf(w, "  turns since last rep: %d / %d\n", st.TurnsSinceRep, cfg.RateLimit.MinTurns)
	today := 0
	if st.RepsDay == dayKey(t) {
		today = st.RepsToday
	}
	fmt.Fprintf(w, "  reps today: %d / %d\n\n", today, cfg.RateLimit.DailyCap)

	if len(st.Pending) == 0 {
		fmt.Fprintln(w, "Pending reps: none")
	} else {
		fmt.Fprintln(w, "Pending reps:")
		for _, r := range st.Pending {
			fmt.Fprintf(w, "  %s  %s  [%s]  started %s\n", r.ID, r.File, strings.Join(r.Blocks, ", "), r.Created.Local().Format("Jan 2 15:04"))
		}
	}

	names := make([]string, 0, len(langs))
	for n := range langs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		L := langs[n]
		level := levels.For(n)
		all, err := LoadBlocks(L.BlocksFile)
		if err != nil {
			fmt.Fprintf(w, "\n%s: can't load blocks (%v)\n", n, err)
			continue
		}
		skip := map[string]bool{}
		for _, s := range L.SkipBlocks {
			skip[s] = true
		}
		var due, fresh []string
		type upcoming struct {
			id  string
			due time.Time
		}
		var next []upcoming
		for _, b := range all {
			if skip[b.ID] || b.Level != level {
				continue
			}
			c := ledger.Card(n, b.ID)
			switch {
			case c == nil:
				fresh = append(fresh, b.ID)
			case !t.Before(c.Due):
				due = append(due, b.ID)
			default:
				next = append(next, upcoming{b.ID, c.Due})
			}
		}
		sort.Slice(next, func(i, j int) bool { return next[i].due.Before(next[j].due) })
		fmt.Fprintf(w, "\n%s, %s (%s): %d blocks: %d new (due on first use), %d due, %d current\n",
			n, level, strings.Join(L.Extensions, " "), len(fresh)+len(due)+len(next), len(fresh), len(due), len(next))
		if len(due) > 0 {
			fmt.Fprintf(w, "  due: %s\n", strings.Join(due, ", "))
		}
		for i, u := range next {
			if i == 5 {
				break
			}
			fmt.Fprintf(w, "  next: %-26s %s\n", u.id, dueLabel(u.due, t))
		}
	}
	return nil
}

func humanDays(d float64) string {
	if d < 1 {
		return fmt.Sprintf("%.0f hours", d*24)
	}
	if d < 1.5 {
		return "1 day"
	}
	return fmt.Sprintf("%.0f days", d)
}

// dueLabel formats a due date, with the time of day when it's close enough
// for that to matter.
func dueLabel(due, t time.Time) string {
	if due.Sub(t) < 48*time.Hour {
		return due.Local().Format("Jan 2 15:04")
	}
	return due.Local().Format("Jan 2")
}
