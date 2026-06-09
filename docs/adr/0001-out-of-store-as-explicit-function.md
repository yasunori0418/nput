# ADR-0001: out-of-store symlink を明示関数に降格し store link に統一する

- ステータス: 採用
- 日付: 2026-06-07
- 関連: `docs/concept.md`, `docs/spec.md`（entries スキーマ / `src`）

## 背景

初期設計では `src` の Nix 型（`builtins.typeOf src`）によって配置挙動を暗黙的に切り替えていた。
`string` 型を渡すと Nix ストアを経由しない out-of-store symlink（ローカルパスへのライブ symlink）になり、
これを「手元 dotfiles のライブ反映」という第一級の機能として concept で打ち出していた。

この暗黙分岐には次の問題がある。

- 同じ `src` フィールドの値が、型という見えにくい属性で全く異なる配置先（ストア vs ローカル FS）に化ける。
- Nix エコシステムの基本である「配置物はストア由来で再現性がある」という前提から外れる経路が、デフォルトで口を開けている。
- `path` リテラルを渡すと暗黙でストアにコピーされる罠（HM issue #2085 と同種）と区別がつきにくい。

## 決定

- **store link をコア・デフォルトとする。** `src` のデフォルト挙動は常に Nix ストアへの symlink。
- **out-of-store は明示関数 `nput.lib.mkOutOfStoreSymlink "/abs/path"` でのみ opt-in する。** 型ベースの暗黙分岐は廃止する。
- 関数はマーカー attrset を返し、`src` に渡す。`src` フィールドは 1 つのまま、型は `path | set | marker`。
- `mkOutOfStoreSymlink` は core lib（nixpkgs のみ依存）では **パスをマーカーに包むだけの純粋関数**とする。実際の link 生成は配置エンジンが担う（ADR-0003）。
- concept では out-of-store を headline から外し、「開発中の dotfiles をライブ編集したいときの明示的退避路」として降格する。

## 根拠

- 「store link に統一」という意図に最も忠実で、型マジックを排除できる。
- HM の `source = config.lib.file.mkOutOfStoreSymlink ...` と同じ「明示的に out-of-store を選ぶ」使い心地に揃う。
- out-of-store はライブな絶対パスへの symlink であり本質的にストア link ではない。「統一」は「デフォルト/コアをストアにし、out-of-store を明示的な例外に降格する」ことを意味する。

## 影響

- 初期 `docs/spec.md` の `src` 型表（`string` → out-of-store）を書き換える。
- concept.md の「src の型による反映タイミング」表を「store デフォルト + 明示関数」の語りに変更。
- out-of-store の内容は設計上ライブであり、世代管理ではリンク先マッピングのみ追跡する（ADR-0002）。

## 棄却した代替案

- **暗黙分岐を残しつつ関数も足す**: 同一目的の経路が 2 つになり「統一」と矛盾。
- **out-of-store 自体を廃止**: 依頼の「mkOutOfStoreSymlink 同等の関数を追加」と矛盾。ライブ編集ユースケースを失う。
- **専用フィールド `outOfStore`**: フィールドが増え `src` との mutually-exclusive バリデーションが必要になる。
