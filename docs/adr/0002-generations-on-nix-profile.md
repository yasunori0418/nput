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
- **stale 除去は state マニフェスト diff で行う。** 全層共通で「配置したもの」を state マニフェスト
  （例: `$root/.local/state/nput/<name>.json`）に書き出し、新旧 diff で消えた entry の symlink を除去する。
  standalone は加えて nix profile を登録する。マニフェスト＝全層共通の stale 源、profile＝standalone 限定のロールバック機構。

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
- spec.md に世代管理仕様（profile 位置・state マニフェスト・CLI）を追加し、`--only` を削除する。
- design.md の mkActivationScript 仕様に `name` 引数と世代生成を反映する。

## 棄却した代替案

- **独自の軽量世代ログ**: nix profile 機構を再発明し、GC 安全性を自前担保する必要がある。
- **世代を取らずバックアップのみ（.bak）**: 「世代管理もできる」要求に応えられない。
- **世代を opt-in（デフォルト世代なし）**: stale 除去のため結局状態管理が要り、二重コードパスになる。
- **全層で nput 独自世代**: ホスト世代と二重化し不整合。
- **copy も世代追跡（再マテリアライズ / target スナップショット）**: ユーザー管理の副作用という位置づけと矛盾し、
  store 外スナップショット管理が重い。
