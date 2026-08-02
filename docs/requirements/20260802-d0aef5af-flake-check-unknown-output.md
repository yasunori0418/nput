---
id: "REQ-d0aef5af-e922-400b-b250-ca38719c480b"
type: requirement
name: "nput カスタム output は nix flake check の unknown 警告を許容し主検証は nix build で行う"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  The consumer-side `outputs.nput.<system>.<name>` (a dedicated namespace that avoids
  polluting `packages`) causes `nix flake check` to emit
  `warning: unknown flake output 'nput'` while exiting 0; this SHALL be accepted as
  expected and harmless, and SHALL NOT break CI. `flake check` inspects the evaluation
  soundness of the attrset directly under `nput`, but SHALL neither build nor evaluate the
  `.<system>.<name>` derivations beneath it, so nothing is built by mistake. Transposing
  to `flake.nput.<system>` with the flake-parts module SHALL NOT remove the warning, since
  nix itself hardcodes the known outputs, and renaming the output SHALL NOT remove it
  either. The primary verification of a `nput` artifact SHALL therefore be
  `nix build .#nput.<system>.<name>`.
specification_ja: |
  consumer 側の `outputs.nput.<system>.<name>`（`packages` 汚染を避ける専用 namespace）は
  `nix flake check` で `warning: unknown flake output 'nput'` を出すが exit 0 になる。
  これは想定内・無害として許容し、CI を壊してはならない。`flake check` は `nput` 直下
  attrset の eval 健全性は検査するが、配下の `.<system>.<name>` derivation は build も
  eval もしない（誤 build しない）。flake-parts module で `flake.nput.<system>` へ
  transpose しても警告は消えない（nix 本体が known-output を hardcode するため）。
  output 名を変えても消えない。`nput` 成果物の主検証は
  `nix build .#nput.<system>.<name>` で行わなければならない。
---
# REQ-d0aef5af: nput カスタム output は nix flake check の unknown 警告を許容し主検証は nix build で行う

## 仕様

**`nix flake check` と `nput` カスタム output**: consumer の
`outputs.nput.<system>.<name>`（`packages` 汚染回避の専用 namespace）は `nix flake check` で
**`warning: unknown flake output 'nput'` を出すが exit 0**（CI を壊さない・想定内・無害）。
`flake check` は `nput` 直下 attrset の eval 健全性は検査するが、**配下の
`.<system>.<name>` derivation は build も eval もしない**（誤 build しない）。
**flake-parts module で `flake.nput.<system>` へ transpose しても警告は消えない**
（nix 本体が known-output を hardcode するため）。output 名を変えても消えない
（`lib` 以外は unknown 警告）。`nput` 成果物の主検証は `nix build .#nput.<system>.<name>`
で行う。将来 upstream の flake-schemas（PR #8892）がマージされたら `schemas.nput` で
消す余地を残す。

専用 namespace を使う判断そのものは REQ-205d744d の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「再現性スタンス」節の blockquote「`nix flake check` と `nput`
カスタム output」。

決定の実体は ADR-0015「flake check」（unknown 警告は想定内・主検証は `nix build`）と、
transpose しても警告が消えないことを確認した ADR-0029。
