---
id: "REQ-8409db86-a1ba-4053-86dc-588985cc1ca7"
type: requirement
name: "target 除去後は空の親ディレクトリチェーンを root 境界まで保守的に剪定する"
specification: |
  After a target has been removed, the chain of its parent directories SHALL be pruned
  conservatively towards the root, because nput creates parent directories automatically
  when placing but does not record them in the manifest, so that without pruning empty
  directories would pile up whenever the entry hierarchy changes and would become a source
  of conflicts blocking later placements. A directory SHALL be removed only while it is
  empty; since `rmdir` succeeds only on an empty directory, a non-empty one (ENOTEMPTY, or
  EEXIST in some implementations) SHALL be treated as success and left silently in place,
  which requires no additional lock and is free of TOCTOU hazards. The root itself SHALL
  never be removed and the walk up the parent chain SHALL always stop at the root
  boundary. Where a symlink appears in the ancestor chain, the walk SHALL stop without
  touching it, since removing a symlink would delete the link itself rather than affect
  the entity behind it. The pruning SHALL apply to the final removal paths: stale removal
  shared by apply and rollback, the symlink removal of `reset`, and the deletion of a copy
  target; the walk SHALL be independent for each removed target, so that a directory
  shared with another entry naturally remains as non-empty. The pre-placement removal
  (PreRemove) SHALL NOT invoke this pruning, since each of its removals is by definition
  followed immediately by a placement at the same location, and since pruning in the
  middle of a bottom-up sequence would pre-empt the subjects of the explicit removal
  actions still to be executed and make the report inaccurate. The pruning SHALL NOT be
  reported as a warning, being intended housekeeping: it SHALL instead be reported as a
  `pruned <path>` line of the placement report, and SHALL thereby fall under the ordinary
  output discipline of that report. A failure of the pruning itself, other than the
  ENOTEMPTY case above, SHALL follow the existing policy of the caller.
specification_ja: |
  target を除去した後、その親ディレクトリチェーンを root 方向へ保守的に剪定しなければならない。
  nput は配置時に親 dir を自動作成するが manifest には記録しないため、剪定しないと entry 階層を
  変更するたびに空 dir が積もり、後続配置を塞ぐ conflict の温床になるためである。剪定は空のときだけ
  rmdir する。rmdir は空ディレクトリでしか成功しないため、非空（`ENOTEMPTY`。一部実装では
  `EEXIST`）は成功扱いとして黙って残す（追加ロック不要・TOCTOU 安全）。root 自体は絶対に消しては
  ならず、親チェーンの走査は root 境界で必ず停止する。祖先チェーンに symlink が現れたら、それに
  触れず停止しなければならない（symlink を rmdir するとリンクそのものを消してしまい実体には
  触れないため）。適用範囲は最終段の除去経路（apply / rollback から共用する stale 除去・`reset` の
  symlink 除去・copy target 削除）とし、除去した target ごとに独立して walk する（別 entry が同じ
  dir を共有していれば非空として自然に残る）。配置前除去（PreRemove）はこの剪定を呼んではならない。
  PreRemove の各除去は必ず直後の配置で同じ場所に再配置される前提であり、また bottom-up 順の処理の
  途中で祖先剪定を挟むと、まだ実行していない後続の明示的な除去アクションの対象を先取りして消して
  しまいレポートが不正確になるためである。剪定は意図された掃除であり warning にしてはならない。
  配置レポートの `pruned <path>` 行として報告し、同レポートの通常の出力規律に従わせる。剪定
  自体の失敗（上記 ENOTEMPTY を除く異常系）は呼び出し元の既存ポリシーに従う。
---
# REQ-8409db86: target 除去後は空の親ディレクトリチェーンを root 境界まで保守的に剪定する

## 仕様

target を除去した後、その親ディレクトリチェーンを root 方向へ保守的に剪定する
（HM の `rmdir -p --ignore-fail-on-non-empty` 対称）。nput は配置時に親 dir を `mkdir -p` 相当で
自動作成するが manifest には記録しないため、剪定しないと entry 階層を変更するたびに空 dir が
積もり、後続配置を塞ぐ conflict の温床になる。

- **空のときだけ** rmdir する。rmdir は空ディレクトリでしか成功しないため、非空（`ENOTEMPTY`。
  一部 rmdir(2) 実装では `EEXIST`）は**成功扱いとして黙って残す**（追加ロック不要・TOCTOU 安全。
  並行書き込みで非空になれば rmdir がそこで失敗し自然に停止する）
- **root 自体は絶対に消さない**。親チェーンの走査は root 境界で必ず停止する
- **祖先チェーンに symlink が現れたら、それに触れず停止する**（symlink 自体を rmdir すると
  リンクそのものを消してしまい実体には触れないため、安全側に倒して何もしない）
- **適用範囲は最終段の除去経路**: removeStale（apply / rollback から共用）・`reset` の symlink
  除去・copy target 削除。除去した target ごとに独立して walk するため、別 entry が同じ dir を
  共有していれば非空として自然に残る
- **PreRemove は本剪定ヘルパを呼ばない**。PreRemove の各除去（Unlink / Rmdir）は必ず直後の配置で
  同じ場所に何かが再配置される前提のため、剪定しても即座に作り直されるだけで無駄な往復になる。
  1 回の PreRemove 呼び出しで複数の Unlink/Rmdir を bottom-up 順に処理するため、途中で祖先剪定を
  挟むと、まだ実行していない後続の明示的な Rmdir アクションの対象を先取りして消してしまい
  レポートが不正確になる
- **出力規律**: 剪定は意図された掃除であり warning にはしない。既定 silent、`-v` で配置レポートに
  `pruned <path>` として可視化する。剪定ヘルパ自体の失敗（権限エラー等の異常系。ENOTEMPTY は
  上記の通り対象外）は、呼び出し元の既存ポリシーに従う: removeStale / `reset` は warning で残置

> **上は原文の写しで、規範は frontmatter が正**。原文が補足する「実 dir migration で除去する
> dir チェーンは planner の `classifyDirMigration` が子から親まで明示的に列挙済みで、剪定ヘルパ
> による探索的な追加除去を必要としない」は実装構造の説明で規範ではない（実 dir migration
> そのものは REQ-7cee95dd）。空 dir の残置が cosmetic な取りこぼしに過ぎないことも同様の注記。
> 写しの「出力規律」の項が言う**既定 silent と `-v` の規律そのもの**（→ ADR-0031）は
> REQ-8ef34101 / REQ-0a123b89 の担当で、本 item の規範ではない。本 item が規範とするのは
> 「剪定を warning にせず配置レポートの `pruned <path>` 行として出す」ことだけで、その
> レポートを既定で出すか `-v` で出すかは委譲する。配置レポートの基本行種別
> （`placed` / `replaced` / `removed` / `skipped`）とストリーム規律も REQ-fea038de の担当で、
> 本 item はそこへ剪定固有の `pruned` 行を足す。stale 除去の不変条件は REQ-16aef46b、
> `reset` の除去は REQ-31f2882e の担当。

## 出典

`docs/spec.md`「世代管理仕様」→「stale 除去の対象と安全不変条件」→「空親ディレクトリ剪定」節。

決定の実体は ADR-0047「配置前除去（PreRemove）の一般化」§5（PreRemove が剪定ヘルパを呼ばない
こと）と、出力規律を定める ADR-0031。剪定そのものは epic #172 D4（Issue #174）で確定した。
