---
id: "ADR-0040"
type: adr
name: "root marker に `subdir` 適用形を追加する（実体は target 前置の eval 時糖衣）"
status: 採用
origin: "次期マイルストーン追加計画の grilling（2026-07-04）。「root マーカーの解決位置を宣言的にずらしたい」要望を、grilling で「root 配下のサブディレクトリを基点にする」意図（配下方向）と確認した"
justifies:
  - "REQ-37b56673-6e40-4a1b-a2a7-5d3c084e3e66"
revises:
  - "ADR-0007"
references:
  - "ADR-0004"
  - "ADR-0014"
  - "ADR-0017"
  - "ADR-0019"
---
# ADR-0040: root marker に `subdir` 適用形を追加する（実体は target 前置の eval 時糖衣）

- ステータス: 採用
- 日付: 2026-07-04
- 関連: ADR-0004, ADR-0014, ADR-0017, ADR-0019, `docs/spec.md`, `lib/types.nix`, `lib/manifest.nix`
- 改訂対象: ADR-0007 の 3 root マーカー（projectRoot / homeRoot / systemRoot）の形を「引数適用可能（`homeRoot { subdir = ...; }`）」へ拡張（決定の反転なし・単体使用は不変）
- 起点: 次期マイルストーン追加計画の grilling（2026-07-04）。「root マーカーの解決位置を宣言的にずらしたい」要望を、grilling で「root 配下のサブディレクトリを基点にする」意図（配下方向）と確認した

## 背景

配置基点を root 配下のサブディレクトリ（例: `$HOME/.config`）に置きたい場合、現状は全 entry の `target` に共通プレフィックスを書く（`target = ".config/nvim"` 等）しかない。config 全体が同一プレフィックスを共有するケース（`~/.config` 専用 manifest 等）では宣言の重複になる。

既存機構との関係: `--root` フラグは実行時の一律上書き（ADR-0017）、絶対パス文字列は eval 時固定の fixed root であり、いずれも「マーカーの実行時解決 + 宣言的な配下オフセット」は表現できない。

設計にあたり重要な事実: **「基点を root/sub にする」ことと「全 target に `sub/` を前置する」ことは nput の意味論で完全に等価**である。stale 除去の diff キー・foreign symlink 判定・祖先 symlink walk・`gitignore` の anchor 形はすべて target 文字列で動くため、root を実際に動かす必要がない。root を本当に動かす実装を選ぶと、`manifest.json` に新フィールドが要り（旧 engine が黙って無視すると誤配置になるため schemaVersion bump が必須）、profileDir キーイングにも「同名 config が subdir 違いで世代系列を共有する」footgun（ADR-0023 §背景 4 と同型）が生まれる。糖衣ならどちらも構造的に発生しない。

なお grilling では「root より親方向（外）へずらす」解釈も検討したが、採らない（ADR-0019 のパス安全性を迂回する escape hatch になる。→ 棄却案）。

## 決定

### 1. API = marker の関数適用形 `homeRoot { subdir = "..."; }`

- 3 マーカー（`projectRoot` / `homeRoot` / `systemRoot`）を `__functor` 付き attrset にし、**単体でも従来通り marker、適用しても marker** を両立する:

  ```nix
  root = nput.lib.homeRoot;                        # 従来形（不変）
  root = nput.lib.homeRoot { subdir = ".config"; } # 適用形（新設）
  ```

- 適用結果は `subdir` フィールドを足した marker attrset。`mkOutOfStoreSymlink` と同じ「関数でマーカーを作る」既存パターンの延長で、`lib/types.nix` の `rootType` は optional な `subdir`（相対パス文字列）を許容するだけ。
- fixed root（絶対パス文字列）には本機構を提供しない（パス連結で直接表現できるため不要）。

### 2. 実体 = `normalizeManifest` での target 前置糖衣

- `normalizeManifest` が `subdir` を検出したら、**全 entry の正規化後 target に `subdir` を前置**して manifest を生成する。root の解決値・`rootKind` は一切変えない。
- したがって **engine 変更ゼロ・`manifest.json` スキーマ変更ゼロ（schemaVersion 据え置き）**。profileDir キーイングも不変（root が動かないため）。`gitignore` の anchor 出力・stale 除去・衝突検出（ADR-0024 §5 / ADR-0035 / ADR-0038）はすべて前置後の target で自然に正しく動く。
- `subdir` には ADR-0019 のパス安全性検査をそのまま適用する: 絶対パス（`/` 始まり）はエラー、正規化で `..` により root の外へ出るものはエラー。既定は「なし」（前置しない）。

## 根拠

- **糖衣にする理由**: 上記の等価性により、root を実際に動かす実装は複雑性（スキーマ bump・キーイング footgun・engine 分岐）だけを増やして何も得ない。糖衣は実装が `normalizeManifest` の 1 箇所に閉じ、既存の検査・意味論が全部そのまま流用できる。
- **marker 関数化を選ぶ理由**（grilling で確定）: 「root をずらす」というユーザーの心的モデルに書き味が一致する。`__functor` により単体使用の互換を壊さず、`rootType` の変更も optional フィールド 1 つに収まる。
- **fixed root に提供しない理由**: `"/abs/path"` を渡すユーザーは `"/abs/path/sub"` と書けばよく、マーカーのような「実行時解決とオフセットの分離」問題が存在しない。

## 影響

- **`lib/default.nix` / `lib/manifest.nix`**: 3 マーカーの `__functor` 化・`normalizeManifest` の前置糖衣・`subdir` の pathChecks 適用。
- **`lib/types.nix`**: `rootType` に optional `subdir` を許容。
- **`docs/spec.md`**: root 節に適用形の構文・糖衣である旨（「root は動かない・target 前置と等価」）・`subdir` のエラー仕様を追記。
- **`CONTEXT.md` / `docs/glossary.md`**: root マーカーの定義に適用形を追記。
- **ADR-0007**: 改訂注記（マーカー形の拡張）を同一 PR で追記。
- **テスト**: nix-unit（前置の正規化・`..` / 絶対パス拒否・単体 marker の不変性）・namaka snapshot。

## 棄却した代替案

- **親方向（root の外）へのオフセット**: ADR-0019 が eval 時に拒否している「root の外への配置」を意図的に開ける escape hatch となり、保守的 stale 除去が守る「root 配下しか触らない」性質を破る。ユースケースも具体化しなかったため不採用（grilling で配下方向の意図と確認済み）。
- **`mkManifest` の引数（`targetPrefix`）**: 実装は同等に単純で fixed root にも一様に効くが、「root をずらす」心的モデルとの一致で marker 適用形を選んだ（grilling で確定）。意味論は同一のため、将来必要なら別名 API として追加できる。
- **root を実際に動かす（manifest に subdir フィールド追加）**: schemaVersion bump 必須・profileDir キーイング footgun・engine 分岐増。糖衣との機能差はゼロで複雑性だけが残る。
- **何もしない（target プレフィックスで書く現状維持）**: 機能的には成立するが、config 全体の共通基点という宣言意図が entries に散る。糖衣のコストが十分小さいため採用に値する。
