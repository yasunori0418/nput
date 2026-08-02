---
id: "REQ-f9920c87-8551-4aa3-bf03-26fdf4191ed6"
type: requirement
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
name: "nix experimental-features は前提条件とし、CLI は自動付与せず案内エラーで停止する"
specification: |
  The CLI uses `nix eval` / `nix build` (the new CLI) internally, and therefore SHALL
  require that the user environment has already enabled
  `experimental-features = nix-command` (and additionally `flakes` for flake entrypoints).
  The CLI SHALL NOT add `--extra-experimental-features` automatically, so that it does not
  silently override nix.conf / Determinate Nix / organizational policy settings. When nix
  returns a feature-not-enabled error, the CLI SHALL stop with an error that states the
  prerequisite and how to enable it, and SHALL NOT swallow the raw nix error.
specification_ja: |
  CLI は内部で `nix eval` / `nix build`（新 CLI）を使うため、ユーザー環境で
  `experimental-features = nix-command`（flake entrypoint はさらに `flakes`）が
  有効化済みであることを前提としなければならない。CLI は
  `--extra-experimental-features` を自動付与してはならない（nix.conf / Determinate Nix /
  組織ポリシーの設定を黙って上書きしないため）。未有効で nix が機能未有効エラーを
  返したときは、前提条件と有効化方法を案内するエラーで停止しなければならず、生の nix
  エラーを握り潰してはならない。
---
# REQ-f9920c87: nix experimental-features は前提条件とし、CLI は自動付与せず案内エラーで停止する

## 仕様

CLI は内部で `nix eval` / `nix build`（新 CLI）を使うため、ユーザー環境で
`experimental-features = nix-command`（flake entrypoint はさらに `flakes`）が
**有効化済みであることを前提**とする。

CLI は `--extra-experimental-features` を自動付与しない（nix.conf / Determinate Nix /
組織ポリシーの設定を黙って上書きしないため）。未有効で nix が機能未有効エラーを
返したら**前提条件と有効化方法を案内する分かりやすいエラーで停止**する（生の nix
エラーを握り潰さない）。

## 出典

`docs/spec.md`「CLI 仕様（一次 UX）」の blockquote「前提条件: nix experimental-features」。

決定の実体は ADR-0025「nix experimental-features 前提」。
