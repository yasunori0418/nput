---
id: "REQ-c2654ca5-62c2-4e4b-ad67-ffc5468f429b"
type: requirement
name: "NixOS / nix-darwin モジュールは配置先ユーザーを特定する user オプションを必須で取る"
specification: |
  The NixOS and nix-darwin modules SHALL take, in addition to the common options, a
  required `nput.user :: string` used to identify the user placement is for. The
  home-manager module and standalone SHALL NOT require `user`, both referring to `$HOME`.
specification_ja: |
  NixOS / nix-darwin モジュールは共通オプションに加えて `nput.user :: string` を必須で
  取らなければならない（配置先ユーザーの特定に使用する）。home-manager と standalone は
  `$HOME` を参照するため `user` を要求してはならない。
---
# REQ-c2654ca5: NixOS / nix-darwin モジュールは配置先ユーザーを特定する user オプションを必須で取る

## 仕様

```
nput.user :: string       # 必須（配置先ユーザーの特定に使用）
```

home-manager と standalone は `$HOME` を参照するため `user` は不要。

> **上は原文の写しで、規範は frontmatter が正**。`user` から実際に root を解決する式
> （`config.users.users.${cfg.user}.home`）は REQ-8d965ca2 の担当。

## 出典

`docs/spec.md`「モジュールオプション仕様」→「NixOS / nix-darwin 追加オプション（将来拡張）」節。

決定の実体は、NixOS / nix-darwin を将来拡張の配線層と位置づけ root を host が供給すると定めた
ADR-0003「配置ロジックは全層 nput エンジンが所有し、モジュールは配線に徹する」と ADR-0004
「root 一般化と「純粋関数プリミティブ」としての positioning」。
