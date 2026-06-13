# ADR-0014: entries を target キーの attrset にし、手動 name と手動一意性チェックを廃する

- ステータス: 採用（2026-06-14 改訂: entry フィールド `mode` を `method` に改名 → ADR-0015）
- 日付: 2026-06-14
- 関連: ADR-0001, ADR-0008, ADR-0010, ADR-0013, ADR-0015, `docs/spec.md`, `docs/design.md`, `docs/concept.md`, `CONTEXT.md`
- 改訂対象: **ADR-0010**「entries を `attrsOf (submodule …)` でモデル化」の棄却を反転。entry submodule の `name` 必須フィールドと重複 `name` の `lib.throwIf` を廃止
- 参照: home-manager `modules/lib/file-type.nix`（`home.file` の識別子モデル）

> **2026-06-14 改訂注記（ADR-0015）**: 本文中の entry フィールド `mode`（`"symlink"` | `"copy"`）は **`method` に改名**された
> （unix file mode との誤読回避）。本文は当時の名のまま歴史的記録として残す（ADR-0008 の慣例）。現行スキーマは `method`（→ ADR-0015）。

## 背景

確定済みの entries API は `list of { name; src; subpath?; target; mode? }` で、`name` は **手動定義必須**かつ **`entries` 内で一意**（重複は `normalizeManifest` の `lib.throwIf`・→ ADR-0010）だった。これはユーザー体験を損なう。

- entry ごとに論理 `name` を手で考えて付ける必要がある。
- その一意性をユーザーが頭で管理し、衝突すると評価時エラーになる。

home-manager の `home.file` の実装を確認したところ、識別子を別フィールドで持たない。

```nix
# home-manager modules/lib/file-type.nix
types.attrsOf (types.submodule …)   # attrset。属性キーが識別子
target = mkDefault name;            # target は属性キー(name)から既定値を取る
```

`home.file` は **attrset で属性キーが識別子**になり、`target` はキーから自動導出される（明示上書き可）。**一意性は Nix の attrset 仕様（キー重複不可）でタダで担保**され、手動チェックは不要。

nput の entry `name` は実質「stale 除去の diff キー + 表示ラベル」でしかなく（`nput.<name>` の config / profile 名とは別物）、**target は本来一意でなければならない**（2 entry が同一 target に置けば衝突）。よって target を自然な識別子にできる。design.md が `name` 必須の根拠にした「index ベース自動命名は並び替えで名前が変わる」懸念も、順序非依存な target をキーにすれば満たされる。

## 決定

### 1. entries を `attrsOf (submodule …)`・キー = target にする（B1）

- `entries` を **attrset** にし、**属性キー = root 相対の target パス**とする。
- entry submodule のフィールドは `src`（必須）/ `subpath`（default `"."`）/ `target`（**省略可・既定 = 属性キー**）/ `mode`（default `"symlink"`）。**`name` フィールドは廃止**。
- `target = mkDefault name`（`name` = 属性キー）。属性キーをそのまま target とするのが canonical。キーを論理ラベルにして `target` を明示上書きするのも可（home-manager と同じ逃げ道）。

```nix
# canonical: キー = target
entries = {
  ".claude/skills/nix" = { src = inputs.claude-skills; subpath = "skills/nix"; };
  ".local/share/nvim/site/pack/foo/start/foo" = { src = inputs.vim-foo; };
};
```

### 2. 一意性は Nix attrset が native 担保、手動 throwIf を撤廃

- name 一意性の `lib.throwIf`（→ ADR-0010）を**撤廃**する。同一キー（= 同一 target）は Nix の attrset 仕様で定義できない。
- 「別名キーで同一 target」（target 明示上書きで衝突させる）だけは attrset では防げないため、その場合のみ engine 実行時 conflict 検出（既存）に委ねる。
- copy + out-of-store marker / `systemRoot` 未実装の `lib.throwIf`（→ ADR-0013）は**残す**。

### 3. stale 除去の identity・manifest の identity = 解決後 target

- stale 除去の diff キーは **解決後 target**。属性キーを上書きしても、実際に配置されるのは target なので identity は target に一本化する。
- `manifest.json` v1 の `entries[]` から **`name` を落とす**（→ ADR-0013 で pin した v1 をリリース前に更新）。identity は `target`。symlink farm の GC アンカー名は target をサニタイズして用いる（home-manager の `storeFileName` 相当）。

### 4. モジュールオプションも attrsOf に揃える

- `modules/common.nix` の `nput.entries` を `attrsOf (submodule …)`（`lib/types.nix` と共有）にする。`nput.<name>` の config 階層が既に attrset キー方式なので、entry 階層も揃い構造の非対称が解消する。

### 5. 動的生成は `listToAttrs` 等で組む

- 素の list `map` から attrset 構築（`builtins.listToAttrs` / `lib.listToAttrs` / `lib.mapAttrs'`）へ移る。キー（target）に名前を補間する。

```nix
entries = builtins.listToAttrs (map (n: {
  name  = ".claude/skills/${n}";          # listToAttrs のキー = target
  value = { src = inputs.claude-skills; subpath = "skills/${n}"; };
}) names);
```

## 根拠

- **UX**: 手動 name 命名と手動一意性管理という 2 つの負担を同時に消す。識別子を考える行為自体が無くなる。
- **home-manager 整合**: 実証済みの `home.file` モデルと同型で、利用者の既知のメンタルモデルに乗る。
- **native 一意性**: Nix attrset がキー重複を許さないため、検査コードを持たずに一意性が成立する。
- **identity の自然さ**: target は配置先として元々一意であるべき値で、stale 除去の diff キーとして過不足ない。`name` 必須の旧根拠（index 不安定回避・design.md）も target キーで満たす。
- **構造の対称**: config 階層（`nput.<name>`）と entry 階層がどちらも attrset キー方式に揃う。

## 影響

- **`docs/spec.md`**: entries スキーマを attrsOf（キー = target・`name` 廃止）に変更。manifest v1 の `entries[]` から `name` を削除し identity = target に。動的生成節を `listToAttrs` に書き換え。エラー仕様の「重複 name」を「同一キー = Nix で表現不可 / 別名キーで同一 target = engine conflict」に更新。`normalizeManifest` の name 一意性 throwIf を削除。モジュールオプション `nput.entries` を attrsOf に。全 entries 例を変換。
- **`docs/design.md`**: 「name フィールドを必須にする理由」節を「target をキーにする理由」に差し替え。entries スキーマ表・`nput.entries` 型・全 entries 例を変換。
- **`docs/concept.md`**: entries 例を target キー attrset に変換。
- **`CONTEXT.md`**: `entry / entries` 定義を「target をキーとする attrset。キーが識別子」に更新。`name` 一意性の記述を除去。
- **ADR-0010**: 「entries を attrsOf でモデル化」棄却・「`name` 必須」・「重複 name の throwIf」を本 ADR が反転した旨の改訂注記を足す。
- **実装フェーズ**: `lib/types.nix`（attrsOf submodule・`target = mkDefault name`）、`lib/manifest.nix`（name throwIf 削除・target identity）、`modules/common.nix`（attrsOf 共有）。

## 棄却した代替案

- **A: `name` を optional にし target 既定（list API 維持）**: 最小変更だが list と attrset の二層で識別子モデルが不揃いのまま残り、一意性も throwIf を温存する。native 担保の利が得られない。
- **B2: キー = 論理名・target 明示必須**: 短いラベルを保てるが「target を自動解決する」という主目的を達せず、target 必須記述が残る。キー上書きで B1 の中に内包できる。
- **C: `name` 廃止・target を list 内 identity に**: list のまま target を identity にできるが、一意性の native 担保（attrset）が得られず、結局 throwIf か engine 検出が要る。
- **現状維持（手動 name + throwIf）**: UX 負担が残る。home-manager より劣る識別子モデルを積極採用する理由がない。
