# ADR-0018: templates の具体内容と `--all` のサブコマンド対応範囲を確定する

- ステータス: 採用
- 日付: 2026-06-14
- 関連: ADR-0005, ADR-0006, ADR-0007, ADR-0013, ADR-0015, ADR-0016, ADR-0017, `docs/spec.md`, `docs/design.md`, `docs/concept.md`
- 起点: ADR-0017 に続く実装前レビューで「templates の具体内容」「`gitignore` / `rollback` / `list-generations` の `--all` 対応」が未定義だった

## 背景

`nput init <template>` は `nix flake init -t <nput>#<template>` の透明なラッパーで（ADR-0006, ADR-0007）、
`templates/standalone` / `templates/project` を展開する。ADR-0015 で project template に pin 版 `nput` 同梱 + shellHook +
`.gitignore` ガイドが入ると決まったが、**各 template の具体ファイル内容**と、**`--all` を `apply` 以外の
サブコマンド（`gitignore` / `rollback` / `list-generations`）にも広げるか**が未定義だった。

## 決定

### 1. template の richness = 最小 + 手厚いコメント

- 各 template は **動く example を 1 config だけ**置き、バリエーション（`subpath` / `method = "copy"` /
  `mkOutOfStoreSymlink` / 複数 entry / 動的生成）は **コメントで示す**。
- `templates/standalone` は `homeRoot` の 1 例、`templates/project` は `projectRoot` の 1 例 + devShell。
- starter を小さく保ち、ユーザーが不要分を削除する手間を最小化する。「nput はファイルを生成しない」thesis 下、
  ファイルを展開するのは nix の templates 機構であり、starter は学習用の最小形に留める。

### 2. `templates/project` の `.gitignore` ガイド = flake.nix コメント + `.gitignore` ヘッダコメント

- `flake.nix` の manifest 定義のそばに「配置物は ephemeral。`.gitignore` へ `nput gitignore <name>` の出力を追記」コメントを置く。
- template に `.gitignore` を同梱し、先頭に `# nput: regenerate with 'nput gitignore <name>'`（再生成手順）ヘッダコメントを置く。
- コマンドと手順がコード近接で明確になる。README 同梱はファイルが増えコードから離れて読まれない risk があるため採らない。

### 3. `templates/project` の shellHook 雛形 = 名指し apply + `--all` はコメント

- `devShell.shellHook = "nput apply <name> --no-wait"`（名指し）を既定にする。example が 1 config なので名指しが最も明確で、
  混在 entrypoint の footgun（ADR-0017）も起きない。
- 「複数 config なら `nput apply --all --project-root --no-wait`」をコメントで示す。
- devShell の `packages` には pin 版 `nput` を同梱する（`packages = [ nput.packages.${system}.nput ]`・ADR-0015）。
- `.envrc`（direnv）は同梱しない（direnv 非利用者に不要ファイルを増やすため。コメントで案内に留める）。

### 4. `--all` のサブコマンド対応 = `gitignore` / `list-generations` に拡張、`rollback` は名指し必須

- **`gitignore --all`**: 対応する。project mode の複数 config の target をまとめて **ソート + 重複除去**して stdout 出力する。
  repo の `.gitignore` は 1 つなので一括列挙が自然。`gitignore` は本来 project mode 向けなので `--all` は **projectRoot の
  config を対象**とする（root モードフィルタは実質冗長だが ADR-0017 のフィルタと併用可）。
- **`list-generations --all`**: 対応する。home mode の全 config の世代を一覧表示する。読み取り専用で無害、全体像の把握に便利。
  home mode 限定コマンドなので `--all` は **homeRoot の config を対象**とする。
- **`rollback --all`**: **対応しない**（名指し必須）。全 config を一斉に前世代へ戻すのは破壊的な一括操作で、通常は役割単位で
  戻したいため footgun。途中失敗で config 間の状態が不揃いになり得る。「ユーザーが配置を明示的に握る」思想からも名指しが安全。

## 根拠

- **最小 + コメント**は nput init の学習価値（書き方を握る）と starter の軽さを両立する。肥大した雛形は削除の手間を生む。
- **flake.nix コメント + .gitignore ヘッダ**はコマンド・手順をコード近接に置く。ephemeral 配置の `.gitignore` 登録という
  「一度きり・手動」の運用を最も発見しやすい場所に置ける。
- **名指し shellHook**は example 1 config に対し最も明示的で、ADR-0017 の混在 footgun を既定から排除する。
- **`gitignore` / `list-generations` の `--all`** は安全（前者は読み取り出力、後者は読み取り専用）で実用的。
  **`rollback --all`** だけは破壊的一括操作なので名指しに限定し、誤った全戻しを構造的に防ぐ。

## 影響

- **`docs/spec.md`**:
  - `nput init` 節に templates の具体内容（standalone / project のファイル構成・最小 + コメント方針）を追記。
  - サブコマンド体系に `gitignore --all` / `list-generations --all` を追加。`rollback` は名指し必須を明記。
  - `gitignore` 節に `--all`（projectRoot config をソート + 重複除去）を、`list-generations` に `--all`（homeRoot config）を追記。
- **`docs/design.md`**: プロジェクト構成の `templates/` 説明に最小 + コメント方針・project template の devShell / .gitignore 同梱を反映。
- **`docs/concept.md`**: 必要なら project mode の使用例コメントに `.gitignore` 再生成手順を反映（任意）。
- **実装フェーズ**: `templates/standalone/flake.nix`、`templates/project/{flake.nix,.gitignore}`、`cmd/nput`
  （`gitignore --all` の集約 + ソート + 重複除去、`list-generations --all`、`rollback` の `--all` 拒否）。

## 棄却した代替案

- **template に複数パターンを同梱**: すぐ使えるが starter が肥大し削除の手間が増える。バリエーションはコメントで足りる。
- **template をほぼ空のプレースホルダに**: 最小だが初見ユーザーが書き方を掴みにくく nput init の学習価値が薄い。
- **`.gitignore` ガイドを README に**: 説明量は取れるがファイルが増え、コードから離れて読まれない。
- **`.envrc` を同梱**: direnv 運用は即戦力だが非利用者に不要ファイルを強いる。コメント案内で足りる。
- **`rollback --all` を対応**: 一括戻しは破壊的で footgun。役割単位の名指しが nput の「明示的に握る」思想に合う。
- **3 コマンドすべてに `--all` を一律対応 / 一律非対応**: コマンドごとに安全性・実用性が異なるため一律は不適切。
