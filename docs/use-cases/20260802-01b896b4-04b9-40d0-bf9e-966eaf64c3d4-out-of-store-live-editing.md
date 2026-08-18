---
id: "UC-01b896b4-04b9-40d0-bf9e-966eaf64c3d4"
type: use_case
name: "開発中の手元 dotfiles を out-of-store symlink で参照し、編集と同時に反映しながら育てる"
refines:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
---
# UC-01b896b4: 開発中の手元 dotfiles を out-of-store symlink で参照し、編集と同時に反映しながら育てる

## 使われ方

手元の dotfiles リポジトリを開発している最中は、編集のたびに flake を更新して再適用するのは
回り道になる。この場合に限り、明示関数 `nput.lib.mkOutOfStoreSymlink "/abs/path"` で Nix
ストアを経由しない symlink を張り、ファイル編集と同時に反映させる。

```nix
".config/nvim" = { src = nput.lib.mkOutOfStoreSymlink "/home/user/dotfiles/nvim"; };
```

これは headline 機能ではなく**明示的な退避路**である。配置先のデフォルトは常に Nix ストアへの
symlink であり、再現性を担保する（→ ADR-0001）。out-of-store は型ベースの暗黙分岐ではなく、
ユーザーが明示関数を書いたときにだけ選ばれる。

| 配置元 | 反映タイミング | 向いている用途 |
|---|---|---|
| store link（デフォルト）| flake 更新 + 適用時 | バージョン固定した外部リポジトリ |
| `mkOutOfStoreSymlink "/abs/path"`（明示）| ファイル編集と同時（ライブ）| 開発中の手元 dotfiles |

> **上表は分割時点の `docs/concept.md`（原文）からの写し**。use_case は規範を持たないため、
> store link が既定で out-of-store が marker による opt-in であることの規範は
> REQ-99ca5381 にある。

指す先の内容は設計上ライブであり、版管理されるのはリンク先のマッピングのみ。世代管理は
リンク先のみが対象になる。

## この使われ方が要求すること

- store link がデフォルトで、out-of-store は明示関数による opt-in であること
- `mkOutOfStoreSymlink` が out-of-store を表すマーカーを返し、`src` がそれを受け取れること
- 指定した絶対パスが評価時に確定し、target の root 解決に影響しないこと
- 版管理の対象がリンク先マッピングに限られること

## 出典

`docs/concept.md`「store link をデフォルトとする（out-of-store は明示的退避路）」と
「想定ユースケース」home mode 節の最終項目。
