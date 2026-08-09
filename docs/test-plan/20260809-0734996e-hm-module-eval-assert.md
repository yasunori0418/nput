---
id: "TP-0734996e-aea9-4229-8075-89a64bdf9f79"
type: test_plan
name: "home-manager モジュールの配線は build sandbox 内の評価アサートで検証する"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
depends_on:
  - "TP-b7f1dc79-0222-4b6e-9e91-0545046e34f2"
  - "TP-229b69c0-cf5e-4fb6-a353-27e5064d93e9"
specification: |
  The home-manager module SHALL have a verification level between pure evaluation of the
  library and the end-to-end harness: a check that evaluates a standalone home-manager
  configuration inside the build sandbox and asserts over the resulting activation script
  and the manifest it references, without activating anything. That level SHALL assert the
  properties that are decided at evaluation time and would otherwise only be observable
  after activation — that the module wires activation to the engine rather than translating
  entries into home-manager's own file placement, that the manifest handed to the engine
  pins the home root, that entries declared on the module reach the manifest, and that
  optional module settings reach the invocation as the flags they correspond to. Anything
  requiring a real activation SHALL remain outside this level, since the sandbox has no
  profile to mutate. The check SHALL use the same evaluation-test double as the rest of the
  evaluation layer, so that its manifest is byte-stable.
specification_ja: |
  home-manager モジュールは、lib の純評価と E2E ハーネスの中間に検証レベルを持たなければ
  ならない。すなわち、standalone な home-manager configuration を build sandbox 内で評価し、
  得られた activation スクリプトとそれが参照する manifest に対して、何も activate せずに
  アサートする check である。このレベルは、評価時に決まり、さもなければ activate 後にしか
  観測できない性質をアサートしなければならない。モジュールが entry を home-manager 自身の
  ファイル配置へ翻訳するのではなく activation をエンジンへ配線していること、エンジンへ渡す
  manifest が home root を pin していること、モジュールで宣言した entry が manifest へ届いて
  いること、そして任意のモジュール設定が対応するフラグとして起動に届いていることである。
  実 activate を要するものはこのレベルの外に留めなければならない（sandbox には変更すべき
  profile が無いため）。この check は評価層の他と同じ test double を使い、manifest をバイト
  安定に保たなければならない。
---
# TP-0734996e: home-manager モジュールの配線は build sandbox 内の評価アサートで検証する

## 仕様

home-manager モジュールの検証は 3 レベルに分かれ、本 item は中間層を定める。

| レベル | 内容 | 担当 item |
|---|---|---|
| 評価アサート（本 item）| build sandbox 内で standalone HM configuration を評価し、activation スクリプトと manifest にアサート。activate しない | 本 item |
| 実 activate（非 NixOS）| E2E で HM configuration を実際に activate し、engine が配置することまで見る | TP-229b69c0（シナリオ `05-hm`）|
| 実 activate（NixOS / nix-darwin）| VM / sandbox を要するためスコープ外 | TP-b7f1dc79 |

中間層でアサートするのは、**評価時に決まるが activate 後にしか観測できない性質**。

- activation が entry を home-manager 自身のファイル配置へ翻訳せず、engine を manifest
  経路で起動する配線であること
- engine へ渡す manifest が home root を pin していること
- モジュールで宣言した entry が manifest へ届いていること
- 任意のモジュール設定が、対応するフラグとして同じ起動に配線されること

実 activate（profile 世代のコミット・FS 配置）はこのレベルに含めない。build sandbox には
変更すべき profile が無いため。そこは E2E（TP-229b69c0）が受ける。

manifest をバイト安定に保つため、評価層の他と同じ test double を使う（→ TP-d3d06fe4）。

## 出典

現況の `flake.nix` `checks.hm-module`。同 check のコメントが「実 activate（`nix-env --set`・
FS 配置）は build sandbox では行えないため E2E へ回す」と、本 item が定める境界をそのまま
述べている。起点は Issue #17（HM モジュール統合の AC）で、実 activate 側は Issue #19。

> **本 item が埋めるのは TP-b7f1dc79 と TP-229b69c0 の隙間**。前者は VM を要する経路の
> スコープ外宣言、後者は E2E ハーネスの検証範囲を定めるが、その中間にある「activate せずに
> 評価だけで確かめられる配線」がどちらの担当でもなかった。
