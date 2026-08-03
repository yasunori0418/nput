---
id: "REQ-8085f194-c903-4ecb-abd8-c719fe7b3292"
type: requirement
name: "home-manager モジュールの engine kick 1 回は activation からビルド済み link-farm を渡し、失敗で switch を止める"
derives_from:
  - "UC-d39c1994-f9a5-4860-80ba-f6e584adaf14"
specification: |
  The home-manager module SHALL kick the engine from `home.activation.nput`
  (`entryAfter ["writeBoundary"]`) with `nput apply --manifest <link-farm>`, the link-farm
  being built by `mkManifest` from the entries of a config at module evaluation time and
  its store path passed from the activation script. The activation SHALL NOT perform
  `nix eval` or `nix build`, not being on the entrypoint path. The flock SHALL be taken
  blocking, the placement report SHALL be visible, and an engine error such as a conflict
  SHALL stop the switch by exiting non-zero, matching the clobber error of `home.file`
  under a declarative switch. Because the pinned nput CLI (`packages.nput`) and
  `mkManifest` come of the same flake input, a schemaVersion skew SHALL NOT arise
  structurally. This states the contract of a single kick; the correspondence of one
  config to one manifest and one profile is stated by REQ-c6891aeb, and the definition of
  the option supplying its entries by REQ-fc1c7ce6, neither being restated here. How many
  times the activation kicks the engine when several configs are present, in what order
  and how a partial failure is aggregated (ADR-0035 §3), is stated by REQ-c847d1af and is
  likewise not restated here.
specification_ja: |
  home-manager モジュールは `home.activation.nput`（`entryAfter ["writeBoundary"]`）から
  `nput apply --manifest <link-farm>` で engine を kick しなければならない。link-farm は
  モジュール評価時に config の entries から `mkManifest` でビルドし、その store パスを
  activation script から渡す。activation は entrypoint 経路ではないため `nix eval` /
  `nix build` を行ってはならない。flock は blocking で取り、配置レポートを可視とし、
  engine error（conflict 等）は非 0 終了で switch を止めなければならない（宣言的 switch・
  `home.file` の clobber エラーと同型）。pin 版 nput CLI（`packages.nput`）と `mkManifest` が
  同一 flake input 由来のため、schemaVersion skew は構造的に起こらないものとする。本 item が
  規定するのは 1 起動あたりの契約であり、1 config が 1 manifest = 1 profile に対応することは
  REQ-c6891aeb、その entries を供給するオプションの定義は REQ-fc1c7ce6 の担当で、いずれも
  本 item では規定しない。複数 config があるとき activation が engine を何回 kick するか・
  その順序・部分失敗の集約（ADR-0035 §3）は REQ-c847d1af の担当で、これも本 item では
  規定しない。
---
# REQ-8085f194: home-manager モジュールの engine kick 1 回は activation からビルド済み link-farm を渡し、失敗で switch を止める

## 仕様

| method | `src` | 動作 |
|---|---|---|
| `"symlink"` | path / set | engine が store link をネイティブ symlink |
| `"symlink"` | marker | engine が out-of-store symlink をネイティブ作成（HM の mkOutOfStoreSymlink には委譲しない）|
| `"copy"` | path / set | engine が place-once ネイティブコピー |

- `home.activation.nput`（`entryAfter ["writeBoundary"]`）から engine を起動する。配置ロジックは
  HM に依存しない。root は `homeRoot` を pin
- engine kick は **`nput apply --manifest <link-farm>`**。モジュール評価時に `nput.entries` から
  `mkManifest` で link-farm をビルドし、その store パスを activation script から渡す。
  activation は `nix eval` / `build` を行わない（entrypoint 経路ではない）
- **blocking flock・配置レポート可視・engine error（conflict 等）は非 0 終了で switch を止める**
  （宣言的 switch・`home.file` の clobber エラーと同型）
- pin 版 nput CLI（`packages.nput`）と `mkManifest` が同一 flake input 由来のため
  schemaVersion skew は構造的に起きない
- 世代は nput 自前 profile（内部機構・前世代マニフェスト + stale 追跡）に乗る。ユーザー向け
  rollback は host（`home-manager --rollback`）に一本化（`nput rollback` 非公開）

> **上は原文の写しで、規範は frontmatter が正**。原文が参照する次の規範は本 item の担当では
> ない。
>
> - method と `src` の組み合わせが選ぶ配置方法 → REQ-77689c68。表が HM でも engine が扱う
>   （HM の `mkOutOfStoreSymlink` へ委譲しない）と述べる点は REQ-eb363122 / REQ-c1b3ca5f
> - `apply --manifest` の契約そのもの（entrypoint 発見・eval・build を行わないこと・
>   `-f` / `--all` との併用エラー）→ REQ-dec58330
> - モジュールが root を pin し利用者が再指定しないこと → REQ-fc1c7ce6。`homeRoot` の
>   層ごとの解決 → REQ-8d965ca2
> - manifest を供給するオプションの定義（canonical は `nput.configs.<name>.entries`・
>   `nput.entries` はその糖衣）→ REQ-fc1c7ce6、1 config が 1 manifest = 1 profile に対応する
>   ことと profile 粒度 → REQ-c6891aeb。本 item の写しにある `nput.entries` は原文逐語であり、
>   規範文では供給元を「config の entries」と述べて ADR-0035 と衝突しない形にしている。
>   activation が profile ごとに engine を kick する規律（ADR-0035 §3・辞書順・部分失敗の
>   集約）は原文に対応記述が無いため本 item の写しには現れないが、規範は REQ-c847d1af が
>   持つ
> - flock を既定 blocking で取ること → REQ-1c1526b1。レポートと warning を stderr へ出す
>   ストリーム規律 → REQ-fea038de。終了コードの体系 → REQ-2c5a10d8
> - 世代が nput 自前 profile に乗ること → REQ-1be4d678。profile 名の次元 → REQ-c6891aeb、
>   profileDir のキー → REQ-d5a2e289。rollback を host へ一本化すること → REQ-844ee375
> - `manifest.json` が唯一の安定契約で `schemaVersion` が 1 に固定であること →
>   REQ-79ce0a09 / REQ-250d936c。本 item はそれを前提に「module 経路では skew が構造的に
>   起きない」ことだけを規定する
>
> 原文が世代の箇条書きに併記する「MVP は固定名 `default` の単一 profile
> （`<state>/nix/profiles/nput/default`）」は、**ADR-0035 が `nput.configs.<name>` の実装を
> 決定済み**で反転しているため写しから落とした（REQ-c6891aeb / REQ-d5a2e289 と同じ扱い）。
> `docs/spec.md` の追従は本 item の担当範囲外。

## 出典

`docs/spec.md`「モジュール別動作仕様」→「home-manager モジュール」節の表と箇条書き。

決定の実体は ADR-0026「モジュール activation の engine kick は `apply --manifest`
（ビルド済み link-farm 直接適用）で行う」。モジュールを配線に徹させることは ADR-0003 が
定めている。
