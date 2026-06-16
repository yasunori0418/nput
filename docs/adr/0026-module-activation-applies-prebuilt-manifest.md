# ADR-0026: モジュール activation の engine kick は `apply --manifest`（ビルド済み link-farm 直接適用）で行う

- ステータス: 採用
- 日付: 2026-06-17
- 関連: ADR-0002, ADR-0003, ADR-0006, ADR-0007, ADR-0011, ADR-0015, ADR-0023, ADR-0024, ADR-0025, `docs/spec.md`, `docs/design.md`, `CONTEXT.md`
- 改訂対象: ADR-0003 / ADR-0007 が「モジュールは switch 時に engine を起動する配線」とした**起動方法**を確定する（決定の反転なし・未定義だった invocation を埋める）。CONTEXT.md の `nput CLI` / `apply` / `module` 定義を拡張する。
- 起点: Issue #17（home-manager モジュール統合）の実装中に判明した計画外の仕様。docs（ADR-0003, ADR-0007, spec のモジュール別動作仕様）は「module は engine を kick する配線」とだけ述べ、**具体的な invocation を一度も規定していなかった**。

## 背景

CLI（ADR-0007）は **entrypoint 駆動**で設計されている。`nput apply <name>` は entrypoint（`flake.nix` 等）を発見し、`nix eval` で rootKind を先取りし、`nix build` で `nput.<system>.<name>` の link-farm を得て engine を駆動する（ADR-0023 実行フロー）。CONTEXT.md も `nput CLI` を「entrypoint を発見し内部で `nix build`/`eval` を回して named manifest を得て engine に配置させる」と定義する。

一方 HM モジュール経路では、manifest は**モジュール評価時に** `nput.entries` から `mkManifest` で生成され、link-farm derivation として**既にビルド済み**で activation script から store path で参照できる。ここで engine を起動するのに entrypoint 駆動の `apply` を使うと:

- entrypoint flake が必要だが、HM の `nput.entries` はモジュール config 内にあり、ユーザー flake の `nput.<system>.<name>` outputs には現れない。
- 既ビルド済み link-farm を再度 `nix eval`/`nix build` し直すのは二度手間で、switch ごとに評価コストがかかる。

つまり**既存 CLI には「ビルド済み link-farm を engine へ直接渡す口」が無かった**。engine.Apply 自体は `Build=nil` + 既ビルド済み `LinkFarm` の経路を持つ（tmpdir テスト用）が、CLI から到達できなかった。これが計画外の仕様欠落である。

## 決定

### 1. `apply --manifest <link-farm>` を engine kick 契約にする

`nput apply` に `--manifest <link-farm>` フラグを足し、**モジュール / host activation が engine を kick する契約**とする。これに伴い `apply` の定義を「**named manifest を適用する。manifest の取得元は entrypoint を build するか、ビルド済み link-farm を直接渡すか**」へ拡張する（CONTEXT.md の `apply` / `nput CLI` 定義を更新）。

- `--manifest` 指定時は **entrypoint 発見・rootKind 先取り eval・`nix build` を行わない**。引数の link-farm を engine.Apply の `Build=nil`（既ビルド済み LinkFarm）経路へ直接渡す。
- **取得後の挙動（flock → 前世代 diff → 配置 → 保守的 stale 除去 → `nix-env --set` → レポート）は通常 apply と完全に同一**。engine の `Build` / `LinkFarm` seam にそのまま対応し、配置ロジックの二重化は無い（ADR-0003：配置は全層 engine が単一所有）。
- 引数は **link-farm**（`mkManifest` 出力の store path = 世代として `nix-env --set` でコミットされる対象）。`manifest.json` はその中から engine が読む。`manifest.json` 単体ファイルではなく link-farm を渡すのは、世代コミットに GC アンカー symlink farm を含む link-farm 全体が要るため。
- **rootKind は `manifest.json` から engine が読む**（先取り eval は不要）。HM は `homeRoot` を pin するため home だが、`--manifest` 自体は全モード（project / fixed / home）の link-farm を受理する汎用フラグ。

### 2. フラグ組合せ: `--manifest` は `-f` / `--all` と排他、位置引数 `name` は直交

`--manifest` は「取得元」を link-farm に固定するため、entrypoint 発見系フラグと意味が衝突する。

- `--manifest` + `-f`（entrypoint 明示）→ **error 停止**（取得元の二重指定）。
- `--manifest` + `--all`（entrypoint の config 列挙適用・将来 #14）→ **error 停止**。
- 位置引数 `name` は **profile 選択**として `--manifest` と直交し両立する（`name` 省略 = `default` = ADR-0024 の HM 固定 profile と一致）。`--manifest` で `name` を黙って無効化しない。
- `--dryrun`（将来）はビルド済み manifest からプラン表示でき、`--manifest` と両立する。

### 3. HM activation の invocation は blocking + 可視 + 失敗で switch 停止

HM モジュールは `home.activation.nput`（`entryAfter ["writeBoundary"]`）から `run ${nput}/bin/nput apply --manifest <link-farm>` を実行する。

- **blocking flock**（`--no-wait` を付けない）。`home-manager switch` は意図的な直列操作で、try-lock skip は「switch 成功だが nput は未配置」の silent footgun を生むため採らない。
- **レポート可視**（`--quiet` を付けない）。placed/replaced/removed を stderr に出し、`home.file` の symlink 配置出力と一貫させる。
- **engine error（conflict 等）は非 0 終了で `run` が伝搬し switch を止める**。宣言的 switch として正しい（`home.file` が clobber で error 停止するのと同型）。
- 設定ノブは持たず `options.nput` は `{ enable; entries; }` のまま（ADR-0024 §2・ADR-0025 §2 の MVP positioning）。

### 4. `--manifest` は公開・安定な CLI seam

`--manifest` は `--help` に載せる**公開フラグ**とし、`manifest.json` の schemaVersion と同じく **Nix↔CLI 契約面の一部**として安定させる（将来の engine も受理を継続）。位置づけは「host/module activation の seam（ビルド済み manifest を CI 等で直接適用する用途にも使える）」。将来の NixOS / nix-darwin モジュール（ADR-0004）も同じ seam で engine を kick する。

## 根拠

- §1 は ADR-0007 の entrypoint 駆動 UX を壊さず、engine が既に持つ `LinkFarm` seam を CLI へ素直に露出するだけの最小追加。配置の単一所有（ADR-0003）も保つ。
- §2 は「取得元の二重指定を黙って解決しない」明示性で、位置引数 `name` を profile 選択として残すことで将来の HM 複数 profile seam（ADR-0024 §棄却）とも素直に接続する。
- §3 は宣言的 switch の安全性（未配置の silent skip を作らない・conflict を握り潰さない）を `home.file` の既存 UX に揃える。
- §4 により module 経路は **schemaVersion skew が構造的に起きない**: CLI（`packages.nput`）と manifest を生んだ `nput.lib`（`mkManifest`）が**同一 nput flake input 由来**で、activation script に焼き込まれる link-farm も同じ input の評価結果だから。standalone のグローバル install で起きうる skew（ADR-0006, ADR-0015）はこの経路には当てはまらない。

## 影響

- **実装**:
  - `cmd/nput`（apply.go / main.go）: `--manifest` フラグ追加・`-f` 併用 error・`runApplyManifest`（engine.Apply の `LinkFarm` 経路駆動）。`internal/engine` は既存 `Build=nil` 経路をそのまま使い未変更。
  - `modules/common.nix`: 共通 `options.nput { enable; entries }`（entry submodule は `lib/types.nix` と共有）。
  - `modules/home-manager.nix`: `homeRoot` pin → `mkManifest` で link-farm → `home.activation` から `apply --manifest` kick。
  - `flake.nix`: `homeManagerModules.default` を pin 版 nput CLI 注入ラッパーで公開。評価専用 `home-manager` input（lib/ は home-manager 非依存のまま）。`checks.hm-module`。
- **`docs/spec.md`**: CLI サブコマンド体系 / グローバルフラグに `apply --manifest` を追記。実行フローに `--manifest` 経路（eval/build を skip）を追記。モジュール別動作仕様の home-manager 節に invocation を明記。エラー仕様表に「`--manifest` + `-f`」行を追加。
- **`docs/design.md`**: モジュール統合設計に engine kick = `apply --manifest` を反映。
- **`CONTEXT.md`**: `nput CLI` / `apply` の定義を「取得元 = entrypoint build | ビルド済み link-farm」へ拡張。`module` に kick 方法（`apply --manifest`）を明記。
- **テスト**: `checks.hm-module` は HM standalone configuration を評価し activation 配線・`homeRoot` pin・entries 反映をアサート。実 activate（`nix-env --set`・FS 配置）は build sandbox では行えないため E2E（#19）へ回す。

## 棄却した代替案

- **専用サブコマンド `nput activate --manifest`**: `apply` を entrypoint 専用に保てるが、apply の全フラグ（`--no-wait` / `--recopy` / `--root` / `--quiet`）とレポートを重複実装する必要があり、`activate` が home-manager の "activation" 語彙と衝突する（`home.activation` 内の `nput activate` が再帰的に読める）。取得後の挙動が apply と完全同一な以上、別動詞にする価値が薄い。
- **生成 entrypoint flake + 既存 `apply`**: 純 Nix で済むが、評価済み `entries`（derivation / marker 混在）を flake source へ直列化する必要があり脆く（`normalizeManifest` の再実装）、`switch` ごとに `nix eval`/`build` が走る。pure-eval での store path 参照可否も不確実。
- **engine kick の CLI 口を別 issue へ分離し #17 は Nix 側のみ**: 境界は綺麗だが、AC「HM standalone configuration を評価・activate して配置をアサート」を #17 単体で満たせず、activation が実装不能のスタブになる。
- **`--manifest` を内部隠しフラグ**: 公開面を狭く保てるが、activation script に `nput apply --manifest <lf>` が生で現れるためユーザーの目に触れる一方で説明が無く、チグハグ。NixOS/darwin（将来）も同 seam を使うため、安定契約として公開する方が一貫する。
