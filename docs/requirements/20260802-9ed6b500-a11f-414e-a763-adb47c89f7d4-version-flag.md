---
id: "REQ-9ed6b500-a11f-414e-a763-adb47c89f7d4"
type: requirement
name: "--version は埋め込みバージョンを cobra 既定書式で表示して終了し短縮形を持たない"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  The `--version` flag SHALL display the embedded version in the cobra default format
  (`nput version X.Y.Z`) and then terminate. It SHALL NOT have a short form, because `-v`
  is already assigned to `--verbose`.
specification_ja: |
  `--version` は埋め込みバージョンを cobra 既定書式（`nput version X.Y.Z`）で表示して
  終了しなければならない。`-v` は `--verbose` に割当済みのため短縮形を持ってはならない。
---
# REQ-9ed6b500: --version は埋め込みバージョンを cobra 既定書式で表示して終了し短縮形を持たない

## 仕様

```bash
--version           # 埋め込みバージョンを表示して終了（cobra 既定書式 `nput version X.Y.Z`。
                    # -v は --verbose に割当済みのため短縮形なし）
```

`--version` がエンベロープを出さないこと（`--json` との関係）は REQ-5c2e64c3 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」のグローバルフラグ表 `--version`。

決定の実体は ADR-0042「リリースを bump PR 起点で自動化する」で、埋め込みバージョンの
供給元（VERSION ファイル由来の ldflags 埋め込み `main.version`）を定めている。
