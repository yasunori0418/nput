# ADR-0043: `--json` 機械可読出力を niface 規約準拠にし、JSON 出力の niface 準拠を恒常原則とする

- ステータス: 採用
- 日付: 2026-07-06
- 関連: ADR-0033, ADR-0023, ADR-0031, ADR-0018, ADR-0042, ADR-0004, `docs/concept.md`, `docs/design.md`, `docs/spec.md`, niface specVersion 1（yasunori0418/niface）, yasunori0418/niface#1
- 改訂対象: ADR-0033 §1-3（独自エンベロープ `{"version":1,...}` を niface エンベロープ準拠へ）/ ADR-0023 §2（「エラーは stdout に畳み込まず stderr 専有」を再改訂）。ストリーム規律の骨子（stdout=機械可読専有・warning/error 常時 stderr）と終了コード表 0/1/2 は不変
- 起点: nput の niface 準拠化 grilling（2026-07-06）と、それを受けた niface 側 grilling による niface#1 の方針確定（batch エンベロープ + subject + §5 参照キー規約の 3 層化）

## 背景

ADR-0033 は `--json` を「全サブコマンド一律のグローバルフラグ・stdout に単一 JSON オブジェクト（`{"version":1,"command":...}`）」として導入する決定をした。これは nput 単独の独自エンベロープであり、当時は外部規格が存在しなかった。

その後、n プレフィックスのツール群（nput / nboot / nwrap / nherd / nshadow / ncompose）が **stdout / stdin の JSON パイプで合成される** ことを前提にした共通規格 **niface**（specVersion 1）が起こされた。エコシステムの北極星は「NixOS とは異なる Nix 版 Arch / Gentoo ——最小のコアとユーザーによる組み立て」であり、その実現手段が「**単一責務のツールを niface 規格のパイプで合成する**」ことである（→ niface `docs/ecosystem/overview.md`・本リポジトリ ADR-0004 の north-star）。nput はこのエコシステムで唯一 active なツールであり、ncompose が nput を含む各ツールの出力を固定順に合成・失敗時逆順 rollback する構想の**最初の適合実装**になる。

したがって nput の `--json` は、nput 単独の都合ではなく **niface 規約の適合ツールとして** 出力すべきである。ADR-0033 の独自エンベロープはこの相互運用の前提を満たさない。

niface#1 の grilling で niface 側の仕様も確定した（batch エンベロープ・subject・§5 の参照キー規約 3 層化。詳細は yasunori0418/niface#1）。これにより nput が niface に準拠するための決定を本 ADR で確定する。

## 決定

### 1. 恒常原則: nput の全 JSON 出力は niface 規約に準拠する（北極星要件）

- nput の `--json` 出力（**現在および将来のすべての機能**）は niface specVersion 1 規約に準拠する。単発の `--json` 機能の決定ではなく、**JSON 出力の niface 準拠を nput の恒常的な設計制約**とする。
- 今後の機能追加（`nput prune`・plan / verify・mkEnv 等）で機械可読出力を持つ場合、その出力は niface エンベロープ規約に収める。ツール固有の情報は niface の `info` 配下にのみ置く。
- 根拠は北極星: ncompose によるツール合成は「規格が契約（ツール間の会話は niface 規約のみに依存）」で初めて成立する。nput が独自形状に逸脱すると合成の前提が崩れる。**niface 準拠はエコシステム構築に向けた設計要件**であり、nput 内部の出力都合より優先する。
- concept.md（北極星節）・design.md（出力規約）・spec.md（出力ストリーム規律）に本原則を明記する（→ 影響）。

### 2. エンベロープと `mode`

- `--json` 指定時、stdout に **niface エンベロープを 1 文書だけ**書く（`specVersion` / `tool{name,version}` / `command` / `mode` / `status` / `dryRun` / `startedAt` / `finishedAt` / `errors` / `result{items,changes,info}`）。命名は camelCase・時刻は RFC 3339。
- `mode` は single / batch の唯一の判別子（必須）。単一主体コマンドは `mode:"single"` + `result`、複数主体（`--all`）は `mode:"batch"` + `results[]`（`result` XOR `results`）。
- `tool.version` は VERSION ファイル → ldflags 埋め込み（ADR-0042・`--version` 新設が前提）。
- niface の `specVersion`（出力規格）・`manifest.json` の `schemaVersion`（engine 入力契約）・`tool.version`（nput リリース）は**独立**に進化する 3 つのバージョンとして扱う。

### 3. item identity（subject-scoped）

- item の `id` は niface の `id = lowercase-hex(sha256(JCS(identity)))` で導出する。`identity = {kind, key}`。
- entry の identity は `kind="entry"`, `key={target}`（root 相対 target のみ）。**config 名は key に含めない**。
- niface#1 §5（参照キー規約 3 層）に従い、id 値は subject を跨いで衝突してよく、consumer は **`(tool.name, subject, id)` の 3 つ組**で参照を解決する。`subject` は id 導出に関与しない弱い識別子。
- `subject` は single モードで optional なトップレベル `subject:{name}`（config 名）、batch モードで各 sub-envelope に必須。

### 4. items / changes マッピングと reversible

- **items = フルインベントリ**: `apply` / `apply --dryrun` は manifest の全 entry を item として列挙する（変更の無い entry も `status:"success"`）。stale 除去された旧 entry も列挙する。
- **changes = 差分のみ**（niface: noop を含めない）: place / copy 新規 → `add`、replace / recopy → `modify`、stale 除去 / reset → `remove`。
- **reversible**: symlink の add/modify/remove と copy の新規配置 = `true`（世代 rollback / 単純除去で戻せる）。copy の上書き（`--recopy`）と削除（`reset`）= `false`（copy は世代外で hash 追跡せず旧内容を復元できない）。
- 可逆性は `change.reversible` のみで表現し、`W_IRREVERSIBLE` 等の警告コードは付けない（niface §4 が「consumer は `reversible:false` を警告として扱うべき」と定めており二重符号化になる・niface#1 F）。

### 5. read-only 列挙は `result.info` インベントリ（id 導出 item にしない）

- `list-generations` の世代・`gitignore` の対象パスは **副作用の無い列挙**であり、`result.info` 配下のツール固有インベントリとして持つ（niface#1 D/E）。id 導出 item にはしない（item は「処理単位の実行結果の記録」の定義のまま）。
  - `list-generations`: `info.generations = [{number, date, current}]`。世代番号はどの key にも入らない。
  - `gitignore`: `info.paths`（anchor 形 target）。**デフォルトの行指向 stdout 出力は不変**（`--json` は opt-in の第 2 契約・ADR-0033 の gitignore 二契約方針を継承）。
- これらのコマンドの `result.items` は空配列でよい。

### 6. エラーと終了コード

- エラーは **niface エンベロープに構造化して載せる**（item 非依存の全体エラーは `errors[]`、item 起因は `item.error`）。**同時に stderr の人間向けテキスト（既存の op + 対象パス wrap 規約）も常時併存**させる（niface §1 が stderr を診断チャネルとして許容）。これは ADR-0023 §2 / ADR-0033 §2 の「エラーは stdout に畳み込まず stderr 専有」を再改訂するもの。
- 終了コード表 0 / 1 / 2 は不変（POSIX・0=成功 / 非 0=失敗。1 = 一般エラー・`--all` 部分失敗、2 = `--dryrun` conflict は nput 内部の意味づけ）。niface の `status` は exit 0 → `success` / exit 1・2 → `error` に連動する。niface 消費側が依存してよいのは「0 ⇔ success / 非 0 ⇔ error」のみ。
- conflict は該当 entry を `item.status:"failed"` + `error.code:"E_NPUT_COLLISION"` で表し、1 件でもあれば `status:"error"`（`--dryrun` でも当該 item は failed）。

### 7. `--all` は batch エンベロープ

- `apply --all` / `list-generations --all` / `gitignore --all` は **常に** `mode:"batch"`（対象が N=0 / 1 でも。起動の性質で形状が決まる）。
- `results[]` の各要素は **完全な `mode:"single"` sub-envelope**（`specVersion` / `tool` / `command` / `mode` / `subject` を自身で持ち、切り出せば単独で valid）。`subject={config名}`。config 単位の build / lock 失敗（item 非依存）は該当 sub-envelope の `errors[]` に置く。top の `errors[]` は主体列挙自体の失敗のみ。
- top の `status` は集約（sibling に 1 つでも error があれば error）。空 batch（`results:[]`）は `success`。`--all --dryrun` の終了コード優先度（error 1 → conflict 2 → 0）は不変（ADR-0024）。
- **`gitignore --all --json` は cross-config dedup をしない**: 各 subject が自 config の paths を持ち、消費側が union + dedup して `.gitignore` を再構成する。一方 **テキスト既定出力は従来通り dedup + sort 済みの単一リスト**（ADR-0018 不変）。テキスト＝集約 / JSON＝per-config という非対称を受け入れる（JSON は「どの config 由来か」を保つ利点があり、機械側の dedup は自明）。

### 8. Go 依存・エラーコード・実装 gate

- `github.com/yasunori0418/niface/go`（Envelope 汎用型 + `DeriveID`）を依存に追加する。ADR-0033 が課した「stdlib-only で emit」制約は、niface/go 自体が stdlib-only の規格参照実装であること、および id 導出（JCS + sha256）を規格実装と共有して適合ベクタ（`id-vectors.json`）との乖離リスクを下げることを理由に緩和する。CLI 出力契約は cmd 層に閉じ、`engine.Result` 等を直接 marshal せず DTO 経由で niface 型へ詰め替える。
- エラーコード: ツール別 `E_NPUT_COLLISION` / `E_NPUT_BUILD`・警告 `W_NPUT_FOREIGN`（foreign symlink）。共通コードは `E_LOCK` / `E_IO` / `E_NOTFOUND` / `E_PERMISSION` を再利用する（niface §6 の二層命名）。
- `reset --json` は破壊的操作の確認を機械消費で扱えないため **`--yes` を必須**とし、無ければ prompt せず `status:"error"`・非 0 で fail fast する。
- **実装の着手**は niface の正式成果物（`spec/v1` 更新・`envelope-batch.schema.json`・go module の mode/batch/subject 型）の完成、および ADR-0042 の VERSION + `--version` を前提とする。本 ADR は決定の記録であり、実装は前提成果物の完成後に #130 以降で進める。

## 根拠

- **恒常原則にする理由**: niface 準拠を「今回の `--json` 機能」に閉じた決定にすると、将来機能が独自形状に逸れる余地が残り、ncompose 合成の前提（規格が契約）が崩れる。北極星に向けては「全 JSON 出力が niface 準拠」を制約として明文化しておく必要がある。
- **独自エンベロープを捨てる理由**: nput 単独の `{"version":1,...}` は niface 消費側（ncompose 等）が解釈できない。エコシステムの相互運用は規格準拠が唯一の前提。
- **エラーをエンベロープに載せる理由（ADR-0033 §2 の反転）**: niface はエラーを規格の一部（`errors[]` / `item.error`）として持つ。適合するには畳み込みが必要。stderr テキストは診断チャネルとして残すため人間向けの可読性は失わない。
- **read-only 列挙を info に置く理由**: 世代一覧・gitignore パスは実行結果ではなく列挙で、id 安定性の機構（§5）を要しない。item 化すると世代番号の key 問題等の無理が生じる。info インベントリが素直。
- **`--all` を常に batch にする理由**: 件数で形状が変わると消費側がコマンド形式から出力形状を予測できない。niface#1 が「起動の性質で mode が決まる」と定めた。

## 影響

- **`docs/concept.md`**: 北極星節に「nput は niface 規約でエコシステムに接続し、JSON 出力は niface 準拠」を追記。
- **`docs/design.md`**: 出力・終了コード規約節の「`--json` は将来送り」を「niface 準拠（→ ADR-0043）」へ更新。
- **`docs/spec.md`**: 出力ストリーム規律節に niface 準拠の `--json` 契約（エンベロープ / mode / エラーはエンベロープ + stderr 併存）を追記し、「`--json` は MVP では持たない」注記を削除。各コマンドの JSON ペイロード詳細は #130 以降で追記。
- **ADR-0033 / ADR-0023**: 本 ADR で改訂した箇所に blockquote 注記を書き戻す（同一 PR）。
- **実装（`cmd/nput/` ほか）**: niface/go 依存追加、エンベロープ emit、engine の full-inventory 化、レポート系の data-first 化。詳細は #130 / #131 / #132 / #164。

## 棄却した代替案

- **ADR-0033 の独自エンベロープを維持し niface は参照に留める**: 相互運用の前提を満たさず、北極星（ncompose 合成）に到達できない。
- **niface 準拠を今回の `--json` 機能に閉じた決定にする**: 将来機能の逸脱余地が残る。恒常原則として明文化する方を採る。
- **エラーを stdout の niface エンベロープに載せず stderr 専有のまま**（ADR-0033 §2 維持）: niface 適合を満たさない。
- **read-only 列挙を id 導出 item にする**: 世代番号の key 問題等の無理が生じ、niface#1 で却下済み。
- **`W_IRREVERSIBLE` 警告コードの追加**: `change.reversible` との二重符号化。niface#1 で却下済み。
