# ADR-0015: 実装前レビューで surfaced した残セマンティクスを確定する（devShell 同梱 / cross-config 衝突 / nix-env 統一 / 祖先 symlink / rollback diff / mode→method / flake check）

- ステータス: 採用
- 日付: 2026-06-14
- 関連: ADR-0002, ADR-0005, ADR-0007, ADR-0010, ADR-0011, ADR-0013, ADR-0014, `docs/spec.md`, `docs/design.md`, `docs/concept.md`, `CONTEXT.md`
- 改訂対象: **ADR-0002**「世代 GC / rollback を `nix profile`（新CLI）/ `nix-collect-garbage` に委譲」を `nix-env` 系統一へ訂正。**ADR-0007 / templates** に devShell 同梱を canonical として追記。**ADR-0013** の配置セマンティクス（foreign symlink・祖先 symlink）を拡張。**ADR-0010 / ADR-0014** の entry フィールド `mode` を `method` に改名
- 起点: ドキュメント計画段階の最終レビュー（grill）で surfaced した「実装前に決めておくべき」残細目の束

## 背景

ADR-0013 / ADR-0014 で engine 実行時セマンティクスと entries モデルの大半が固まったが、
実装着手前のレビューで次の 5 点に未定義 / 仕様の揺れが残っていた。

1. **project mode で `nput` CLI をどう PATH に載せるか未定義**。中心トリガの devShell `shellHook = "nput apply ..."` が
   `nput` を前提にするのに、devShell が `nput` を PATH に供給する方法が docs に無い。グローバル install 依存だと
   プロジェクトが pin した `nput.lib`（manifest schemaVersion）とグローバル CLI のバージョンが skew しうる。
2. **別 config（別 profile）が同一 target に書き込むときの挙動が未定義**。spec は同一 manifest 内の衝突しか扱わず、
   `nput.vim` と `nput.zsh` が同じ target を狙うと silent clobber + ping-pong が起きる。
3. **世代操作の CLI が混在**。コミット点は `nix-env --profile <dir> --set`（spec/ADR-0002）なのに、GC / rollback で
   `nix profile`（新CLI）/ `wipe-history` に委譲と書かれている。新 `nix profile` は profile-manifest を要求し、
   `nix-env --set` 製 profile（中身が link-farm そのもの）では動かない。
4. **target の祖先 component が symlink のとき未定義**。entry A が全体 symlink を置き、entry B がその配下を狙うと、
   `os.MkdirAll` が symlink-to-store を「既存 dir」と見なし、続く配置が read-only store へ書こうとして失敗 / store 汚染。
5. **`nput rollback` の stale 除去 diff 基準が未定義**。nput は profile dir でなく任意 root に配置するため、profile ポインタ
   移動だけでは FS が変わらず再配置が必須だが、diff の baseline / target / 順序が spec に無い。

## 決定

### 1. project mode は devShell に pin 版 `nput` を同梱する（canonical）

- `templates/project` の devShell は `packages = [ nput.packages.${system}.nput ]` を含め、`nix develop` / direnv 入室時に
  **flake.lock で pin した `nput`** を PATH へ載せる。CLI と `nput.lib`（manifest）が同一 flake 入力から来るため
  `schemaVersion` が必ず一致し、project-first / 再現性（flake.lock 固定）の thesis と整合する。
- グローバル install（`nix profile install github:<owner>/nput`）は **standalone（home mode）の利便**として残すが、
  project mode の canonical 経路ではない。spec / design / concept に「project mode は devShell 同梱が canonical」と明記する。

### 2. cross-config の同一 target は後勝ち維持 + foreign symlink 上書き時に warning

- 別 config（別 profile）が同一 target を狙うのは基本「衝突させない前提」とし、**後勝ち**（last-writer-wins）を維持する
  （ADR-0013 の同一 profileDir 後勝ちポリシーと同思想）。グローバル registry での所有者管理は「store 外可変 state を持たない」
  方針に反するため採らない。
- ただし engine が **自身の前世代 manifest に記録の無い symlink（foreign = 他 nput profile / 他ツール / 手動）を上書きするときは
  warning を出す**。silent clobber を避けつつ停止はしない。通常ファイル / ディレクトリは従来通り error 停止（上書きしない）。
- 配置時の判定を次に対称化する。

```
target が:
  不在                                  → 配置
  自分の前世代 manifest が記録した symlink → 置換（silent・従来通り）
  上記以外の symlink（foreign）           → warning を出して置換（後勝ち）← 本 ADR
  通常ファイル / ディレクトリ             → error 停止（従来通り）
```

### 3. 世代操作は `nix-env --profile <dir>` 系で統一する

- コミット・rollback・任意世代切替・一覧・間引きを **すべて `nix-env --profile <dir>` 系**に統一する。

```
コミット   : nix-env --profile <dir> --set <link-farm>
rollback  : nix-env --profile <dir> --rollback
任意切替  : nix-env --profile <dir> --switch-generation N
一覧      : nix-env --profile <dir> --list-generations
間引き    : nix-env --profile <dir> --delete-generations <gens>
store GC  : nix-collect-garbage
```

- ADR-0002 / spec から `nix profile`（新CLI）/ `nix profile wipe-history` の記述を除去する。新 `nix profile` は profile 内に
  自身の profile-manifest（要素一覧）を要求するが、`nix-env --set <drv>` 製 profile はそれを持たず（profile 直下が
  link-farm = nput の `manifest.json` を含む）、`nix profile list / wipe-history / rollback` が「profile manifest が無い」で
  失敗 / legacy 扱いになる。`nix-env` 系は世代 symlink のみを見るため任意 profile で動く。
- 「nput の profile は `nix-env` 式であり `nix profile` サブコマンドとは非互換」と spec に明記する。

### 4. target の祖先 component が symlink なら配置前に error 停止

- 配置前に target の各祖先 component を **lstat で walk** し、いずれかが symlink なら明確なメッセージで error 停止する
  （例: `祖先 .claude/skills は symlink 配置。配下に nix をネストできない`）。`os.MkdirAll` が symlink-to-store を
  既存 dir と見なして store へ書く / dangling を作るのを防ぐ。
- `--dryrun` でも同条件を **conflict** として検出し非ゼロ終了する。
- docs に回避策を明記する: 全体 symlink 配置の配下に別 entry をネストできない。entry を細分化する（親を置かず子 subdir ごとに置く）か、
  親 entry を `mode = "copy"` にする。

### 5. `nput rollback` は baseline=離れる世代、ポインタ移動を最後にする

- rollback を「baseline = 現世代 N、target = 戻る世代 N-1」の一般配置ルーチンとして定義する。

```
nput rollback <name>  (gen N → N-1)
  1. baseline = 現世代 N の manifest（FS の現状）
  2. target   = N-1 の manifest
  3. 除去     = N ∖ N-1 の entry（保守的不変条件下で stale 除去）
  4. 配置     = N-1 の entry を place / replace
  5. ポインタ = nix-env --profile <dir> --rollback（または --switch-generation N-1）を最後に
```

- apply エンジンを `(baseline, target)` を差し替えて再利用する。profile ポインタを先に動かすと baseline が N-2 にずれて
  誤った stale 除去になるため、**FS 収束を先に・ポインタ移動を最後に**する。home mode 限定（project mode は rollback 非公開・ADR-0005）。

### 6. entry の `mode` を `method` に改名する

- entry フィールド `mode`（`"symlink"` | `"copy"`）を **`method`** に改名する。`mode` は unix file mode（`0644` 等）と誤読の
  余地があり、配置種別の enum と紛らわしい。`manifest.json` の `mode` も `method` に揃える（Nix 記述・契約・engine で名を一致）。
- 旧名 `mode` は submodule が strict（未知キー拒否・ADR-0010）なので eval 時エラーになる。実装が無く schemaVersion 初期の今が
  改名コスト最小（ADR-0008 / ADR-0009 の改名と同じ理屈）。

### 7. `schemaVersion` は MVP では v1 のみ。前方移行は v2 で考える

- engine は自身の対応版より新しい `schemaVersion` を拒否する（ADR-0006）。**MVP は v1 のみ**を対象とし、v1→v2 の migration
  方針は本 ADR のスコープ外とする。将来 v2 を切るときに別 ADR で移行戦略（engine 側の後方互換読み込み等）を決める。

### 8. `nput` カスタム output の `nix flake check` 警告は無害として許容する

- consumer flake の `outputs.nput.<system>.<name>`（ADR-0007 で `packages` 汚染回避のため採用したカスタム namespace）は、
  `nix flake check` で **`warning: unknown flake output 'nput'` を出すが exit 0**（CI を壊さない）。これを**想定内・無害として許容**し、docs に明記する。
- nix 2.27 / 2.34 実測 + nix ソース（`src/nix/flake.cc` の `CmdFlakeCheck`）で確認した事実:
  - `nix flake check` は各 output の**トップレベル値を force する**ので `nput` 直下 attrset の eval 健全性は検査される（むしろ利点）。
    一方 **未知 output は再帰検査せず**、配下の `.<system>.<name>` derivation は **build も eval もされない**（ADR-0007 の「`packages` 流用は誤 build しうる」回避を支持）。
  - **flake-parts を経由しても警告は消えない**（freeform で最終 outputs へ素通しするだけ。警告抑制機構なし）。
  - 警告を消す upstream 機構は **flake-schemas（PR #8892）だが未マージ**で Determinate Nix 限定。`homeConfigurations` 等は nix 本体の
    community-known リスト（`flake.cc` ハードコード）に載るため警告が出ないが、`nput` は未登録。output 名を `lib` 以外へ変えても unknown 警告は出る。
- 将来 PR #8892 が upstream マージされたら `schemas.nput` を提供して警告を消す余地を残す（現時点では Determinate 限定依存になるため採らない）。

## 根拠

- **devShell 同梱**は project-first の再現性を CLI レベルまで延ばす。グローバル CLI 依存は flake.lock の pin を CLI で破る穴になり、
  schemaVersion skew を招く。同梱なら CLI と lib が単一入力で一致する。
- **cross-config 後勝ち + warning** は ADR-0013 の後勝ちポリシーと一貫させつつ、per-profile manifest だけで「把握できていない上書き」を
  可視化する。error 停止まで強めると正当な後勝ち運用（意図的な再配置）を阻害するため、warning に留める。
- **`nix-env` 統一**は「コミットを `--set` で行う」既定（ADR-0002）から論理的に必然。`nix profile` 併記は実際には動かない仕様の揺れだった。
- **祖先 symlink で停止**は ADR-0013 の「未定義挙動を早期に弾く」「保守的・誤破壊しない」と一直線。store 汚染 / dangling は黙って起きると発見が遅い。
- **rollback の baseline 明文化**は nput が任意 root に配置する（profile dir ≠ 配置先）構造から必然的に要る順序制約。書かないと
  「ポインタを動かせば戻る」という `~/.nix-profile` 的直感で実装すると FS が収束しない / stale が誤る。
- **`mode` → `method` 改名**は誤読の芽を実装前に摘む。`mode` は file permission を強く連想させ、配置種別 enum と意味が混線する。
- **v1 のみ**は MVP のスコープを締め、まだ存在しない v2 の移行戦略を先取りで設計しない（YAGNI）。
- **flake check 警告の許容**は実測に基づく。警告は exit 0 で CI を壊さず、子 derivation の誤 build も起きず、`nput` 直下の eval 健全性は
  むしろ検査される。回避策（flake-parts / 別名 / flake-schemas）はいずれも upstream 安定版で警告を消せないか過剰依存を招くため、許容が最小コスト。

## 影響

- **`docs/spec.md`**: CLI 仕様 / 実行フローに devShell 同梱を明記。配置動作仕様（symlink モード）に foreign symlink warning と
  祖先 symlink error を追記。世代管理仕様 / GC / rollback の CLI を `nix-env` 系へ統一し `nix profile`（新CLI）記述を除去。
  rollback 節に baseline / 順序を明記。エラー仕様表に「祖先 component が symlink」を追加。
- **`docs/design.md`**: プロジェクト構成 / 使用パターン 1 の devShell に `packages` 同梱を反映。世代管理の commit / GC を `nix-env` 系へ統一。
- **`docs/concept.md`**: project mode 説明と使用例 devShell に pin 版 `nput` 同梱を反映。
- **`CONTEXT.md`**: `engine` / `generation` 周辺に nix-env 統一・foreign symlink warning・祖先 symlink 停止を反映。
  `entry / entries` の語に `mode` → `method` を反映。Flagged ambiguities に「project mode の `nput` は devShell 同梱が canonical」を追加。
- **`mode` → `method` 改名の伝播**: 生きた docs を更新する——spec.md（entries スキーマ / `mode` 節 / manifest.json v1 / エラー仕様表）、
  design.md（entries スキーマ表 / 配置例）、concept.md（copy 表）、CONTEXT.md（Flagged ambiguities の `mode = "symlink"` 言及）。
  ADR-0010 / ADR-0013 / ADR-0014 の**本文は当時の名のまま歴史的記録として残し**（ADR-0008 の慣例）、フィールド定義を持つ
  ADR-0010 / ADR-0014 には改訂注記で `mode` → `method`（→ ADR-0015）を指すに留める。
- **flake check スタンス**: spec.md（CLI 仕様 / 再現性スタンス周辺）に「`nix flake check` は `nput` を unknown output として警告するが exit 0・無害・
  子 derivation は build/eval されない」を明記。design.md（flake.nix outputs 設計）に同注記。`nput` 主検証は `nix build .#nput.<sys>.<name>` で行う。
- **ADR-0002**: 改訂注記で GC / rollback の `nix profile` 委譲を `nix-env` 系へ訂正。
- **ADR-0007**: 改訂注記で「project mode は devShell に pin 版 `nput` を同梱」を追記。
- **ADR-0013**: 改訂注記で配置セマンティクスに foreign symlink warning・祖先 symlink error を追加した旨を記録。
- **実装フェーズ**: `templates/project`（devShell `packages` + shellHook）、`internal/`（祖先 lstat walk・foreign symlink 判定 + warning・
  rollback の baseline 差し替え）、`cmd/nput`（rollback / GC の `nix-env` 呼び出し）。

## 棄却した代替案

- **グローバル install 前提を維持（devShell 非同梱）**: 1 回の導入で済むが flake.lock の pin を CLI で破り schemaVersion skew を招く。
- **cross-config を error 停止**: silent clobber は確実に防げるが、意図的な後勝ち再配置（同一 target を別 config で張り替える運用）を阻害する。
- **cross-config をグローバル所有者 registry で管理**: store 外可変 state を持つことになり ADR-0002/0006 の方針に反する。
- **cross-config を完全 silent（現状維持）**: ping-pong / 黙った上書きが発見しづらい。
- **世代 CLI の混在を維持**: `nix profile`（新CLI）が `nix-env --set` 製 profile で動かない仕様の揺れを残す。
- **`nix profile`（新CLI）へ全寄せ**: `nix profile` は profile 全体を任意 drv に `--set` する nput の使い方（profile = link-farm）に合わず、
  要素 install モデルとも噛み合わない。`nix-env --set` が正しいプリミティブ。
- **祖先 symlink を黙って解決（store へ書く / 親を実体化）**: store 汚染 / dangling / 暗黙の振る舞いを生む。
- **rollback でポインタを先に動かす**: baseline が N-2 にずれ stale 除去が誤る。FS が profile と乖離する。
- **`mode` を維持**: 誤読の芽を残す。実装・schemaVersion が初期の今なら改名コストが最小。
- **flake check 警告を flake-schemas で消す**: PR #8892 未マージで Determinate Nix 限定。upstream nix ユーザーを切り捨てる。
- **flake check 警告のため別 output 構造へ逃げる**: `lib` 以外はどの名でも unknown 警告になり、`packages` 流用は誤 build リスク（ADR-0007 棄却済み）。警告自体は消せない。
