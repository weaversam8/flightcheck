# Go building blocks

Each `## id: title` section is one block. The description says what counts
as *using* the block: the code being written has to exercise it, not just
sit near it. Edit freely; ids are what the ledger tracks, so renaming an id
resets its schedule.

## structs_methods: Structs and methods
Defining a struct type or methods on it: choosing value vs pointer receivers, struct embedding and promoted fields, or constructor functions (`NewX`) that set up invariants.

## interfaces: Interfaces
Defining an interface or deliberately satisfying one: small consumer-side interfaces, implicit satisfaction, embedding interfaces, compile-time checks like `var _ io.Reader = (*T)(nil)`, or the nil-interface-vs-nil-pointer trap.

## type_switches: Type assertions and type switches
`v.(T)` assertions (especially the comma-ok form) or `switch x := v.(type)` with multiple cases.

## errors: Error wrapping and inspection
Wrapping with `fmt.Errorf("...: %w", err)`, `errors.Is`/`errors.As`/`errors.Join`, sentinel errors (`var ErrX = errors.New(...)`), or custom error types with an `Error()` method. A plain `if err != nil { return err }` does not count.

## defer_panic_recover: defer, panic, and recover
`defer` used for cleanup with its ordering and argument-evaluation rules in mind (e.g. deferring `Close` and checking its error, deferred closures modifying named results), or `panic`/`recover`.

## slices: Slices
Non-trivial slice work: `make` with length vs capacity, `append` aliasing, re-slicing, `copy`, in-place filtering, 2D slices, or the `slices` package (`slices.Sort`, `slices.Index`, `slices.Contains`, `slices.SortFunc`). A single `append` does not count.

## maps: Maps
Map literals, comma-ok lookups, `delete`, maps as sets (`map[string]struct{}`), maps of slices or structs, iterating in sorted order, or the `maps` package.

## strings_runes: Strings, bytes, and runes
`strings.Builder`, `bytes.Buffer`, the `strings`/`bytes` packages beyond a single call, `strconv` conversions, or bytes vs runes (`range` over a string, `[]rune`, `utf8`).

## closures: Function values and closures
Functions as values, closures capturing state, loop-variable capture semantics, or patterns like functional options (`func WithTimeout(d time.Duration) Option`).

## generics: Generics
Type parameters on functions or types, constraints (`any`, `comparable`, `cmp.Ordered`, `~int | ~string` unions), or generic data structures.

## iterators: Range-over-func iterators
Writing or consuming `iter.Seq`/`iter.Seq2` iterators: functions of the form `func(yield func(T) bool)`, respecting `yield`'s return value, or `slices.Collect`/`maps.Keys` style helpers.

## constants_iota: Constants and iota
`const` blocks, `iota` enumerations, typed constants, or enum-like types with a `String()` method.

## goroutines: Goroutines and WaitGroups
Starting goroutines with `go`, coordinating them with `sync.WaitGroup` or `errgroup.Group`, bounding concurrency, or avoiding goroutine leaks.

## channels: Channels
Making buffered or unbuffered channels, sending and receiving, `close` and `range` over a channel, directional channel types (`<-chan T`, `chan<- T`), or pipeline / fan-out / fan-in patterns.

## select: select statements
`select` over multiple channels, including `default` for non-blocking operations, timeouts with `time.After`, or cancellation via `ctx.Done()`.

## sync_primitives: Mutexes and sync primitives
`sync.Mutex`/`RWMutex` guarding shared state, `sync.Once`, `sync/atomic` types, `sync.Pool`, or `sync.Map`.

## context: context.Context
Accepting and threading `ctx` as the first parameter, `context.WithCancel`/`WithTimeout`/`WithDeadline`, checking `ctx.Err()`/`ctx.Done()`, or context values with private key types.

## io_files: io, bufio, and files
Composing `io.Reader`/`io.Writer` (`io.Copy`, `io.TeeReader`, `io.MultiWriter`), `bufio.Scanner`/`bufio.Writer`, or opening, creating, and walking files with `os` and `filepath`/`fs.WalkDir`.

## json: encoding/json
Struct tags (`json:"name,omitempty"`), `Marshal`/`Unmarshal`, streaming `Encoder`/`Decoder`, `json.RawMessage`, or custom `MarshalJSON`/`UnmarshalJSON` methods.

## http_server: net/http servers
`http.Handler`/`HandlerFunc`, `ServeMux` routing (including method and wildcard patterns like `"GET /items/{id}"` and `r.PathValue`), middleware wrapping, writing status codes and headers, or configuring `http.Server` timeouts and shutdown.

## http_client: net/http clients
Building requests with `http.NewRequestWithContext`, a configured `http.Client` with timeouts, checking status codes, and closing response bodies.

## time: time and durations
`time.Duration` arithmetic, `Timer`/`Ticker`, parsing or formatting with reference layouts (`2006-01-02`), time zones, or monotonic-clock comparisons.

## cli_os: Command-line programs
The `flag` package or hand-rolled argument parsing, `os.Args`, `os.Getenv`/`LookupEnv`, exit codes with `os.Exit`, stdin/stdout/stderr handling, or `os/exec` to run subprocesses.

## slog: Structured logging
`log/slog`: loggers and handlers, key-value attrs, `slog.With`, levels, or `slog.Group`.

## database_sql: database/sql
`sql.DB` queries (`QueryContext`, `QueryRowContext`, `ExecContext`), scanning rows and checking `rows.Err()`, `sql.Null*` types, transactions, or prepared statements.

## reflection: Reflection
The `reflect` package: inspecting types and struct tags, setting values via `reflect.Value`, or `Kind` switches.

## testing: Tests
Writing tests: table-driven tests with `t.Run` subtests, `t.Helper`, `t.TempDir`/`t.Cleanup`/`t.Setenv`, `t.Parallel`, `httptest`, testable examples, benchmarks, or fuzz tests.

## packages_modules: Packages and modules
Package layout and visibility (exported vs unexported, `internal/`), `init` functions, `go.mod` edits (`require`, `replace`, Go version), or splitting code across packages to avoid import cycles.

## directives: Compiler directives and embedding
`//go:embed` with `embed.FS` or strings, `//go:build` constraints and `_os.go`/`_arch.go` file naming, or `//go:generate`.
