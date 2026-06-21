# nput の project mode config（dogfood）を flake-parts module として切り出す（→ ADR-0029）。
# root flake が公開する flakeModules.default（perSystem.nput.<name> を flake.nput.<system>.<name>
# へ転置する機構）を前提に、perSystem.nput.skills へ mattpocock/skills の manifest を宣言する。
# dev/flake.nix の imports に並べて読み込む。
#
# `nput apply skills -f <dev flake>` でビルドし、各 skill を .claude/skills/<name> へ
# store-symlink 配置する。root = projectRoot（git toplevel）なので配置先は repo root 配下。
# 配置物は .gitignore 済み（.claude/skills/*）の ephemeral。
{ inputs, ... }:
let
  nputLib = inputs.root.lib;

  # 展開する skill を明示列挙する（mattpocock/skills の skills/ 配下の相対パス）。
  # 本来は skills/<category> を lib.listFilesInSrc で動的列挙したいが、その API は
  # 本ブランチ未実装のため、現行運用（skills-lock.json）の skill 集合を明示列挙して
  # 忠実に再現する。skills-lock.json は vercel skills 用に残置（両者は別経路）。
  skillSubpaths = [
    "engineering/grill-with-docs"
    "engineering/improve-codebase-architecture"
    "engineering/prototype"
    "engineering/setup-matt-pocock-skills"
    "engineering/tdd"
    "engineering/to-issues"
    "engineering/to-prd"
    "engineering/triage"
    "productivity/grilling"
    "productivity/handoff"
  ];

  # skill ごとに { ".claude/skills/<name>" = entry; } を組む。
  # target = .claude/skills/<skill 名>、配置元は skills/<category>/<name> の subpath。
  skillEntries = builtins.listToAttrs (
    map (p: {
      name = ".claude/skills/${baseNameOf p}";
      value = {
        src = inputs.matt-skills;
        subpath = "skills/${p}";
      };
    }) skillSubpaths
  );
in
{
  perSystem =
    { pkgs, ... }:
    {
      # perSystem.nput.skills → flake.nput.<system>.skills へ自動転置される（root flakeModule）。
      # pkgs は perSystem 由来（= nixpkgs.legacyPackages.<system>）で packages.nput と一貫する。
      nput.skills = nputLib.mkManifest {
        inherit pkgs;
        root = nputLib.projectRoot;
        entries = skillEntries;
      };
    };
}
