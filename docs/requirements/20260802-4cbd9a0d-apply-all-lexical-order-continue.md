---
id: "REQ-4cbd9a0d-9f94-4747-8881-56020dc6d5af"
type: requirement
derives_from:
  - "UC-1c280dce-7c72-44c0-95ea-d06344f62a47"
name: "apply --all は辞書順に適用し部分失敗でも続行して最後に集約する"
specification: |
  `nput apply --all` SHALL apply all of `nput.*` in lexical (key-sorted, deterministic)
  order, and SHALL continue with the remaining configs even when some of them fail,
  because each config is an independent atomic profile. It SHALL display an aggregated
  success / failure summary at the end, and SHALL exit non-zero when at least one config
  failed. `--all` itself SHALL NOT be atomic as a whole, because project mode does not
  expose rollback and the semantics would break.
specification_ja: |
  `nput apply --all` は `nput.*` を辞書順（キーソート・決定的）に適用しなければならず、
  一部が失敗しても残りを続行しなければならない（各 config は独立 profile で atomic なため）。
  最後に成功 / 失敗を集約表示し、1 つでも失敗なら非ゼロ終了しなければならない。
  `--all` 自体は全体 atomic にしてはならない（project mode は rollback 非公開で
  意味論が崩れるため）。
---
# REQ-4cbd9a0d: apply --all は辞書順に適用し部分失敗でも続行して最後に集約する

## 仕様

`apply --all` は **entrypoint の `nput.*` を辞書順（キーソート・決定的）に適用**し、
**一部が失敗しても残りを続行**する（各 config は独立 profile で atomic なため）。
Nix attrset は定義順を保持せず `builtins.attrNames` が辞書順を返すため適用順は辞書順に
なるが、各 config が独立 atomic なので順序は結果に影響しない（表示・失敗集約のための
決定的順序）。最後に成功 / 失敗を集約表示し、**1 つでも失敗なら非ゼロ終了**する。
`--all` 自体は全体 atomic にしない（project mode は rollback 非公開で意味論が崩れるため）。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `apply --all` の箇条書き。

決定の実体は ADR-0016「`--all` 適用順」（辞書順・決定的）と ADR-0013（`--all` を全体
atomic にしない・部分失敗でも続行し集約表示）。
