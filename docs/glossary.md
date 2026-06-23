# Glossary

Canonical English terms for nput. This is the authoritative spelling reference for README text, code comments, and command output. When a concept appears in prose or output, use the **canonical term** defined here and avoid the listed alternatives so wording stays consistent across the project.

Each entry fixes one canonical spelling. Definitions are intentionally short; for full rationale see `docs/spec.md` (specification) and `docs/adr/` (design decisions). The Japanese-language counterpart lives in `docs/glossary.ja.md`.

## Core placement abstraction

### placement primitive
The core of nput: a pure function that places a Nix store path at a `root`-relative `target`. It is not hidden behind a module abstraction — users compose it directly.
- **Avoid**: "placement framework", "configuration management" (nput does not generate configuration).

### engine
The placement core that owns both placement (native filesystem operations) and stale removal. It takes `manifest.json` as input, invokes only `nix` (profile) and `git` (toplevel), and is implemented as a Go library driven by the **nput CLI**. It does not generate a bash script per config. "Library" here means internal in-binary layering under `internal/`, not a publicly importable, reusable module — the stable surface is the `manifest.json` contract. The engine is stdlib-only.
- **Avoid**: "per-config generated bash script", "per-layer placement logic", "a single flat implementation fused with the CLI", "importing the engine as a public Go module".

### nput CLI
The primary user-facing UX; the `packages.nput` binary on `PATH`. It discovers an **entrypoint**, runs `nix build` / `eval` internally to obtain a named manifest, and has the engine place it. Subcommands include `apply [<name>]`, `apply --all`, `rollback`, `list-generations`, `gitignore`, and `init`.
- **Avoid**: describing a per-config `nix run .#x` wrapper as the primary UX; describing `apply` as "always builds the entrypoint" (a built link-farm can be applied with `--manifest`).

### entrypoint
The Nix config file the nput CLI reads: one of `flake.nix`, `shell.nix`, or `default.nix`. It exposes a named manifest under `nput.<name>`. The config is still written in Nix and evaluated by `nix build`.
- **Avoid**: saying "nput discovers the config contents from the CWD" — it discovers the entrypoint *file*; the config is fixed by Nix evaluation.

### module
An integration layer such as standalone, home-manager, a future NixOS module, or a devShell `shellHook`. It is purely the wiring that kicks the engine; it never places files itself and never translates to `home.file` or `systemd.tmpfiles`.
- **Avoid**: "the module places files", "the module translates to native mechanisms".

## Placement input and output

### entry / entries
A placement definition. `entries` is an attrset keyed by `target`, where each value is an entry (`{ src; subpath?; target?; method?; }`). The attribute key is the identifier and supplies the default `target`. There is no manual `name` field; key uniqueness is guaranteed natively by Nix attrset keys.
- **Avoid**: "file entry" (the source may be a directory); giving an entry a `name` field; calling `entries` a list of `{ name; … }` (the old form).

### src
The placement source of an **entry** (which store path or repository). The default is a **store link** into the Nix store; out-of-store is opt-in via an explicit marker. It is orthogonal to **subpath**; do not read `src` as an abbreviation of `source`.
- **Avoid**: confusing it with **subpath**; calling it `source`.

### subpath
The relative path selecting which path inside **src** to take, for an **entry**. Works for both files and directories; default `"."`. Omitting `subpath` is the canonical way to select the whole repository (`subpath = "."` is the explicit form).
- **Avoid**: the old names `source` / `dir`; confusing it with **src**; assuming a dedicated token/marker is needed for "whole repo" (omission expresses it).

### target
The placement destination of an **entry**, specified as a path relative to **root**. The `entries` attribute key is the default `target` and is also the entry's identity (the diff key for stale removal, and its uniqueness key).

### root
The base path for placement. Chosen explicitly via the `root` argument of the public API (there is **no implicit default**). Its type is a union of `string` (absolute path, fixed at evaluation time) and `marker` (resolved at runtime). The markers are **projectRoot**, **homeRoot**, and **systemRoot**.
- **Avoid**: "`$HOME` is fixed", "defaults to home mode", "the default when `root` is omitted"; describing a marker as "sugar that returns a path string" (it is a runtime-resolution kind, not expanded to a path at evaluation time).

## Placement modes

### home mode
The placement mode where **root** = `$HOME`, selected explicitly with the `homeRoot` marker. Used by both standalone and modules such as home-manager. It commits a generation each time and exposes `--rollback` to the user.
- **Avoid**: "standalone-only", "the default when `root` is omitted".

### project mode
The placement mode where **root** = the project root, selected with the `projectRoot` marker. Its placements are **ephemeral placement** (not committed); generations and rollback are not exposed to the user. The root resolves to the git toplevel.
- **Avoid**: "relative to CWD", "relative to the config file".

### system mode
The placement mode where **root** = `/`, selected with the `systemRoot` marker. Used for the distro concept (root = `/`).

### projectRoot
The root marker selecting **project mode**. At runtime it resolves the git toplevel as **root** (overridable with `--root`). One of the root markers alongside `homeRoot` / `systemRoot`, following the same "pass a marker to opt into behavior" pattern as `mkOutOfStoreSymlink`.
- **Avoid**: interpreting it as pointing to the config file location.

### homeRoot
The root marker selecting **home mode**. At runtime it resolves `$HOME` as **root**. It promotes the previously implicit `$HOME` default into an explicit marker.

### systemRoot
The root marker selecting **system mode** (root = `/`). Promotes what ADR-0004 called a "future absolute-path string seam" into a first-class marker.

### ephemeral placement
The nature of placements in **project mode**: regenerated on each clone and never committed to the project, so activation never touches git state.
- **Avoid**: confusing it with "vendoring" or "placement that commits artifacts".

## Placement kinds

### store link
The core, default placement: a symlink whose destination is a Nix store path. The default path that guarantees reproducibility. "Unification" means making the store the default/core and demoting out-of-store to an explicit exception.
- **Avoid**: confusing it with out-of-store symlink; calling it a "copy".

### out-of-store symlink
A live symlink to a local absolute path, opted into only via `nput.lib.mkOutOfStoreSymlink "/abs/path"` (for live editing of dotfiles under development). An explicit escape hatch, not a first-class feature.
- **Avoid**: treating it as default behavior; producing it via implicit branching on the type of `src`.

## State management

### generation
The unit of rollback, managed on nput's own Nix profile (the `nix-env --profile <dir>` style). Commit (`--set`), rollback, switching to an arbitrary generation, listing, and pruning are all unified under the `nix-env --profile <dir>` family; only store GC uses `nix-collect-garbage`.
- **Avoid**: narrating it under a "stateless script" premise (reversed since the initial direction); managing it with the new `nix profile` CLI (which requires a profile-manifest and does not work on `nix-env --set` profiles).

### store manifest
The generation-derived record of "what nput placed". Concretely the `manifest.json` (carrying a `schemaVersion`) inside the link-farm derivation, generated by Nix (`lib.mkManifest`) and read by the Go engine — the Nix↔Go contract. It underpins the engine's conservative stale removal (it deletes only nput-managed symlinks that point where the record says, and never touches the user's real files).

### method
The entry field that selects the placement kind (renamed from the old `mode`). Renamed to avoid misreading it as a unix file mode.
- **Avoid**: the old name `mode`.

## Flagged ambiguities

- **"symlink" alone is ambiguous.** Always disambiguate to either **store link** or **out-of-store symlink**. The default store link is itself realized as a symlink, so the bare word does not identify the kind.
- **"unification" is not "removal".** Store-link unification does not delete out-of-store; it demotes it from the default and isolates it behind an explicit function.
- **`src` and `subpath` are distinct.** `src` = which thing (store path / repository); `subpath` = which path inside it. Similar names, orthogonal concepts. Do not use the old name `source` (= today's `subpath`).
- **"standalone" is a launch form, not a placement mode.** Standalone (invoking the CLI directly) is orthogonal to placement mode (home / project / system); the mode is decided by the `root` marker.
