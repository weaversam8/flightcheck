# flightcheck

Stay current at writing code by hand while you work with Claude Code.

flightcheck breaks a language into building blocks and tracks how recently
you've written each one yourself. When the agent is about to write code that
uses a block you're overdue on, flightcheck swaps that edit for a TODO and
hands it to you. You write it, say "ok, done," and the agent grades your
version and schedules the block's next review with spaced repetition.

Reps are rate-limited (at most one every 15-30 minutes of work), so it stays
out of the way most of the time.

## Install

flightcheck runs on Linux and macOS (amd64 and arm64). Install the binary
one of these ways, then run `flightcheck install` to add the hooks.

**Install script** (no Homebrew or Go needed; downloads a release binary and
verifies its checksum):

```sh
curl -fsSL https://raw.githubusercontent.com/weaversam8/flightcheck/main/install.sh | sh
flightcheck install
```

It installs to `~/.local/bin` and tells you if that isn't on your `PATH`.
Set `FLIGHTCHECK_INSTALL_DIR` to pick another directory,
`FLIGHTCHECK_VERSION=v0.1.0` to pin a release, or `FLIGHTCHECK_SETUP=1` to
run `flightcheck install` for you. Re-run the script to upgrade.

**Homebrew** (macOS or Linux):

```sh
brew tap weaversam8/flightcheck https://github.com/weaversam8/flightcheck
brew install --HEAD weaversam8/flightcheck/flightcheck   # drop --HEAD once a release is tagged
flightcheck install
```

**Go** (1.24+):

```sh
go install github.com/weaversam8/flightcheck/cmd/flchk@latest
flchk install
```

This installs only the short name, `flchk`.

**From a checkout:** `make install` builds, copies `flightcheck` and `flchk`
to `~/.local/bin`, and adds the hooks. Use `sudo make install PREFIX=/usr/local`
for a system-wide install, and `make uninstall` to remove it.

### What `flightcheck install` does

`flightcheck install` adds three hooks to `~/.claude/settings.json` (it
keeps your existing settings and backs the file up first) and seeds
`~/.flightcheck/`. Re-running it never overwrites your config or blocks
files (unless you pass `--force`), but it does add any new built-in
languages to `languages.json`. Restart open Claude Code sessions afterward. Use
`--scope project` to write `.claude/settings.json` in the current repo
instead; everyone working in that repo will then need flightcheck installed.

The binary is also installed under the short name `flchk`, so
`flchk status` works anywhere `flightcheck status` does. The hooks and the
agent always call it by its full path, so they work even if the install
directory isn't on the `PATH` that Claude Code sees.

## How a "rep" works

1. **The agent writes an edit as usual.** Before it's applied, the
   `PreToolUse` hook checks the file extension. If the language is tracked,
   the rep gate is open, and some of its blocks are due, it asks Haiku which
   due blocks the new code actually uses. Haiku sees only the code and the
   block definitions, never the agent's commentary.

2. **If any are due, the edit becomes a TODO.** Instead of applying the
   change, flightcheck adds a TODO comment on its own line above the code
   being changed and leaves the rest of the file as it was, so you can see
   what you're rewriting. The agent's version is saved in
   `~/.flightcheck/pending/`.

3. **The agent coaches and stops.** A `PostToolUse` hook tells it to describe
   what the code must do, which blocks it exercises, and where the docs are,
   without writing the code. Edits are blocked until you reply, and the rep
   file stays blocked until it's graded.

4. **You write it, delete the TODO comment, then tell the agent you're done**
   in plain language. A
   `UserPromptSubmit` hook reminds the agent of the pending rep and the
   grading rubric on each message.

5. **The agent grades before touching anything.** It runs
   `flightcheck grade <id> <easy|good|hard|fail> --blocks ...`, listing only
   the blocks your code actually exercised. Then it explains what you missed,
   shows its version, and fixes the code if needed. Grading refuses to run
   while the TODO marker is still in the file.

Want to skip one? Tell the agent. It runs `flightcheck defer <id>`, which
removes the TODO and applies its edit. The block stays due and comes back later.

### Grades

| Grade | Meaning                                          |
|-------|--------------------------------------------------|
| easy  | correct and idiomatic; the agent changes nothing |
| good  | correct with minor touch-ups                     |
| hard  | needed hints, or real corrections                |
| fail  | incomplete, wrong, or mostly rewritten           |

## Scheduling

Each block has a [SuperMemo-2][sm-2] card in `~/.flightcheck/ledger.json`:

- **First rep places the block** by grade: easy → 21 days, good → 7,
  hard → 2, fail → 1. Experienced developers clear the basics quickly.
- **After that**, interval × ease (starting ease 2.5). Ease moves +0.10 on
  easy, 0 on good, −0.14 on hard, −0.20 on fail, with a floor of 1.3. A fail
  resets the interval to 1 day.
- **Intervals cap at 90 days**, so every block resurfaces at least quarterly.
- **"Due" only matters when you touch it.** A due block you never write
  costs nothing.

Every review is logged to `~/.flightcheck/history.jsonl`, so FSRS
parameters can be fitted later from real data.

[sm-2]: https://en.wikipedia.org/wiki/SuperMemo#Algorithms

### Rate limits

A new rep needs all of:

- **Active time**: a random 15-30 minutes since the last rep. Gaps between
  hook events count for at most 5 minutes, so you shouldn't be hit with a rep
  on both sides of an hour-long lunch break.
- **Turns**: at least 8 turns since the last rep. Each message you send is
  a turn. In auto mode, where the agent can work a long time between your
  messages, every 4 file edits it makes also count as a turn.
- **Daily cap**: fewer than 8 reps today.

The gates reset when a rep starts. If a due block appears while a gate is
closed, the edit goes through normally.

## Configuration

`~/.flightcheck/config.json` (written by `install`, all fields optional):

```json
{
  "rate_limit": { "min_active_minutes": 15, "max_active_minutes": 30,
                  "idle_cap_minutes": 5, "min_turns": 8,
                  "auto_edits_per_turn": 4, "daily_cap": 8 },
  "srs": { "initial_ease": 2.5, "min_ease": 1.3, "max_interval_days": 90,
           "fail_ease_penalty": 0.2, "fail_interval_days": 1,
           "placement_days": { "easy": 21, "good": 7, "hard": 2, "fail": 1 } },
  "rep": { "max_lines": 30 },
  "classifier": { "backend": "auto", "cli_command": "claude",
                  "cli_args": ["-p", "--model", "haiku", "--output-format", "json"],
                  "api_model": "claude-haiku-5-5", "timeout_seconds": 25 }
}
```

**Classifier backend.** `auto` uses the Anthropic API when `ANTHROPIC_API_KEY`
is set, otherwise `claude -p` with your existing Claude Code login. The
nested `claude` runs with `FLIGHTCHECK_INTERNAL=1`, so flightcheck's own
hooks skip it. The classifier only runs when a rep could actually start
(tracked language, gate open, a block due). Every call is logged to
`~/.flightcheck/classifier.log` for spot-checking.

**Edit size.** Edits longer than `max_lines` that hit a due block are
refused with a request to split them, so a rep stays small.

**Auto mode.** The `PostToolUse` hook counts the agent's file edits while
Claude Code reports auto mode, and every `auto_edits_per_turn` of them
counts as one turn. The agent keeps working without stopping until a rep
starts. The edit that starts a rep doesn't count. Set it to `0` to count only
your messages.

**Pausing.** Set `FLIGHTCHECK_DISABLE=1` to turn every hook into a no-op.

## Adding a language

Add an entry to `~/.flightcheck/languages.json` and a blocks file in
`~/.flightcheck/blocks/`:

```json
"python": {
  "extensions": [".py"],
  "blocks_file": "python.md",
  "comments": { "default": ["#", ""] },
  "skip_blocks": []
}
```

```markdown
## comprehensions: Comprehensions
List, dict, and set comprehensions, generator expressions, nested `for` clauses.

## context_managers: Context managers
`with` blocks, writing `__enter__`/`__exit__`, `contextlib.contextmanager`.
```

Block ids are what the ledger tracks, so renaming an id resets its schedule.
`skip_blocks` is for blocks that appear in nearly every edit (Elixir skips
`pattern_matching` and `pipelines`).

## Commands

```shell
flightcheck install [--scope user|project|local] [--settings PATH] [--force]
flightcheck uninstall [--scope ...] [--settings PATH] [--purge]
flightcheck status
flightcheck grade REP_ID easy|good|hard|fail [--blocks id,id]   # run by the agent
flightcheck defer REP_ID                                         # run by the agent
flightcheck hook pre-tool-use|post-tool-use|user-prompt-submit   # run by Claude Code
```

## Limits

- **Shell writes skip the classifier.** Claude writing files via Bash isn't
  turned into a rep, though writes to a file with a pending rep are blocked.
- **The classifier is a small model.** It can mislabel. Grading asks the
  agent to confirm which blocks your code actually exercised.
- **Languages are matched by file extension only.** Ansible claims every
  `.yml` and `.yaml` file, so other YAML (Kubernetes, CI config) also goes to
  the classifier when the rep gate is open. Its blocks are worded so that YAML
  shouldn't match, but it can. To opt out, set `ansible`'s `extensions` to
  `[]` in `languages.json`. Don't delete the entry, because `install` adds
  missing languages back.
- **It fails open.** Any internal error is logged to
  `~/.flightcheck/errors.log` and the edit proceeds untouched.

## Development

```sh
make test     # go vet + tests, including a simulated hook session
make build    # bin/flightcheck and bin/flchk
make dist     # release tarballs for linux/darwin × amd64/arm64 in dist/
```

CI (`.github/workflows/test.yml`) runs the tests on Ubuntu and macOS.

To release:

1. Tag and push: `git tag v0.1.0 && git push origin v0.1.0`. The release
   workflow tests, builds the four tarballs with `scripts/dist.sh`, and
   publishes them with `checksums.txt` as a GitHub release, which is what
   `install.sh` downloads.
2. Run `scripts/release.sh 0.1.0` to point the Homebrew formula at the
   tagged source tarball, then commit and push `Formula/flightcheck.rb`.
