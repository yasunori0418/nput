---
id: "REQ-c847d1af-a437-46bb-bd64-42083810d034"
type: requirement
name: "HM の activation は configs を辞書順に走査して profile ごとに engine を起動し、部分失敗を最後に集約する"
derives_from:
  - "UC-d39c1994-f9a5-4860-80ba-f6e584adaf14"
  - "UC-1c280dce-7c72-44c0-95ea-d06344f62a47"
specification: |
  The `home.activation.nput` of the home-manager module SHALL walk `nput.configs` and kick
  the engine once per config, so that each profile takes its own engine invocation. The
  order of execution SHALL be the lexical order of `<name>`, so that the run is
  deterministic and its log reproducible. A failure of one profile SHALL NOT stop the
  profiles that follow; every failure SHALL instead be aggregated and the activation SHALL
  be failed at the end, as `apply --all` does on a partial failure. No CLI extension
  passing several manifests to a single engine invocation SHALL be introduced, so that the
  unit of atomicity — one profile — is kept at the CLI boundary as well. The contract of a
  single kick is stated by REQ-8085f194, and the correspondence of one config to one
  manifest and one profile by REQ-c6891aeb; neither is restated here. The posture of
  continuing past a partial failure and aggregating it is common with the `apply --all` of
  the standalone path stated by REQ-4cbd9a0d.
specification_ja: |
  home-manager モジュールの `home.activation.nput` は `nput.configs` を走査し、config
  ごとに 1 回ずつ engine を起動しなければならない（profile ごとに独立した engine 起動を
  取るため）。実行順は `<name>` の辞書順としなければならない（実行を決定的にしログを
  再現可能にするため）。1 profile の失敗は後続の profile を止めてはならず、失敗は最後に
  集約して activation を失敗させなければならない（`apply --all` の部分失敗と同じ姿勢）。
  複数 manifest を 1 回の engine 起動へ渡す CLI 拡張を導入してはならない（atomic 性の
  単位である profile を CLI 界面にも保つため）。1 起動あたりの契約は REQ-8085f194、
  1 config が 1 manifest = 1 profile に対応することは REQ-c6891aeb の担当で、いずれも
  本 item では規定しない。部分失敗で続行し集約する姿勢は、standalone 経路の
  `apply --all` を定める REQ-4cbd9a0d と共通である。
---
# REQ-c847d1af: HM の activation は configs を辞書順に走査して profile ごとに engine を起動し、部分失敗を最後に集約する

## 仕様

- `home.activation.nput` は `configs` を走査し、**profile ごとに 1 回ずつ**
  `nput apply --manifest <link-farm-N> <name>` を実行する
- 各起動は profileDir 単位の flock・前世代 diff・保守的 stale 除去・`nix-env --set` が
  **profile ごとに独立して**走る
- 実行順は `<name>` の**辞書順**で決定的にする（Nix の attrset 走査順と一致・ログの
  再現性のため）
- **1 profile の失敗は後続 profile を止めず、最後に集約して activation を失敗させる**
- 複数 manifest を 1 回の engine 起動に渡す CLI 拡張は行わない（atomic 性の単位 =
  profile を CLI 界面にも保つ）

> **本 item の出典は ADR-0035 §3 であり、`docs/spec.md` に対応記述は無い**。原文が
> ADR-0035 に未追従で、複数 config があるとき activation が engine を何回 kick するかを
> 述べていないため、#209 の分割では item 化されず REQ-8085f194 / REQ-c6891aeb の注記に
> 申し送りとして残っていた（epic #203 / issue #228 で回収）。
>
> **他 item との担当分界**: 1 起動あたりの activation 契約（`entryAfter ["writeBoundary"]`・
> `apply --manifest` でビルド済み link-farm を渡すこと・activation が `nix eval` /
> `build` を行わないこと・engine error が非 0 終了で switch を止めること）は
> REQ-8085f194。1 config = 1 profile = 1 manifest の対応と `<name>` 次元そのものは
> REQ-c6891aeb。`nput.configs` オプションの定義は REQ-fc1c7ce6。`apply --manifest` と
> 位置引数 `name` の直交・両立は REQ-dec58330 / REQ-c2d44626。profileDir 単位の flock は
> REQ-1c1526b1、profileDir のキーは REQ-d5a2e289。世代が nput 自前 profile に積まれる
> ことは REQ-1be4d678、module 経路で rollback を host へ一本化することは REQ-844ee375。
> 辞書順・部分失敗続行・最後に集約という姿勢を standalone の `apply --all` について
> 定めるのは REQ-4cbd9a0d。本 item は同じ姿勢を HM activation の configs ループについて
> 独立に規定する。文面は重なるが、適用先（CLI の `--all` か activation の configs ループか）
> が別経路であり、片方の規範が他方を含意しないため、規範としては別立てになる。
>
> **本文箇条書き第 2 項に対応する規範文を持たない理由**: 「各起動は profileDir 単位の
> flock・前世代 diff・保守的 stale 除去・`nix-env --set` が profile ごとに独立して走る」は、
> 本 item の「config ごとに 1 回 engine を起動する」と、profileDir 単位の flock を定める
> REQ-1c1526b1・profileDir のキーを定める REQ-d5a2e289・世代機構を定める REQ-1be4d678 の
> 合成から従うため、規範として再掲しない。

## 出典

ADR-0035「HM モジュールに `nput.configs.<name>` を導入し複数 profile（役割分離）を
可能にする」§3「activation は profile ごとに独立した engine 起動」。

`docs/spec.md` には対応記述が無いため、原文の写しは持たない（規範は frontmatter が正で、
上の箇条書きは ADR 本文の要約）。部分失敗を集約する姿勢の淵源は ADR-0018、rollback を
host へ一本化する点は ADR-0002 / ADR-0024、`apply --manifest` と位置引数の両立は
ADR-0026 が定めている。
