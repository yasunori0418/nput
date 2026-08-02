---
id: "UC-d39c1994-f9a5-4860-80ba-f6e584adaf14"
type: use_case
name: "既に home-manager / NixOS / nix-darwin を使っている環境へ nput をモジュールとして組み込む"
refines:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
---
# UC-d39c1994: 既に home-manager / NixOS / nix-darwin を使っている環境へ nput をモジュールとして組み込む

## 使われ方

既に home-manager / NixOS / nix-darwin で環境を管理しているユーザーが、nput を standalone
CLI としてではなくモジュールとして組み込み、host の activation（`home-manager switch` /
`nixos-rebuild switch` 等）に nput の配置を載せる。flake-parts を使っているプロジェクト向け
には `perSystem.nput` を `flake.nput.<system>` へ transpose する flakeModule も提供する
（→ ADR-0029）。

統合はあくまで**オプション**である。コアロジックはモジュールシステムに依存しない純粋関数
として実装され、統合層はコアの薄いラッパーに過ぎない。配置の実体は全層で nput 自身の固定
Go エンジンが実行し、`home.file` / `systemd.tmpfiles` などプラットフォームのネイティブ機構
へは委譲しない（→ ADR-0003・ADR-0006）。

```
nput コア（配置エンジン）= 振る舞いの単一の源
        ↑ 起動するだけ
モジュール層 = root と activation hook を供給する薄い配線
```

モジュール経由でも役割ごとに名前つき config を分けて独立した profile を取れる。ただし
profile は**内部機構**（stale 追跡）に留め、ユーザー向け rollback は host に一本化する
（host rollback は旧 config の再 activate で nput が自動追従する）。

## この使われ方が要求すること

- モジュールが root と activation タイミングを供給して engine をキックするだけの配線であり、
  ネイティブ機構へ翻訳しないこと
- 全モジュールが共通オプションの同一集合を公開すること
- モジュール経路では entrypoint 発見・eval を経ず、ビルド済み link-farm を engine へ直接
  適用すること
- 配置に失敗したとき host の switch が止まること
- モジュール経由でも名前つき config ごとに独立した profile を取れること
- ユーザー向け rollback を host へ一本化し、nput profile は前進のみで追従すること
- flake-parts 経路が直書きと同一の derivation を生み、CLI のアドレッシングを変えないこと

## 出典

`docs/concept.md`「設計の哲学」の「配置ロジックはコアが所有し、モジュールは配線に徹する」
「home-manager に依存しない」「統合は『オプション』」、および「世代管理（standalone）」の
モジュール時の profile の扱いに関する記述。

Issue #211 の素材リストにはこの項目が無いが、モジュール統合は concept.md が繰り返し述べる
使われ方であり、統合層の requirement 群がこの use_case を持たないと orphan になるため
item 化した（2026-08-02 確定）。
