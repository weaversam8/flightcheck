# AGENTS.md

## What this is

flightcheck is a set of Claude Code hooks that keeps a developer's hand-coding
skills from atrophying while they work with an agent. Each language is split
into named building blocks (e.g. Go `channels`, `generics`). When the agent is
about to write code that uses a block the developer is overdue on, the edit is
swapped for a `TODO(flightcheck <id>)` stub, the developer writes it, and the
agent grades the result. Grades feed an SM-2 spaced-repetition schedule per
block.

## Goals

- **Stay out of the way.** Reps are rare and small: rate-limited by active
  time, turns, and a daily cap, and capped at `rep.max_lines`. Most edits pass
  through untouched.
- **Practice only what's real.** A rep triggers only when the agent's actual
  code exercises a due block (judged by a Haiku classifier that sees only the
  code and block definitions).
- **Fail open.** Any internal error is logged to `~/.flightcheck/errors.log`
  and the edit proceeds. flightcheck must never break a coding session.
- **Respect the user's setup.** `install` merges into `settings.json` without
  reordering or clobbering, backs it up, and never overwrites user config or
  blocks files without `--force`.
- **Zero dependencies.** Single static Go binary, stdlib only.

## Layout

- `cmd/flchk/main.go`: CLI entry point (`install`, `status`, `grade`, `defer`,
  `hook ...`). The binary ships as both `flightcheck` and `flchk`.
- `internal/fc/hooks.go`: the `PreToolUse` / `PostToolUse` /
  `UserPromptSubmit` hook logic and the rep lifecycle. The core of the project.
- `internal/fc/classify.go`: the classifier (Anthropic API or `claude -p`).
- `internal/fc/srs.go`, `ratelimit.go`: scheduling and rep gating.
- `internal/fc/store.go`, `paths.go`, `config.go`: on-disk state under
  `~/.flightcheck/` (or `$FLIGHTCHECK_HOME`).
- `internal/fc/install.go`: settings merge and seeding of defaults.
- `internal/fc/defaults/`: embedded `languages.json` and per-language blocks
  files (`## id: Title` + a description of what counts as *using* the block).

## Working on it

- `make test` runs `go vet` and the tests; `make build` writes `bin/`.
- Tests isolate state with `FLIGHTCHECK_HOME=<tempdir>` and stub the
  classifier with `FLIGHTCHECK_FAKE_CLASSIFY=block_a,block_b`. Never touch the
  real `~/.flightcheck` or call a real model from tests.
- Block ids are the ledger keys: renaming one resets users' schedules.
- Keep `README.md` in sync with user-visible behavior (commands, config
  fields, defaults).
- If flightcheck is installed in your own session, edits here can become reps
  too. Set `FLIGHTCHECK_DISABLE=1` to turn the hooks off.
- Open ideas live in `TODO`.
