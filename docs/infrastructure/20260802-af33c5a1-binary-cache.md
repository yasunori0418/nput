---
id: "INF-af33c5a1-f846-4f09-8e43-c6ea042d6e06"
type: infrastructure
name: "バイナリキャッシュ（main push 時の cachix 投入）"
depends_on:
  - "INF-d1230e1f-8ba8-49d8-8386-409bfbb7dd27"
satisfies:
  - "QA-58522afb-31d5-4a1f-a7df-0858efa9e44b"
---
# INF-af33c5a1: バイナリキャッシュ（main push 時の cachix 投入）

## 構成

cachix のキャッシュ `yasunori0418` へ `packages.<system>.nput` のビルド生成パスを投入する。
consumer の CI / ローカルが「最新 main の `nput` を実ビルドせずキャッシュから引く」ための基盤。

- 構成は CI パイプライン（INF-d1230e1f）と同じ os×system の 3 環境マトリクス
  （`ubuntu-latest`=x86_64-linux / `ubuntu-24.04-arm`=aarch64-linux /
  `macos-latest`=aarch64-darwin）で `nix build .#packages.<system>.nput` をネイティブ実行する。
  ただし**同じ値を持つ別々の定義**で、この workflow 自身の `strategy.matrix` に直書きしてある
  （下の「CI パイプラインへの依存」を参照）
- 投入は `.github/actions/setup-nix` が内包する `cachix-action` が authToken 指定で生成パスを
  自動 push する経路に任せる。明示の push ステップは置かない
- 単一パッケージのため、未キャッシュのものだけを選別する plan / collect の多段構成は採らない

## トリガ

`push: main` + `paths` + `workflow_dispatch`。nput バイナリの内容を決める入力
（`**.nix` / `**.go` / `go.mod` / `go.sum` / `flake.lock` / `dev/flake.lock`）が変わった
main push でのみ投入し、docs だけの変更では走らせない。

投入契機を tag push にしない理由は、tag リリースに紐づけるとリリース間の main 変更がキャッシュ
されず、nput が flake input として消費される形と噛み合わないため。ここは INF-9878e9f5
（リリース自動化）とは独立に動く — キャッシュは main の最新、リリースは `VERSION` の変更が
それぞれ駆動する。

なお required status check を持つ CI パイプライン（INF-d1230e1f）と異なり、この workflow は
main への push 後に走るため、トリガ段の `paths` フィルタをそのまま使える。

## CI パイプラインへの依存

投入の実行契機は独立（PR 対 main push）だが、**実行基盤は CI パイプライン（INF-d1230e1f）に
乗っている**。この workflow は自前の nix セットアップを持たず、`.github/actions/setup-nix`
（`install-nix-action` + cachix 認証込みの `cachix-action`）へ丸ごと委ねる。投入経路そのものが
その composite action の中にあるため、`setup-nix` の構成が変われば投入も直接影響を受ける。

os×system マトリクスも CI パイプラインと同一の内容を踏襲するが、こちらは共有部品ではない。
`setup-nix` と違って reusable workflow / composite action のような共有機構は挟んでおらず、
**同じ 3 組をこの workflow の `strategy.matrix` へ独立にハードコードした写し**である。したがって
**片方を変えたらもう片方の追従が要る** — CI パイプライン側で環境を増減しても、この workflow は
黙って旧構成のままビルドし続ける（キャッシュに載る system が CI の検証対象とずれる）。
INF-8b97573f が「ジョブ名 / matrix を変えると check 名も変わる。ruleset 側の追従が要る」と
言うのと同じ種類の追従点で、マトリクスの変更はこの 2 箇所へ同時に波及する。

## 出典

ADR-0028（cachix push を main push の os マトリクスで実装する）。
