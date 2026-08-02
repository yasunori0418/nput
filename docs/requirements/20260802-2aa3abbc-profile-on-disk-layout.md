---
id: "REQ-2aa3abbc-90b2-486e-92de-d785554bdeb3"
type: requirement
name: "profileDir は config 専用ディレクトリとし、profile リンク・世代・pending out-link をその中に並べる"
specification: |
  `profileDir` SHALL be a directory dedicated to each config, holding within it the
  profile link, the generations and the build out-link. Its base `<state>` SHALL be
  `$XDG_STATE_HOME` when that is set and `~/.local/state` otherwise, consistently with the
  default profile location of nix itself. The profile link SHALL be `<profileDir>/profile`
  and SHALL be the subject of `nix-env --profile`; the generations `profile-N-link`
  created by `nix-env` and the `nix build --out-link` target SHALL sit as its siblings
  inside the same directory, so that neither passes through the profile link and no
  breakage arising from passing through a read-only store path can occur structurally.
  The pending out-link SHALL be named `.pending`, there being at most one per config
  inside the dedicated directory. The backref file `.root`, which records the absolute
  path of the original root, SHALL be placed at the roothash level — the parent of the
  `<name>` directory — and SHALL be shared by the several `<name>` directories under it.
  `profileDir` SHALL also be the flock key, serializing apply / reset / rollback for the
  same config while remaining independent of a different `<name>` directory under the same
  roothash.
specification_ja: |
  `profileDir` は各 config 専用のディレクトリでなければならず、その中に profile リンク・世代・
  build out-link を置く。基底 `<state>` は `$XDG_STATE_HOME` があればそれ、無ければ
  `~/.local/state` とする（nix 本体の profile 既定と整合）。profile リンクは
  `<profileDir>/profile` とし、これを `nix-env --profile` の対象とする。`nix-env` が作る世代
  `profile-N-link` と `nix build --out-link` の対象は、同じディレクトリ内の兄弟として並べ、
  profile リンクを貫通させてはならない（read-only な store パスを貫通する破綻を構造的に
  起こさないため）。pending out-link は専用ディレクトリ内に config あたり最大 1 個であり、
  名は `.pending` とする。元 root の絶対パスを記録した backref ファイル `.root` は roothash 階層
  （`<name>` dir の親）に置き、複数の `<name>` で共有する。`profileDir` は flock キーでもあり、
  同一 config の apply / reset / rollback を直列化しつつ、同 roothash でも別 `<name>` dir とは
  独立させる。
---
# REQ-2aa3abbc: profileDir は config 専用ディレクトリとし、profile リンク・世代・pending out-link をその中に並べる

## 仕様

```
<state>/nix/profiles/nput/<roothash>/.root              # backref（roothash 階層・複数 <name> で共有）
<state>/nix/profiles/nput/<roothash>/<name>/             # ← profileDir（flock キー）
<state>/nix/profiles/nput/<roothash>/<name>/profile        # profile リンク（nix-env --profile <profileDir>/profile の対象）
<state>/nix/profiles/nput/<roothash>/<name>/profile-N-link # 世代（nix-env が profile の兄弟に作成）
<state>/nix/profiles/nput/<roothash>/<name>/.pending       # nix build --out-link（profile を貫通しない兄弟）
# home（--root なし）: <state>/nix/profiles/nput/<name>/{profile, profile-N-link, .pending}
```

> profile の基底 `<state>` は **`$XDG_STATE_HOME` があればそれ、無ければ `~/.local/state`**
> （nix 本体の profile 既定と整合）。

- profile 操作は `nix-env --profile <profileDir>/profile ...`、build は
  `nix build --out-link <profileDir>/.pending`。世代兄弟 `profile-N-link` と `.pending` が
  profile リンクを貫通せず兄弟として並ぶため、read-only な store パスを貫通する破綻が構造的に
  起きない
- **pending out-link は専用ディレクトリ内に 1 個なので名は `.pending`**（`<name>` 次元は
  ディレクトリ階層が表す）。`--set` 前失敗で残る pending は config あたり最大 1
- **flock キー = profileDir（専用ディレクトリ）**。同一 config の apply / reset / rollback を
  直列化し、同 roothash でも別 `<name>` dir とは独立する
- **backref `.root` は roothash 階層**（`<name>` dir の親）に置き、複数 `<name>` で共有する
  （孤児 profile 逆引き seam）

> **上は原文の写しで、規範は frontmatter が正**。`profileDir` の**キー**（home は `<name>`・
> project / fixed / `--root` 上書きは `<roothash>/<name>`）そのものは REQ-d5a2e289 の担当で、
> project mode で解決済み root をキーにする理由と `<roothash>`（解決後の絶対 root パスの
> sha256 短縮 hex）の定義は REQ-46fccb80、孤児 profile の扱いは REQ-d41b1d0a の担当。
> HM モジュール経由も home mode の `<name>` 直キーに乗る（原文が挙げる「固定名 `default`」の
> MVP 限定は ADR-0035 が反転済み・REQ-c6891aeb）。本 item は「キーで指す先が
> ディレクトリであり、その中に何をどう並べるか」という物理形を規定する。flock を blocking で
> 取るか try-lock で取るかは REQ-1c1526b1、pending out-link が gcroot を持ち config あたり
> 最大 1 に抑えられることは REQ-840b3641 の担当。

## 出典

`docs/spec.md`「世代管理仕様」→「機構」→「profile のオンディスクレイアウト」節と、
同「機構」節の機構表直後の blockquote（`<state>` の定義）。

決定の実体は ADR-0025「実装前残セマンティクス第7巡」の「profile 専用ディレクトリレイアウト」
（ADR-0022 / ADR-0023 の `.pending-<name>` を改訂）。`<state>` の定義は ADR-0022、backref
`.root` による孤児逆引きは ADR-0013 が定めている。
