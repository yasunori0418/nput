---
id: "REQ-622787dc-4512-4ce9-9c7d-7b32bbb70557"
type: requirement
name: "symlink 配置は親 dir を作り配置元/subpath を指すリンクを張り、foreign symlink は警告して後勝ちする"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  To place a symlink the engine SHALL first create the parent directory of the target
  (equivalent to `mkdir -p`), then handle an existing symlink at the target: one recorded
  by the entry's own previous-generation manifest SHALL be replaced without a warning,
  falling under the ordinary output discipline of the placement report, whereas an
  unrecorded (foreign) symlink — belonging to another nput profile, another tool, or
  placed by hand — SHALL be replaced as well but SHALL emit a warning, the last writer
  thereby winning. It SHALL then create a symlink at `<root>/<target>` pointing at
  `<placement source>/<subpath>`, the placement source being the Nix store path for a
  store link and the absolute path held by the marker for an out-of-store symlink. The
  processing SHALL be the same whether the subpath denotes a file or a directory. Two
  different configs (profiles) aiming at the same target SHALL be treated as something not
  meant to happen; the last writer SHALL be tolerated and the overwriting of a foreign
  symlink SHALL be made visible by the warning.
specification_ja: |
  symlink の配置にあたり engine は、まず target の親ディレクトリを作成し（`mkdir -p` 相当）、
  次に target の既存 symlink を扱わなければならない。当該 entry 自身の前世代 manifest が記録
  した symlink は warning を出さずに置き換え（配置レポートの通常の出力規律に従う）、記録の無い
  （foreign な）symlink——他 nput profile / 他ツール / 手動——も置き換えるが warning を出す
  （後勝ち）。そのうえで `<配置元>/<subpath>` を
  指す symlink を `<root>/<target>` に作成しなければならない。配置元は store link では Nix
  ストアパス、out-of-store symlink では marker の絶対パスでなければならない。subpath が
  ファイル・
  ディレクトリのいずれでも処理は同じでなければならない。別 config（別 profile）が同一
  target を狙うことは起きない前提として扱わなければならず、後勝ちを許容しつつ foreign
  symlink の上書きは warning で可視化しなければならない。
---
# REQ-622787dc-4512-4ce9-9c7d-7b32bbb70557: symlink 配置は親 dir を作り配置元/subpath を指すリンクを張り、foreign symlink は警告して後勝ちする

## 仕様

```
1. target の親ディレクトリを作成（mkdir -p 相当。緩和対象の祖先 symlink / 実 dir target は PreRemove 除去済み・foreign は 0 で弾き済み）
2. target が既存 symlink のとき:
   - 自身の前世代 manifest が記録した symlink → そのまま置き換える（silent）
   - 記録の無い symlink（foreign = 他 nput profile / 他ツール / 手動）→ warning を出して置き換える（後勝ち）
3. <配置元>/<subpath> を指す symlink を <root>/<target> に作成（os.Symlink）
   - store link:        配置元 = Nix ストアパス
   - out-of-store:      配置元 = marker の絶対パス
```

- subpath がファイル・ディレクトリどちらでも同じ処理
- 別 config（別 profile）が同一 target を狙うのは基本「衝突させない前提」。後勝ちを許容しつつ
  foreign symlink 上書きは warning で可視化する

> **上は原文の写しで、規範は frontmatter が正**。手順 0〜0.7（配置前除去・退避）は
> REQ-c9ab91c1-f778-4f87-a2ea-c66d6b3c2575 / REQ-7cee95dd-bc5a-4e86-bebc-6080ef78fe26 / REQ-2b48620a-abaa-43df-a106-954bbba3de56 / REQ-9b0046e0-8ddc-4c0b-940e-3fe6f36d0e98、張替えを unlink + symlink の
> 2 操作で行うことは REQ-61856da1-8883-401e-ad57-9f326b96d400 の担当。`shellHook` 再入で起きる cross-config の振動は
> REQ-fc1118b1-b0e8-4ddf-80f6-c70956651693 の担当。記録済み symlink の張替えを既定 silent とし `-v` で可視化するという
> 出力規律そのもの（→ ADR-0031）は REQ-8ef34101-8150-4124-92d5-94fabe6b5d90 / REQ-0a123b89-0399-4f76-b988-56a5f7e0becf の担当で、本 item は
> 「記録済みは warning にせず配置レポート側で扱い、foreign は warning を出す」という
> 対比だけを規範とする。

## 出典

`docs/spec.md`「配置動作仕様」→「symlink モード」の手順 1〜3 と、同節の箇条書き
第 3・4 項。

決定の実体は ADR-0015「実装前レビューで surfaced した残セマンティクス」の foreign symlink
warning と cross-config 衝突（後勝ち）。
