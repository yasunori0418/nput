---
id: "DSG-bb6e03fd-e1c4-47c4-bfa0-a73e9457b184"
type: design
name: "E2E に legacy entrypoint シナリオを置き、NIX_PATH を flake.lock の nixpkgs に pin して検証する"
satisfies:
  - "REQ-c890ce4a-6528-4ab3-ac86-23d7aebff7da"
  - "REQ-da253cab-34d4-4d6e-96f0-de99e012b376"
  - "REQ-67095391-eab2-45d2-b75b-b428d481bcc2"
  - "TP-229b69c0-cf5e-4fb6-a353-27e5064d93e9"
---
# DSG-bb6e03fd-e1c4-47c4-bfa0-a73e9457b184: E2E に legacy entrypoint シナリオを置き、NIX_PATH を flake.lock の nixpkgs に pin して検証する

## 設計

TP-229b69c0-cf5e-4fb6-a353-27e5064d93e9 がハーネスに課すシナリオ 5 件（project mode / home mode / stale 除去 /
copy place-once・out-of-store / HM module）に加えて、**legacy entrypoint シナリオ**を
E2E に置く。

| 項目 | 内容 |
|---|---|
| entrypoint | `shell.nix`（passthru 形）|
| 検証する経路 | `nput apply` / `nput apply --all` / 素の `nix-shell` 互換 |
| nixpkgs の供給 | `NIX_PATH` を flake.lock の nixpkgs に pin |

**このシナリオを足す理由**は、legacy entrypoint が flake 経路とは**別の実行パスを通る**
ことにある。REQ-c890ce4a-6528-4ab3-ac86-23d7aebff7da が canonical と定める `mkShell` passthru 形は、
`nix build -f <ep> nput.<name>` という flake とは異なる attr path 解決を経る
（内部の共通化は DSG-92f54490-872a-42ac-bbd7-d06e9ee381c6）。attr path の同一性を CLI 内部で保っていても、
実際に nix が解決できるかは実 nix で叩かないと分からない。

**素の `nix-shell` 互換まで検証範囲に含める**のは、passthru 形を canonical にした
狙いが「`shell.nix` が mkShell を兼ねられる」ことだからである。`nput` を通さない
素の `nix-shell` が壊れていたら、passthru 形を選んだ意味が失われる。

**`NIX_PATH` を flake.lock の nixpkgs に pin する**のは、legacy entrypoint が
REQ-67095391-eab2-45d2-b75b-b428d481bcc2 の通り NIX_PATH 依存の impure eval を許容する best-effort だからである。
テスト側で pin しないと、実行マシンのチャンネル状態でシナリオの結果が変わり、
再現性のない E2E になる。pin する先を flake.lock に揃えれば、flake 経路の
シナリオと同じ nixpkgs で走る。

REQ-da253cab-34d4-4d6e-96f0-de99e012b376 が定める「legacy では相対 path の src が自動で store 化されない」性質も、
このシナリオで実際に踏まれる（fixture の組み方がそれを前提にする必要がある）。

## 出典

`docs/design.md`「テスト戦略」の E2E シナリオ列挙のうち legacy entrypoint の項。
