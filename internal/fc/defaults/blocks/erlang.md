# Erlang building blocks

Each `## id: title` section is one block. The description says what counts
as *using* the block: the code being written has to exercise it, not just
sit near it. Edit freely; ids are what the ledger tracks, so renaming an id
resets its schedule.

## pattern_matching: Pattern matching
Destructuring with `=`, matching in function heads and `case` clauses, single-assignment variables, or matching against an already-bound variable.

## guards: Guards
`when` clauses on functions, `case`, or `receive`, combining guards with `,` and `;` vs `andalso`/`orelse`, or knowing which BIFs are allowed in guards (`is_*`, `map_get`, `is_map_key`, `element`).

## multi_clause: Multi-clause functions and arity
Writing a function as several `;`-separated clauses dispatched by pattern, ordering clauses so specific ones come first, or same-named functions of different arity used as helpers and defaults.

## funs: Funs and higher-order functions
`fun(X) -> ... end` (including multi-clause and named funs), `fun Mod:Name/Arity` references, closures over bound variables, or passing funs to other functions.

## recursion: Recursion and accumulators
Hand-written recursive functions: base cases, accumulators, tail recursion, or reversing an accumulated list with `lists:reverse/1`.

## comprehensions: Comprehensions
List comprehensions, or binary and map comprehensions (`<< <<X>> || <<X>> <= Bin >>`, `#{K => V || K := V <- Map}`), especially with filters, multiple generators, or pattern-matching generators.

## lists_module: Lists and the lists module
List operations (`[H | T]`, `++` and its cost), or non-trivial use of `lists:foldl`, `keyfind`, `keysort`, `flatmap`, `partition`, or `zip`. A single trivial `lists:map` does not count.

## tuples_atoms: Tuples and atoms
Tagged tuples as data, `element`/`setelement`, or atoms as tags and enums, including the atom table limit (avoiding `list_to_atom` on untrusted input, using `list_to_existing_atom`).

## maps: Maps
Map literals, update syntax `M#{k := v}` vs `M#{k => v}`, matching on map keys, or the `maps` module beyond a single get (`maps:fold`, `maps:update_with`, `maps:with`).

## records: Records
Defining `-record`, creating, updating, and matching `#rec{}` values, field access with `R#rec.field`, sharing records through `.hrl` files, or `record_info`.

## strings_binaries: Strings, binaries, and iolists
Strings as lists vs UTF-8 binaries, the `string` and `unicode` modules, building output as iolists with `iolist_to_binary`, `io_lib:format`, or `re` regular expressions.

## bit_syntax: Bit syntax
Binary syntax `<<...>>` with sizes, types, and segments (`X:16/big`, `Rest/binary`, `C/utf8`), for parsing or building binary data and protocols.

## error_tuples: Error tuples and maybe
Returning `{ok, V}`/`{error, Reason}`, chaining fallible calls through nested `case` or `maybe ... ?= ... else ... end`, or choosing between returning an error and crashing.

## exceptions: Exceptions
`try ... of ... catch ... after`, the `throw`/`error`/`exit` classes, capturing stack traces with `Class:Reason:Stack`, or deliberate let-it-crash design.

## modules_attributes: Modules and attributes
`-module`, `-export`, `-import`, `-export_type`, `-compile`, custom attributes, or hot code loading concerns such as fully qualified `?MODULE:loop(State)` calls.

## macros: Macros and the preprocessor
`-define` with or without arguments, `-include`/`-include_lib`, `-ifdef` conditional compilation, or predefined macros like `?MODULE`, `?FUNCTION_NAME`, and `?LINE`.

## typespecs: Specs and Dialyzer
`-spec`, `-type`, `-opaque`, type unions and `when` constraints in specs, or writing code shaped for Dialyzer.

## behaviours: Custom behaviours
Defining a behaviour with `-callback` and `-optional_callbacks`, implementing one with `-behaviour`, or dispatching to a callback module held in state.

## processes: Processes and messages
`spawn`/`spawn_link`/`spawn_monitor`, sending with `!`, `receive` loops that carry state through recursion, or registering names with `register`.

## receive_patterns: Selective receive and request/reply
Selective `receive`, `after` timeouts, tagging requests with `make_ref()` and matching on the reply, process aliases, or avoiding mailbox buildup with a catch-all clause.

## links_monitors: Links, monitors, and exit signals
`link`, `monitor`, handling `'DOWN'` and `'EXIT'` messages, `process_flag(trap_exit, true)`, or `exit/2` with its signal semantics (`normal`, `kill`, `killed`).

## gen_server: gen_server
Implementing or calling a `gen_server`: `init`, `handle_call`, `handle_cast`, `handle_info`, `handle_continue`, `terminate`, state shape, or `gen_server:call` semantics and timeouts.

## gen_statem: gen_statem
Implementing a `gen_statem`: `callback_mode`, `state_functions` vs `handle_event_function`, actions such as `{reply, ...}`, `postpone`, and state or event timeouts.

## supervisors: Supervisors
A `supervisor` callback module: `init/1` returning sup flags and child specs, restart strategies and intensity, `simple_one_for_one` or `start_child`, or restart and shutdown values on children.

## otp_applications: OTP applications
The `application` behaviour (`start/2`, `stop/1`), `.app.src` contents, `application:get_env`, or `sys.config` and `vm.args` settings.

## ets: ETS
Creating or querying tables (`ets:new`, `insert`, `lookup`, `update_counter`, match specs, `select`), table types and options (`set`, `ordered_set`, `named_table`, `read_concurrency`), or table ownership and heirs.

## timers: Timers and time
`erlang:send_after`/`start_timer` and cancelling them, `timer` functions, `erlang:monotonic_time` vs `system_time`, or `calendar` conversions.

## sockets: Sockets
`gen_tcp`/`gen_udp` listen, accept, and receive loops, socket options like `{active, once}`, `{packet, N}`, or `binary`, or handling `{tcp, ...}` and `{tcp_closed, ...}` messages.

## ports_nifs: Ports and NIFs
`open_port` with `{packet, N}` or `exit_status`, handling port messages, or loading NIF stubs with `-on_load` and `erlang:load_nif`.

## distribution: Distribution
Node connections and `net_kernel`, `rpc`/`erpc` calls, sending to `{Name, Node}`, `global` names, `pg` groups, or `monitor_node`.

## logger_io: Logging and formatted output
`logger` calls and metadata, the `?LOG_*` macros, or `io:format` and `io_lib:format` control sequences (`~p`, `~s`, `~ts`, `~w`).

## eunit_ct: EUnit and Common Test
Writing tests: EUnit `_test` functions and `_test_` generators, `?assert*` macros and fixtures, or Common Test suites with `all/0`, `groups/0`, and `init_per_testcase/2`.

## rebar: rebar3 and project configuration
Editing `rebar.config` (deps, profiles, plugins, relx releases, overrides) or structuring an umbrella project under `apps/`.
