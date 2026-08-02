---
id: "REQ-61c05e09-0bde-4f74-9a96-03185f9df606"
type: requirement
name: "--root は全モード共通で解決 root を明示上書きし profileDir を roothash でキーする"
specification: |
  The `--root <path>` flag SHALL explicitly override the resolved root, in every mode
  alike: with it, project mode SHALL NOT use the git toplevel and home mode SHALL NOT use
  `$HOME`. When it is given, `profileDir` SHALL be overridden in every mode and keyed by
  the `<roothash>` of the overriding root.
specification_ja: |
  `--root <path>` は解決 root を明示上書きするフラグでなければならず、全モード共通で
  効く（project は git toplevel を、home は `$HOME` を使わない）。明示時は全モードで
  profileDir を上書き後 root の `<roothash>` でキーしなければならない。
---
# REQ-61c05e09: --root は全モード共通で解決 root を明示上書きし profileDir を roothash でキーする

## 仕様

```bash
--root <path>       # 解決 root を明示上書き（全モード共通。project は git toplevel を、home は $HOME を使わない）
                    # 明示時は全モードで profileDir を上書き後 root の <roothash> でキー
```

profileDir のレイアウト（`<roothash>/<name>` の形）そのものは REQ-2aa3abbc の担当で、
本 item は `--root` が全モードで root を上書きし profileDir のキーを roothash に切り替える
ことを規定する。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」のグローバルフラグ表 `--root <path>`。

決定の実体は ADR-0017「`--root` の適用範囲」（全モード共通の上書き）と ADR-0023
（`--root` 明示時の profileDir キーイング）。
