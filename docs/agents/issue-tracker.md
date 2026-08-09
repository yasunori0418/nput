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
