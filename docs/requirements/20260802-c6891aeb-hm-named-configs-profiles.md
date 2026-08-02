---
id: "REQ-c6891aeb-13c0-4ae7-9ad1-5c343735266a"
type: requirement
name: "HM モジュール経由でも名前つき config ごとに役割分離した独立 profile を取れる"
specification: |
  A user of the home-manager module SHALL also be able to hold several independent
  profiles separated by role, each with its own generations, and role separation SHALL NOT
  be confined to the standalone CLI path. One `<name>` of `nput.configs` SHALL yield one
  manifest and one profile; the atomicity of "one config = one profile = one manifest"
  SHALL be preserved and no helper for composing several configs SHALL be provided. The
  profile directory SHALL sit on the `<name>` key of home mode, so that a config named
  `default` resolves to the same place as before and no change of layout is entailed. The
  definition of the options themselves — that `nput.configs.<name>.entries` is the
  canonical form and the bare `nput.entries` a deprecated sugar onto
  `configs.default.entries` — is stated by REQ-fc1c7ce6, being common to every module, and
  is not restated here.
specification_ja: |
  home-manager モジュールの利用者も、役割分離された複数の独立 profile（それぞれが自分の
  世代を持つ）を取れなければならず、役割分離を standalone CLI 経路に限ってはならない。
  `nput.configs` の `<name>` 1 つが 1 manifest = 1 profile に対応するものとする。
  「1 config = 1 profile = 1 manifest」の atomic 性は保ち、複数 config を合成するヘルパを
  提供してはならない。profile ディレクトリは home mode の `<name>` キーに乗せ、`default`
  という名の config は従来と同じ場所に解決されるものとし、レイアウトの変更を伴っては
  ならない。オプション自体の定義（canonical な形が `nput.configs.<name>.entries` であり、
  素の `nput.entries` が `configs.default.entries` への deprecated 糖衣であること）は
  全モジュール共通であり REQ-fc1c7ce6 の担当で、本 item では規定しない。
---
# REQ-c6891aeb: HM モジュール経由でも名前つき config ごとに役割分離した独立 profile を取れる

## 仕様

- `nput.configs` の属性キー `<name>` が profile 名で、standalone の entrypoint `nput.<name>` と
  同じ次元。`<name>` ごとに独立した link-farm を生成し、1 config = 1 profile = 1 manifest の
  atomic 性を保つ（合成ヘルパは提供しない）
- HM モジュール経由でも**役割分離ができる**。役割ごとに分けたいユーザーが standalone CLI 経路へ
  回る必要は無い
- profile dir は home mode の `<name>` 直キー（`<state>/nix/profiles/nput/<name>`）にそのまま
  乗る。`default` 以外の `<name>` が増えるだけでレイアウトの変更は無い

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
> 留め、§3〜§4 の item 化は原文の追従（段階 7）とあわせて別途扱う。なお §4 の eval 停止は、
> 別 entrypoint・別 manifest の cross-config を「eval では検出不可・実行時の後勝ち + foreign
> warning」とする REQ-5c6b07da / REQ-fc1118b1 に対する**例外**（HM の `configs` は全 config が
> 単一のモジュール eval に載るため静的検出が可能）であり、item 化の際はこの関係を明示する
> 必要がある。

## 出典

`docs/spec.md`「モジュールオプション仕様」→「共通オプション（全モジュール）」節の
blockquote「HM モジュールの profile 粒度（MVP）」。

決定の実体は ADR-0035「HM モジュールに `nput.configs.<name>` を導入し複数 profile
（役割分離）を可能にする」§1（ADR-0024 §2 / ADR-0025 §2 の MVP 限定を改訂）。
