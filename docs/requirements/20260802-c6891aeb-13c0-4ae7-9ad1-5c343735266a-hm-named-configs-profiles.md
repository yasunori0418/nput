---
id: "REQ-c6891aeb-13c0-4ae7-9ad1-5c343735266a"
type: requirement
name: "HM モジュール経由でも名前つき config ごとに役割分離した独立 profile を取れる"
derives_from:
  - "UC-d39c1994-f9a5-4860-80ba-f6e584adaf14"
  - "UC-1c280dce-7c72-44c0-95ea-d06344f62a47"
specification: |
  A user of the home-manager module SHALL also be able to hold several independent
  profiles separated by role, each with its own generations, and role separation SHALL NOT
  be confined to the standalone CLI path. One `<name>` of `nput.configs` SHALL yield one
  manifest and one profile; the atomicity of "one config = one profile = one manifest"
  SHALL be preserved and no helper for composing several configs SHALL be provided.
  Introducing the `<name>` dimension SHALL be non-breaking: a config named `default` SHALL
  go on resolving to the same profile as before, and no change to the layout of the
  profile directory SHALL be entailed. How the profile directory is keyed is stated by
  REQ-d5a2e289, and the definition of the options themselves — that
  `nput.configs.<name>.entries` is the canonical form and the bare `nput.entries` a
  deprecated sugar onto `configs.default.entries` — by REQ-fc1c7ce6, being common to every
  module; neither is restated here.
specification_ja: |
  home-manager モジュールの利用者も、役割分離された複数の独立 profile（それぞれが自分の
  世代を持つ）を取れなければならず、役割分離を standalone CLI 経路に限ってはならない。
  `nput.configs` の `<name>` 1 つが 1 manifest = 1 profile に対応しなければならない。
  「1 config = 1 profile = 1 manifest」の atomic 性は保たなければならず、複数 config を
  合成するヘルパを提供してはならない。`<name>` 次元の導入は非破壊でなければならず、`default`
  という名の config は従来と同じ profile に解決され続け、profile ディレクトリのレイアウト
  変更を伴ってはならない。profile ディレクトリをどうキーするかは REQ-d5a2e289、
  オプション自体の定義
  （canonical な形が `nput.configs.<name>.entries` であり、素の `nput.entries` が
  `configs.default.entries` への deprecated 糖衣であること）は全モジュール共通であり
  REQ-fc1c7ce6 の担当で、いずれも本 item では規定しない。
---
# REQ-c6891aeb: HM モジュール経由でも名前つき config ごとに役割分離した独立 profile を取れる

## 仕様

- `nput.configs` の属性キー `<name>` が profile 名で、standalone の entrypoint `nput.<name>` と
  同じ次元。`<name>` ごとに独立した link-farm を生成し、1 config = 1 profile = 1 manifest の
  atomic 性を保つ（合成ヘルパは提供しない）
- HM モジュール経由でも**役割分離ができる**。役割ごとに分けたいユーザーが standalone CLI 経路へ
  回る必要は無い
- `<name>` 次元の導入は**非破壊**。`default` という名の config は従来と同じ profile へ解決され
  続け、`default` 以外の `<name>` が増えるだけで profile dir のレイアウト変更は無い
  （キーの体系そのものは REQ-d5a2e289 の担当）

> **`docs/spec.md` 原文と異なる理由**: 原文の blockquote は「`nput.entries` は単一 attrset =
> 単一 manifest = 1 profile（固定名 `default`）で `<name>` 次元を持たず、**HM モジュール経由では
> 役割分離はできない**。役割ごとに分けたいユーザーは standalone CLI 経路を使う。HM の複数
> profile 化（`nput.configs.<name>.entries` 等）は将来拡張の seam として残す」と述べるが、
> この MVP 限定は **ADR-0035 が実装決定済み**（seam を実装へ進め、`nput.entries` を
> `configs.default.entries` への糖衣にする）で反転している。`docs/spec.md` 側がこの改訂に
> 追従できていないため、分割にあたって原文の MVP 限定を規範へ持ち込まない判断をした
> （REQ-37b56673 / REQ-16faf428 で ADR-0036 由来の未追従を扱ったのと同じ扱い）。
> `docs/spec.md` の追従は本 item の担当範囲外。
>
> **原文の残る規範の所在**: `nput.configs` / `nput.entries` のオプション定義そのものは
> REQ-fc1c7ce6（ADR-0035 §1 が定義先を `modules/common.nix` としているため全モジュール共通の
> 集合に属する）、profile dir レイアウトの物理形は REQ-2aa3abbc、home mode が `<name>` 直キーで
> あることは REQ-d5a2e289、standalone が `nput.<name>` で複数 profile を持てることは
> REQ-496b1a07、1 起動あたりの activation 契約は REQ-8085f194、module 経路で rollback を host へ
> 一本化することは REQ-844ee375 の担当。
>
> **ADR-0035 §3〜§4 を本 item に含めない理由**: 同 ADR は §3 で activation が `configs` を
> 走査して profile ごとに engine を kick すること（辞書順・1 profile の失敗で後続を止めず
> 最後に集約）、§4 で単一 HM config 内の `configs` 間の正規化後 target 衝突を eval 時
> assertion で停止することも決めているが、いずれも `docs/spec.md` に対応記述が無い
> （原文が ADR-0035 未追従のため）。本 item は原文 blockquote の担当範囲＝profile 粒度に
> 留め、**§3 の規範は REQ-c847d1af、§4 の規範は REQ-5923ac79 が持つ**。なお §4 の eval
> 停止は、別 entrypoint・別 manifest の cross-config を「eval では検出不可・実行時の
> 後勝ち + foreign warning」とする REQ-5c6b07da / REQ-fc1118b1 に対する**例外**（HM の
> `configs` は全 config が単一のモジュール eval に載るため静的検出が可能）であり、この
> 関係は REQ-5923ac79 の注記が明示している。

## 出典

`docs/spec.md`「モジュールオプション仕様」→「共通オプション（全モジュール）」節の
blockquote「HM モジュールの profile 粒度（MVP）」。

決定の実体は ADR-0035「HM モジュールに `nput.configs.<name>` を導入し複数 profile
（役割分離）を可能にする」§1〜§2（ADR-0024 §2 / ADR-0025 §2 の MVP 限定を改訂）。`<name>` 次元と
atomic 性・合成ヘルパ不提供は §1、`default` が従来と同じ profile に解決され続ける非破壊性は
§1（profile dir が `<name>` 直キーに乗ること）と §2（`nput.entries` を `configs.default.entries`
への糖衣とすること）の双方に依る。
