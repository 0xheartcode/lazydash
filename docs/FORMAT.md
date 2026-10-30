# Git-native issue format

lazydash's local backend reads and writes issues stored directly in the Git
repository, with no database and no network. The format is **owned and specified
by git-local-issue (`gli`)**, which is the reference implementation:

- Specification: https://github.com/0xheartcode/git-local-issue/blob/main/FORMAT.md
- `gli` on crates.io / GitHub: https://github.com/0xheartcode/git-local-issue

lazydash conforms to that spec rather than defining its own; this document is a
short orientation, and the `gli` spec is authoritative wherever they differ.

## Storage model (overview)

- Each issue is one ref: `refs/issues/<uuid>`.
- The issue is an append-only chain of commits; each commit is one immutable
  operation (create, comment, set-state, add-label, add-assignee, and so on).
  The root commit is the `create` operation.
- Commits normally point at the empty tree: the issue's data lives in the commit
  message, not in files.
- The current state of an issue is not stored. It is computed by folding the
  operation log with the CRDT rules in the `gli` spec (last-writer-wins registers
  for single values like title and state, observed-remove sets for labels and
  assignees), ordered by each operation's logical `(Lamport, Op-Id)`.

Because reads depend only on Git, viewing issues never requires any external
binary and works fully offline.

## Reading and writing

- **Reading:** lazydash folds the operation log directly from Git.
- **Writing:** lazydash delegates writes to the `gli` binary when it is
  installed, so `gli` remains the single owner of operation encoding (operation
  ids, logical clocks, set semantics). Write actions are advertised only when
  `gli` is available, the same way GitHub write actions require the `gh` CLI.

## Compatibility

Per the `gli` spec, readers ignore unknown trailers and never reject a higher
`Format-Version`, so the format can grow additively. The current format version
is `1`.
