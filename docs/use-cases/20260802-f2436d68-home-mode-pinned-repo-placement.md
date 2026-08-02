---
id: "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
type: use_case
name: "home mode で外部リポジトリの中身をバージョン固定して $HOME 配下の任意パスへ配置する"
refines:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
---
# UC-f2436d68: home mode で外部リポジトリの中身をバージョン固定して $HOME 配下の任意パスへ配置する

## 使われ方

ユーザーが `root = nput.lib.homeRoot` を明示した config を書き、外部リポジトリの内容
（リポジトリ全体・サブディレクトリ・単一ファイル）をバージョン固定した状態で `$HOME` 配下の
任意パスへ配置する。`nput apply <name>` で適用し、`src` の更新（flake input の更新 /
npins update 等）後に再適用すると新しい内容へ切り替わる。

具体例は次の 3 つで、いずれも「取得済みのストアパスを `$HOME` 配下の決まった場所へ置く」
という同じ使われ方の変奏である。

**vim / neovim プラグイン**。従来 `git clone` で特定ディレクトリへ配置するパターンを Nix で
再現し、バージョンを固定する。

```nix
# entries の 1 entry（属性キー = target）
".local/share/nvim/site/pack/foo/start/foo" = { src = inputs.vim-plugin-foo; };
```

**コーディングエージェントのスキル**。Claude Code などのエージェントはスキルをリポジトリで
管理することがある。リポジトリ全体ではなく、特定のサブディレクトリだけを取り出して配置する。

```nix
".claude/skills" = { src = inputs.skills-repo; subpath = "skills/nix"; };
```

**zsh / bash プラグイン**。プラグインを特定ディレクトリへ配置し、設定ファイルから `source`
で読み込む。

```nix
".zsh/plugins/autosuggestions"     = { src = inputs.zsh-autosuggestions; };
".zsh/plugins/syntax-highlighting" = { src = inputs.zsh-syntax-highlighting; };
```

このほか、カラーテーマリポジトリからの配置や、複数マシン（Linux / macOS）で同一の配置設定を
共有する使い方も同じ形に収まる。

## この使われ方が要求すること

- config を Nix で書き、`nput` CLI が entrypoint を発見して評価・適用すること
- `src` にストアパスを取り、`subpath` でリポジトリ内の一部だけを選べること。リポジトリ全体は
  `subpath` の省略で表せること
- 配置先が `$HOME` 配下として解決されること（`root = homeRoot` の明示・暗黙デフォルトなし）
- symlink 配置が冪等で、既存の配置物と衝突したときの振る舞いが決まっていること
- 配置をやめたくなったときに、配置物を無い状態へ戻す手段があること（`reset`。copy を消す
  唯一の手段でもあるため copy の使われ方が主な動機になるが、symlink 配置の撤去としても使う）

## 出典

`docs/concept.md`「役割の分離 — 具体的な用途」（vim プラグイン管理 / コーディングエージェント
のスキル / zsh・bash プラグイン）と「想定ユースケース」の home mode 節。

Issue #211 の素材リストはこの 3 例を別々に挙げるが、3 例とも紐づく requirement 群が重なる
ため 1 件の use_case に統合した（2026-08-02 確定）。分けても derives_from が同一の
requirement 集合を三重に指すだけで traceability の解像度が上がらないため。
