---
id: "DSG-92f54490-872a-42ac-bbd7-d06e9ee381c6"
type: design
name: "legacy entrypoint の分岐は attr path 組み立てに閉じ、nix 呼び出しヘルパを共通で再利用する"
satisfies:
  - "REQ-c890ce4a-6528-4ab3-ac86-23d7aebff7da"
  - "REQ-496b1a07-5b74-416b-9e5f-3952b4c03737"
  - "REQ-60c6b7ea-e936-4ce8-bd75-ad35e9c693b9"
---
# DSG-92f54490: legacy entrypoint の分岐は attr path 組み立てに閉じ、nix 呼び出しヘルパを共通で再利用する

## 設計

flake entrypoint と legacy entrypoint（`shell.nix` / `default.nix`）は、CLI が叩く
nix コマンドの形が異なる。

| entrypoint | build | rootKind eval |
|---|---|---|
| flake | `nix build <ep>#nput.<system>.<name>` | `nix eval <ep>#nput.<system>.<name>.rootKind` |
| legacy | `nix build -f <ep> nput.<name>` | `nix eval -f <ep> nput.<name>.rootKind` |

**CLI 内部でこの差を吸収するのは attr path（およびフラグ）の組み立て 1 箇所だけ**とし、
実際の nix 実行を担うヘルパ（`runNixCapture` / `runNixStream`）とエラー処理は
両経路で共通のものを再利用する。

経路ごとにコマンド実行から書き下ろすのではなく組み立てだけを分岐させる理由は次の 2 つ。

- **REQ-c890ce4a が「legacy でも CLI の attr path を分岐させない」**（`nput.<name>` の
  attr path 自体は passthru 形でもトップレベル attrset 形でも同一）ことを定めている。
  分岐が組み立て 1 箇所に閉じていれば、この同一性が構造として保たれる。実行系まで
  経路ごとに分かれると、片方だけ挙動が変わる余地が生まれる
- **REQ-60c6b7ea が定める実行フローの順序（eval 先行 → flock → build）を経路によらず
  1 つに保てる**。順序を守るのはフローを組む側の責務で、そこが共通なら legacy 経路にだけ
  順序違反が入り込むことがない。REQ-496b1a07 が求める「形ごとの attr path で build する」も、
  形の違いが組み立てに局所化されている限り満たされる

## 出典

`docs/design.md`「flake.nix outputs 設計」末尾（L159）。
