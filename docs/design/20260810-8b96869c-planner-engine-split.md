---
id: "DSG-8b96869c-842e-4f78-8ff7-df1f1d6c1a68"
type: design
name: "配置ロジックを plan を計算する planner と FS 意味論を実行する engine に分け、dryrun と apply が同一の plan を共有する"
satisfies:
  - "REQ-fa181aa6-29a2-48c3-ae07-cc1b9a3b0303"
  - "TP-e7c25263-6d2d-4a37-8275-26906889d912"
---
# DSG-8b96869c: 配置ロジックを plan を計算する planner と FS 意味論を実行する engine に分け、dryrun と apply が同一の plan を共有する

## 設計

`internal/` の配置ロジックを 2 層に分ける。

```
internal/planner/   ← 分類判定に閉じた純粋層。(前世代 manifest, 新 manifest, root, FS プローブ)
                       から plan（Place / Copies / Remove / PreRemove / Backup / Conflicts /
                       Warnings）を計算する。FS は Lstat / Readlink / ReadDir の
                       プローブ用インターフェース越しにのみ読み、書き込みは一切しない
internal/engine/    ← plan を実 FS へマテリアライズする層。symlink 生成・unlink・rename・
                       copy・世代切替を実行し、除去直前に安全不変条件を実 FS に対して再検証する
```

**planner が持つのは「前世代の記録 × 現状の実体」の分類**であり、FS の意味論そのものは持たない。
planner が読む FS は判定材料としてのプローブ（実体が symlink か通常ファイルか、リンク先が記録
どおりか）に限り、実行時は実 FS の実装を差し込む。**engine は planner の分類をやり直さず、
plan をそのまま実行する**。例外は `apply --recopy` の上書きで、これは plan を経由せず engine が
manifest の copy entry を直接走査する（配置の可否は plan の conflict gate が先に判定済み・
→ ADR-0020）。

この分け方が実現手段として効くのは次の点。

- **dryrun と apply が同一の planner の plan を共有できる**。REQ-fa181aa6 が要求する
  dryRun パリティは、「dryrun 用に予定差分を別途組み立てない」ことで構造的に保証される。
  plan の算出が engine の実行経路から切り離されていなければ、dryrun は実行しない代わりに
  予定を再現する第 2 の実装を持つことになり、パリティは実装の一致を人手で維持する約束に
  落ちる。`--recopy` の上書きが dryrun の plan に現れないこと（同 REQ）も、recopy が
  planner の分類ではなく engine の materialize 側経路にあるというこの分割の帰結である
- **分類層を fake FS の table-driven で覆える**。TP-e7c25263 がユニットレベルの table-driven を
  規範化する対象（同 item の言う「FS に触れずに判断する層」＝ FS 意味論を実行せず判定だけを
  下す層で、判定材料のプローブ読み取りは含む）を、この分割が構造として作る。規範とその理由は
  TP-e7c25263 が持つ
- **除去の安全不変条件が 2 度検査される**。planner が plan を組む時点と、engine が unlink する
  直前の 2 回で、記録どおりを指す symlink であることを確かめる。plan の算出から実行までの
  間に FS が変わりうる以上、実行側の再検証は省けない

分割の代償は、planner の分類と engine の実行が乖離しうること（→ RISK-f8e14849）。planner が
プローブ越しでなく実 FS を直接触るようになる、あるいは engine が plan を無視して独自に分類を
やり直すと、table-driven テストは緑のまま実挙動が壊れる。

### DSG-7d354fe0 / DSG-17db0831 とのスコープの違い

本 item が述べるのは **`internal/` の内部をどう割るか**であり、既存の 2 item とは扱う境界が違う。
DSG-7d354fe0 は `cmd/nput`（CLI 面）と `internal/`（配置ロジック）の 2 パッケージ境界を定める
もので、`internal/` の中身には踏み込まない。DSG-17db0831 は CLI / engine / lib / `common.nix` /
統合層という言語横断の 5 段と、その間の依存の向きを定めるもので、同じく engine を 1 段として
扱い内部を割らない。本 item はその engine 1 段の内側に planner / engine の境界を引く。
両 item の主張は本 item の新設によって変わらない。
