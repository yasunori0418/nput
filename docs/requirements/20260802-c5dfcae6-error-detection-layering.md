---
id: "REQ-c5dfcae6-6094-4850-99e5-bf14530bc60a"
type: requirement
name: "設定の誤りは評価時に、実体の不整合は engine 実行時に検出する層分けを守る"
specification: |
  Errors SHALL be detected in the layer that can decide them. An error that is decidable
  from the configuration alone — an omitted `root`, an ill-typed value, an unknown key, a
  contradictory combination, an absolute path or a `..` escape — SHALL be detected at Nix
  evaluation time by `evalModules` / `lib.throwIf` inside `mkManifest`, so that the engine
  is never started. An error that is only decidable against the actual filesystem SHALL be
  detected by the engine at run time. Uniqueness of entry identifiers SHALL NOT be checked
  at all, being guaranteed natively by the impossibility of duplicate attrset keys, so an
  evaluation-time error for a duplicate `name` SHALL NOT exist. This item states the
  layering alone; each individual condition and its behaviour are stated by the items
  responsible for them and SHALL NOT be restated here.
specification_ja: |
  エラーは、それを判定できる層で検出しなければならない。設定だけで判定できるもの
  （`root` 省略・型不正・未知キー・矛盾する組み合わせ・絶対パス / `..` エスケープ）は
  `mkManifest` の `evalModules` / `lib.throwIf` により Nix 評価時に検出し、engine を
  起動させてはならない。実際のファイルシステムと突き合わせて初めて判定できるものは
  engine 実行時に検出する。entry 識別子の一意性は attrset のキー重複不可により native に
  担保されるため検査そのものを行わず、重複 `name` という評価時エラーは存在してはならない。
  本 item は層分けのみを規定し、個々の条件とその動作は各担当 item の規範であって
  ここでは再掲しない。
---
# REQ-c5dfcae6: 設定の誤りは評価時に、実体の不整合は engine 実行時に検出する層分けを守る

## 仕様

評価時エラー（`root` 省略・不正な型・未知キー・copy+marker・絶対パス / `..` エスケープ）は
`mkManifest` の `evalModules` / `lib.throwIf` が検出する。識別子の一意性は entries が
attrset のため Nix のキー重複不可で担保され、**重複 `name` という評価時エラーは存在しない**。

判定に実体（ファイルシステム）を要する条件は評価時には決まらないため、engine 実行時の
検出に回る。

> **上は原文の写しで、規範は frontmatter が正**。原文が列挙する個々の条件は、それぞれの
> 担当 item が規範を持つ。
>
> - `root` の明示必須 → REQ-4ec3accc
> - 未知キー・旧名の strict 拒否 → REQ-3e446ad9
> - `src` が素の文字列を取らないこと → REQ-99ca5381
> - 絶対パス / `..` エスケープの拒否 → REQ-6911eab6
> - copy + out-of-store marker の矛盾拒否 → REQ-16faf428
> - 識別子 = 属性キーと一意性の native 担保 → REQ-cb77ea05
> - 同一 manifest 内の正規化後 target 重複 → REQ-5c6b07da
> - `mkManifest` が入力検査の単一ゲートであること → REQ-d1b5b3f5
>
> 原文が同じ列挙に含める「`root = systemRoot`（system mode は未実装）を評価時エラーに
> する」は規範に採らない。この決定（ADR-0013 §5）は ADR-0036 が撤回済みで、現行は
> `rootKind = "system"` を正規の値として通すため（REQ-16faf428 の注記と同じ理由）。

## 出典

`docs/spec.md`「エラー仕様」節の導入文と、同節の表のうち評価時エラー行・engine 実行時
エラー行の別。

決定の実体は ADR-0010「manifest の型検査を `evalModules` で行う」（設定の誤りを評価時に
閉じる）と ADR-0013「engine のランタイム意味論」（実体依存の判定を engine に置く）で、
一意性を検査しない根拠は ADR-0014「entries は target キーの attrset」。
