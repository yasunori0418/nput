---
id: "CASE-276945a3-59c2-493d-9971-c080a9a8c066"
type: test_case
name: "nix-unit: aggregator-merge.nix — 衝突検査をダミー入力で検証する"
target: "tests/nix-unit/aggregator-merge.nix"
covers:
  - "TC-539ce047-3ec3-4fb2-a4d1-9f83a4f75101"
---
# CASE-276945a3: nix-unit aggregator-merge.nix

## 対象

`tests/nix-unit/aggregator-merge.nix`（`tests/nix-unit.nix` のディレクトリ列挙で自動搭載。
TP-36e90d5d が定めるとおりテスト名はファイル横断で一意でなければならず、本ファイルは
`testAggregatorMerge*` 接頭辞を使う）

検証するのは同じスイートの組み立て器である `tests/nix-unit-lib.nix` の `mergeTests`。
アグリゲータ本体（`tests/nix-unit.nix`）は `builtins.readDir` に直結していて注入点を
持たないため、Issue #308 で検査を `{ file, tests }` のリストを引数に取る純関数へ切り出し、
本ファイルがそこへダミー入力を渡す。純関数は `tests/nix-unit/` の**外**に置く（内側は
readDir が列挙するので、置くとそれ自身がテストファイルとして import される）。

## 検証内容

合成した `{ file, tests }` のリストを `mergeTests` へ渡し、衝突なし側は返り値を、衝突あり側は
`expectedError.type = "ThrownError"` + `expectedError.msg` で throw を見る。

- 衝突が無ければマージ結果のテスト名が全ファイルぶん揃う（`attrNames` を固定値で比較）
- 値もそのまま写る（attrset 全体を exact 一致で見る）
- 空入力は空 attrset（衝突なしの境界の下端）
- 1 ファイルだけならテストを何件持っていても衝突しない（`length owners > 1` の境界の下側）
- 2 ファイルが同名 `testDup` を定義したら throw し、報告に `  - testDup: alpha.nix, beta.nix`
  の行が出る
- 衝突していない `testAlphaOne` は報告に現れない（negative lookahead で不在を見る）
- 3 ファイルが同名なら owners が 3 件とも列挙される
- 衝突するテスト名が 2 組あれば報告が 2 行になる（行の連結をパターンに直接書いて見る）

ダミーの `tests` の値は `mergeTests` が名前しか見ないので任意でよいが、マージ結果が名前と値の
対応を取り違えていないか見分けられるよう全て違う値を置いている。`//` の後勝ちがどちらを選ぶかは
検証しない（衝突する入力は必ず throw するので到達しない）。

`expectedError.msg` の照合は nix-unit が `std::regex` を既定文法（ECMAScript）で構築するため
negative lookahead が使え、`$` は行末ではなく文字列末尾を指す。`.` は改行を跨がないので複数行の
報告を舐めるパターンは `(.|\n)` と書く。

`nput` 引数は使わない（検証対象がスイートの組み立てで manifest 生成関数ではないため）が、
アグリゲータが全 leaf を `{ lib, nput }` で import するのでシグネチャは他 leaf に揃える。

**このテストの検出力はミューテーションで確認した**（→ Issue #308 の実装時）。`collisions` を
常に空にすると throw 側 4 件が落ち、owners の列挙を先頭 1 件へ切ると報告内容を見る 3 件が
落ちる。判定を owners の件数ではなくテストの総数で見る誤実装は、非衝突側のどれか（多くは
`disjoint` を使う 2 件）が落とす。単一ファイルのケースはそこへ独自の検出力を足すものではなく、
**境界の下側を明示的に固定する**位置づけである。

一方、衝突判定を外して全テスト名を衝突扱いにするミューテーションは本番アグリゲータ自身が
throw してスイート全体が評価不能になるため、本 CASE の評価には到達しない（`mergeTests` を
`nix eval` で直接叩けば確認できるが、スイート内では検知できない）。

## 出典

Issue #308（#287 で入れた重複検査に、検出器自身を検証するテストが無い。検出元は #287 の
レビュー収束）。緩和する脅威は RISK-3de9753f、一意性の規約は TP-36e90d5d が持つ。
