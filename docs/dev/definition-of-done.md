# 完成の定義

- 改訂日: 2026-07-26

プロジェクトに 1 つの恒久ドキュメント。配置は `docs/dev/definition-of-done.md` 固定で、
機能単位の作業ディレクトリ（`docs/dev/<対象>/`）の削除対象に含めない。

項目は**機械判定・人判定を合わせて最大 5 件**（機械パース契約の一部）。
項目が多いほど各項目のクリアが重くなり完成が遠のくため、増やしたくなったら統合するか、
優先度の低い項目を外す。

## 機械判定

各項目に判定手段を宣言する。dev-pipeline 等の決定論スクリプトがこのテーブルを
パースして項目別判定を報告するため、列構成・種別・条件の語彙は変えない。

| ID | 項目 | 種別 | 対象 | 条件 |
|---|---|---|---|---|
| DOD-01 | E2E テストが通っていること | ci-check | e2e | success |
| DOD-02 | nix flake check が通っていること | ci-check | flake-check (ubuntu-latest, x86_64-linux) | success |

DOD-01 / DOD-02 は ruleset `main branch protection` の required status check でもあり、
未達ならそもそも `main` へマージできない（→ ADR-0030）。DOD-02 は matrix の代表 1 件のみを
挙げているが、ruleset 側では残る 2 プラットフォーム
（`flake-check (ubuntu-24.04-arm, aarch64-linux)` / `flake-check (macos-latest, aarch64-darwin)`）
も必須のまま強制される。job 名 / matrix を変えると check 名が変わるため、
ruleset と本テーブルの両方に追従が要る。

`go-coverage` は計測・表示のみで閾値ゲートを持たない設計（マージ順依存の回避）であり、
required check にも含めていないため、完成の定義にも入れない。

### 種別と条件の語彙（機械パース契約）

- `ci-check`: 対象 = GitHub Actions の check 名。条件 = `success`（成功していること）。
- `artifact`: 対象 = リポジトリ相対パス。対象パスには `{target}` プレースホルダを使える
  （判定時に対象機能名へ展開される。例: `docs/test/{target}/test-summary-report.md`）。
  条件は次のいずれか:
  - `exists`: ファイルが存在する。
  - `contains:<文字列>`: ファイルが `<文字列>` を含む。

## 人判定

機械化できない項目のみを列挙する。人間が最終確認するチェックリストになる唯一の部分。

- [ ] DOD-03: 設計判断を伴う変更であれば `docs/adr/` に ADR を追加し、既存 ADR と矛盾していない
      （連番ファイル名のため、どの ADR が当該変更の分かを機械が特定できない）
- [ ] DOD-04: `docs/design.md` / `docs/spec.md` / `docs/concept.md` が実装の変更に追従している
      （追従の要否と十分性が意味判断になる）
