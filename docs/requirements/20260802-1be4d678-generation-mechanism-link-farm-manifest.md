---
id: "REQ-1be4d678-959c-44d7-a346-44bfd95af56e"
type: requirement
name: "世代は link farm derivation を nput 自前 profile へコミットして積み、前世代 manifest から stale を除去する"
specification: |
  The manifest of what has been placed SHALL be embedded inside the store, as part of the
  link farm derivation produced by `lib.mkManifest` (what that derivation contains is
  stated by REQ-60e6b49c and is NOT restated here); no mutable JSON SHALL be kept outside
  the store, and `manifest.json` SHALL be immutable.
  At run time the engine SHALL, in this order: acquire a flock keyed on the resolved
  `profileDir`; read the store manifest of the previous generation and diff it against the
  new generation so as to determine which entries have disappeared; place the symlinks,
  out-of-store symlinks and place-once copies, doing the new placements and replacements
  first and the stale removal last; and only once everything has succeeded, commit by
  updating the nput nix profile through `nix-env --profile <profileDir>/profile --set
  <link-farm-drv>`, that being the commit point in every mode, so that a partial failure
  never reaches the commit and the previous generation is preserved. The previous
  generation SHALL be read from the previous generation of nput's own profile in every
  mode, standalone and module alike, and SHALL NOT depend on the host's `oldGenPath`.
  nput SHALL hold its own profile in every mode: in standalone it SHALL serve
  user-facing rollback, whereas in a module (home-manager / NixOS / nix-darwin) it SHALL
  be confined to an internal mechanism holding the previous-generation manifest and stale
  tracking. One profile SHALL correspond to one config, each config being atomic.
specification_ja: |
  「配置したもの」のマニフェストは、純粋関数 `lib.mkManifest` が生成する link farm derivation の
  一部として store 内に埋め込まなければならない（当該 derivation が何を含むかは REQ-60e6b49c が
  規定し、ここでは繰り返さない）。store 外の可変 JSON は持たず、`manifest.json` は不変とする。
  engine は実行時に、解決後 `profileDir` 単位の flock を
  取得し、前世代の store マニフェストと新世代を diff して消えた entry を判定し、symlink /
  out-of-store symlink / place-once copy を配置し（新規・張替を先に、stale 除去を最後に）、
  全て成功してから `nix-env --profile <profileDir>/profile --set <link-farm-drv>` で nput の nix
  profile を更新しなければならない。この `--set` が全モード共通のコミット点であり、途中失敗は
  そこへ到達せず前世代を保つ。前世代は standalone / module を問わず全モード共通で nput 自身の
  profile の前世代から読み、ホストの `oldGenPath` に依存してはならない。nput は全モードで自前
  profile を持ち、standalone ではユーザー向け rollback に使い、module（HM / NixOS / darwin）では
  前世代マニフェストと stale 追跡のための内部機構に留める。1 profile は 1 config に対応し、
  config 単位で atomic とする。
---
# REQ-1be4d678: 世代は link farm derivation を nput 自前 profile へコミットして積み、前世代 manifest から stale を除去する

## 仕様

- 純粋関数 `lib.mkManifest` が **link farm derivation**（`manifest.json` + 配置元への symlink
  ツリー）を生成する。「配置したもの」のマニフェストは `manifest.json` として link farm の
  一部に **store 内に**埋め込む（store 外の可変 JSON は持たない。`manifest.json` は不変）
- **engine**（Go ライブラリ）が実行時の副作用として:
  0. 解決後 `profileDir` 単位の flock を取得
  1. **前世代の store マニフェスト**（`manifest.json`）と新世代を diff し、消えた entry の
     **symlink を除去**（stale 除去）
     - 前世代は **全モード共通で nput 自身の profile の前世代**から読む（standalone も module も
       同一。ホストの oldGenPath には依存しない）
  2. symlink / out-of-store / place-once copy を配置（新規・張替を先に、stale 除去を最後に）
  3. 全て成功してから `nix-env --profile <profileDir>/profile --set <link-farm-drv>` で nput の
     nix profile を更新（コミット点・全モード）。途中失敗は 3 に到達せず前世代を保つ

**nput は全モードで自前 profile を持つ**。standalone では profile をユーザー向け rollback に
使い、module（HM/NixOS/darwin）では profile を**内部機構**（前世代マニフェスト + stale 追跡）に
留める。

| 機構 | 役割 | 適用層 | 位置 |
|---|---|---|---|
| 世代由来の store マニフェスト | stale 除去のための前回状態（不変・GC-root 済み）| 全層共通 | `manifest.json` として link farm derivation 内に埋め込み |
| nput の nix profile | 前世代の保持・世代番号・GC root | 全モード（standalone はユーザー向け / module は内部）| `profileDir` |

> **上は原文の写しで、規範は frontmatter が正**。原文が参照する次の規範は本 item の
> 担当ではない。
>
> - flock を blocking で取るか try-lock で取るか → REQ-1c1526b1
> - ネイティブ FS 操作で配置すること → REQ-35f485ff、各配置方法の手順 → REQ-622787dc /
>   REQ-d2277c7a / REQ-a8a923ad
> - stale 除去の保守的不変条件 → REQ-16aef46b
> - `profileDir` のオンディスクレイアウトと基底 `<state>` → REQ-2aa3abbc
> - project mode の世代スキップと lstat ドリフト修復 → REQ-46fccb80
> - module 時にユーザー向け rollback を host へ一本化すること → REQ-844ee375
> - `lib.mkManifest` が純粋関数であること・その戻り値 → REQ-2b0c2bb8 / REQ-60e6b49c
>
> 原文が併記する「配置・cleanup アルゴリズムは home-manager の `linkGeneration` /
> `cleanup` を参考に Go で再実装する（`home.file` 自体は再利用しない）」は実装方針の注記で
> 要求ではなく、「`nix` / `git` 以外はサブプロセスを使わない」は REQ-6c4e174a の担当。

逆に、上の写しには現れないが規範文が持つものが 1 つある。**1 profile = 1 config の atomic
性**で、原文はこれを本節ではなく「CLI 仕様」→「サブコマンド体系」の `apply` の箇条書き
（「profile は config 単位で atomic」）で述べる。当該箇所を分割した REQ-c2d44626 と、
これを論拠として引く REQ-c890ce4a がいずれも規範の所在を本 item に置いているため、profile の
機構を持つ本 item が引き受けて規範化した。

## 出典

`docs/spec.md`「世代管理仕様」→「機構」節の箇条書きと機構表、および同節の
「nput は全モードで自前 profile を持つ」段落。atomic 性のみ同「CLI 仕様」→
「サブコマンド体系」の `apply` の箇条書き（上の注記を参照）。

決定の実体は ADR-0002「世代管理を nix profile に乗せる（全モード自前 profile / rollback は
standalone 中心）」と ADR-0006「エンジンを固定の Go バイナリにし、lib はデータ生成に徹する」
（link farm derivation への manifest 埋め込み・コミット最後）。全モードのコミットを
`nix-env --profile <profileDir>/profile --set` に統一することは ADR-0025 が定めている。
