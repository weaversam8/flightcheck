// Command flightcheck (also installed as flchk) keeps hand-coding skills
// current while you work with Claude Code. See README.md.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/weaversam8/flightcheck/internal/fc"
)

var version = "dev"

const usage = `flightcheck: stay current at writing code by hand while using Claude Code.

Usage:
  flightcheck install [--scope user|project|local] [--settings PATH] [--force]
  flightcheck uninstall [--scope user|project|local] [--settings PATH] [--purge]
  flightcheck status
  flightcheck grade REP_ID easy|good|hard|fail [--blocks id,id]
  flightcheck defer REP_ID
  flightcheck hook pre-tool-use|post-tool-use|user-prompt-submit
  flightcheck version

grade and defer are normally run by the agent, not by you.

flchk is a short alias for flightcheck.
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "flightcheck:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return nil
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "install", "uninstall":
		fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
		scope := fs.String("scope", "user", "settings scope: user, project, or local")
		settings := fs.String("settings", "", "explicit settings.json path")
		force := fs.Bool("force", false, "overwrite config, languages, and block files with defaults")
		purge := fs.Bool("purge", false, "also delete ~/.flightcheck (uninstall only)")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		path, err := fc.SettingsPath(*scope, *settings)
		if err != nil {
			return err
		}
		if cmd == "install" {
			return fc.Install(stdout, stdin, stdinIsTerminal(), path, *force)
		}
		return fc.Uninstall(stdout, path, *purge)

	case "status":
		return fc.Status(stdout, version)

	case "grade":
		pos, flags := splitFlags(rest)
		fs := flag.NewFlagSet("grade", flag.ContinueOnError)
		blocks := fs.String("blocks", "", "comma-separated blocks the user's code exercised")
		if err := fs.Parse(flags); err != nil {
			return err
		}
		if len(pos) != 2 {
			return fmt.Errorf("usage: flightcheck grade REP_ID easy|good|hard|fail [--blocks id,id]")
		}
		g, err := fc.ParseGrade(strings.ToLower(pos[1]))
		if err != nil {
			return err
		}
		var ids []string
		for _, b := range strings.Split(*blocks, ",") {
			if b = strings.TrimSpace(b); b != "" {
				ids = append(ids, b)
			}
		}
		return fc.GradeRep(stdout, pos[0], g, ids)

	case "defer":
		if len(rest) != 1 {
			return fmt.Errorf("usage: flightcheck defer REP_ID")
		}
		return fc.DeferRep(stdout, rest[0])

	case "hook":
		if len(rest) != 1 {
			return fmt.Errorf("usage: flightcheck hook EVENT")
		}
		fc.RunHook(rest[0], stdin, stdout)
		return nil

	case "version", "--version", "-v":
		fmt.Fprintln(stdout, "flightcheck", version)
		return nil

	case "help", "--help", "-h":
		fmt.Fprint(stdout, usage)
		return nil
	}
	return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
}

// stdinIsTerminal reports whether install can ask questions. /dev/null also
// passes this check, but reads from it hit EOF and keep the current levels.
func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// splitFlags lets flags appear after positional args (`grade ID good --blocks x`).
func splitFlags(args []string) (pos, flags []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		pos = append(pos, a)
	}
	return pos, flags
}
