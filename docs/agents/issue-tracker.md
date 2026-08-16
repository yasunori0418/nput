# Issue tracker: GitHub

Issues and PRDs for this repo live as GitHub issues. Use the `gh` CLI for all operations.

## Conventions

- **Create an issue**: `gh issue create --title "..." --body "..."`. Use a heredoc for multi-line bodies.
- **Read an issue**: `gh issue view <number> --comments`, filtering comments by `jq` and also fetching labels.
- **List issues**: `gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, body, labels: [.labels[].name], comments: [.comments[].body]}]'` with appropriate `--label` and `--state` filters.
- **Comment on an issue**: `gh issue comment <number> --body "..."`
- **Apply / remove labels**: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **Close**: `gh issue close <number> --comment "..."`

Infer the repo from `git remote -v` — `gh` does this automatically when run inside a clone.

## Epics

An epic bundles sub-issues under a milestone-level tracking issue (#147). Epics are
plain issues — there is no separate issue type — distinguished by convention:

- **Title prefix** `epic:` (e.g. `epic: sara によるドキュメントのグラフ構造化`).
- **Label** `epic`, plus a topical label (`enhancement` / `documentation`).
- **Registered** in the tracking issue's epic table. That table is the index; the label
  makes the same set queryable with `gh issue list --label epic --state all`.

Apply `epic` when creating an issue that the tracking issue will list as an epic. Keep it
on the issue after it closes — the label is how completed epics stay discoverable.

An epic row in the tracking table may exist before its issue does (the table can name a
planned epic with "issue は未起票"). Such a row has nothing to label until it is filed.

Epics may nest: an epic can be filed under another epic rather than directly under the
tracking issue (#283 under #203, for example). Label it `epic` all the same — depth is
expressed by which issue lists it, not by the label.

## Tasks

A task is a sub-issue under an epic or tracking issue — the unit actual work and PRs are
scoped to. Label it `task`.

- **Apply** `task` when decomposing an epic into sub-issues. This is required, not
  optional: `label:task` is how the work units stay enumerable.
- **Do not apply** it to a standalone issue that belongs to no epic (a one-off bug report
  or docs fix). Those carry only topical and triage labels.
- **Keep** it after the issue closes, same as `epic`.

`tracking` / `epic` / `task` are mutually exclusive — an issue is at most one of them.

PRs reference both levels: `Closes #<task>` + `Refs #<epic>`.

## When a skill says "publish to the issue tracker"

Create a GitHub issue.

## When a skill says "fetch the relevant ticket"

Run `gh issue view <number> --comments`.

## Defect issues

Defects found in the product are managed here, not in the document graph — there is no
`defect` item type (→ ADR-0051). The graph holds norms (what should hold); a defect is an
event, and events belong to the tracker.

- **File** a defect as a GitHub issue with the `bug` label.
- **Trace** in the issue body, when the defect was revealed by executing a test case:
  the revealing test case (`CASE-xxxxxxxx`) and the requirement or design it traces back
  to (`REQ-xxxxxxxx` / `DSG-xxxxxxxx`). References are one-way — items in `docs/` never
  link back to defect issues.
- **Feed back** before closing: ask whether a `risk` or `test_condition` was missing that
  would have caught this defect earlier. If so, add the missing item(s) in `docs/` — this
  shift-left loop is what keeps the risk and test analysis from decaying into a static
  snapshot.
