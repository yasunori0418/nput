---
id: "REQ-fc1118b1-b0e8-4ddf-80f6-c70956651693"
type: requirement
name: "同一 target を複数 config で狙うことによる振動はユーザー責任とし warning で可視化するに留める"
specification: |
  Where two different configs aim at the same target, the drift repair may oscillate on
  every re-entry of the `shellHook` — A places it, B detects it as foreign and takes it,
  A takes it back — which is an active oscillation rather than a single instance of the
  last writer winning. Not aiming at the same target from several configs SHALL be the
  responsibility of the user, and nput SHALL make the situation visible through the
  foreign symlink warning and SHALL NOT hold any mechanism that detects and stops it.
  The foreign warning during an oscillation will keep appearing under the high frequency
  at which a `shellHook` runs, and this SHALL be regarded as correct, being the signal of
  a misconfiguration; the warning SHALL be outside the scope of silence on success and
  SHALL therefore appear regardless of `-v`. The MVP SHALL NOT hold any mechanism for
  suppressing or aggregating it, the situation being resolved by removing the duplicate
  target from the configs.
specification_ja: |
  別 config が同一 target を狙うと、ドリフト修復が「A が置く → B が foreign 検知して奪う →
  A が再奪取」と `shellHook` 再入のたびに振動しうる（単発の後勝ちではなく能動的オシレーション）。
  「同一 target を複数 config で狙わない」はユーザー責任とし、nput は foreign symlink warning で
  可視化するに留め、検知して止める機構を持ってはならない。振動中の foreign warning は
  `shellHook` の高頻度実行で出続けるが、これは設定ミスのシグナルとして正しい。この warning は
  成功時沈黙の対象外であり、`-v` の有無に関わらず常時出る。MVP では抑制 / 集約機構を持たず、
  config の同一 target 重複を解消して直す。
---
# REQ-fc1118b1: 同一 target を複数 config で狙うことによる振動はユーザー責任とし warning で可視化するに留める

## 仕様

- **cross-config 同一 target の振動はユーザー責任**: 別 config A / B が同一 target を狙うと、
  lstat 修復が「A が置く → B が foreign 検知して奪う → A が再奪取」と `shellHook` 再入のたびに
  振動しうる（単発「後勝ち」ではなく能動的オシレーション）。「同一 target を複数 config で
  狙わない」をユーザー責任とし、foreign symlink warning で可視化する。検知して止める機構は
  持たない
- **振動中の foreign warning は `shellHook` 高頻度実行で出続けるが、これは設定ミスのシグナルと
  して正しい**。warning は成功時沈黙の対象外なので `-v` の有無に関わらず常時出る。抑制 / 集約
  機構は MVP で持たず、config の同一 target 重複を解消して直す（document-only）

> **上は原文の写しで、規範は frontmatter が正**。foreign symlink warning そのものと単発の
> 後勝ちは REQ-622787dc、lstat ドリフト修復そのものは REQ-46fccb80、warning が成功時沈黙の
> 対象外であることの出力規律そのものは REQ-8ef34101 の担当。同一 config 内での target 衝突を
> 評価時に検出することは REQ-5c6b07da の担当（本 item は config をまたぐ場合を扱う）。

## 出典

`docs/spec.md`「世代管理仕様」→「project mode の世代」節の、世代スキップ時 FS 検査に付随する
入れ子箇条書き 2 項。

決定の実体は ADR-0023「実装前残セマンティクス第5巡」（cross-config 同一 target をユーザー責任と
すること）と ADR-0024「実装前残セマンティクス第6巡」（振動 warning を抑制しないこと）。
