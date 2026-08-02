---
id: "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
type: use_case
name: "プロジェクト repo 内へ nix store の物を devShell 入室のたびに配置してチームで共有する"
refines:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
---
# UC-19a90989: プロジェクト repo 内へ nix store の物を devShell 入室のたびに配置してチームで共有する

## 使われ方

nput の**中心的な配置モード**。任意のプロジェクトに nput を組み込み、root をプロジェクト
ルートに解決して repo 内の任意パスへ nix store の物を配置する（→ ADR-0005 / ADR-0007）。
新規プロジェクトは `nput init project`、既に `flake.nix` がある既存 repo へは手動で組み込む
（→ ADR-0024）。

具体例:

- repo 内の `.claude/skills/` を nix store から配置してチームで共有する
- project-local な tool 設定・hook をバージョン固定で repo 内へ配置する
- 社内共有の設定リポジトリから特定ディレクトリだけを取り出してプロジェクトへ配置する

主トリガは devShell。`devShells.<name>` の `shellHook` から nput をキックし、`nix develop` /
direnv でプロジェクトに入った瞬間に配置される。CLI 本体は devShell の `packages` に pin 版
`nput` を同梱するのが canonical（→ ADR-0015）。

```nix
# entrypoint(flake.nix)が manifest を公開し、devShell で nput apply する
nput.${system}.skills = nput.lib.mkManifest {
  inherit pkgs;
  root = nput.lib.projectRoot;
  entries = {
    ".claude/skills/nix" = { src = inputs.claude-skills; subpath = "skills/nix"; };
  };
};
devShells.${system}.default = pkgs.mkShell {
  packages  = [ nput.packages.${system}.nput ];   # pin 版 nput を PATH へ
  shellHook = "nput apply skills --no-wait";
};
```

> **上は `docs/concept.md` からの写し**（コメントの出典注記は省いた）。use_case は使われ方を
> 述べる層で規範を持たないため、この例が示す各要素の規範は requirement 側にある
> （devShell 同梱 → REQ-14f0aec9、`nput.<name>` のアドレッシング → REQ-496b1a07、
> devShell からの engine 起動 → REQ-a0bdf6db）。

配置物は per-clone で再生成される前提の **ephemeral** であり、プロジェクトにはコミットされ
ない。したがって activation は git 状態に干渉せず、`.gitignore` に入れるべき target は専用
コマンド `nput gitignore` で列挙してプロジェクト管理者が一度登録する。

## この使われ方が要求すること

- root がプロジェクトルートとして解決されること（git toplevel 相対・`--root` で上書き可）
- 配置物が ephemeral として扱われ、activation が git 状態に干渉しないこと
- `.gitignore` へ登録すべき target を列挙する手段があること
- devShell の `shellHook` からキックする配線が成立すること。入室のたびに走るため、変化が
  無いときの再配置コストが抑えられていること
- 世代は内部機構に留め、ephemeral な配置に意味の薄い rollback を公開しないこと

## 出典

`docs/concept.md`「プロジェクトに閉じた配置（project mode）」と「想定ユースケース」の
project mode 節。
