# nput

*Read this in [Japanese (日本語)](README.ja.md).*

> Place fetched git repositories at arbitrary paths via symlink or copy.

nput is a Nix library and module set that **places the contents of an already-fetched
Nix store path at a `root`-relative target** — as a symlink or a copy. It does **not**
generate configuration. It puts a repository's contents where you ask, untouched.

The core is a **placement primitive**: a pure function that places a Nix store path at a
`root`-relative `target`. It is not hidden behind a module abstraction — you compose it
directly. `home.file`-style placement (`root` = `$HOME`) is just one application; `root`
is chosen explicitly with the `projectRoot` / `homeRoot` / `systemRoot` markers (there is
**no implicit default**).

> **Status: MVP / implementation phase.** The implemented scope is the standalone CLI +
> **project mode** as the core, with **home mode** also supported. NixOS / nix-darwin
> modules and **system mode** are future work. See [MVP status](#mvp-status) for the full
> matrix. APIs may still change.

---

## Why nput

Nix can *fetch* a repository (`fetchFromGitHub`, `fetchGit`, flake inputs, npins, …) but
*placing* its contents onto the filesystem is a separate problem. The usual answers all
have costs:

- **home-manager `home.file`** places files, but requires home-manager and assumes a
  *whole-environment* model: one `home-manager switch` updates everything at once, so one
  change can ripple across — or break — unrelated tools. The file module cannot be
  extracted as a standalone library.
- **A shell `git clone`** works but loses version pinning, reproducibility, and Nix
  integration.
- **Module abstractions** (home-manager / NixOS / nix-darwin / system-manager) declare
  placement through the Nix module system and hide the behavior behind an abstraction,
  translating to a platform-native mechanism (`home.file`, `systemd.tmpfiles`, …). That
  takes the control of "what goes where, and how" out of your hands and duplicates
  behavior per layer.

nput separates **fetching** (Nix evaluation: `src` is a store path) from **placement**
(a fixed runtime engine), and keeps placement behavior in a **single core** that you drive
explicitly:

- **No configuration generation.** nput never translates module options into config
  files. It places what the repository already contains.
- **Independent units.** Each placement config (`nput.<name>`) is its own Nix profile.
  Update and apply each role independently — one update never ripples to another.
- **No home-manager dependency.** The `lib/` core depends only on nixpkgs. It runs
  standalone; module integrations (home-manager, devShell, future NixOS/nix-darwin) are
  thin wiring that only *kick* the engine — they never place files themselves.

---

## How it works

nput has two layers:

```
[nput CLI]  packages.nput — on PATH, the primary UX
  · discovers an entrypoint (flake.nix / shell.nix / default.nix)
  · runs `nix build` / `nix eval` internally to obtain a named manifest
  · drives the engine to place, prune stale links, and swap the profile
   ↓ manifest.json
[engine]    a Go library (stdlib-only)
  · takes manifest.json as input; invokes only `nix` (profile) and `git` (toplevel)
  · native filesystem operations for place / replace / remove + conservative stale removal
```

- The **engine** owns placement and stale removal. It reads `manifest.json` — the stable
  Nix↔Go contract — and performs native filesystem operations.
- `lib.mkManifest` is a **pure function** that produces a link-farm derivation
  (`manifest.json` + a symlink farm). It has no side effects.
- An **entrypoint** is the Nix file the CLI reads (`flake.nix`, `shell.nix`, or
  `default.nix`); it exposes a named manifest under `nput.<name>`. The config is still
  written in Nix and evaluated by `nix build`.

---

## Requirements

- **Nix** with experimental features enabled in your environment:
  `experimental-features = nix-command` (and `flakes` for flake entrypoints).
  nput does **not** silently inject `--extra-experimental-features`; if a feature is not
  enabled it stops with a clear message explaining the prerequisite and how to enable it.
- **git** on `PATH` (used to resolve the project root in project mode).

---

## Install

### Standalone (home mode)

Install the CLI globally so `nput` is on `PATH`:

```bash
nix profile install github:yasunori0418/nput
```

To avoid `schemaVersion` skew, align the CLI and the `nput` your flake pins to the **same
input** (the global CLI and your flake's `nput.lib` are otherwise separate inputs that can
drift; the engine rejects a `schemaVersion` newer than its own).

### Project mode (canonical: pin in the devShell)

For project mode, the canonical form is to **bundle a pinned `nput`** in the project's
devShell so the CLI and `nput.lib` come from the same flake input (locked by `flake.lock`):

```nix
devShells.${system}.default = pkgs.mkShell {
  packages  = [ nput.packages.${system}.nput ];   # pinned nput on PATH
  shellHook = "nput apply <name> --no-wait";        # placed on `nix develop` / direnv entry
};
```

### Scaffold a new project

`nput init` is a transparent wrapper over `nix flake init -t`; nput itself generates
nothing, and existing files are never overwritten:

```bash
nput init standalone   # homeRoot example
nput init project      # projectRoot example + devShell wiring + .gitignore guide
```

---

## Quickstart

### Project mode (the central use)

In **project mode** the root is the project root (the git toplevel). Placements are
**ephemeral** — regenerated on each clone, never committed — so activation never touches
git state. This is the central way to use nput: embed it in a repo and place store paths
at arbitrary in-repo paths, kicked from a devShell.

```nix
# flake.nix — entering the repo places .claude/skills/nix from a fetched store path
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.nput.url    = "github:yasunori0418/nput";

  inputs.claude-skills.url   = "github:someone/claude-skills";
  inputs.claude-skills.flake = false;

  outputs = { self, nixpkgs, nput, ... }@inputs:
    let
      system = "x86_64-linux";
      pkgs   = nixpkgs.legacyPackages.${system};
    in
    {
      nput.${system}.skills = nput.lib.mkManifest {
        root = nput.lib.projectRoot;        # resolves to the git toplevel at runtime
        entries = {
          ".claude/skills/nix" = { src = inputs.claude-skills; subpath = "skills/nix"; };
        };
      };

      devShells.${system}.default = pkgs.mkShell {
        packages  = [ nput.packages.${system}.nput ];   # pinned nput (canonical)
        shellHook = "nput apply skills --no-wait";       # place on shell entry
      };
    };
}
```

```bash
# Enter the shell — the placement runs automatically via the shellHook
nix develop          # or: direnv allow

# List the targets the project owner should add to .gitignore (stdout only; no writes)
nput gitignore skills >> .gitignore
```

- `--root <path>` overrides the resolved root in any mode (escape hatch for git-less
  trees, debugging, etc.).
- Generations are an internal mechanism here; `rollback` / `list-generations` are **not**
  exposed in project mode (rollback is meaningless for ephemeral placements).
- In a devShell, use a **named apply** (`nput apply skills`) or
  `nput apply --all --project-root`. A bare `--all` would also place any home-mode configs
  into `$HOME` — a footgun in mixed entrypoints.

### Home mode (standalone, roles as separate profiles)

In **home mode** the root is `$HOME`, selected with `homeRoot`. Each config is its own
profile, committed every apply, with user-facing `rollback`.

```nix
# flake.nix — each role is a named manifest = an independent profile
outputs.nput.${system} = {
  vim-plugins = nput.lib.mkManifest {
    root = nput.lib.homeRoot;
    entries = {
      ".local/share/nvim/site/pack/foo/start/foo" = { src = inputs.vim-foo; };
      ".local/share/nvim/site/pack/bar/start/bar" = { src = inputs.vim-bar; };
    };
  };

  zsh-plugins = nput.lib.mkManifest {
    root = nput.lib.homeRoot;
    entries = {
      ".zsh/plugins/autosuggestions"     = { src = inputs.zsh-autosuggestions; };
      ".zsh/plugins/syntax-highlighting" = { src = inputs.zsh-syntax-highlighting; };
    };
  };
};
```

```bash
# Update / apply / roll back each role independently (separate profiles)
nput apply vim-plugins
nput rollback vim-plugins          # home mode only
nput apply zsh-plugins
nput list-generations vim-plugins
```

After updating a `src` (a flake input update, `npins update`, …), re-apply only the
affected config to keep the change from touching any other tool.

### home-manager module

The module pins `root = homeRoot` (you do not re-specify `root`). It kicks the engine from
`home.activation` via `nput apply --manifest <link-farm>` — it never delegates to
`home.file`. nput keeps its own profile as an **internal mechanism**; user-facing rollback
is unified on the host (`home-manager --rollback`).

```nix
imports = [ inputs.nput.homeManagerModules.default ];

nput = {
  enable = true;
  entries = {
    # external repo (store link)
    ".claude/skills/nix" = { src = inputs.skills-repo; subpath = "skills/nix"; };
    # theme as a copy (place-once, then user-managed)
    ".local/share/themes/dark" = { src = inputs.themes; subpath = "dark"; method = "copy"; };
    # live editing of local dotfiles via out-of-store symlink
    ".config/nvim" = { src = nput.lib.mkOutOfStoreSymlink "/home/me/dotfiles"; subpath = "home/.config/nvim"; };
  };
};
```

> The home-manager module is a **single manifest = one profile** (fixed name `default`) in
> the MVP — it has no `<name>` dimension, so **role separation is not available through the
> module**. Use the standalone CLI path (`nput.<name>` entrypoints) for multiple
> independent profiles.

### Adding nput to an existing flake

`nput init` is for new projects. To retrofit an existing `flake.nix`, do these four steps
by hand (nput never auto-merges your flake — "do not generate configuration"):

1. **Add the input**: `inputs.nput.url = "github:yasunori0418/nput";`
2. **Expose a manifest**:
   `outputs.nput.<system>.<name> = nput.lib.mkManifest { root = nput.lib.projectRoot; entries = { ... }; };`
3. **Bundle pinned nput in the devShell**: `packages = [ nput.packages.${system}.nput ];`
4. **Wire a named apply**: `shellHook = "nput apply <name> --no-wait";`

If the repo uses **flake-parts**, write step 2 via the flake module instead (keeps `pkgs`
consistent with `perSystem`):

```nix
imports = [ inputs.nput.flakeModules.default ];
perSystem = { pkgs, ... }: {
  nput.<name> = inputs.nput.lib.mkManifest {
    inherit pkgs;
    root = inputs.nput.lib.projectRoot;
    entries = { ... };
  };
};
# flake-parts transposes this to flake.nput.<system>.<name> — the CLI addressing is unchanged.
```

> `nix flake check` reports `warning: unknown flake output 'nput'` but exits 0 (harmless,
> expected — the `nput` namespace keeps manifests out of `packages`). Verify artifacts with
> `nix build .#nput.<system>.<name>`.

---

## `entries` schema

`entries` is an **attrset keyed by `target`** — the attribute key is the identifier and
supplies the default `target`. Uniqueness is guaranteed natively by Nix attrset keys; there
is no manual `name` field.

```nix
entries = {
  "<target>" = {
    src     = ...;          # required
    subpath = ".";          # optional, default "." (the whole repository)
    target  = "<target>";   # optional, default = the attribute key
    method  = "symlink";    # optional, default "symlink" | "copy"
  };
  # ...
};
```

The entry submodule is **strict** (unknown keys are rejected); typos and old names
(`name` / `source` / `dir` / `mode`) are evaluation-time errors.

### `src` — required

The placement source. The default is a **store link** (a symlink into the Nix store, which
guarantees reproducibility). Out-of-store is opt-in via an explicit marker.

| `src` value | Symlink points to | Use |
|---|---|---|
| `path` (e.g. `inputs.myrepo`) | Nix store (immutable) | version-pinned external repo |
| `builtins.path { path = /home/...; name = "..."; }` | Nix store (local copied in) | a local tree via the store |
| `set` (e.g. `pkgs.fetchFromGitHub { ... }`) | Nix store (immutable) | version-pinned external repo |
| `marker` (`nput.lib.mkOutOfStoreSymlink "/abs/path"`) | local FS (live) | local dotfiles under development |

```nix
src = inputs.myrepo;                                    # store link
src = pkgs.fetchFromGitHub { owner = "..."; repo = "..."; rev = "..."; hash = "..."; };
src = builtins.path { path = /path/to/dotfiles; name = "dotfiles"; };
src = nput.lib.mkOutOfStoreSymlink "/path/to/dotfiles"; # out-of-store (live), explicit

# Removed: passing a bare string for implicit out-of-store is not supported.
# src = "/path/to/dotfiles";   # error
```

### `subpath` — default `"."`

The relative path selecting which path **inside `src`** to take. Works for files and
directories. Omitting it (or `"."`) selects the whole repository.

```nix
subpath = ".";                  # whole repository (explicit form)
subpath = "skills/nix";         # a subdirectory
subpath = "themes/dark.json";   # a single file
```

`src` and `subpath` are orthogonal: `src` = *which thing* (store path / repository),
`subpath` = *which path inside it*.

### `target` — default = the attribute key

The destination, **relative to `root`**. The attribute key is the default `target` and is
also the entry's identity (the diff key for stale removal, and its uniqueness key). You can
make the key a logical label and override `target` explicitly (like `home.file`).

### `method` — default `"symlink"`

Selects the placement kind:

| `method` | `src` kind | Behavior | Generations |
|---|---|---|---|
| `"symlink"` | path / set | symlink into the Nix store (read-only) | yes (profile) |
| `"symlink"` | marker | out-of-store symlink to a local path (live) | yes (link target only) |
| `"copy"` | path / set | **place-once** copy (writable, user-managed) | **no** |
| `"copy"` | marker | evaluation-time error (contradictory) | — |

**copy is place-once and user-managed.** Once materialized, nput does not touch the target.
The store's read-only mode (`0444` / `0555`) is preserved but owner-write is added so the
copy is editable. To follow an upstream `src` update, use `nput apply --recopy` (overwrites
all copy targets unconditionally) or `nput reset` then re-apply. Copies are not generation-
managed and are never rolled back. If a foreign real file already occupies a copy target,
nput skips it but emits a **warning** (it never overwrites your file).

### Generating entries dynamically

Interpolate names into the target key. Derive the target from a variable **you** control —
not from `baseNameOf src` (a store path resolves to `/nix/store/<hash>-source`, so
`baseNameOf` yields `<hash>-source`):

```nix
let plugins = [ "telescope" "treesitter" "cmp" ]; in
nput.lib.mkManifest {
  root = nput.lib.homeRoot;
  entries = builtins.listToAttrs (map (n: {
    name  = ".local/share/nvim/site/pack/plugins/start/${n}";  # key = target
    value = { src = inputs.${n}; };
  }) plugins);
}
```

To enumerate subdirectories, `builtins.readDir` an **already-realised store path /
`flake = false` input** (a raw `fetchFromGitHub` derivation would trigger import-from-
derivation and break pure flake eval):

```nix
let
  skills = builtins.readDir "${inputs.claude-skills}/skills";
  names  = builtins.attrNames (nixpkgs.lib.filterAttrs (_: t: t == "directory") skills);
in
nput.lib.mkManifest {
  root = nput.lib.homeRoot;
  entries = builtins.listToAttrs (map (n: {
    name  = ".claude/skills/${n}";
    value = { src = inputs.claude-skills; subpath = "skills/${n}"; };
  }) names);
}
```

---

## Command reference

The CLI discovers the entrypoint in the CWD (`flake.nix` → `shell.nix` → `default.nix`),
overridable with `-f`. Each `nput.<name>` is an independent profile; `<name> = default` is
resolved by `nput apply` when the name is omitted.

```bash
nput apply [<name>]            # apply nput.<name> (omitted = nput.default); builds, commits a new generation, places
nput apply <name> --dryrun     # read-only plan: place/replace/remove/conflict/no-op, zero side effects
nput apply <name> --recopy     # also overwrite every copy target from its src unconditionally
nput apply --manifest <farm>   # apply a pre-built link-farm directly (no entrypoint discovery / eval / build)
nput apply --all               # apply every nput.* in lexicographic order; continues past failures
nput apply --all --project-root # apply only projectRoot configs (also --home-root / --system-root)
nput reset <name> [target...]  # tear down placements (no profile change); target omitted = all entries
nput reset <name> --dryrun     # show what would be removed; zero side effects
nput rollback <name>           # roll back to the previous generation (home mode only; name required)
nput list-generations <name>   # list generations (home mode only)
nput list-generations --all    # list generations for all home-mode configs
nput gitignore <name>          # print placement targets for .gitignore to stdout (no writes; project mode only)
nput gitignore --all           # sorted + deduped targets for all projectRoot configs
nput init <template>           # wrapper over `nix flake init -t github:yasunori0418/nput#<template>`
```

### Global flags

```text
-f, --file <path>   # specify the entrypoint explicitly (overrides auto-discovery)
--root <path>       # override the resolved root in any mode
--no-wait           # on lock contention, skip instead of waiting (for shellHook; explicit apply blocks by default)
-v, --verbose       # print the placement report (summary + per-target lines); default is silent on success
--debug             # reveal the internal nix commands on stderr (for troubleshooting)
--project-root      # --all qualifier: only projectRoot configs (also --home-root / --system-root)
--recopy            # apply qualifier: overwrite every copy target from src
--manifest <path>   # apply only: apply a pre-built link-farm directly
-y, --yes           # skip reset's confirmation prompt (for scripts / CI)
```

### Output and exit codes

- **Silent on success by default** ("silence is golden"). The placement report, try-lock
  skip notices, and the `apply --all` summary are **not** printed unless you pass `-v` /
  `--verbose`. Pass `-v` to opt into the report on stderr.
- **`--debug`** reveals the internal nix commands (verbosity `-v` and debugging are
  orthogonal). There is **no `--quiet`** (removed when success became silent by default),
  and **no `--json`** in the MVP.
- **Stream discipline**: stdout is reserved for machine-readable output (`gitignore`
  listings, `apply --dryrun` plans) — printed even at the default verbosity, so
  `nput gitignore <name> >> .gitignore` and `nput apply <name> --dryrun | ...` pipe safely.
  **Warnings (e.g. foreign symlinks) and errors always go to stderr** and are never
  silenced.

| Exit code | Meaning |
|---|---|
| `0` | success / no-op / `--no-wait` try-lock skip |
| `1` | general error (eval error, engine runtime error, `apply --all` partial failure) |
| `2` | `apply --dryrun` detected a conflict (usable as a CI pre-gate) |

### Behavior notes

- **Idempotent.** Re-applying converges to the same result. For symlinks, nput
  conservatively removes only the stale links it recorded as placing that still point where
  the record says — it never touches your real files or foreign links. Existing nput
  symlinks are replaced; a foreign symlink is replaced with a warning; a real file or
  directory at the target is an error (no overwrite).
- **Generations** ride on nput's own Nix profile (`nix-env --profile <dir>`). Switch to an
  arbitrary generation, prune, and GC via the standard `nix-env` / `nix-collect-garbage`
  against the profile path. project mode skips committing a new generation when the
  link-farm is unchanged (but still repairs drifted entries via `lstat`).
- **`apply --all`** applies each config independently (each is atomic on its own profile),
  continues past failures, and exits non-zero if any failed. It is **not** atomic as a
  whole.
- **`reset`** is a filesystem-only teardown: it removes nput-managed symlinks
  (conservatively) **and** deletes copy targets (the only explicit way to remove a copy). It
  requires a name (`--all` is not supported), requires confirmation or `-y`, and leaves the
  profile/generations untouched — entries still in the config are re-placed on the next
  apply.

---

## Comparison with other tools

The axis is not "feature presence" but **"hide placement behind a module abstraction, or
expose it as a pure function you control."**

| Tool | Role | Approach | Difference from nput |
|---|---|---|---|
| npins / niv | source version pinning | — | does not place files (orthogonal — compose with nput) |
| home-manager `home.file` | file placement + generations | module (generate / declare) | requires HM; whole-environment model; the file module cannot be extracted standalone |
| `mkOutOfStoreSymlink` (HM) | out-of-store symlink | helper inside a module | HM-only; nput provides an equivalent as a dependency-free explicit function |
| nixpkgs `linkFarm` / `symlinkJoin` | store-internal symlink trees | pure function | output stays *inside* the store; never placed at an arbitrary out-of-store path (nput uses it internally) |
| `nix profile` | generation management | — | placement target is fixed at `~/.nix-profile`; no arbitrary-path placement (nput rides on it) |
| `systemd.tmpfiles` (`L`) | declarative symlink to an arbitrary path | module (NixOS) | low-level, NixOS-only, no copy / generations / fetch abstraction |
| numtide/system-manager | non-NixOS `/etc` + systemd + packages | module (`lib.evalModules`) | overlapping domain but the **opposite** approach; no arbitrary-path placement, HOME dotfiles, or subdirectory extraction |
| `git clone` (shell) | clone and place | imperative | no reproducibility or Nix integration |
| **nput** | independent placement of fetched sources + generations + explicit out-of-store | **pure function, user-managed** | — |

No single existing tool is nearly identical. The building blocks (symlink farm, nix
profile, out-of-store, arbitrary-path symlink) all exist, but nput is the only one that
bundles them as "fetch-agnostic + non-generating + per-entry application + HM-independent
pure-function core + cross-platform shared schema + arbitrary-path placement × generation
management." In particular nput does **not** compete with system-manager: the domains
(packages / systemd / `/etc`) overlap, but the philosophy ("hide in a module" vs. "expose
as a pure function") differs at the design level. For a real distro base, system / service /
package layers would be delegated to or combined with system-manager, while nput stays a
**granular arbitrary-path placement primitive**.

---

## MVP status

| Area | Status |
|---|---|
| Standalone CLI (`apply` / `reset` / `rollback` / `list-generations` / `gitignore` / `init`) | implemented (core) |
| project mode (`projectRoot`) | implemented (core) |
| home mode (`homeRoot`) | implemented |
| home-manager module | implemented — single profile (fixed name `default`); no role separation |
| Generations / rollback (home mode) | implemented |
| copy (place-once) / out-of-store symlink | implemented |
| flake-parts module | implemented |
| `manifest.json` schema | v1 only; no migration / backward-compat machinery yet |
| `--json` machine-readable output | not in the MVP (future) |
| NixOS / nix-darwin modules | future |
| system mode (`systemRoot` = `/`) | future (seam only; evaluation-time error if selected today) |

**Known limitations / honest caveats**

- The function-based "package install + PATH" mechanism for the distro north-star is
  undefined and out of scope.
- Boot / init / filesystem / partition layers are not nput's domain.
- Removing a clone leaves an orphan profile directory under
  `<state>/nix/profiles/nput/` (the store is freed by `nix-collect-garbage`, but the
  profile directory remains). There is no `prune` command in the MVP; remove it manually.
- The home-manager module cannot separate roles into multiple profiles in the MVP — use the
  standalone CLI for that.

---

## Documentation

The full design and specification documents are currently maintained in Japanese:

- `docs/concept.md` — concept, design philosophy, comparison with existing tools
- `docs/design.md` — design (layers, flake outputs, module design, usage patterns)
- `docs/spec.md` — specification (lib API, entries schema, placement behavior, error spec)
- `docs/glossary.md` — canonical English terminology
