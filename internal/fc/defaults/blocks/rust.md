---
schema_version: 2
---
# Rust building blocks

Each `## id: title` section is one block. The description says what counts
as *using* the block: the code being written has to exercise it, not just
sit near it. Edit freely; ids are what the ledger tracks, so renaming an id
resets its schedule.

Blocks under `# Beginner` are for people who set this language to beginner
during install, and are one small piece of syntax each. Blocks under
`# Intermediate` are for everyone else.

# Beginner

## let_bindings: let bindings
Declaring variables with `let`, `let mut`, a type annotation like `let x: i32 = 1`, or shadowing a name with a second `let`.

## scalar_types: Scalar types and casts
Using `i32`, `u64`, `f64`, `bool`, or `char` explicitly, or converting with `as`.

## if_expressions: if expressions
Writing `if`/`else`, including using `if` as a value: `let x = if a { 1 } else { 2 };`.

## loop_basics: Loops
Writing `for x in items`, `for i in 0..n`, `while`, or `loop` with `break`.

## function_basics: Functions
Declaring `fn` with typed parameters and a `-> T` return type, including returning the final expression without a semicolon.

## reference_basics: Reference basics
Passing or accepting `&T` or `&mut T` in a simple call, or dereferencing with `*`. Lifetime annotations do not count.

## struct_basics: Struct basics
Defining a small struct, building one with field syntax, or reading and assigning its fields.

## match_basics: match basics
A `match` on a simple value or small enum with a few arms and optionally `_`.

## option_basics: Option basics
Writing `Some(x)` or `None`, or handling an `Option` with `if let Some(x) = value`.

## vec_basics: Vec basics
Creating a vector with `vec![]` or `Vec::new()`, `push`, indexing, or `len`.

## string_basics: String basics
Creating text with `String::from` or `.to_string()`, passing `&str`, or combining values with `format!`.

## printing_macros: Printing macros
`println!` or `eprintln!` with `{}` or `{:?}` placeholders.

# Intermediate

## ownership: Ownership and moves
Code shaped by move semantics: deciding whether a function takes or returns ownership, `Clone` vs `Copy`, partial moves out of structs, or `std::mem::take`/`replace`. Passing a value along without any such decision does not count.

## borrowing: Borrowing and references
Deciding between `&T`, `&mut T`, and slices (`&str`, `&[T]`) in signatures, satisfying the one-writer-or-many-readers rule, reborrowing, or splitting borrows with `split_at_mut` or by field. A plain `&self` receiver does not count.

## lifetimes: Lifetimes
Explicit lifetime parameters on functions, structs, or impls, reasoning about elision, `'static` bounds, or higher-ranked bounds (`for<'a>`).

## structs_impls: Structs and impl blocks
Defining structs with `impl` blocks: associated functions and constructors, `self` vs `&self` vs `&mut self` receivers, tuple and newtype structs, or struct update syntax.

## enums: Enums
Defining enums with data-carrying variants and methods on them, or modeling states so invalid ones can't be represented.

## pattern_matching: Pattern matching
`match` with destructuring, guards, `@` bindings, ranges, and nested patterns, or `if let`, `while let`, `let else`, and `matches!`.

## option_result: Option and Result combinators
`map`, `and_then`, `ok_or`, `unwrap_or_else`, `filter`, `transpose`, `as_ref`, `take`, or `collect` into a `Result`. A single `unwrap` does not count.

## error_propagation: Error propagation and error types
Returning `Result` with the `?` operator, `map_err`, custom error enums implementing `std::error::Error` and `From`, `Box<dyn Error>`, or `thiserror`/`anyhow`.

## traits: Defining and implementing traits
Defining a trait with required and default methods, implementing one for a type, supertraits, associated types and constants, or the orphan rule and newtype workarounds.

## std_traits: Standard library traits
Implementing or deriving with intent: `Display`, `From`/`Into`, `Default`, `PartialEq`/`Eq`/`Hash`, `Ord`, `Deref`, `Drop`, `AsRef`, `FromStr`, `TryFrom`, or operator traits like `Add` and `Index`.

## generics: Generics and trait bounds
Type parameters on functions, structs, or impls, `where` clauses, `impl Trait` in argument or return position, blanket impls, const generics, or `PhantomData`.

## trait_objects: Trait objects
`dyn Trait` behind `Box`, `&`, or `Arc`, choosing dynamic vs static dispatch, dyn compatibility, or downcasting with `Any`.

## closures: Closures
Writing closures that capture state, `Fn`/`FnMut`/`FnOnce` bounds, `move` closures, returning closures, or storing `Box<dyn Fn>`.

## iterators: Iterators
Non-trivial adapter chains (`filter_map`, `flat_map`, `fold`, `zip`, `windows`, `peekable`), `iter` vs `iter_mut` vs `into_iter`, `collect` with a target type, or implementing `Iterator` for your own type. A single trivial `map` does not count.

## collections: Collections
`Vec` beyond push (`retain`, `drain`, `dedup`, `sort_by_key`), `HashMap`/`BTreeMap` and the entry API, `HashSet`, `VecDeque`, or `BinaryHeap`.

## strings: Strings and text
`String` vs `&str`, UTF-8 and char boundaries, `chars`/`bytes`/`split`/`parse`, `format!` and format specifiers, `Cow<str>`, or `OsStr`/`Path` conversions.

## smart_pointers: Smart pointers
`Box` for recursive or heap-allocated types, shared ownership with `Rc`/`Arc`, `Weak` to break cycles, or `Cow` for clone-on-write.

## interior_mutability: Interior mutability
`Cell`, `RefCell` and its runtime borrow panics, `OnceCell`/`OnceLock`/`LazyLock`, or the `Rc<RefCell<T>>` pattern.

## threads: Threads
`std::thread::spawn` with `move`, `thread::scope`, joining and handling panics, or `Send` and `Sync` bounds.

## sync_primitives: Shared state and channels
`Arc<Mutex<T>>`, `RwLock`, atomics and memory ordering, `Condvar`, `Barrier`, or `mpsc` channels, including keeping lock guards short-lived.

## async_await: async and await
`async fn` and `.await`, spawning and joining tasks, `tokio::select!`, async channels, cancellation, not blocking the runtime, or `Send` issues with values held across `.await`.

## pinning_futures: Pinning and manual futures
Implementing `Future` by hand, `Pin` and `Unpin`, `Poll` and `Waker`, or implementing `Stream`.

## macros: Declarative macros
`macro_rules!`: fragment specifiers, repetitions, hygiene, or recursive macros.

## proc_macros: Procedural macros
Writing a `derive`, attribute, or function-like proc macro with `syn` and `quote`.

## modules_visibility: Modules and visibility
`mod` trees across files, `pub`, `pub(crate)`, and `pub(super)`, `use` re-exports with `pub use`, or designing a crate's public API.

## cargo: Cargo and features
Editing `Cargo.toml` (dependencies, feature flags, workspaces, profiles), `#[cfg(feature = ...)]` and other conditional compilation, or `build.rs`.

## unsafe_ffi: unsafe and FFI
`unsafe` blocks and functions, raw pointers, upholding safety invariants in a safe wrapper, `extern "C"`, `#[repr(C)]`, or `MaybeUninit`.

## io_files: I/O and files
`std::io::Read`/`Write`/`BufRead`, `BufReader`, `std::fs`, `Path`/`PathBuf`, or stdin, stdout, and stderr handling.

## serde: Serde
`Serialize`/`Deserialize` derives, attributes such as `rename_all`, `default`, `flatten`, and `tag`, or a custom `Serializer` or `Deserializer`.

## cli: Command-line programs
`std::env::args` or `clap` derive parsing, subcommands, exit codes with `std::process::ExitCode`, or `std::process::Command` to run subprocesses.

## testing: Tests
`#[cfg(test)]` modules, `#[test]` with `assert!`/`assert_eq!` and `#[should_panic]`, tests returning `Result`, integration tests in `tests/`, doc tests, or property tests with `proptest`.
