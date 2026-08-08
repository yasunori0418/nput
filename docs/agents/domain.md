# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- **`CONTEXT.md`** at the repo root, or
- **`CONTEXT-MAP.md`** at the repo root if it exists — it points at one `CONTEXT.md` per context. Read each one relevant to the topic.
- **`docs/adr/`** — read ADRs that touch the area you're about to work in. In multi-context repos, also check `src/<context>/docs/adr/` for context-scoped decisions.
- **`docs/` の item** — in this repo the normative content lives in per-item files, not in prose documents. See "Item graph" below.

If any of these files don't exist, **proceed silently**. Don't flag their absence; don't suggest creating them upfront. The producer skill (`/grill-with-docs`) creates them lazily when terms or decisions actually get resolved.

## Item graph

`docs/` is a sara knowledge graph: one file per item, each carrying YAML frontmatter with an `id`
and its relations. The normative statements live in the items; `docs/spec.md` / `docs/design.md` /
`docs/concept.md` are **overviews that index them** and deliberately carry no detail.

**`docs/concept.md` is the one exception.** Four sections in it — the design philosophy, the
comparison against existing tools, the north-star, and how the design evolved — were deliberately
left un-itemised (they don't reduce to requirements), so that prose is the primary source. Read it,
don't skim it as an index.

| Directory | Type | Prefix | Holds |
|---|---|---|---|
| `docs/solution/` | solution | `SOL` | What nput is and what it solves |
| `docs/use-cases/` | use_case | `UC` | How it gets used |
| `docs/requirements/` | requirement | `REQ` | What must hold of the product (normative, RFC2119) |
| `docs/design/` | design | `DSG` | How a requirement or a test_plan is realised |
| `docs/quality/` | quality | `QA` | Norms binding the development process (normative, RFC2119) |
| `docs/test-plan/` | test_plan | `TP` | Scope, levels and approach of testing (normative, RFC2119) |
| `docs/infrastructure/` | infrastructure | `INF` | The machinery upholding a quality — CI, release, docs site, merge gate |
| `docs/risks/` | risk | `RISK` | What threatens a requirement or a design — **not started** |
| `docs/test/<subject>/` | test_condition | `TC` | What a risk needs covered — **not started** |
| `docs/test/<subject>/` | test_case | `CASE` | A concrete case under a condition — **not started** |
| `docs/test/<subject>/` | defect | `D` | What a case revealed — **not started** |
| `docs/adr/` | adr | `ADR` | Decisions, linked by `justifies` |

The four types marked **not started** are defined in `docs/model.yaml` but have neither items nor
directories yet; they get created when the test process is taken up, at which point the granularity
of `<subject>` (per feature or per requirement) is decided.

The Holds column above describes each type as the graph currently stands, not the full set of
parents `docs/model.yaml` permits: `infrastructure` may hang off a `design` as well as a
`quality`, though every `INF` item satisfies a `QA` today.

Every type but `solution` and `adr` hangs off at least one upstream relation, so a missing link
shows up as an orphan warning. `defect` is the exception: it has no way to declare one, and so
always raises the warning — see the header of `docs/model.yaml`.

Start from the overview that matches the topic (spec for behaviour, design for structure, concept
for positioning), follow its link into the item, then traverse relations for the surrounding context.
Full conventions — ID format, placement rules, which type owns what — are in `CLAUDE.md`. Which of
`requirement` / `quality` / `test_plan` a norm belongs to, and whether a risk attaches to a
`requirement` or a `design`, are decided by `docs/agents/sara-graph.md`.

## Traverse with `sara query`

`sara` lives in the dev shell, so prefix commands with `nix develop ./dev --command`.

```bash
sara query <full-id> -u      # upstream: requirement -> use_case -> solution,
                             #           quality / test_plan -> solution
sara query <full-id> -d      # downstream: requirement / test_plan -> design, quality -> infrastructure
                             #             (risk and below not started yet)
sara check                   # validate the whole graph (broken refs / duplicate IDs / cycles)
sara report coverage         # coverage
sara report matrix           # traceability matrix
```

**`sara query` only accepts the full ID.** Prose references use the 8-character short form
(`REQ-2b0c2bb8`), which sara rejects with "Item not found". The short form is a prefix of the full
ID, so grep for it to resolve both the declaring item and everything that references it:

```bash
rg -l 2b0c2bb8 docs/
rg -o 'REQ-2b0c2bb8[0-9a-f-]*' docs/requirements/20260802-2b0c2bb8-*.md | head -1
```

Use this before claiming something isn't specified — a requirement you can't find in the overview
may still exist as an item.

## File structure

This repo (single-context, with the item graph under `docs/`):

```
/
├── CONTEXT.md
├── CLAUDE.md                          ← placement rules and ID conventions
├── sara.toml                          ← which paths the graph covers
└── docs/
    ├── concept.md  design.md  spec.md ← overviews; index the items (concept.md: see exception above)
    ├── glossary.md  glossary.ja.md    ← canonical spelling of each term
    ├── model.yaml                     ← the graph's type definitions
    ├── solution/  use-cases/          ← SOL / UC
    ├── requirements/  design/         ← REQ / DSG
    ├── quality/  test-plan/           ← QA / TP
    ├── infrastructure/                ← INF
    ├── risks/                         ← RISK (not started; no directory yet)
    ├── test/<subject>/                ← TC / CASE / D (not started; no directory yet)
    └── adr/                           ← ADR (sequential IDs, unlike the rest)
```

Generic single-context repo (no item graph) — read `CONTEXT.md` and `docs/adr/` only:

```
/
├── CONTEXT.md
├── docs/adr/
│   ├── 0001-event-sourced-orders.md
│   └── 0002-postgres-for-write-model.md
└── src/
```

Multi-context repo (presence of `CONTEXT-MAP.md` at the root):

```
/
├── CONTEXT-MAP.md
├── docs/adr/                          ← system-wide decisions
└── src/
    ├── ordering/
    │   ├── CONTEXT.md
    │   └── docs/adr/                  ← context-specific decisions
    └── billing/
        ├── CONTEXT.md
        └── docs/adr/
```

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a test name), use the term as defined in `CONTEXT.md`. Don't drift to synonyms the glossary explicitly avoids.

In this repo two files cover the vocabulary and they are not interchangeable. `CONTEXT.md` is the domain glossary — Japanese, with the reasoning and ADR references behind each term; read it to understand what a term *means*. `docs/glossary.md` (`docs/glossary.ja.md` for Japanese) fixes the canonical **spelling** of each term in short entries; consult it when writing README text, code comments, or command output, so wording stays consistent.

If the concept you need isn't in the glossary yet, that's a signal — either you're inventing language the project doesn't use (reconsider) or there's a real gap (note it for `/grill-with-docs`).

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than silently overriding:

> _Contradicts ADR-0007 (event-sourced orders) — but worth reopening because…_

The same applies to a `requirement` item: its `specification` is normative, so contradicting it is a
spec change, not an implementation detail. Say which item you're contradicting by its short ID.
