# ADR-0019: パス安全性・copy の farm/gitignore 扱い・空 entries の挙動を確定する

- ステータス: 採用
- 日付: 2026-06-14
- 関連: ADR-0002, ADR-0005, ADR-0006, ADR-0010, ADR-0013, ADR-0014, ADR-0016, `docs/spec.md`, `docs/design.md`, `docs/concept.md`
- 改訂対象: **ADR-0010**（normalizeManifest の検査）にパス安全性検査を追加。**ADR-0006**（GC アンカー = manifest + farm）に copy 非アンカーを明記。**ADR-0005**（project mode ephemeral）に copy target の扱いを明記
- 起点: ADR-0018 後の横断検証で surfaced した「3 巡で詰め切れていなかった」残 4 点

## 背景

ADR-0016 / 0017 / 0018 で実装前残セマンティクスの大半が固まったが、docs 横断検証で次の 4 点が**記述ゼロ**だった。

1. **target / subpath のパス安全性が未定義**。target は「root 相対」、subpath は「src 内相対」だが、`..` で root / src の
   外へ出る入力や絶対パスの検証がどこにも無い。`.claude/../../etc/x` のような target が通ると root 外へ書ける。
2. **copy entry の store src を symlink farm が GC アンカーするか未定義**。farm アンカーは「store-backed entry」と書くが、
   copy は place-once でマテリアライズ後は store src が不要。
3. **project mode の copy target が ephemeral か committed か未定義**。project mode = ephemeral（untracked）だが
   copy = ユーザー所有・編集可。`gitignore` が copy target を含めるかで意味論が変わる。
4. **空 entries（`entries = {}`）の挙動が未定義**。apply すると前世代の全 symlink が stale 除去され全クリアになる。

## 決定

### 1. target / subpath の絶対パス・`..` エスケープを eval 時に拒否する

- `normalizeManifest` の `evalModules` / `lib.throwIf` で **target と subpath を検査**する（ADR-0010 の検査ゲートに追加）。
  - **絶対パス拒否**: target / subpath が `/` 始まりならエラー（target は root 相対・subpath は src 内相対が不変条件）。
  - **`..` エスケープ拒否**: `filepath.Clean` 相当で正規化し、`..` で root（target）/ src（subpath）の外へ出るものを拒否。
- eval 時に弾くため CLI / module 両経路で効き（ADR-0010 の「単一検査ゲート」と一貫）、engine まで不正パスが漏れない。
- root 自体の実体パスは実行時解決（マーカー）だが、**target の静的な `..` / 絶対判定は root 値に依らず eval 時に可能**。

### 2. copy entry は symlink farm の GC アンカーを持たない

- farm の GC アンカーは **symlink method の store-backed entry 限定**とし、**copy entry の store src はアンカーしない**。
- copy は place-once でマテリアライズ後は store から独立（世代外・ADR-0002）なので、store src を掴み続ける必要がない。
  アンカーすると不要に store を保持し GC を妨げる。copy の store src は `nix-collect-garbage` で解放されてよい。
- copy entry は `manifest.json` には記録する（orphan 警告・stale 判定のため）が、farm アンカーは持たない。

### 3. project mode の copy target は ephemeral 扱い（gitignore に含める）

- project mode = ephemeral（untracked）の不変条件を優先し、**copy target も ephemeral 扱い**とする。
- `gitignore`（`<name>` / `--all`）は **method を区別せず全 target を列挙**する（copy target も含む）。
- 各 clone で copy は place-once で再マテリアライズされ、**編集は clone local / 使い捨て**（`git clean` で消える）。
  「project mode の配置物はコミットしない」concept と整合する。docs に「project mode の copy 編集は clone local」と明記する。
- copy を committed（vendoring）にしたい場合は nput の責務外（手動コミット）とし、project mode の ephemeral 原則を崩さない。

### 4. 空 entries は正当な全クリアとして許可する（警告なし）

- `entries = {}`（空 manifest）の apply を「**この config の配置を全撤去する**」正当な表現として許可する。
- 保守的 stale 除去（nput 管理 symlink のみ・実ファイル不可触・ADR-0002）なので安全。冪等・予測可能で特別扱い不要。
- warning も eval エラーも出さない。spec に「空 entries は全 stale 除去（config の配置を消す手段）」と明記する。

## 根拠

- **パス安全性の eval 拒否**は root 外書き込みという最も危険な未定義挙動を構造的に防ぐ。eval ゲート（ADR-0010）に寄せることで
  CLI / module 両経路を一度に守れる。静的に判定可能な絶対 / `..` を実行時まで遅延する理由がない。
- **copy 非アンカー**は copy の「世代外・place-once」性質（ADR-0002）から必然。アンカーは store を無駄に保持するだけ。
- **copy ephemeral**は project mode の ephemeral 原則（ADR-0005）を全 method に一貫させる。method ごとに tracked/untracked を
  分けると「project mode は untracked」という単純な不変条件が壊れ、place-once と git 管理の二重管理を生む。
- **空 entries 許可**は「entries = {} は全撤去」という素直な意味を保つ。最低 1 entry 必須にすると正当な撤去手段を失う。

## 影響

- **`docs/spec.md`**:
  - 入力検査（`normalizeManifest`）節に target / subpath の絶対パス・`..` エスケープ拒否を追記。
  - エラー仕様表に「target / subpath が絶対パス」「target / subpath が `..` で root / src を越える」の行を追加。
  - manifest.json の symlink farm 対応節に「copy entry は farm アンカーを持たない（store src は GC 可）」を追記。
  - `gitignore` 節に「method を区別せず全 target を列挙・copy target も含む・project mode の copy 編集は clone local」を追記。
  - 配置動作 / 世代管理に「空 entries は全 stale 除去（config の配置を消す手段・警告なし）」を一行注記。
- **`docs/design.md`**: copy の世代外表 / farm 記述に copy 非アンカーを反映（任意）。
- **`docs/concept.md`**: copy の project mode 利用に「編集は clone local」を反映（任意）。
- **ADR-0010**: 改訂注記で「normalizeManifest に target / subpath のパス安全性検査（絶対 / `..` エスケープ拒否）を追加」。
- **ADR-0006**: 改訂注記で「GC アンカー = symlink store-backed entry 限定。copy entry は farm アンカーしない」。
- **ADR-0005**: 改訂注記で「project mode の copy target も ephemeral（gitignore に含む・編集は clone local）」。
- **実装フェーズ**: `lib/`（normalizeManifest のパス検査 throwIf・farm 構築で copy を除外）、`cmd/nput`（gitignore は全 target）、
  `internal/`（空 manifest の全 stale 除去）。

## 棄却した代替案

- **パス検査を engine 実行時のみ**: dryrun / CI gate でしか見えず、eval 時に弾ける静的不正を遅延する。
- **パス検査を二段（eval + engine）**: 最も堅牢だが、静的判定で十分な絶対 / `..` に engine 側の重複検査は過剰。
  （root 解決後の最終パスが root 配下かは engine の責務として残せるが、本 ADR の主眼は eval ゲートでの早期拒否。）
- **copy も farm アンカー**: 一貫するが世代外の copy に store 保持を強い GC を妨げる。
- **copy target を committed（gitignore 除外）**: 編集共有はできるが project mode の ephemeral 原則と矛盾し二重管理を生む。
- **空 entries を warning / eval エラー**: 誤クリア検知にはなるが、意図的な全撤去という正当用途を阻害 / 毎回ノイズ。
