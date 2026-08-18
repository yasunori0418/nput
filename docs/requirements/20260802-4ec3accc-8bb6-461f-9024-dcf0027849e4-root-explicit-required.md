---
id: "REQ-4ec3accc-8bb6-461f-9024-dcf0027849e4"
type: requirement
name: "root は明示必須で暗黙デフォルトを持たない"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  `root` SHALL be REQUIRED and stated explicitly, and SHALL NOT have an implicit default
  such as `$HOME`. Omitting `root` SHALL be an error at Nix evaluation time.
specification_ja: |
  `root` は明示必須でなければならず、暗黙デフォルト（`$HOME` 等）を持ってはならない。
  `root` を省略した場合は Nix 評価時にエラーとしなければならない。
---
# REQ-4ec3accc-8bb6-461f-9024-dcf0027849e4: root は明示必須で暗黙デフォルトを持たない

## 仕様

`root` は**明示必須**で、暗黙デフォルトを持たない。

- `root` を省略すると Nix 評価時にエラーになる。
- 「省略したら `$HOME`」のような暗黙の既定値は置かない。
- 明示必須は `mkManifest` / CLI entrypoint の層に課す。モジュール（home-manager →
  `homeRoot` / devShell → `projectRoot`）は自分の性質で root を pin するため、
  モジュール利用者は root を再指定しない（→ ADR-0007 §2）。

> **原文「root の解決」節の導入文の残る規範の所在**: root が target を絶対パスへ変換する基準で
> あることは target を root 相対と定める REQ-0b0cd1e3-bfeb-45c1-978d-e2e11c568336、配置の実体を全層で engine が実行する
> ことは REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9 / REQ-6c4e174a-4d16-477a-96ff-17cb4eb5b564 の担当。
>
> 上の第 3 項が述べるモジュール側の root pin（HM → `homeRoot` / devShell → `projectRoot`・
> モジュール利用者は root を再指定しない）は **REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10 の担当**で、本 item の規範には
> 含めない。本 item が規定するのは `mkManifest` / CLI entrypoint の層で root が明示必須で
> あることに限る。

## 出典

`docs/spec.md`「lib API」→「`lib.mkManifest`」→「`root` の値」および
「`lib.projectRoot` / `lib.homeRoot` / `lib.systemRoot`」。「root の解決」節の導入文
（root を `mkManifest` の `root` 引数で明示必須に選ぶ）も同じ規範を述べており、独立 item を
立てず本 item に畳んだ。
