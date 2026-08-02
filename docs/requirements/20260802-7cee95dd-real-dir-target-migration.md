---
id: "REQ-7cee95dd-bc5a-4e86-bebc-6080ef78fe26"
type: requirement
name: "実 dir の target は全 leaf が除去可能なときだけ全体を配置前除去して symlink 化する"
specification: |
  When the target itself is a real directory, the engine SHALL classify every leaf below
  it at any depth. Only when every leaf is either a recorded and stale symlink — recorded
  by the entry's own previous generation, matching that record on disk, and absent from
  the next generation — or an empty sub-directory of any origin, SHALL the whole target be
  removed before placement (PreRemove: leaves by unlink, directories by rmdir from child
  to parent) and the symlink newly placed; this removal SHALL be silent by default and
  SHALL be made visible under `-v`. If even one other leaf exists — a real file or
  directory with content, a foreign symlink, or a self-contradictory entry that also
  remains in the next generation — the command SHALL stop with an error for the target as
  a whole and SHALL NOT remove it partially.
specification_ja: |
  target 自身が実 dir のとき、engine は配下（任意深さ）の全 leaf を判定しなければならない。
  全 leaf が「recorded かつ stale な symlink（当該 entry 自身の前世代が記録・on-disk 一致・
  次世代に無い）」または「空の sub dir（由来を問わない）」であるときに限り、target 全体を
  配置前に除去（PreRemove: leaf は unlink・dir は子から親へ rmdir）してから symlink を新規
  配置する。この除去は既定で silent とし、`-v` で可視化する。それ以外の leaf が 1 つでも
  ある（中身のある実 file / dir・foreign symlink・次世代にも残る自己矛盾）ときは target 全体
  をエラーで停止しなければならず、部分除去をしてはならない。
---
# REQ-7cee95dd: 実 dir の target は全 leaf が除去可能なときだけ全体を配置前除去して symlink 化する

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
> REQ-9b0046e0 の担当（退避そのものの契約は REQ-5dd5a4e9）。

祖先 symlink の migration は REQ-c9ab91c1、method 変更 symlink→copy の migration は
REQ-2b48620a の担当。

## 出典

`docs/spec.md`「配置動作仕様」→「symlink モード」の手順 0.5 と、同節の箇条書き
「target に通常ファイルまたはディレクトリが存在する場合はエラーで停止（上書きしない）。
ただし実 dir は §0.5 の条件を満たせば例外的に PreRemove で除去して配置する」。

決定の実体は ADR-0047「配置前除去（PreRemove）の一般化」の D2。
