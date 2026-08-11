---
id: "CASE-4179dcb2-f199-4469-ae87-0927ece06e65"
type: test_case
name: "manifest_test.go — v1 の復元と版・未知フィールドの拒否"
target: "internal/manifest/manifest_test.go"
covers:
  - "TC-172548ea-154a-4e22-a169-8252a43e3781"
---
# CASE-4179dcb2: manifest_test.go — v1 の復元と版・未知フィールドの拒否

## 対象

`internal/manifest/manifest_test.go`（対象は `internal/manifest/manifest.go` の読み取りと検証）

## 検証内容

- v1 の文書の全フィールド（配置元の種別・配置元・部分パス・配置先・配置方式・root の種別）が
  定義どおりに読み取れること
- 消費側が知らない版を拒否し、しかも判別可能な値を包んで返すこと。呼び出し側が「pin した
  flake と CLI の版がずれている」という案内を出せることが契約であり、テストは包まれた値を
  判別できることまで確かめる
- 未知のキーを黙って無視せず拒否すること
- 評価時に絶対パスが決まる root の種別がパスを持つこと
- 文書が存在しないときに失敗すること

一時ディレクトリへ文書を直接書き出す fixture 方式で、nix も外部プロセスも起動しない。

> 現状では、版が下限を下回る場合の拒否と、root の種別が空である場合の拒否は覆われていない。
> 実行時に解決する種別（project / home / system）がパスを持たないことも、読み取れること自体は
> 確かめているがパスの不在まではアサートしていない。
