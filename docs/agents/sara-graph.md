# Sara Document Graph

How to place items in this repo's sara knowledge graph (`docs/model.yaml`, validated by
`sara check`). Covers the conventions that the model schema itself cannot enforce.

## Where a risk attaches: `requirement` or `design`

A `risk` may point at either a `requirement` or a `design` via `threatens`. The model
permits both, so the choice is a judgement call — make it by this rule.

> A risk attaches to `requirement` by default. Attach it to `design` only when it is a
> risk that **would disappear if the design were replaced by an alternative** — that is, a
> risk intrinsic to the design choice. A concern that the requirement itself might not be
> met always attaches to `requirement`.

The test: imagine the design being swapped for a different one that satisfies the same
requirement. If the risk goes away with it, it belongs to the design. If it survives the
swap, it belongs to the requirement.

Examples:

| Risk | Attaches to | Why |
|---|---|---|
| The unwind of the undo journal fails partway through | `design` | The undo journal is one design for "leave no trace on failure". Replace it with another (e.g. staging into a temp tree and renaming into place) and this particular failure mode is gone. |
| `apply` leaves traces on the filesystem when it fails | `requirement` | This is the requirement itself being broken. Every design has to answer for it; no change of design makes the concern go away. |

Attaching a design-specific risk to the requirement loses the information that the risk is
a cost of one particular choice. Attaching a requirement-level risk to a design hides it
from every other design that has the same exposure.

## Why this matters

Risks are where test conditions hang from (`test_condition --mitigates--> risk`), so
misplacing a risk misplaces the tests that cover it. A requirement-level risk parked under
one design leaves the other designs looking untested for a concern they share.

## How a requirement states its norm: `specification` / `specification_ja`

A `requirement` carries its norm twice — `specification` in English (which sara validates
for the presence of an RFC2119 keyword) and `specification_ja` in Japanese (which sara does
not inspect at all). Both are normative; neither is a translation gloss of the other. The
rules below keep the two readable as one norm.

Nothing enforces them mechanically. sara only asks whether *some* RFC2119 keyword is
present, never which spelling, and it does not look at `specification_ja` at all. These
rules are therefore upheld by review. Should that prove too loose, the check belongs in a
prose linter built for the job (textlint and the like) rather than in a hand-rolled script.

### `specification` uses the SHALL family only

Write `SHALL` / `SHALL NOT`. Do not write `MUST` / `MUST NOT`, even though RFC2119 makes
them synonyms — mixing both spellings for one strength makes the corpus read as if the two
differ. `SHOULD` / `SHOULD NOT` / `MAY` stay as they are; they carry strengths of their own.
The `placeholder` in `docs/model.yaml` uses SHALL for the same reason.

### `specification_ja` uses normative auxiliaries, not the plain declarative

A sentence that states the norm must end in a normative auxiliary. The plain declarative
("〜する", "〜となる") reads as a description of the current implementation rather than as a
demand on it, which is exactly the distinction the requirement exists to record.

This applies per sentence, not per item: within one `specification_ja`, the sentences that
state the norm take an auxiliary, while sentences that supply background, rationale, or a
worked example stay in the declarative. Do not force an auxiliary onto a sentence that is
not itself normative.

### The strength mapping is fixed

| `specification` | `specification_ja` |
|---|---|
| SHALL (= MUST) | 〜しなければならない |
| SHALL NOT (= MUST NOT) | 〜してはならない |
| SHOULD / SHOULD NOT | 〜すべきである / 〜すべきでない |
| MAY | 〜してもよい |

Inflected forms of these are fine where the sentence needs them (e.g. 「〜してはならず、」 to
join a following clause), as long as the strength is unchanged. What must not happen is a
strength drifting across the two fields — a `SHALL` rendered as 「〜すべきである」 weakens the
requirement in the field most readers of this repo actually read.
