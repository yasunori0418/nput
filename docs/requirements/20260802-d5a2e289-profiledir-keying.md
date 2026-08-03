---
id: "REQ-d5a2e289-40bc-45a9-9d44-21b8dc561b81"
type: requirement
name: "profileDir は home のみ name 直キーとし、fixed root と --root 上書きは roothash でキーする"
derives_from:
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
  - "UC-403fbe32-b146-401b-8b53-fe67c1e169c5"
specification: |
  The key of `profileDir` SHALL be determined by the root. Keying directly on `<name>`
  SHALL be confined to home mode without `--root`, where "one profile per user" holds.
  Where `--root` is given, `profileDir` SHALL be keyed on the `<roothash>` of the
  overriding root in every mode alike, home and fixed included. Fixed root mode — an
  absolute path string as `root`, without `--root` — SHALL likewise always be keyed on
  `<roothash>`, a fixed root being an arbitrary absolute path settled at evaluation time
  and therefore separated into an independent series per root, as project mode and a
  `--root` override are, which structurally prevents the silent orphan of a config of the
  same name under a different root sharing a series of generations. The keying SHALL be
  consistent across `apply` / `reset` / `rollback` / `list-generations`, so that operating
  on a generation made with `--root` requires the same `--root` again. The computation of
  `<roothash>` and the backref `.root` SHALL be the same mechanism as project mode.
specification_ja: |
  `profileDir` のキーは root によって定めなければならない。`<name>` 直キーは「1 ユーザー
  1 profile」が成立する home（`--root` なし）に限る。`--root` 明示時は home / fixed を含む
  全モードで、上書き後 root の `<roothash>` でキーしなければならない。fixed root mode
  （`root` に絶対パス文字列・`--root` なし）も常に `<roothash>` でキーしなければならない。fixed root は
  評価時確定の任意絶対パスであり、project / `--root` 上書きと同じく root ごとに独立系列へ
  分離することで、別 root の同名 config が世代系列を共有する silent orphan を構造的に
  防ぐためである。このキーイングは `apply` / `reset` / `rollback` / `list-generations` で
  一貫させなければならず、`--root` を付けた世代を操作するには再び同じ `--root` を要する。
  `<roothash>` の算出と backref（`.root`）は project mode と同一機構でなければならない。
---
# REQ-d5a2e289: profileDir は home のみ name 直キーとし、fixed root と --root 上書きは roothash でキーする

## 仕様

| 状況 | profileDir |
|---|---|
| home（`--root` なし）| `<state>/nix/profiles/nput/<name>` |
| home / fixed（`--root /p`）| `<state>/nix/profiles/nput/<roothash(/p)>/<name>` |
| fixed（`--root` なし・`root = "/abs"`）| `<state>/nix/profiles/nput/<roothash(/abs)>/<name>` |
| project（`--root` 有無）| `<state>/nix/profiles/nput/<roothash>/<name>` |

- **`--root` 明示時は全モードで profileDir を上書き後 root の `<roothash>` でキーする**。
  home / fixed mode も `<state>/nix/profiles/nput/<roothash>/<name>` になり、異なる
  オーバーライド root が独立した世代系列に分離されて silent orphan を防ぐ。`apply` /
  `reset` / `rollback` / `list-generations` で一貫し、`--root` を付けた世代を操作するには
  再び同じ `--root` が要る。`--root` なしの通常 home は従来どおり `<name>` キー
  （1 ユーザー 1 profile）
- **fixed root mode（`root` に絶対パス文字列・`--root` なし）も常に `<roothash>/<name>` で
  キーする**。fixed root は評価時確定の任意絶対パスなので project / `--root` 上書きと同じく
  root ごとに独立系列へ分離し、別 root の同名 config が世代系列を共有する silent orphan を
  構造的に防ぐ。`<name>` 直キーは「1 ユーザー 1 profile」が成立する home（`--root` なし）に限る
- `<roothash>` 算出・backref（`.root`）は project mode と同一機構

> **上は原文の写しで、規範は frontmatter が正**。原文の表が持つ「HM モジュール経由（MVP）は
> `<state>/nix/profiles/nput/default`（固定名 1 profile）」の行は、**ADR-0035 が HM の
> `nput.configs.<name>` 実装を決定済み**で反転している（HM も `<name>` 次元を持ち、home mode の
> `<name>` 直キーにそのまま乗る。`default` は糖衣が指す config 名に過ぎない）。`docs/spec.md`
> 側がこの改訂に追従できていないため、分割にあたって MVP 限定の行を規範へ持ち込まない判断を
> した（REQ-c6891aeb と同じ扱い）。`docs/spec.md` の追従は本 item の担当範囲外。
>
> 原文が参照する次の規範は本 item の担当ではない。
>
> - profileDir の中に何をどう並べるか（profile リンク・世代・`.pending`・backref `.root` の
>   置き場所）と `<state>` の定義 → REQ-2aa3abbc
> - `<roothash>`（解決後の絶対 root パスの sha256 短縮 hex）の定義と、project mode で
>   解決済み root をキーにする理由 → REQ-46fccb80
> - `--root` が全モードの解決 root を一律上書きすること自体 → REQ-61c05e09。ただし
>   **`--root` 明示時のキーイングは本 item の担当**で、REQ-61c05e09 の規範からは落として
>   ある（profileDir のキーは root の種別ごとに定まる一つの体系であり、`--root` の行だけを
>   切り出すと ADR-0023 §3 の改訂時に片方だけ直る事故が起きるため一本化した）
> - 孤児 profile の放置許容と backref による逆引き → REQ-d41b1d0a

## 出典

`docs/spec.md`「root の解決」→「project mode（`root = projectRoot`）」節の箇条書き
第 4〜5 項（`--root` 明示時のキーイング・fixed root mode のキーイング）と、
第 5 項に付随する profileDir 表。

決定の実体は ADR-0023「実装前残セマンティクス第5巡」§3（`--root` 明示時の profileDir
キーイングを home / fixed へ拡張）と ADR-0024「実装前残セマンティクス第6巡」§1
（fixed root mode も常に `<roothash>` でキーする）。
