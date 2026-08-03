---
id: "REQ-67095391-eab2-45d2-b75b-b428d481bcc2"
type: requirement
name: "flake は pure eval で flake.lock が固定し legacy は impure を許容しユーザー責任とする"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  A `flake.nix` entrypoint SHALL be evaluated purely (root resolution happens at engine
  runtime, so the evaluation stays pure) and its reproducibility SHALL be fixed by
  flake.lock. A `shell.nix` / `default.nix` entrypoint SHALL be allowed to evaluate
  impurely (depending on NIX_PATH / channels), and its reproducibility SHALL be the user's
  responsibility; pinning nixpkgs, including the nput lib, with npins / fetchTarball /
  flake-compat or similar SHOULD be recommended.
specification_ja: |
  `flake.nix` entrypoint は pure eval とし（root 解決はエンジン実行時なので eval は
  pure のまま）、再現性は flake.lock で固定するものとしなければならない。
  `shell.nix` / `default.nix` entrypoint は impure eval（NIX_PATH / channels 依存）を
  許容しなければならず、その再現性はユーザー責任としなければならない。nput lib を含め
  nixpkgs を npins / fetchTarball / flake-compat 等で固定することを推奨すべきである。
---
# REQ-67095391: flake は pure eval で flake.lock が固定し legacy は impure を許容しユーザー責任とする

## 仕様

| entrypoint | eval | 再現性 |
|---|---|---|
| `flake.nix` | pure（root 解決はエンジン実行時なので eval は pure のまま）| flake.lock で固定 |
| `shell.nix` / `default.nix` | impure（NIX_PATH / channels 依存）を許容 | **ユーザー責任**。nput lib を含め nixpkgs を npins / fetchTarball / flake-compat 等で固定することを推奨 |

legacy entrypoint で `src` が自動で store 化されないことは REQ-da253cab の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「再現性スタンス」の表。

決定の実体は ADR-0007 §5「shell.nix / default.nix は best-effort（再現性はユーザー責任）」。
