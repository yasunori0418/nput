---
id: "REQ-b4e4b65d-6e35-40c3-a00e-20c14043df6f"
type: requirement
name: "recopy の上書きは削除ではなく同一親内への rename 退避で行う"
specification: |
  When `apply --recopy` overwrites an existing copy target, it SHALL do so not by deleting
  it but by renaming it aside to a temporary name within the same parent directory, and
  SHALL then re-copy `<src>/<subpath>`. A rename involves metadata only, so the object set
  aside SHALL not be corrupted even under ENOSPC. When the target is absent, an ordinary
  place-once copy SHALL be performed instead. Where a later stage of the same apply run
  fails, the original content SHALL be restored from the object set aside; where the run
  finally succeeds, that object SHALL be deleted.
specification_ja: |
  `apply --recopy` が既存の copy target を上書きするとき、削除ではなく同一親ディレクトリ内の
  一時名への rename 退避で行い、そのうえで `<src>/<subpath>` を再コピーしなければならない。
  rename はメタデータ操作のみであり、ENOSPC 下でも退避物が壊れないためである。target が不在の
  ときは通常の place-once コピーを行う。同一 apply run 内で後続の段が失敗した場合は退避物から
  元の内容を復元し、run が最終的に成功した場合は退避物を削除する。
---
# REQ-b4e4b65d: recopy の上書きは削除ではなく同一親内への rename 退避で行う

## 仕様

```
config 内の各 copy entry について:
  target が存在 → 同一親ディレクトリ内の一時名へ rename 退避してから <src>/<subpath> を再コピー。
                  この apply run が最終的に成功すれば退避物を削除、途中失敗して巻き戻すときは rename back
  target が不在 → 通常の place-once コピー
```

上書きは削除ではなく **rename 退避**で行う（rename はメタデータ操作のみで ENOSPC 下でも
退避物が壊れない）。同一 apply run 内で後続の段が失敗した場合、退避物から元の内容が復元される。

> **上は原文の写しで、規範は frontmatter が正**。`--recopy` が全 copy target を無条件上書き
> すること・レポート表示・世代を増やさないことは REQ-7cc32a2b の担当で、本 item は上書きを
> どう実現するか（削除ではなく rename 退避）だけを規定する。退避物の巻き戻し / 削除を担う
> undo ジャーナルそのものは REQ-5e75aabc、コピー時の mode 規則と symlink 複製は REQ-84e3c717 /
> REQ-0bd55dfc、place-once は REQ-d2277c7a の担当。

## 出典

`docs/spec.md`「配置動作仕様」→「recopy（`apply --recopy`）・reset（`nput reset`）」節の
recopy コードブロックと、同節箇条書き第 3 項。

決定の実体は ADR-0044「apply 途中失敗の完全巻き戻し — インメモリ undo ジャーナル」（rename
退避による上書きと復元）で、`--recopy` の追加自体は ADR-0020 が決めている。
