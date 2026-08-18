---
id: "REQ-7cee95dd-bc5a-4e86-bebc-6080ef78fe26"
type: requirement
name: "実 dir の target は全 leaf が除去可能なときだけ全体を配置前除去して symlink 化する"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  When the target itself is a real directory, the engine SHALL classify every leaf below
  it at any depth. Only when every leaf is either a recorded and stale symlink — recorded
  by the entry's own previous generation, matching that record on disk, and absent from
  the next generation — or an empty sub-directory of any origin, SHALL the whole target be
  removed before placement (PreRemove: leaves by unlink, directories by rmdir from child
  to parent) and the symlink newly placed; this removal SHALL NOT be reported as a warning,
  being an intended migration, and SHALL instead fall under the ordinary output discipline
  of the placement report. If even one other leaf exists — a real file or directory with
  content, a foreign symlink, or a self-contradictory entry that also remains in the next
  generation — the command SHALL stop with an error for the target as a whole and SHALL
  NOT remove it partially.
specification_ja: |
  target 自身が実 dir のとき、engine は配下（任意深さ）の全 leaf を判定しなければならない。
  全 leaf が「recorded かつ stale な symlink（当該 entry 自身の前世代が記録・on-disk 一致・
  次世代に無い）」または「空の sub dir（由来を問わない）」であるときに限り、target 全体を
  配置前に除去（PreRemove: leaf は unlink・dir は子から親へ rmdir）してから symlink を新規
  配置しなければならない。この除去は意図された移行であり warning にしてはならず、配置レポート
  の通常の出力規律に従わせなければならない。それ以外の leaf が 1 つでもある（中身のある実
  file / dir・foreign symlink・
  次世代にも残る自己矛盾）ときは target 全体をエラーで停止しなければならず、部分除去を
  してはならない。
---
# REQ-7cee95dd-bc5a-4e86-bebc-6080ef78fe26: 実 dir の target は全 leaf が除去可能なときだけ全体を配置前除去して symlink 化する

## 仕様

```
target 自身が実 dir のとき、配下（任意深さ）の全 leaf を判定する:
  - 全 leaf が「recorded ∧ stale な symlink（自身の前世代が記録・on-disk 一致・次世代に無い）」
    または「空の sub dir（由来を問わない）」なら → target 全体を配置前に除去（PreRemove: leaf は Unlink・
    dir は子から親へ Rmdir・silent・`-v` で可視）してから symlink を新規配置
  - 上記以外の leaf が 1 つでもある（中身のある実 file/dir・foreign symlink・次世代にも残る自己矛盾）
    → target 全体をエラーで停止（部分除去はしない）
```

> **上は原文の写しで、規範は frontmatter が正**。この migration が失敗して「エラーで停止」
> と判定された実 dir も `--backup` の退避対象に入る。退避対象を判定各段の結論
> （エラー停止 / copy foreign スキップ）で括る規範と、退避が配置手順のどの段に入るかは
> REQ-9b0046e0-8ddc-4c0b-940e-3fe6f36d0e98 の担当（退避そのものの契約は REQ-5dd5a4e9-6162-4fa5-b295-66844f5a4f3b）。

祖先 symlink の migration は REQ-c9ab91c1-f778-4f87-a2ea-c66d6b3c2575、method 変更 symlink→copy の migration は
REQ-2b48620a-abaa-43df-a106-954bbba3de56 の担当。除去を既定 silent とし `-v` で可視化するという出力規律そのもの
（→ ADR-0031）は REQ-8ef34101-8150-4124-92d5-94fabe6b5d90 / REQ-0a123b89-0399-4f76-b988-56a5f7e0becf の担当で、本 item は「この除去を warning に
せず配置レポート側で扱う」ことだけを規範とする。

## 出典

`docs/spec.md`「配置動作仕様」→「symlink モード」の手順 0.5 と、同節の箇条書き
「target に通常ファイルまたはディレクトリが存在する場合はエラーで停止（上書きしない）。
ただし実 dir は §0.5 の条件を満たせば例外的に PreRemove で除去して配置する」。

決定の実体は ADR-0047「配置前除去（PreRemove）の一般化」の D2。
