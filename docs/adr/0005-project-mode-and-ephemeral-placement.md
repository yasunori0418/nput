# ADR-0005: project mode（プロジェクト相対配置）と ephemeral 配置原則

- ステータス: 採用（2026-06-13 改訂: 「既定 `$HOME`（home mode）」前提を撤回し root 明示必須へ → ADR-0007）
- 日付: 2026-06-11
- 関連: ADR-0002, ADR-0003, ADR-0004, ADR-0007, `docs/concept.md`, `docs/design.md`, `docs/spec.md`

> **2026-06-13 改訂（ADR-0007）**: 本 ADR は「これまで root = `$HOME` 固定（home mode 既定）で、project mode を `projectRoot` で opt-in する」前提で書かれている。ADR-0007 で暗黙デフォルトを撤廃し、project mode は `projectRoot` / `homeRoot` / `systemRoot` の 3 マーカーの一つになった（home が既定でも project が既定でもない）。project mode の機構（git toplevel 解決・root キー profile・ephemeral・世代スキップ・非公開世代）自体は不変。
>
> **2026-06-14 改訂注記（ADR-0017）**: 本 ADR の **世代スキップを「完全 no-op」→「lstat 検査 + 必要時のみ再張り」**へ精緻化した。新 link-farm が前世代と同一なら新世代は積まない（世代無限増殖の回避は不変）が、各 target を lstat 検査し foreign に書き換えられた / 消えた entry はその entry だけ再張りしてドリフトを収束させる（→ ADR-0017）。
>
> **2026-06-14 改訂注記（ADR-0019）**: project mode の **copy target も ephemeral 扱い**と確定した。`gitignore` は method を区別せず全 target を列挙し、
> copy target も各 clone で place-once 再マテリアライズされ編集は clone local / 使い捨て（`git clean` で消える）。committed（vendoring）は nput の責務外（→ ADR-0019）。

## 背景

これまで nput の公開 API は root = `$HOME` 固定で、配置先は `$HOME` 相対の target だけを想定していた（ADR-0004）。
ADR-0004 が seam として残した root 一般化は、将来の system 配置（root = `/`、distro 構想）だけを念頭に置いていた。

これとは別に、**任意プロジェクト内に nput を組み込み、repo 内の任意パスへ nix store の物を配置したい**用途が出た。
具体例は「repo 内の `.claude/skills/` をチームで共有する」「project-local な tool 設定・hook を nix store から置く」など。
配置先はクローンした作業ツリーの中であり、`$HOME` でも `/` でもない**第三の root = プロジェクトルート**になる。
これは ADR-0004 が想定していなかった root 種別である。

## 決定

### project mode を導入する

- **root = プロジェクトルート**の配置モードを導入する。`$HOME` 相対（home mode）とは別系統の root。
- root の特定は **実行時に解決**する。クローン場所はクローンごとに違い、Nix 評価時に絶対パスを焼き込めないため。
  - **既定: `git rev-parse --show-toplevel`**（git toplevel）。どのサブディレクトリから叩いても同じ root に解決され、
    nput が「git リポジトリを扱う」前提とも一致する。
  - **`--root <path>` で上書き可能**。git 外で使う場合や別ルートを指したい場合に明示する。

### profile は解決済み root でキーする

- project mode の profile/世代は **profile key に解決済み root を含める**（例: `~/.local/state/nix/profiles/nput/<roothash>/<name>`）。
- これにより同一 flake を複数箇所にクローンしても profile が衝突せず、stale 除去が互いの配置を掃除し合う事故を防ぐ。
- profile・store マニフェストの不変条件（ADR-0002）はそのまま再利用する。repo 内に可変 state は持たない。

### project mode の世代はユーザー露出しない

- project mode の profile は **stale 除去 + 世代スキップ判定の内部機構**に留める。
- `--rollback` / `--list-generations` は project mode では公開しない。配置物が ephemeral（後述）で rollback の意味が薄く、
  devShell キック時には戻し先となる host 世代も存在しないため。
- standalone の home mode は従来通り `--rollback` / `--list-generations` を公開する（ADR-0002 のまま）。

### 世代スキップ短絡を project mode の必須要件にする

- project mode では **新しい link farm derivation が前世代と同一なら、新世代を積まずに no-op で終える**。
- devShell / direnv 運用では `shellHook` がシェル再入のたびに走るため、毎回新世代を積むと profile 世代が無限増殖する。
- standalone の home mode は従来通り「適用のたびに新世代」（ADR-0002）のままとし、世代スキップは project mode 限定とする。

### devShell shellHook を正式な配線レイヤーに加える

- ADR-0003 の「モジュール（HM / NixOS / nix-darwin）= エンジンを起動する配線」に、**devShell の `shellHook`** を同型の配線として追加する。
- `nix develop` / direnv（`use flake`）でシェルに入った瞬間に nput エンジンをキックし、git toplevel を root に解決して配置する。
- devShell は配置ロジックを持たず、root（git toplevel）と activation タイミング（シェル入室）を供給するだけ。HM モジュールと同じ位置づけ。

### ephemeral 配置原則 — 配置物はコミット対象外

- **project mode で nput が配置する物は、プロジェクトにコミットされるべきでない**（per-clone でクローンごとに再生成する ephemeral な物）。
- したがって **activation（`nput`）は git 状態に一切干渉しない**。`.gitignore` に触れず、target が git tracked かのチェックもしない。
- 正しい運用では配置物は常に untracked なので、保守的 stale 除去（ADR-0002 / ADR-0003）が git-tracked file と衝突する事態は原理的に起きない。

### `.gitignore` 生成は専用コマンドで stdout 出力のみ

- 配置 target を `.gitignore` 向けに列挙する**専用コマンド**（例: `nput gitignore`）を設ける。出力は **stdout のみ**でファイルは書き込まない。
- nput は「symlink / copy を target に置く」以外のファイル改変をしないツールであり続ける（`.gitignore` 自動追記はしない）。
- `.gitignore` の更新は一度きりで足り、定義変更が起きた場合の更新責務はプロジェクト管理者が持つ。

### copy / out-of-store も同原則を継承

- project mode の copy（place-once）・out-of-store symlink も ephemeral・untracked 前提を継承し、新しいセマンティクスは持ち込まない。

## 根拠

- **config ファイル相対 root を採らない理由**: Nix の評価モデルでは `nix run` 時に flake のソースが nix store にコピーされる。
  実行時に「設定ファイルの場所」を取ると `/nix/store/...` になり、ユーザーの作業ツリーを指さない。
  `--impure` + `$PWD` を要求するか破綻するため、config 相対は成立しない。
- **CWD（`$PWD`）既定を採らない理由**: 実行場所で配置先が変わり、サブディレクトリから叩くと配置がズレ、stale 除去が別ディレクトリを掃除しうる。
  「同じ config → 同じ結果」という冪等性原則（concept.md）と矛盾する。git toplevel なら実行場所に依らず安定する。
- **profile を root でキーする理由**: root でキーしないと同一 flake の複数クローンが 1 profile を奪い合い、
  stale 除去が相互のクローンの配置を破壊する。home mode（1 ユーザー 1 つ）では起きなかった問題。
- **devShell を配線に加える理由**: devShell shellHook は activation タイミングを供給するだけで配置ロジックを持たない。
  ADR-0003 の「モジュール = 配線」と同型であり、project mode の主用途（repo に入ると自動配置）に最も自然なトリガ。
- **ephemeral 原則の効用**: 配置物が常に untracked であることを設計原則として固定すると、stale 除去と git-tracked file の衝突が原理的に消え、
  activation 側に git チェックを入れる必要がなくなる（activation は純粋に「置くだけ」に保てる）。

## 影響

- ADR-0004 を改訂する（root を公開引数へ昇格し `projectRoot` マーカーを導入。本 ADR と対）。
- `docs/spec.md` / `docs/design.md` に以下を反映する。
  - `mkActivationScript` の `root` 引数（既定 `$HOME` / `projectRoot` マーカー / 絶対パス）。
  - `projectRoot` の実行時解決（git toplevel 既定）と `--root` CLI。
  - `nput gitignore`（stdout 出力のみ）コマンド。
  - project mode の世代スキップ短絡と、世代の非公開（`--rollback` / `--list-generations` を出さない）。
- `docs/concept.md` に project-scoped placement 節を追加する。
- `CONTEXT.md` に用語（`project mode` / `projectRoot` / `ephemeral placement`）を追加し、`root` の定義を更新する。
- **orphan profile（フォローアップ）**: profile を root パスでキーするため、クローンを削除すると `~/.local/state` に
  profile が孤児として残る。store は `nix-collect-garbage` で解放されるが profile ディレクトリは残る。
  これは放置許容（または手動削除）とし、`README.md` / 公開用ドキュメントに注記する方針とする（本リポジトリでは未作成のため後続作業）。

## 棄却した代替案

- **config ファイル相対 root**: Nix で flake source が store にコピーされるため store path 化し、作業ツリーを指せない。成立しない。
- **CWD（`$PWD`）既定 root**: 実行場所で配置先が変わり冪等性を破壊。stale 除去が別ディレクトリを誤って掃除しうる。
- **repo 内に可変 state（`.nput/` 等）を置く**: ADR-0002（store マニフェストは不変・GC-root）と衝突。配置 symlink の誤コミット懸念も生む。
  profile を root でキーして `~/.local/state` に置く方式を採用。
- **project mode で世代を取らない**: stale 除去（entry が消えたときの掃除）を同時に失い、entry の増減を扱えなくなる。
- **`.gitignore` への自動追記 / `--write`**: nput が常にファイルを生成・改変するツールになり、「設定を生成しない」核心思想に反する。stdout 出力のみに留める。
- **project mode で `--rollback` を公開**: per-clone な ephemeral 配置で rollback の意味が曖昧。devShell キック時は戻し先 host 世代も無い。内部機構に留める。
