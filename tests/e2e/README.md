# nput E2E ハーネス（非 NixOS）

実 nix を使って `nput` を end-to-end に駆動し、**「非 NixOS でも nix さえあれば動く」**という
主張を検証する bash ハーネス（→ `docs/design.md`「テスト戦略」・ADR-0012）。

`lib`（nix-unit / namaka の評価テスト）や配置エンジン（Go の tmpdir 統合テスト）が
**nix を使わずに**純ロジック / FS 操作を検証するのに対し、ここでは **flake entrypoint からの
`nix build` / `nix eval` / `nix-env --set` を含む実経路**を一気通貫で回す。

## 実行

```bash
# CI（ubuntu-latest）と同じ起動方法。ci devShell が nput / git / jq / coreutils を提供する。
nix develop '.?dir=dev#ci' -c tests/e2e/run.sh
```

`run.sh` は `scenarios/*.sh` を辞書順に独立プロセスで実行し、1 つでも失敗すれば非ゼロ終了する。
各シナリオは隔離した一時 `$HOME` / `$XDG_STATE_HOME`（`mktemp -d`）下で動き、ランナーの実
profile / home を汚さない（`tests/e2e/lib.sh` の `e2e_isolate`）。偽 src は fixture flake
ディレクトリ内の相対パス（eval 時に store へコピー）か、out-of-store 用の live ディレクトリで用意する。
fixture flake は `nput` を `path:<repo>` input で参照し、`nixpkgs` / `home-manager` は nput の
`flake.lock` pin に `follows` させてオフライン評価する。

## シナリオ範囲

| シナリオ | 検証内容 |
|---|---|
| `01-project` | project mode。一時 git repo で `nput apply` → git toplevel 配下に store symlink 配置・再 apply の冪等性 |
| `02-home`    | home mode。仮 `$HOME` で apply → `$HOME` 配下配置 + profile 世代コミット、entry 入替で世代を進め `nput rollback` で前世代へ復帰 |
| `03-stale`   | stale 除去。entry を config から削除 → 再 apply で旧 symlink が消える（保守的不変条件） |
| `04-copy`    | copy place-once / out-of-store。copy が通常ファイル（書込可）・place-once 冪等（ローカル編集を破棄しない）・out-of-store の live symlink |
| `05-hm`      | HM module。home-manager standalone configuration を非 NixOS で評価・activate し、activation が engine を起動して配置すること |

## 将来拡張

- **NixOS VM テスト（`runNixOSTest`）は将来拡張に回す**（→ `docs/design.md`・ADR-0012）。
  NixOS / nix-darwin モジュール経路の実 activate は VM / sandbox を要し、本ハーネス（非 NixOS の
  ubuntu-latest 単一ジョブ）のスコープ外。モジュール経路を本実装する段で別途追加する。
