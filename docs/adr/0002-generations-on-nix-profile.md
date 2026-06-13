# ADR-0002: 世代管理を nix profile に乗せる（全モード自前 profile / rollback は standalone 中心）

- ステータス: 採用（2026-06-11 改訂: 実装機構を固定 Go エンジンに変更 → ADR-0006／2026-06-14 改訂: 世代操作 CLI を `nix-env` 系統一・rollback diff 基準を確定 → ADR-0015）
- 日付: 2026-06-07
- 関連: ADR-0001, ADR-0003, ADR-0006, ADR-0015, `docs/concept.md`, `docs/spec.md`（世代管理仕様）

> **2026-06-14 改訂注記（ADR-0015）**: 本 ADR の profile / 世代 / 保守的 stale 除去という **decision 自体は不変**。
> ただし世代操作の CLI は **`nix-env --profile <dir>` 系で統一**された——本文中の「任意世代切替・GC を `nix profile` /
> `nix-collect-garbage` に委譲」「`nix profile wipe-history`／`nix-env --delete-generations`」は、`nix-env --profile <dir>`
> 系（`--set` / `--rollback` / `--switch-generation` / `--list-generations` / `--delete-generations`）+ store GC は `nix-collect-garbage`、
> と読むこと。新 `nix profile` は profile-manifest を要求し `nix-env --set` 製 profile では動かないため。
> `nput rollback` の stale 除去は「baseline=離れる世代・ポインタ移動は最後」で行う（→ ADR-0015）。

> **2026-06-11 改訂注記（ADR-0006）**: 本 ADR の profile / 世代 / store マニフェスト / 保守的 stale 除去という
> **decision 自体は不変**。ただし実装機構は変わった——「activation スクリプト（生成 bash）」は**固定 Go エンジン
> （`packages.nput`）**に置き換わり、`manifest.json` を入力に取る。フラグ式 CLI（`nput --rollback` 等）はサブコマンド体系
> （`nput rollback` / `list-generations` / `gitignore`、dry-run は `--dryrun`）に置き換わる。以下の本文中の「activation スクリプト」は
> Go エンジン、CLI はサブコマンド体系として読むこと。

## 背景

初期方針は「世代管理をしない」だった（冪等な symlink/rsync のみの stateless スクリプト）。
これを覆し、ロールバック可能な世代管理を追加する。世代管理は本質的に副作用（profile symlink の差し替え・
GC root 登録・状態書き込み）を要するため、「純粋関数」という方針との両立方法を定める必要がある。

## 決定

- **nix profile に乗せる（home-manager 方式）。**
  - 純粋関数が link farm derivation（ストア内の symlink ツリー）を生成する。
  - activation スクリプトが `nix-env --profile <dir> --set <drv>` で nix profile に登録する。
  - 世代番号・GC root・ロールバックを Nix 標準機構から得る。
- **純粋性と副作用の両立**: 純粋関数は「link farm derivation と activation スクリプトを生成するだけ」。
  実際の世代 swap は activation 実行時の副作用とする。
- **nput は全モードで自前 profile を持つ（standalone / module 共通）。** 前世代マニフェストの出所を統一するため、
  モジュール時もホストの世代に依存せず nput 自身の profile を保持する（home-manager が NixOS submodule でも自前 profile
  `profiles/home-manager` を持つのと同じ）。host から要るのは「switch 時に nput を kick する」ことだけで、ホストの oldGenPath 配管は不要。
  - **standalone**: profile はユーザー向け。`--rollback` / `--list-generations` を提供する。
  - **module**: profile は**内部機構**（前世代マニフェスト + stale 追跡）に留める。ユーザー向けロールバックは host
    （`home-manager --rollback` 等）に一本化し、`nput --rollback` は standalone 限定。host rollback は旧 config を再 activate
    し nput を再 kick することで自動追従する（nput profile は前進のみ＝旧内容を持つ新世代を積む）。desync を避けるため module で独立 rollback は公開しない。
- **粒度 = `mkActivationScript` 単位 = 1 profile。** 役割ごとの独立は別スクリプト＝別 profile で担保する。
  この決定に伴い `mkActivationScript` は profile を一意特定する `name` 引数を取る。
- **standalone は世代を常時 ON。** profile が新しいデフォルトであり、世代なしモードは持たない（コードパスを 1 本にする）。
- **CLI は最小**: `activate`（デフォルト） / `--rollback`（前世代へ） / `--list-generations`。
  任意世代切替・世代 GC は標準の `nix profile` / `nix-collect-garbage` を profile パスに対して使う（委譲）。
- **`--only` を廃止する。** profile 世代はマニフェスト全体で atomic であり、一部 entry だけの適用は世代状態を壊す。
- **copy は世代から除外する。** copy は place-once（target が在れば触らない）・ユーザー管理の副作用として扱い、
  ロールバック対象外。詳細は ADR-0004 と本 ADR の「copy の扱い」を参照。
- **out-of-store は世代追跡する**が、リンク先マッピングのみ版管理し、指す先の内容は設計上ライブで永遠にスナップショットしない。
- **stale 除去は「世代由来の store マニフェスト」を diff して行う（home-manager 方式）。**
  「配置したもの」のマニフェストは link farm derivation の一部として **store 内に**埋め込む。可変 JSON ファイルは持たない。
  - **全モード共通**: nput 自身の profile の**前世代**の store マニフェストと新世代を diff する（standalone も module も同一）。
    ホストの oldGenPath には依存しない。
  - 記録は不変・GC-root 済み（世代に捕捉される）であり、rollback で記録も巻き戻る。
  - home.file モジュール自体は再利用しないが、配置・cleanup 機構は home-manager の `linkGeneration`/`cleanup` を参考に再実装する。
- **削除の安全不変条件（保守的・HM 流）。** stale 除去で削除してよいのは、**前世代マニフェストが「nput が配置した」と記録し、
  かつ現状もその記録通りの先（その世代の store パス／記録された out-of-store パス）を指す symlink** のみ。
  通常ファイルや nput 非管理の link には触れない。記録と実体が不一致なら削除せず警告する。
  初回／前世代マニフェスト不在時は「記録ゼロ＝削除対象ゼロ」で何も消さない（既存 target との衝突は従来通りエラー停止）。
- **GC とストレージ解放。** profile の各世代は GC root。`nix profile wipe-history`／`nix-env --delete-generations` で
  旧世代を間引き、`nix-collect-garbage` で無参照になった store パスを解放する。
  可変 JSON 方式は GC root を作らず、参照中 store パスが GC で消えて rollback が壊れる／自前 gcroot 管理が要るため採らない。

## copy の扱い

- copy は「初回マテリアライズしたら以後 nput は触らない」place-once。target が在れば上書きしない。
- ストア更新の反映は明示再適用（target 削除後に再実行、等）に委ねる。
- stale 除去の対象外（entry が消えても copy target は削除しない）。ただし orphan 化した copy は警告で通知する。
- 理由: copy は元々 `rsync --delete` で再適用のたびに手編集を上書きしており、「ユーザー管理の副作用」と明示するのが整合的。

## 根拠

- nix profile は profile symlink の差し替えだけで atomic な switch/rollback を実現し、GC root にもなる。
  「Nix エコシステムに則る」要求に最も合う。
- 「任意パス配置 × 世代管理」を組合せた既存ツールは存在しない（調査確認済み）。この空白を埋める価値がある。
- nput が全モードで自前 profile を持つことで、前世代マニフェストの出所が「自 profile の前世代」に統一され、
  ホストごとに異なる oldGenPath 配管を持たずに済む（「他モジュールの内部事情を考えない」方針と整合）。
- module 時に nput profile と host 世代が併存しても不整合は起きない。host rollback は旧 config を再 activate し
  nput を再 kick することで FS を自動収束させる（home-manager が NixOS submodule で自前 profile を持つのと同じ実績ある構図）。
  desync は「ユーザーが module で独立 rollback を打つ」場合のみ起きるため、module では nput --rollback を公開しないことで排除する。

## 影響

- concept.md の「世代管理をしない」記述を撤回し、世代管理節を追加する。
- spec.md に世代管理仕様（profile 位置・store マニフェスト・保守的削除・CLI）を追加し、`--only` を削除する。
- design.md の mkActivationScript 仕様に `name` 引数と世代生成を反映する。

## 棄却した代替案

- **可変 JSON マニフェスト（`$root/.local/state/nput/<name>.json`）**: 実装は単純だが store 外可変・世代に捕捉されず、
  standalone では profile と二重。GC root を作らないため参照中 store パスが GC で消えて rollback が壊れる／自前 gcroot 管理が要る。
  → 世代由来の store マニフェスト（HM 方式）を採用。
- **独自の軽量世代ログ**: nix profile 機構を再発明し、GC 安全性を自前担保する必要がある。
- **世代を取らずバックアップのみ（.bak）**: 「世代管理もできる」要求に応えられない。
- **世代を opt-in（デフォルト世代なし）**: stale 除去のため結局状態管理が要り、二重コードパスになる。
- **module 時は nput 独自 profile を作らずホスト世代に委譲**（当初案）: 前世代マニフェストの出所がホストごとに異なる
  oldGenPath に依存し、配管が per-host で増える。「他モジュールの内部事情を考えない」方針と矛盾するため、全モード自前 profile に変更。
- **module 時は stale 除去しない（previous を持たない）**: dangling link が残り standalone と振る舞いが不一致になる。
- **stale 除去で新マニフェストに無い target を一律削除**: ユーザーが差し替えた実ファイルをクロバーするため、保守的削除を採用。
- **copy も世代追跡（再マテリアライズ / target スナップショット）**: ユーザー管理の副作用という位置づけと矛盾し、
  store 外スナップショット管理が重い。
