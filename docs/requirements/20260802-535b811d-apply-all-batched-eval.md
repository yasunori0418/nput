---
id: "REQ-535b811d-dfc5-4eac-92db-737e70eb5415"
type: requirement
derives_from:
  - "UC-1c280dce-7c72-44c0-95ea-d06344f62a47"
name: "apply --all は rootKind を 1 回の一括 eval で取る"
specification: |
  `apply --all` SHALL obtain the rootKinds in a single batched eval: a map from config
  name to rootKind SHALL be obtained at once with
  `nix eval <ep>#nput.<system> --apply 'cs: builtins.mapAttrs (_: c: c.rootKind) cs' --json`
  (for a legacy entrypoint, which has no per-system dimension,
  `nix eval -f <ep> nput --apply 'cs: …' --json`), and each `profileDir` SHALL be
  determined from it. Filters such as `--project-root` SHALL also be dispatched from this
  result. Only the build SHALL be performed N times, once per config, for the sake of
  atomicity. This SHALL fix the eval process startup cost at 1 rather than N.
specification_ja: |
  `apply --all` は rootKind を 1 回の一括 eval で取らなければならない。
  `nix eval <ep>#nput.<system> --apply 'cs: builtins.mapAttrs (_: c: c.rootKind) cs' --json`
  （legacy は per-system 次元なし: `nix eval -f <ep> nput --apply 'cs: …' --json`）で
  config 名 → rootKind マップを 1 回で取得し、各 profileDir を確定する。`--project-root` 等の
  フィルタもこの結果で振り分ける。build だけは atomic 性のため config ごと N 回行う。
  eval プロセス起動コストを N→1 に固定する。
---
# REQ-535b811d: apply --all は rootKind を 1 回の一括 eval で取る

## 仕様

**`apply --all` は rootKind を 1 回の一括 eval で取る**。
`nix eval <ep>#nput.<system> --apply 'cs: builtins.mapAttrs (_: c: c.rootKind) cs' --json`
（legacy は per-system 次元なし: `nix eval -f <ep> nput --apply 'cs: …' --json`）で
config 名 → rootKind マップを 1 回で取得し、各 profileDir を確定する。`--project-root` 等の
フィルタもこの結果で振り分ける。build だけは atomic 性のため config ごと N 回。
eval プロセス起動コストを N→1 に固定する。

フィルタそのものは REQ-d95b814f の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「実行フロー」の `apply --all` 一括 eval の箇条書き。

決定の実体は ADR-0024「`--all` 一括 eval」で、legacy entrypoint の eval 形は ADR-0032。
