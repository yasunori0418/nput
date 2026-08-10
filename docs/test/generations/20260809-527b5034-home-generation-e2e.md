---
id: "TC-527b5034-715a-4df1-871a-072dc9062704"
type: test_condition
name: "実 nix を通した home mode で世代がコミットされ、rollback が前世代の配置へ戻る"
mitigates:
  - "RISK-fbf029f6-866c-4a08-a4eb-f09e3c7e907e"
  - "RISK-cdcc6faf-9164-409b-b584-2921fa036d10"
---
# TC-527b5034: 実 nix を通した home mode で世代がコミットされ、rollback が前世代の配置へ戻る

## テスト条件

Go テストが double へ差し替えている層（`nix build` / `nix-env --set` / 世代一覧）を
実物に戻し、home mode の世代機構が一気通貫で動くことを検証する。仮 `$HOME` と
`$XDG_STATE_HOME` で隔離した環境に fixture flake を書き、entry を入れ替えながら
apply を重ねる。

- **配置と世代コミット** — apply が `$HOME` 配下へ配置し、`<state>/nix/profiles/nput/
  home/profile` に profile リンクが作られ、`list-generations` が世代を返す
- **世代の積み上がり** — entry を入れ替えて再 apply すると、旧 entry が stale 除去され、
  新 entry が配置され、世代が 2 つ以上になる
- **rollback の往復** — `rollback` が前世代の配置状態（旧 entry あり・新 entry なし）へ
  戻す
- **成功時沈黙** — 既定では配置レポートを出さず、`-v` で出す（→ ADR-0031）
- **`--json` の世代観測** — `apply --dryrun --json` の generation が before == after
  （非 null）を運び、`list-generations --json` が `{number, date, current}` の 2 世代を
  返し、current が rollback 先の 1 世代だけであること。`items` は空で generation スロットを
  持たず `dryRun` が false であること（→ issue #132）

## 対応する CASE

CASE-2e7b9ea2（`tests/e2e/scenarios/02-home.sh`）。E2E ハーネス全体のスコープは
TP-229b69c0、`--json` の検証範囲は TP-d3000054 が定める。
