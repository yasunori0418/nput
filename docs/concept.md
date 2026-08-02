# nput コンセプト

nput が何であり何を解決するかの全体像と、solution / use_case item への索引。

解決したい課題・解決策の核心・想定する使われ方は **`docs/solution/` と `docs/use-cases/` の
item が持つ**。本文書は通読の入口として全体像を述べ、詳細は item へのリンクで示す
（README → 本文書 → item の 3 層構造）。

一方で「設計の哲学」「既存ツールとの比較」「north-star」「設計の変遷」は **item を立てず
本文書に残す**（→ Issue #211）。要求へ落ちない位置づけ・展望・経緯であり、item にすると
requirement を持たない use_case を作ることになるため。

> **この文書の書き方（規約）**
>
> - 散文は **見出し 1 つ（h2 / h3 の最も内側）につき 20 行以内**（item 化しない節を抱えるため
>   spec / design より緩い）。表・リンク列挙・コードブロックはこの制限に含めない
> - item 化された内容は **リンクで示し、本文に書き下さない**
> - **item を立てず散文で書き下してよいのは上記 4 節（哲学・比較・north-star・変遷）に限る**。
>   索引の節（課題と核心・想定する使われ方）は本文書に置くが、内容は item が持つ。この 4 節
>   以外で書き下したい内容が出てきたら、item を立てて本文書からはリンクへ置き換える
> - item を横断して検索・追跡するには `sara query`（→ `docs/agents/domain.md`）を使う
> - item の `## 出典` は本文書の現行章立てを指さない（理由は `docs/spec.md` の同項）

---

## 解決したい課題とコンセプトの核心

Nix で外部リポジトリを取得する手段はあるが、任意パスへ配置する既製手段は無い。home-manager は
配置できるが「全体管理」モデルであり、1 つの変更が全体に影響する。さらにモジュール抽象は
「何がどこにどう置かれるか」を内部実装に隠し、ユーザーから配置の制御を奪う。

nput はフレームワークではなく **配置プリミティブ**である。「nix store のパスを root 相対の
任意パスへ symlink または copy で置く」という単一の責務を、テスト可能な純粋関数として提供し、
設定は生成しない。

- [SOL-9fcd1d6e](solution/20260802-9fcd1d6e-nput-placement-primitive.md) — nput は nix store の物を任意パスへ置く配置プリミティブであり、設定を生成せずユーザーが配置を明示的に握る

---

## 想定する使われ方

中心は project mode（プロジェクト内配置）。`$HOME` 配置（home mode）・system 配置は明示マーカーで
opt-in する例外として位置づける（→ ADR-0007）。

- [UC-19a90989](use-cases/20260802-19a90989-project-mode-in-repo-placement.md) — プロジェクト repo 内へ nix store の物を devShell 入室のたびに配置してチームで共有する
- [UC-f2436d68](use-cases/20260802-f2436d68-home-mode-pinned-repo-placement.md) — home mode で外部リポジトリの中身をバージョン固定して $HOME 配下の任意パスへ配置する
- [UC-1c280dce](use-cases/20260802-1c280dce-independent-update-cycles.md) — 役割ごとに config を分けて更新を独立させ、1 つの更新を他の役割へ波及させない
- [UC-0b6f60cb](use-cases/20260802-0b6f60cb-generation-rollback-standalone.md) — 配置に失敗・後悔したとき standalone で前の世代へロールバックして元の状態へ戻す
- [UC-403fbe32](use-cases/20260802-403fbe32-copy-place-once-user-managed.md) — リポジトリの内容を copy で初回だけ配置し、その後はユーザーが手で編集して育てる
- [UC-01b896b4](use-cases/20260802-01b896b4-out-of-store-live-editing.md) — 開発中の手元 dotfiles を out-of-store symlink で参照し、編集と同時に反映しながら育てる
- [UC-d39c1994](use-cases/20260802-d39c1994-module-integration.md) — 既に home-manager / NixOS / nix-darwin を使っている環境へ nput をモジュールとして組み込む

---

## 設計の哲学

**取得と配置の分離**: 取得は Nix の評価フェーズ（`src` = ストアパス）、配置は実行フェーズ。
取得手段（npins / flake inputs / `fetchFromGitHub` 等）をツール側が抱えず「フェッチ済みの
ストアパスを受け取る」設計にすることで、取得方法の変化から独立する。

**配置ロジックはコアが所有し、モジュールは配線に徹する**: 配置の実体は全層で nput 自身の
固定 Go エンジンが実行する（→ ADR-0003, ADR-0006）。`home.file` / `systemd.tmpfiles` などの
ネイティブ機構には委譲しない。ネイティブ統合の恩恵は捨てるが、振る舞いが単一コアに集約され、
テスト可能性とクロスプラットフォームの一貫性を得る。

**home-manager に依存しない**: NixOS サーバー・最小構成の環境でも同じ設定定義で動くことを
優先する。統合は「オプション」であり、統合層はコアの薄いラッパーに過ぎない。

**冪等性と粒度の柔軟性**: 同じ設定を何度実行しても同じ結果になる。リポジトリ全体・
サブディレクトリ・単一ファイルを同一インターフェースで扱い、呼び出し側は型を意識しなくてよい。

---

## north-star: 配置プリミティブから組むミニマル distro

長期的な狙いは、nixpkgs のパッケージ群（＝ストアパス）を活かしつつ配置だけをユーザーに操作させ、
Arch / Gentoo 的なミニマル Linux ディストリビューションの基盤を作ること（→ ADR-0004）。NixOS は
同じことを巨大なモジュールシステムで行うが、その代償として nixpkgs の PR / リリースサイクルに
縛られる。このためコアの中心抽象は root を `$HOME` に固定せず一般化する。

nput は単独ツールに留まらず、n プレフィックスのツール群（nput / nboot / nwrap / nherd /
nshadow / ncompose）を stdout / stdin の JSON パイプで合成するエコシステムの一員でもある。
その前提は「規格が契約」に尽きるため、nput の機械可読出力（`--json`）は **今回の機能に閉じず、
今後追加するすべての機能で** niface specVersion 1 規約に準拠する（→ ADR-0043）。

**スコープの線引き**: 実装スコープは standalone CLI + project mode をコアとし home mode も
対象、system 配置は将来拡張（→ ADR-0007）。「関数ベースのパッケージ導入・PATH 追加」の具体機構は
未定義。ブート / init / FS / パーティションの層は本構想でも空白であり、nput が担う領域ではない。

---

## 既存ツールとの比較

比較軸は「機能の有無」ではなく **「モジュール抽象で隠すか、純粋関数でユーザーに握らせるか」**。

| ツール | 役割 | アプローチ | nput との違い |
|---|---|---|---|
| npins / niv | ソースのバージョン固定 | — | 配置は行わない（nput と直交・併用可）|
| home-manager `home.file` | ファイル配置 + 世代 | モジュール（生成・宣言）| HM 必須。全体管理。file モジュールを standalone 切り出し不能 |
| `mkOutOfStoreSymlink`（HM）| out-of-store symlink | モジュール内ヘルパ | HM 文脈限定。nput は同等を非依存の明示関数で提供 |
| nixpkgs `linkFarm` / `symlinkJoin` | store 内 symlink ツリー生成 | 純粋関数 | 出力が**ストア内に閉じる**。store 外の任意パスへは置かない（nput は内部で利用可）|
| `nix profile` | 世代管理機構 | — | 配置先が `~/.nix-profile` 固定。任意パス配置はしない（nput が乗る対象）|
| `systemd.tmpfiles`（`L`）| 任意パスへの宣言的 symlink | モジュール（NixOS）| 低レベル・NixOS 専用・copy/世代/取得抽象なし |
| numtide/system-manager | 非NixOS の `/etc` + systemd + パッケージ | モジュール（`lib.evalModules`）| **ドメインは重なるがアプローチが逆**。任意パス配置・HOME dotfiles・サブディレクトリ取り出しはしない |
| `git clone`（シェル）| クローンと配置 | 命令的 | 再現性・Nix との統合がない |
| **本ツール** | 取得済みソースの独立配置 + 世代 + 明示 out-of-store | **純粋関数・ユーザー管理** | — |

nput とほぼ同一のツールは存在しない。構成要素（symlink farm / nix profile / out-of-store /
任意パス symlink）はすべて既存だが、それらを「取得手段非依存 + 生成しない + エントリ個別適用 +
HM 非依存の純粋関数コア + クロスプラットフォーム共通スキーマ + 任意パス配置 × 世代管理」として
束ねたものは無い。特に system-manager とはドメインこそ重なるが、思想レベルでアプローチが
異なるため競合しない。

### home-manager `home.file` との配置意味論の差

`home.file` と nput はどちらも「symlink を配置し前世代との diff で stale を除去する」同型の
モデルを持つ。しかし nput は配置のたびに**自己記録の manifest**（前世代 `manifest.json`）を
持つのに対し、home-manager の cleanup 判定は on-disk の readlink パターンマッチに依存する。
この一次情報の有無が意味論の差として表れる（HM 現行実装〔2026-07 時点〕との比較・
→ ADR-0046, ADR-0047）。

1. **同名 leaf を含む per-file → dir symlink 遷移の自動移行** — HM は旧 leaf の残存を誤認して
   部分適用で失敗しうるが、nput は manifest 記録との一致判定（recorded ∧ stale）で安全に移行する
2. **所有判定の厳密さ** — HM は readlink の glob パターンマッチ、nput は「記録した配置先 +
   on-disk の readlink が記録 dest と完全一致」で判定する
3. **配置を塞ぐ空 dir の由来を問わない自動除去** — HM は collision で停止するが、nput は
   rmdir が空 dir にしか成功しない＝損失ゼロを利用し、由来を問わず配置前除去の対象に含める
4. **祖先 symlink の安全性** — HM は祖先 component が symlink でも無検査で辿るが、nput は
   foreign な祖先 symlink を conflict で停止し、自己記録の stale のみ配置前除去で移行する
5. **rename 可用性 + fail-fast drift** — nput は「配置を塞ぐ依存除去のみ」を前段化し、
   前段化した除去が drift を検出したら skip せず error で停止する

空親ディレクトリ剪定・conflict 全件報告は nput も同等の挙動を持つ（パリティ項目・HM 超えでは
ない）。method 変更を跨ぐ自動移行や copy を含めた `reset` は、copy という概念自体が
home-manager に存在しないため比較の対象外。

---

## 設計の変遷（会話の流れ）

| フェーズ | 検討内容 | 採用した方向 |
|---|---|---|
| 起点 | `fetchFromGitHub` + `lock.json` + シェルスクリプト | ロック管理をシェルで実装 |
| ロック管理 | シェルスクリプト vs Nix 関数 | 副作用が必要なため `npins` 等を使う |
| 配置手段 | home-manager 依存可否 | コアを純粋関数として切り出し、HM 非依存と HM 統合を両立 |
| src 設計 | npins を内包するか | `src` をストアパスとして受け取り取得手段を問わない設計に |
| 役割分離 | 全体管理 vs 役割ごとの独立管理 | エントリに `name` を持たせ、個別更新・個別適用できる設計に |
| out-of-store（ADR-0001）| 型ベース暗黙分岐 vs 明示関数 | store link をデフォルトに統一し、out-of-store は明示関数へ降格 |
| 世代管理（ADR-0002）| 世代を取らない vs 取る | nix profile に乗せた standalone 世代管理を追加。copy は世代外 |
| 層モデル（ADR-0003）| ネイティブ翻訳 vs エンジン所有 | 配置ロジックは全層 nput エンジンが所有、モジュールは配線に徹する |
| 抽象（ADR-0004）| `$HOME` 固定 vs root 一般化 | root を一般化し配置プリミティブに。distro は純粋関数の合成で組む |
| project mode（ADR-0005）| root=`$HOME` 固定 vs プロジェクト相対 | root を公開引数へ昇格し git toplevel 相対の project mode を追加 |
| エンジン実装（ADR-0006）| 生成 bash vs 固定バイナリ | 配置ロジックを固定 Go エンジンに集約。契約は manifest.json |
| 露出 / root（ADR-0007）| per-config ラッパー vs 汎用 CLI | 汎用 `nput` CLI を一次 UX に昇格。root は明示必須。project-first へ |
| src/subpath 分離（ADR-0008）| `source` を `src` と誤読される問題 | `source` を `subpath` に改名。全体選択は省略で表現 |
| entries 識別子（ADR-0014）| `name` フィールド vs attrset キー | `entries` を target キーの attrset に変更し一意性を native に担保 |
| copy と reset（ADR-0019〜0021）| symlink 以外の配置・撤去手段 | `method = "copy"`（世代外・place-once）・`--recopy`・`reset` を追加 |
| module activation（ADR-0026）| モジュールも entrypoint 発見経由か | ビルド済み link-farm を `apply --manifest` で直接適用する経路を新設 |
| flake-parts 統合（ADR-0029）| flake-parts 向けの output 形 | `perSystem.nput` を `flake.nput.<system>` へ transpose する module を追加 |
| 出力規律（ADR-0031）| 成功時に配置レポートを出すか | 成功時はデフォルト沈黙。`-v` で opt-in 表示（`--quiet` は廃止）|

上記に挙げていない ADR は実装確定に伴う詳細な意味論整備（CI・型検査・flock・root 解決の細部等）。
個々の内容は `docs/adr/` を参照。

---

## 関連文書

- `README.md` / `README.ja.md` — 3 層構造の最上段（導入と使い方）
- `docs/spec.md` — 仕様（requirement item への索引）
- `docs/design.md` — 設計（design item への索引）
- `docs/adr/` — 意思決定の記録
- `docs/model.yaml` — sara の型定義（item の型・関係・ID 形式）
