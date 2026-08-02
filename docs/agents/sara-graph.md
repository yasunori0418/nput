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
