---
id: "ADR-0035"
type: adr
name: "HM モジュールに `nput.configs.<name>` を導入し複数 profile（役割分離）を可能にする"
status: 採用
origin: "次期マイルストーン計画の grilling（2026-07-04）。ADR-0025 §2 が「消費側の要求が出た時点で追加」とした HM 複数 profile 化を実施する"
justifies:
  - "REQ-c6891aeb-13c0-4ae7-9ad1-5c343735266a"
  - "REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10"
  - "REQ-5c6b07da-3d06-414d-8770-4f438234b322"
  - "REQ-fc1118b1-b0e8-4ddf-80f6-c70956651693"
  - "REQ-c847d1af-a437-46bb-bd64-42083810d034"
revises:
  - "ADR-0024"
  - "ADR-0025"
references:
  - "ADR-0007"
  - "ADR-0010"
  - "ADR-0014"
  - "ADR-0026"
---
# ADR-0035: HM モジュールに `nput.configs.<name>` を導入し複数 profile（役割分離）を可能にする

- ステータス: 採用
- 日付: 2026-07-04
- 関連: ADR-0007, ADR-0014, ADR-0026, ADR-0010, `docs/spec.md`, `modules/common.nix`, `modules/home-manager.nix`
- 改訂対象: ADR-0024 §2 の「HM モジュール経由は MVP で固定名 1 profile（`default`）、複数化は将来」と ADR-0025 §2 の「単一 `nput.entries` 据え置き・role 分離不可・将来 `nput.configs.<name>.entries` seam」を、本 ADR で seam の実装決定へ進める。「1 config = 1 profile = 1 manifest」の atomic 性（ADR-0002）は不変
- 起点: 次期マイルストーン計画の grilling（2026-07-04）。ADR-0025 §2 が「消費側の要求が出た時点で追加」とした HM 複数 profile 化を実施する

## 背景

ADR-0024 §2 / ADR-0025 §2 は HM モジュールを単一 `nput.entries` = 1 profile（固定名 `default`）に限定し、役割分離（役割ごとに独立した profile・個別 rollback）は standalone CLI 経路のみとした。将来 seam として `nput.configs.<name>.entries`（attrsOf）形が明記されている。

standalone で複数 profile を使い分けるユーザーが HM に移行する（または併用する）際、この表現力の非対称が移行障壁になる。seam の設計は既に確定しており、options 設計と activation の複数 profile kick を実装するだけの状態にある。

## 決定

### 1. `nput.configs.<name>.entries` を新設する

- `modules/common.nix` に `nput.configs = attrsOf (submodule { entries = entriesType; })` を追加する。属性キー `<name>` が profile 名（= standalone の entrypoint `nput.<name>` と同じ次元）。
- `<name>` ごとに `mkManifest { root = homeRoot; entries = ...; }` を呼び、**profile ごとに独立した link-farm** を生成する。1 config = 1 profile = 1 manifest の atomic 性（ADR-0002・ADR-0032 §3 の合成拒否）は保つ。合成ヘルパは提供しない。
- profile dir は home mode の `<name>` 直キー（`<state>/nix/profiles/nput/<name>`・ADR-0024 §2）にそのまま乗る。`default` 以外の `<name>` が増えるだけで、レイアウトの変更は無い。

### 2. `nput.entries` は `configs.default.entries` への糖衣として残し、deprecated にする

- 既存の `nput.entries` は `lib.mkRenamedOptionModule [ "nput" "entries" ] [ "nput" "configs" "default" "entries" ]` で **`configs.default.entries` への rename 糖衣**にする。既存ユーザーの config は無変更で動き続ける（非破壊）。
- rename 警告（home-manager 標準の deprecation warning）がそのまま **deprecated 告知**になる。docs は `configs.<name>` を canonical として記載し、`nput.entries` は「deprecated・後方互換のための糖衣」と注記する。削除時期は定めない（利用状況を見て別 ADR で判断）。

### 3. activation は profile ごとに独立した engine 起動

- `home.activation.nput` は `configs` を走査し、**profile ごとに 1 回ずつ** `nput apply --manifest <link-farm-N> <name>` を実行する（`apply --manifest` と位置引数 `name` の直交・両立は ADR-0026 で既定）。
- 各起動は profileDir 単位の flock・前世代 diff・保守的 stale 除去・`nix-env --set` が **profile ごとに独立して**走る。複数 manifest を 1 回の engine 起動に渡す CLI 拡張は行わない（atomic 性の単位 = profile を CLI 界面にも保つ）。
- 実行順は `<name>` の辞書順で決定的にする（Nix の attrset 走査順と一致・ログの再現性のため）。1 profile の失敗は後続 profile を止めず、最後に集約して activation を失敗させる（`apply --all` の部分失敗と同じ姿勢・ADR-0018）。
- rollback のユーザー面は従来通り **host（`home-manager --rollback`）に一本化**（ADR-0002・ADR-0024）。nput 内部世代が profile ごとに積まれる点も従来と同じで、`nput rollback` は HM 経路では引き続き公開しない。

### 4. configs 間の target 衝突は eval 時に検出して停止する

- 同一 HM config 内の `configs.<A>` と `configs.<B>` が**正規化後 target**（属性キー既定値・明示 `target` 上書きを解決した後の文字列）を重複させた場合、**モジュール eval 時に assertion で停止**する。
- 一般の cross-config 衝突（別 entrypoint・別マシン・別ツール）が「engine 実行時の後勝ち + foreign symlink warning」（ADR-0015）であることは不変。HM の `configs` は**全 config が単一のモジュール eval に載る**ため例外的に静的検出が可能であり、「eval で分かる衝突は eval で止める」（ADR-0024 §5 の同一 manifest 内衝突検出）の自然な延長として扱う。
- 検出しなければ、activation のたびに A と B が同じ target を交互に奪い合う（毎回 foreign symlink warning + 上書き）フリップフロップが恒常化する。役割分離の導入自体が衝突の温床になるため、eval 停止が正しい。

## 根拠

- **`configs` 命名**: ADR-0025 §2 の seam 記述・spec の用語（「config = named manifest」）と一致し、standalone の `nput.<name>` と同じ「config 名」次元であることが名前から読める。`profiles` は世代管理の内部語（profile dir）と紛れる。
- **糖衣を残す理由**: リポジトリは公開済みで利用者ゼロと断定できず、単一 profile ユーザーには `nput.entries` が今後も最短の書き味。`mkRenamedOptionModule` 一行の互換コストで破壊的変更を避けられる。deprecated 警告により新規ユーザーは canonical へ誘導される。
- **CLI 拡張（複数 manifest 一括渡し）をしない理由**: `apply --manifest` の契約（1 link-farm = 1 profile 世代）を保てば、複数化は呼び出し回数の問題であって界面の問題ではない。ADR-0032 §3 が合成を拒否したのと同じく、世代の単位を曖昧にしない。

## 影響

- **`modules/common.nix`**: `configs` オプション追加・`entries` の rename 糖衣化・configs 間 target 衝突 assertion。
- **`modules/home-manager.nix`**: `mkManifest` 呼び出しと activation スクリプトの configs ループ化・部分失敗集約。
- **`docs/spec.md`**: モジュールオプション仕様を `configs.<name>` canonical へ更新、「HM = 単一 entries・role 分離不可」の制約記述を撤去し、糖衣・deprecated 注記と eval 衝突検出を追記。
- **`docs/design.md`**: モジュール統合表の HM 行（`default` 固定・役割分離不可注記）を更新。
- **`CONTEXT.md` / `docs/glossary.md`**: `module` 定義の「MVP では単一 `nput.entries`・役割分離は不可」を更新。
- **ADR-0024 / ADR-0025**: §2 への改訂注記を同一 PR で追記。
- **テスト**: hm-module テスト（`tests/`）に複数 configs の activation・rename 糖衣の等価性・衝突 assertion の eval エラーを追加。

## 棄却した代替案

- **`nput.entries` を廃止して `configs.<name>` に一本化（破壊的変更）**: 公開済み API の破壊。糖衣 1 行で避けられるコストではない。deprecated 警告で同じ誘導効果を得る。
- **`nput.profiles.<name>` 命名**: spec / ADR-0025 が既に `configs` の語で seam を記述しており、変更する積極的理由が無い。世代管理の内部語 `profile` との混同も招く。
- **configs 間 target 衝突を実行時後勝ちのまま放置**: 同一 eval 内で静的に分かる衝突を実行時のフリップフロップに落とすのは、ADR-0010「未定義挙動を早期に弾く」・ADR-0024 §5 と一貫しない。
- **activation を 1 回の engine 起動に集約（複数 manifest 渡しの CLI 拡張）**: 世代コミットの単位が曖昧になり、部分失敗時の状態が説明困難になる。profile ごと独立起動が atomic 性を素直に保つ。
