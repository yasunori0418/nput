---
id: "REQ-61c05e09-0bde-4f74-9a96-03185f9df606"
type: requirement
name: "--root は全モード共通で解決 root を明示上書きする"
specification: |
  The `--root <path>` flag SHALL explicitly override the resolved root, in every mode
  alike: with it, project mode SHALL NOT use the git toplevel and home mode SHALL NOT use
  `$HOME`. How `profileDir` is keyed when it is given is stated by REQ-d5a2e289 and is not
  restated here.
specification_ja: |
  `--root <path>` は解決 root を明示上書きするフラグでなければならず、全モード共通で
  効く（project は git toplevel を、home は `$HOME` を使わない）。明示時に profileDir を
  どうキーするかは REQ-d5a2e289 の担当で、本 item では規定しない。
---
# REQ-61c05e09: --root は全モード共通で解決 root を明示上書きする

## 仕様

```bash
--root <path>       # 解決 root を明示上書き（全モード共通。project は git toplevel を、home は $HOME を使わない）
                    # 明示時は全モードで profileDir を上書き後 root の <roothash> でキー
```

> **上は原文の写しで、規範は frontmatter が正**。写しの 2 行目（明示時の profileDir キーイング）は
> **REQ-d5a2e289 の担当**で、本 item の規範には含めない。profileDir のキーは root の種別ごとに
> 定まる一つの体系（home 直キー / fixed・project・`--root` は roothash）であり、`--root` の行だけを
> 切り出して二重に規範化すると ADR-0023 §3 の改訂時に片方だけ直る事故が起きるため、REQ-d5a2e289 へ
> 一本化した。profileDir のレイアウト（`<roothash>/<name>` の中に何を並べるか）は REQ-2aa3abbc、
> `<roothash>`（解決後の絶対 root パスの sha256 を短縮した hex）の定義は REQ-46fccb80 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」のグローバルフラグ表 `--root <path>`、および
「root の解決」→「project mode（`root = projectRoot`）」節の箇条書き第 3 項（`--root` は
project mode に限らず全モードの解決 root を一律上書きする）。

決定の実体は ADR-0017「`--root` の適用範囲」（全モード共通の上書き）。
