---
id: "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
type: use_case
name: "配置に失敗・後悔したとき standalone で前の世代へロールバックして元の状態へ戻す"
refines:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
---
# UC-0b6f60cb: 配置に失敗・後悔したとき standalone で前の世代へロールバックして元の状態へ戻す

## 使われ方

`src` を更新して適用したところ、新しいバージョンのプラグインが壊れていた・意図しない内容が
配置された、といった場面で、ユーザーは前の世代へ戻して復旧する。

```bash
nput apply <name>            # 新世代を作って適用（name 省略時は nput.default）
nput rollback <name>         # 前世代へ戻す
nput list-generations <name> # 世代一覧
```

世代は nix profile に乗る（→ ADR-0002）。純粋関数 `lib.mkManifest` が link farm derivation
（`manifest.json` + ストア内の symlink ツリー）を生成し、engine が nix profile に登録する
ことで、世代番号・GC root・ロールバックを Nix 標準機構から得る。粒度は `nput.<name>`
単位 = 1 profile なので、役割ごとに独立したロールバック系列を持つ。

```
純粋関数:  lib.mkManifest が link farm derivation（manifest.json + symlink farm）を生成（副作用なし）
実行時:    固定 Go エンジンが配置し profile を swap（副作用はここだけ・→ ADR-0006）
```

世代管理の対象になるのは symlink 配置。out-of-store symlink はリンク先のみが世代管理され
（指す先の内容は設計上ライブ）、copy は世代外である。

| 配置種別 | 世代管理 | 備考 |
|---|---|---|
| symlink（store）| あり（profile）| ロールバック可能 |
| out-of-store symlink | あり（リンク先のみ）| 指す先の内容は設計上ライブ。版管理しない |
| copy | **なし（世代外）** | place-once・ユーザー管理の副作用 |

この使われ方は standalone（home mode）が対象。project mode は配置物が ephemeral なため
rollback を公開せず、モジュール経由のときはユーザー向け rollback を host に一本化する
（→ UC-19a90989・UC-d39c1994）。

## この使われ方が要求すること

- 適用のたびに新世代が積まれ、profile の差し替えで atomic に切り替わること
- 前世代へ戻すコマンドと、世代を一覧するコマンドがあること
- ロールバックが FS の状態まで含めて収束すること
- 適用やロールバックが途中で失敗しても中途半端な状態を残さないこと
- 前世代の記録から消えた entry の配置物が保守的に除去されること（stale 除去）
- 世代を積み続けたときの GC・不要世代の間引き手段があること

## 出典

`docs/concept.md`「世代管理（standalone）」。
