# Triage Labels

The skills speak in terms of five canonical triage roles. This file maps those roles to the actual label strings used in this repo's issue tracker.

| Label in mattpocock/skills | Label in our tracker | Meaning                                  |
| -------------------------- | -------------------- | ---------------------------------------- |
| `needs-triage`             | `needs-triage`       | Maintainer needs to evaluate this issue  |
| `needs-info`               | `needs-info`         | Waiting on reporter for more information |
| `ready-for-agent`          | `ready-for-agent`    | Fully specified, ready for an AFK agent  |
| `ready-for-human`          | `ready-for-human`    | Requires human implementation            |
| `wontfix`                  | `wontfix`            | Will not be actioned                     |

When a skill mentions a role (e.g. "apply the AFK-ready triage label"), use the corresponding label string from this table.

Edit the right-hand column to match whatever vocabulary you actually use.

## Structural labels (not triage roles)

These are orthogonal to the triage roles above — they describe what an issue *is*, not
what it needs next. An issue carries both (e.g. `epic` + `needs-triage`).

| Label      | Color     | Meaning                                                        |
| ---------- | --------- | -------------------------------------------------------------- |
| `tracking` | (default) | Milestone-level issue aggregating epics (e.g. #147)            |
| `epic`     | `#6f42c1` | An epic reachable from a tracking issue; bundles sub-issues    |

`epic` is applied to closed epics as well, so `label:epic --state all` lists every epic
the project has had. See `issue-tracker.md` for when to apply it.
