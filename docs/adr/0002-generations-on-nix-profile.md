# ADR-0002: 世代管理を nix profile に乗せる（standalone 専用）

- ステータス: 採用
- 日付: 2026-06-07
- 関連: ADR-0001, ADR-0003, `docs/concept.md`, `docs/spec.md`（世代管理仕様）

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
- **standalone 専用。** HM / NixOS / nix-darwin モジュール時は nput 独自 profile を作らず、
  ホストの世代システム（home-manager generations / nixos generations）に委譲する。
  ロールバックはホスト世代が旧 config で nput エンジンを再実行することで担保する。
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
  - standalone: 自 profile の**前世代**の store マニフェストと新世代を diff する。
  - モジュール時: ホスト世代が与える**旧世代パス（`$oldGenPath` 相当）**の store マニフェストと diff する。
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
- モジュール時に nput 独自 profile を作るとホスト世代と二系統になり、ロールバックが不整合になる。

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
- **全層で nput 独自世代**: ホスト世代と二重化し不整合。
- **stale 除去で新マニフェストに無い target を一律削除**: ユーザーが差し替えた実ファイルをクロバーするため、保守的削除を採用。
- **copy も世代追跡（再マテリアライズ / target スナップショット）**: ユーザー管理の副作用という位置づけと矛盾し、
  store 外スナップショット管理が重い。
