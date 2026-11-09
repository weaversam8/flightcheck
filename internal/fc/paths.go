package fc

import (
	"os"
	"path/filepath"
)

// Home returns the flightcheck data directory. FLIGHTCHECK_HOME overrides
// the default of ~/.flightcheck (useful for tests and multi-profile setups).
func Home() string {
	if h := os.Getenv("FLIGHTCHECK_HOME"); h != "" {
		return h
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".flightcheck")
}

func ConfigPath() string    { return filepath.Join(Home(), "config.json") }
func LanguagesPath() string { return filepath.Join(Home(), "languages.json") }
func LevelsPath() string    { return filepath.Join(Home(), "levels.json") }
func BlocksDir() string     { return filepath.Join(Home(), "blocks") }
func StatePath() string     { return filepath.Join(Home(), "state.json") }
func LedgerPath() string    { return filepath.Join(Home(), "ledger.json") }
func HistoryPath() string   { return filepath.Join(Home(), "history.jsonl") }
func ClassifierLogPath() string {
	return filepath.Join(Home(), "classifier.log")
}
func PendingDir() string { return filepath.Join(Home(), "pending") }
func RepsDir() string    { return filepath.Join(Home(), "reps") }
func lockPath() string   { return filepath.Join(Home(), ".lock") }

// EnsureDirs creates the data directory layout.
func EnsureDirs() error {
	for _, d := range []string{Home(), BlocksDir(), PendingDir(), RepsDir()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}
