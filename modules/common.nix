# 全モジュール共通の nput オプション定義（→ ADR-0003, ADR-0007, ADR-0010, ADR-0014）。
#
# HM / NixOS / nix-darwin が import する共通の options。配置ロジックは持たず、
# 「何を・どこへ・どう置くか」のデータ（entries）だけを宣言させる。各モジュールは
# 自分の性質で root を pin する（HM → homeRoot）ため、利用者は root を再指定しない。
#
# entry submodule の型は lib/types.nix と共有する（mkManifest の evalModules と同一の
# entriesType）。これにより未知キー（タイポ・旧名）は strict submodule で eval エラーに
# なり、検査の二重定義を避ける（→ ADR-0010, docs/spec.md「モジュールオプション仕様」）。
#
# > HM モジュール経由は MVP で単一 nput.entries = 1 profile（固定名 default）に限り、
# > role 分離（複数 profile）はできない。役割分離が要るユーザーは standalone CLI 経路
# > （entrypoint の nput.<name>）を使う。複数 profile 化は将来 seam（→ ADR-0024, ADR-0025）。
{ lib, ... }:
let
  nputTypes = import ../lib/types.nix lib;
in
{
  options.nput = {
    enable = lib.mkEnableOption "nput（フェッチ済み git リポジトリの symlink / copy 配置）";

    entries = lib.mkOption {
      type = nputTypes.entriesType;
      default = { };
      example = lib.literalExpression ''
        {
          # 属性キー = root 相対 target（識別子・→ ADR-0014）
          ".claude/skills/nix" = {
            src = inputs.claude-skills;
            subpath = "skills/nix";
          };
        }
      '';
      description = ''
        配置定義の attrset。属性キー = 配置先 target（識別子）で、各値は entry submodule
        （src / subpath / target / method）。型は lib/types.nix と共有し、未知キーは eval
        エラーになる（→ docs/spec.md「entries スキーマ仕様」）。
      '';
    };
  };
}
