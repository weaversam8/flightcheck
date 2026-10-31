# Elixir building blocks

Each `## id: title` section is one block. The description says what counts
as *using* the block: the code being written has to exercise it, not just
sit near it. Edit freely; ids are what the ledger tracks, so renaming an id
resets its schedule.

## pattern_matching: Pattern matching
Destructuring with `=`, matching in function heads or case clauses, and the pin operator `^`.

## guards: Guards
`when` clauses on functions or case/receive branches, including knowing which functions are allowed in guards (`is_*`, comparisons, `in`, etc.) or defining `defguard`.

## multi_clause: Multi-clause functions and defaults
Writing a function as several clauses dispatched by pattern, a bodiless head for default arguments (`def f(x, opts \\ [])`), or ordering clauses so specific ones come first.

## anon_functions: Anonymous functions and captures
Writing `fn ... -> ... end` (including multi-clause anonymous functions) or the capture operator: `&Mod.fun/1`, `&(&1 + 1)`, `&fun/2`.

## pipelines: Pipelines
Chaining calls with `|>` and shaping function arguments so the subject comes first.

## control_flow: Control flow
`case`, `cond`, `if`/`unless` expressions, including using their return values.

## with_chains: with chains
`with` expressions that chain matches, especially with an `else` block handling non-matching results.

## recursion: Recursion and accumulators
Hand-written recursive functions: base cases, accumulators, tail recursion, reversing an accumulated list.

## enum_stream: Enum and Stream
Non-trivial use of `Enum` or `Stream`: reduce/reduce_while, group_by, chunk_every, zip, flat_map, or lazy `Stream` pipelines. A single trivial `Enum.map` does not count.

## comprehensions: Comprehensions
`for` comprehensions, especially with filters, multiple generators, pattern-matching generators, `into:`, `uniq:`, or `reduce:`.

## collections: Lists, tuples, and keyword lists
Working with list operations (`[h | t]`, `++`), tuples (`elem`, tagged tuples as data), or keyword lists as options (`Keyword.get/validate`).

## maps: Maps
Map literals, update syntax `%{m | k: v}`, nested updates (`put_in`, `update_in`, `get_in`, `Access`), or `Map` module functions beyond a single get.

## structs: Structs
Defining a struct (`defstruct`, `@enforce_keys`, defaults) or matching/updating struct values.

## strings_binaries: Strings, binaries, and charlists
String manipulation with `String` functions, interpolation subtleties, UTF-8 vs bytes, charlists vs strings, or `IO.iodata` building.

## bitstrings: Bitstring pattern matching
Binary syntax `<<...>>` with sizes, types, or segments (`::16`, `::binary`, `::utf8`, `::size(n)`), for parsing or building binary data.

## sigils: Sigils
Using sigils such as `~w`, `~r`, `~S`, `~D`/`~U`, or defining a custom sigil.

## modules: Modules and directives
Defining modules and module attributes, or using `alias`, `import`, `require`, and `use` deliberately (e.g. `import Mod, only: ...`, nested module aliases, compile-time attributes).

## protocols: Protocols
Defining a protocol with `defprotocol` or implementing one with `defimpl`, including `@derive` and fallbacks.

## behaviours: Behaviours
Defining callbacks with `@callback`, implementing a behaviour with `@behaviour` and `@impl true`, or dispatching to a module that implements one.

## macros: Macros and metaprogramming
`defmacro`, `quote`/`unquote`, `__using__`, `Macro` functions, or compile-time code generation.

## typespecs: Typespecs
`@spec`, `@type`, `@typep`, `@opaque`, or writing code shaped for Dialyzer.

## error_handling: Error handling conventions
Returning `{:ok, _}`/`{:error, _}`, choosing between returning errors and raising, bang (`!`) function variants, `try`/`rescue`/`catch`/`after`, custom exceptions with `defexception`, or deliberate let-it-crash design.

## exunit: ExUnit tests
Writing tests: `describe`, `setup`/`setup_all`, `async: true`, assertions like `assert_raise`/`assert_receive`, tags, or doctests.

## mix: Mix and project configuration
Editing `mix.exs` (deps, aliases, project config) or writing a custom `Mix.Task`.

## processes: Raw processes and messages
`spawn`/`spawn_link`, `send`/`receive` (including `after` timeouts), links, monitors, `Process` functions, or trapping exits.

## genserver: GenServer
Implementing or calling a GenServer: `init`, `handle_call`, `handle_cast`, `handle_info`, `handle_continue`, state shape, `GenServer.call/cast` semantics and timeouts.

## supervisors: Supervisors and child specs
`Supervisor` or `DynamicSupervisor` modules, child specs (`child_spec/1`, `use GenServer, restart: ...`), restart strategies and intensity, or adding children to a supervision tree.

## tasks: Task and Task.Supervisor
`Task.async`/`await`, `Task.async_stream`, `Task.start`, or supervised tasks via `Task.Supervisor`.

## registry: Registry and process naming
Naming processes (`name:` options, `{:via, Registry, ...}`, `:global`) or using `Registry` for lookup or pub/sub.

## ets: ETS
Creating or querying ETS tables (`:ets.new`, `insert`, `lookup`, match specs, table options like `:read_concurrency`).

## app_config: Application and config
`Application` callbacks (`start/2`), `Application.get_env`/`fetch_env!`, or editing `config/*.exs`, especially the runtime vs compile-time distinction (`runtime.exs`, `compile_env`).

## ecto_schemas_changesets: Ecto schemas and changesets
`schema`/`embedded_schema` definitions, associations, `cast`, `validate_*`, `put_change`, constraints, or custom changeset functions.

## ecto_queries: Ecto queries and Repo
`Ecto.Query` (`from`, keyword or pipe syntax, joins, preloads, `select`, fragments) or `Repo` operations, transactions, and `Ecto.Multi`.

## ecto_migrations: Ecto migrations
Writing migrations: `create table`, `alter`, indexes, constraints, `execute`, or reversible `change` vs `up`/`down`.

## phoenix_routing: Phoenix router, plugs, and controllers
Router scopes and pipelines, writing a plug (function or module), controller actions, `conn` manipulation, or JSON/HTML rendering from controllers.

## liveview_lifecycle: LiveView lifecycle
LiveView callbacks (`mount`, `handle_params`, `handle_event`, `handle_info`), assigns and `assign`/`update`, streams, PubSub-driven updates, or `push_patch`/`push_navigate`.

## function_components: Function components and HEEx
Function components with `attr`/`slot`, `render_slot`, HEEx templates with `:if`/`:for`, or component composition.
