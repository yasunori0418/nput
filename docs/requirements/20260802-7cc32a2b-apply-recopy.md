---
id: "REQ-7cc32a2b-eee4-4a29-8dc1-a1dc23e7a065"
type: requirement
name: "apply --recopy は config 内の全 copy target を src から無条件に上書き再コピーする"
specification: |
  In addition to an ordinary apply, `nput apply <name> --recopy` SHALL unconditionally
  overwrite and re-copy every copy target in the config from its current `src` / `subpath`.
  Because copies live outside generations and are not hash-tracked, no difference check
  SHALL be performed and the overwrite SHALL be unconditional. The overwritten targets
  SHALL be shown in the placement report, and no confirmation SHALL be requested because
  the flag is itself opt-in. Local edits to copies SHALL therefore be discarded and
  reverted to the content of `src` (the intent being to follow upstream). The generation
  commit behaviour of the symlink part SHALL be unchanged, and copies SHALL NOT add a
  generation.
specification_ja: |
  `nput apply <name> --recopy` は通常 apply に加えて、config 内の全 copy target を現在の
  `src` / `subpath` から無条件に上書き再コピーしなければならない。copy は世代外で hash 追跡
  しないため差分判定はせず無条件とする。上書きした target はレポート表示し、フラグ自体が
  opt-in なので確認は出さない。したがってローカルの copy 編集は破棄され src の内容に戻る
  （upstream 追従の意図）。symlink 部の世代コミット挙動は不変で、copy は世代を増やさない。
---
# REQ-7cc32a2b: apply --recopy は config 内の全 copy target を src から無条件に上書き再コピーする

## 仕様

```bash
nput apply <name> --recopy     # 通常 apply に加え config 内の全 copy target を src から無条件上書き再コピー
```

`apply <name> --recopy` は通常 apply に加え **config 内の全 copy target を現 `src`/`subpath`
から無条件に上書き再コピー**する。copy は世代外で hash 追跡しないため差分判定はせず
無条件。上書きした target をレポート表示し、フラグ自体が opt-in なので確認は出さない。
**ローカルの copy 編集は破棄され src 内容に戻る**（= upstream 追従の意図）。symlink 部の
世代コミット挙動は不変（copy は世代を増やさない）。

`--all` との合成可否は REQ-687e225f の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `apply <name> --recopy` の箇条書き。

決定の実体は ADR-0020「copy の明示上書き（`apply --recopy`）と配置物のリセット
（`nput reset`）を追加する」。
