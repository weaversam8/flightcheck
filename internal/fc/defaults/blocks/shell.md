---
schema_version: 2
---
# Shell building blocks

Each `## id: title` section is one block. The description says what counts
as *using* the block: the code being written has to exercise it, not just
sit near it. Edit freely; ids are what the ledger tracks, so renaming an id
resets its schedule.

These cover POSIX sh and bash. Blocks that are bash-only say so.

Blocks under `# Beginner` are for people who set this language to beginner
during install, and are one small piece of syntax each. Blocks under
`# Intermediate` are for everyone else.

# Beginner

## variable_assignment: Assigning variables
Writing `name=value` with no spaces around `=`, or assigning a quoted string.

## variable_references: Reading variables
Using `"$name"` or `"${name}"`, including placing a variable inside a double-quoted string.

## echo_output: Printing with echo
Writing `echo` to show a message that includes literal text or a variable.

## if_test: if statements
An `if ...; then ... fi` with optional `else` or `elif`, using `[ ... ]` or a command as the condition.

## compare_tests: String and number tests
A single `[ "$a" = "$b" ]`, `[ -z "$a" ]`, `[ -n "$a" ]`, or numeric test such as `-eq` or `-gt`.

## file_tests: File tests
A test like `[ -f "$path" ]`, `[ -d "$dir" ]`, or `[ -e "$path" ]`.

## for_list: for loops over a list
Writing `for item in a b c; do ... done` or looping over `"$@"`.

## while_basics: while loops
A `while` loop with a simple condition and body, ended with `done`.

## function_basics: Simple functions
Defining `name() { ... }`, calling it, or reading its arguments through `$1` and `$2`.

## script_args: Script arguments
Reading `$1`, `$2`, `$#`, or `"$@"` in a script, without an option parser.

## simple_redirects: Simple redirects
Sending output to a file with `>` or `>>`, or reading a file with `<`.

## command_capture: Capturing command output
Assigning output to a variable with a single `x=$(command)`.

## pipes: Pipes
Connecting two or three commands with `|`, such as `grep foo file | wc -l`.

# Intermediate

## quoting: Quoting and word splitting
Choosing between single quotes, double quotes, and `$'...'`, quoting expansions to prevent word splitting and globbing, or `"$@"` vs `$*`.

## parameter_expansion: Parameter expansion
`${var:-default}`, `${var:=default}`, `${var:?message}`, `${var#prefix}`/`${var%suffix}` (and `##`/`%%`), `${#var}`, or bash's `${var/a/b}`, `${var:offset:len}`, and `${var^^}`. A plain `$var` or `${var}` does not count.

## conditionals: Tests and conditionals
`if` with `[ ]`, `[[ ]]`, or a command's exit status, test operators (`-f`, `-d`, `-z`, `-n`, `-eq` vs `=`), `[[ ... =~ regex ]]` with `BASH_REMATCH`, or `&&`/`||` chains used as conditionals.

## case: case statements
`case ... in` with glob patterns, alternation (`a|b)`), a catch-all `*)` branch, or bash's `;&`/`;;&` fallthrough.

## loops: Loops
`for` over arguments, globs, or arrays, C-style `for ((i=0; ...))`, `while`/`until` loops, or `break`/`continue` with counts.

## read_input: Reading input
`while IFS= read -r line` loops over files or command output, `read -p`/`-s`/`-t`/`-a`, splitting fields with `IFS`, `mapfile`/`readarray`, or handling a final line without a trailing newline.

## functions: Functions
Defining functions with `local` variables, returning status vs printing output, using positional parameters and `shift` inside a function, or returning values through a nameref (`local -n`).

## exit_status: Exit status and strict mode
`set -euo pipefail` with its pitfalls in mind, checking `$?` or `PIPESTATUS`, explicit `exit` codes, `|| true` or `if ! cmd` to allow failure deliberately, or `trap ... ERR`.

## traps: Traps and cleanup
`trap` on `EXIT`, `INT`, or `TERM` for cleanup, temp files and directories with `mktemp`, or cleaning up child processes on exit.

## redirection: Redirection and file descriptors
`>`, `>>`, `2>&1`, `&>`, `>/dev/null`, ordering of redirections, `exec` to redirect a whole script or open file descriptors (`exec 3>file`), or `tee` for logging.

## heredocs: Here-documents and here-strings
`<<EOF` vs `<<'EOF'` (expansion vs literal), `<<-` with tab stripping, here-docs fed to commands like `cat > file` or `ssh`, or `<<<` here-strings.

## substitution: Command and process substitution
`$(...)` (and why not backticks), nested substitutions, trailing-newline stripping, process substitution `<(...)`/`>(...)`, or the subshell trap where variables set in a pipeline are lost.

## arrays: Indexed arrays
Bash arrays: `arr=(...)`, `arr+=(...)`, `"${arr[@]}"` vs `"${arr[*]}"`, `${#arr[@]}`, slicing `${arr[@]:1}`, building a command line in an array, or iterating indices with `${!arr[@]}`.

## assoc_arrays: Associative arrays
`declare -A`, setting and reading keys, checking membership with `-v` or `${map[k]+set}`, or iterating keys with `${!map[@]}`.

## arithmetic: Arithmetic
`$(( ))` expressions, `(( ))` as a condition, increment/compound assignment, integer-only limits, or falling back to `bc`/`awk` for floating point.

## arg_parsing: Argument parsing
`getopts` loops with an options string and `OPTARG`/`OPTIND`, a manual `while [ $# -gt 0 ]; case "$1"` parser with `shift`, long options, `--` handling, or a `usage` function.

## printf: printf and output
`printf` with format strings and `%s`/`%d`/`%q`/`%b`, why `printf` over `echo` for arbitrary data, `printf -v`, or writing errors to stderr (`>&2`).

## globbing: Globbing
Globs with `*`, `?`, and `[...]`, handling a glob that matches nothing (`nullglob`, or checking `-e`), `extglob` patterns like `!(*.txt)`, or `globstar` for `**`.

## find_xargs: find and xargs
`find` with predicates (`-name`, `-type`, `-mtime`, `-prune`), `-exec ... {} +`, `-print0` with `xargs -0` or `while IFS= read -r -d ''`, or `xargs -P` for parallelism.

## text_tools: grep, sed, cut, sort, and friends
Non-trivial text pipelines: `grep -E`/`-o`/`-q`, `sed` substitutions with capture groups or in-place edits (and the GNU vs BSD `-i` difference), `cut`, `tr`, `sort -k`, `uniq -c`, `comm`, or `paste`. A single plain `grep` does not count.

## awk: awk
`awk` programs beyond printing a field: `-F` separators, patterns and conditions, `BEGIN`/`END` blocks, accumulating into associative arrays, or `printf` formatting.

## jq: jq
Querying or building JSON with `jq`: filters and paths, `select`, `map`, `-r` raw output, `--arg`/`--argjson` to pass shell values safely, or constructing objects.

## jobs: Background jobs and subshells
Running commands in the background with `&`, `wait` (including `wait -n` and collecting exit codes), `$!`, `( ... )` subshells vs `{ ...; }` groups, or simple parallelism with a job limit.

## environment: Environment and sourcing
`export` vs shell variables, `VAR=value cmd` one-shot environments, `source`/`.` for config or library files, `readonly`, or resolving the script's own directory (`${BASH_SOURCE[0]}`).

## portability: Portability
Writing POSIX `sh` without bashisms, choosing a shebang (`#!/bin/sh` vs `#!/usr/bin/env bash`), `command -v` to check dependencies, or handling GNU vs BSD tool differences (macOS `date`, `sed`, `stat`).
