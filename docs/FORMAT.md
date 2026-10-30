# Git-native issue format

lazydash's local backend stores issues directly in the git repository, with no
database and no network. This document specifies the on-disk format so other
tools can read and write the same issues. The format is versioned; issues
created by lazydash carry `Format-Version: 1`.

## Storage model

Each issue is an append-only chain of commits under its own ref:

```
refs/issues/<uuid>
```

`<uuid>` is a random RFC 4122 version-4 UUID. The ref points at the tip of the
chain. Every commit in the chain is one event:

- The **root commit** (no parent) creates the issue.
- Each **later commit** records a mutation (a comment, a state change, a field
  change) and has the previous tip as its single parent.

Commits carry no file content: every commit points at the repository's empty
tree (`git mktree </dev/null`). All issue data lives in the commit message. This
keeps the object store small and makes an issue's history a plain `git log`.

Writes update the ref with a compare-and-swap (`git update-ref <ref> <new> <old>`)
so a concurrent write is rejected rather than silently lost.

## Commit message format

A commit message is a subject line, an optional body, and a trailing block of
`Key: value` trailers:

```
<subject>

<optional body, one or more paragraphs>

State: open
Priority: P1
Labels: bug, ui
Assignee: alice
Format-Version: 1
```

Parsing rules:

- The first line is the subject. For the root commit it is the issue title.
- The trailers are the trailing contiguous run of lines matching
  `^[A-Za-z][A-Za-z0-9-]*:[ \t]*.*$`. Everything between the subject and that
  block is the body (the issue description for the root commit).
- A later commit whose subject is, for example, `Close issue` or `Update labels`
  records an event; its trailers carry the new values.

## Trailers

Trailers are written in a stable order and read case-sensitively by key:

| Key              | Meaning                                              |
| ---------------- | ---------------------------------------------------- |
| `State`          | `open` or `closed` (lower case)                      |
| `Priority`       | free-form priority label (for example `P1`)          |
| `Labels`         | comma-separated label names                          |
| `Assignee`       | comma-separated assignee names                       |
| `Milestone`      | milestone name                                       |
| `Fixed-By`       | reference to a fixing commit or change               |
| `Format-Version` | format version of the issue, currently `1`           |

List values (`Labels`, `Assignee`) are comma-separated and trimmed; empty
entries are dropped.

## Reconstructing an issue

Reading an issue means walking its chain from root to tip and folding the events:

1. Start from the root commit's title, body, and trailers.
2. Apply each later commit in order. A `State` trailer updates the open/closed
   state; other trailers replace the corresponding field; a plain event with a
   body is a comment.
3. The latest value of each field wins.

Because the chain is append-only, undo is a ref reset to the tip's parent
(`git update-ref <ref> <parent> <tip>`); the popped commit stays in the object
store and remains recoverable through the reflog. The root commit has no parent
and so cannot be undone this way.

## Interoperability

Any tool that writes commits in this layout is readable by lazydash, and issues
lazydash writes are readable by any tool that follows these rules. Reading
depends only on git plumbing, so viewing issues never requires an external
binary.
