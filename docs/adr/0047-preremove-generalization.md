# ADR-0047: 配置前除去（PreRemove）の一般化 — 配置を塞ぐ自己記録 stale の migration と空 dir の除去

- ステータス: 採用
- 日付: 2026-07-12
- 関連: ADR-0002, ADR-0003, ADR-0006, ADR-0015, ADR-0017, ADR-0020, ADR-0031, ADR-0046, `docs/spec.md`, GitHub Issue #175, #172
- 改訂対象: **ADR-0046**「自己記録の祖先 symlink 配下ネストを許可する」を「配置を塞ぐ自己記録 stale 全般（実 dir target・method 変更）」へ一般化。**ADR-0015 §2**「前世代 manifest 記録の symlink は置換可（silent・後勝ち）」の判定を method 横断（symlink→copy）へ拡張。
- 起点: epic #172 の grilling セッション（2026-07-12）で確定した D1〜D6 のうち、PreRemove 一般化本体（D2, D3, D5）

## 背景

ADR-0046 は「配置 target の**祖先**が自己記録 stale symlink のとき、それを配置前に除去（PreRemove）してネスト移行する」ことだけを許した。しかし home-manager との意味論突合（epic #172 背景）で、それだけでは足りない 2 つの実運用パターンが判明した。

1. **target 自身が実 dir のケース**: 前世代が `foo/main.sh`（per-file）を配置し、新世代がその親 `foo` 自体を dir symlink にしようとすると、`foo` は実 dir として存在しているため「target already has an existing file/directory」で conflict になる。ADR-0046 は祖先の緩和しか扱わないため、**target そのもの**が実 dir のケースは救えない。2026-07-12 の実障害（`.claude/hooks/<name>/main.sh` → `.claude/hooks/<name>` の dir symlink 化）はこのパターン。
2. **method 変更のケース**: 同一 target で `method` を `symlink` から `copy` へ変えると、前世代が置いた symlink が target を占有しており、copy 側の分類は「structure mismatch」または「foreign」として conflict / skip になる。symlink はユーザーデータを持たないため、これも自動移行して安全なはずだが、現状は手動 `rm` が要る。

nput は manifest 記録を持つため、これらも「自分が置いて自分が消す」の範囲内であれば安全に自動化できる。ADR-0046 の枠組み（`Plan.PreRemove` による配置前除去）を、対象を「祖先」から「配置を塞ぐ自己記録 stale 全般」へ広げることで両方を解決する。

## 決定

### 1. 実 dir target は「配下全 leaf が recorded∧stale または空 dir」なら migration

symlink method の entry の target に実 dir が既存でも、次の条件を**すべて**満たせば conflict にせず migration する。

- 配下の各 leaf（任意深さ）が、以下のいずれか:
  - **recorded ∧ stale**（前世代 manifest が記録した symlink・on-disk が記録 dest と一致・次世代に無い）→ 配置前に Unlink
  - **空の sub dir**（由来を問わない。rmdir は空でしか成功せず、データ損失が原理的にゼロなため、nput が作った dir かどうかを判別する必要がない）→ 配置前に Rmdir
- 上記以外の leaf が 1 つでもあれば、**dir 全体**を conflict にする（部分除去はしない）:
  - 中身のある実 file・実 dir（foreign or 判別不能）
  - foreign symlink（記録なし / 記録 dest と不一致）
  - 次世代にも同じ target が entry として残る symlink（自己矛盾 manifest）

判定は plan 時に「全 leaf 除去後に dir チェーン全体が空になる」ことを先に確定させてから行う。除去は子から親へ（bottom-up）、Unlink を先に・Rmdir を後に並べる。

copy method の entry には適用しない（copy target は本 ADR のスコープ外・place-once の対象外構造）。

### 2. method 変更は symlink→copy のみ自動移行、copy→symlink は非対称に conflict のまま

同一 target で `method` が変わったとき:

- **symlink → copy**: 前世代が記録した symlink で on-disk が記録 dest と一致するなら、配置前に Unlink してから place-once copy を新規配置する（silent）。symlink はユーザーデータを持たないため損失ゼロ。
- **copy → symlink**: **自動移行しない**。copy はユーザーが編集した可能性のある実データであり、symlink 化のために黙って消すのは危険。従来通り「target already has an existing file/directory」で conflict のまま。`--backup`（#169・別 issue）が脱出ハッチになる。

readlink が記録先と一致しない（drift）場合は、symlink→copy 方向でも自動移行せず通常の foreign 判定（skip + `WarnCopyForeign`）へフォールバックする。

### 3. drift 時の意味論は PreRemove 系全体で一律 error 停止・最終段 removeStale は不変

ADR-0046 §3 で導入した「PreRemove は最終段の removeStale と違い、drift を skip せず error で停止する」非対称を、一般化した対象（実 dir 配下の leaf Unlink・空 dir の Rmdir・method 変更の Unlink）**全てに適用**する。

- **Unlink drift**（symlink が記録 dest と不一致 / 消失）→ error 停止。
- **Rmdir drift**（rmdir 直前に dir が非空になっていた = ENOTEMPTY/EEXIST）→ 事前の空チェックはせず、`os.Remove` の結果そのものを再検証として使う（TOCTOU 安全）。非空なら error 停止。

理由は ADR-0046 §3 と同根: 後続の配置は「PreRemove で対象が消えている」ことを前提に無条件で行われるため、drift を skip して配置を続行すると、drift 後の実体（例: foreign な書き込み可能 dir）の配下へ書き込んでしまい、ADR-0015 §4 が防いだ store 汚染を再び開く。冪等な再実行で現状の FS に対し再計画され収束する（→ ADR-0017）。

最終段 `removeStale`（プレースメント本流の最後の独立 stale 除去）は本 ADR で意味論を変えない。drift は従来通り warning で残置する。

### 4. データモデル: RemoveAction に Kind を追加

`planner.RemoveAction` に `Kind`（`RemoveUnlink` / `RemoveRmdir`）を追加する。`RemoveUnlink` は従来通り `Entry` を伴う記録済み symlink の除去。`RemoveRmdir` は `Entry` を持たない空 dir の除去（rmdir に再検証すべき manifest 記録が無いため、直前の `os.Remove` 結果自体を再検証として扱う）。

実行順序は **スライス順**で表現する: planner は子を親より先に積む（Unlink → 深い dir から浅い dir への Rmdir の順）。同一 target の重複除去は ADR-0046 由来の `preRemoved` map をそのまま使って dedup する。

`planner.FS` に `ReadDir(path string) ([]os.DirEntry, error)` を追加し、実 dir 配下の走査を可能にする。走査は lstat ベースで symlink の中へ降りない（ADR-0046 §2 と同じ安全規則）。

### 5. PreRemove は空祖先ディレクトリ剪定（#174）を呼ばない

`preRemove`（engine 実行段）は、Unlink・Rmdir どちらの成功時も #174 の `pruneEmptyAncestors` ヘルパを呼ばない。理由は、PreRemove の各アクションは必ず直後の `place` / `materializeCopies` による再配置を前提としており、剪定してもその場で作り直されるだけの無駄な往復になるため。加えて、一般化後は 1 バッチ内に複数の Unlink/Rmdir が bottom-up 順で積まれており（`classifyDirMigration` が子から親まで明示的に列挙する）、途中のアクション成功時に `pruneEmptyAncestors` を呼ぶと、まだ実行していない後続の明示的 Rmdir アクションの対象を先取りして消してしまい、レポート（`result.Pruned`）が不正確になる、または祖先チェーンの外側（placement target とは無関係な祖先）まで意図せず剪定してしまう。

空祖先ディレクトリ剪定そのものは #174 の役割のまま **最終段 `removeStale` と `reset`** に閉じており、PreRemove 段は「配置を妨げる自己記録 stale を計画通りに除去する」責務に純化する。

## 根拠

- **空 dir を由来問わず許す**のは、rmdir が空でしか成功せずデータ損失が原理的にゼロなため。ユーザーは config でその target を（symlink または copy として）宣言済みであり、nput が作ったかどうかの判別コストを払う理由がない。
- **中身のある foreign が 1 つでも dir 全体を conflict にする**のは、部分除去がユーザーの意図しない中間状態を生むため。「移行できるところだけ移行する」設計はエラーの見落としを誘発する。
- **symlink→copy のみ自動化し copy→symlink はしない**のは非対称だが、データ損失リスクの非対称性を反映している。symlink はデータを持たないが copy はユーザー編集を含みうる。この非対称は ADR-0020（copy の place-once・ユーザー管理）と一貫する。
- **drift 一律 error 停止**は ADR-0046 §3 の踏襲。skip 続行は `MkdirAll` の symlink 追従による store/foreign 汚染窓を開く。
- **`RemoveAction.Kind` によるデータモデル拡張**は ADR-0003（planner=純粋にプラン決定 / engine=FS 反映）の役割分担に沿う。Rmdir という「Entry を伴わない除去」を明示的な型として表現することで、engine 側の実行ロジックが分岐に迷わない。

## 影響

- **`internal/planner/planner.go`**: `RemoveAction.Kind`（`RemoveUnlink`/`RemoveRmdir`）追加。`FS.ReadDir` 追加（`osFS` 実装含む）。`Compute` の実 dir 分岐（symlink method・target が実 dir のとき `classifyDirMigration` で判定）追加。`classifyCopy` に method 変更（symlink→copy）の migration 分岐を追加。
- **`internal/engine/staleremove.go`**: `preRemove` を `RemoveAction.Kind` で分岐。Rmdir は事前空チェックなしで `os.Remove` の結果を再検証として使う。Unlink・Rmdir とも drift は一律 error 停止。`preRemove` は #174 の `pruneEmptyAncestors` を呼ばない（§5）。
- **`internal/engine/drift.go`**: gen-skip 経路の `plan.PreRemove` 空不変条件チェックを一般化後の全 PreRemove 発生源に対応するようコメント更新（判定ロジック自体は変更なし: 全発生源が manifest 差分を伴い derivation を変えるため引き続き gen-skip 経路に到達しない）。
- **`internal/engine/engine.go`**: dryrun の PreRemove レポートを Kind で分岐（Rmdir は `Entry` を持たないため `result.Pruned` へ、Unlink は従来通り `result.Removed` へ）。
- **`docs/adr/0046-*.md`**: 改訂対象の back-reference 注記を追記。
- **`docs/adr/0015-*.md`**: §2 の method 横断拡張の back-reference 注記を追記。
- **`docs/spec.md`**: 配置動作仕様（実 dir target の migration・method 変更）・エラー仕様表・実行フローへ反映。
- **cross-epic**: #173（Rollback への PreRemove 配線）は本 ADR の一般化 executor（`preRemove`）をそのまま利用できる設計にしてある。マージ順による rebase 調整は後工程。#176（conflict 全件報告）はスコープ外のまま。#168（undo ジャーナル）は本 ADR の Unlink/Rmdir 両方を undo 対象に含める前提で設計する。

> **2026-07-12 改訂注記（ADR-0044）**: 本 ADR が影響節で予告した「#168（undo ジャーナル）は本 ADR の Unlink/Rmdir
> 両方を undo 対象に含める前提で設計する」は #168（ADR-0044）で実装された。PreRemove の `RemoveUnlink`（記録
> symlink 除去）・`RemoveRmdir`（空 dir 除去）はいずれも apply / Rollback 途中失敗時にインメモリ undo ジャーナルで
> 自動巻き戻しされる（Unlink → 記録 dest で symlink 再作成・Rmdir → `os.Mkdir` で再作成）。本 ADR の migration 判定
> 条件（D1〜D5）・drift 時 error 停止（D3）自体は不変（→ ADR-0044）。

> **2026-07-12 改訂注記（ADR-0045）**: 本 ADR §2 が「copy→symlink は自動移行しない。`--backup`（#169・別 issue）が
> 脱出ハッチになる」と予告した動作は #169（ADR-0045）で実装された。`apply --backup` は copy→symlink の method 変更
> だけでなく、本 ADR がスコープ外のまま残した「配置を塞ぐ記録外（foreign）実体」全般（symlink モードの foreign
> 実 file/dir・実 dir migration 失敗・copy 構造不一致・copy foreign skip）を、ユーザーの明示 opt-in のもとで
> `<target>.<suffix>` へ rename 退避してから配置する。本 ADR の migration 判定条件（D1〜D5。「foreign は対象外」の
> 既定・祖先 symlink conflict の扱い）自体は不変 — `--backup` はこれらの conflict を消すのではなく、conflict の代わりに
> 退避という選択肢を追加するオプトイン層である（→ ADR-0045）。

## 棄却した代替案

- **全面反転（home-manager 型の remove→place の完全対称・2 段化）**: 依存除去のみ前段化する現行方式（ADR-0006 の本流順序は不変）ではなく、独立 stale 除去まで含めて全面的に「先に全部消してから全部置く」設計にすると、rename（target A 削除 + B 追加）のような独立除去まで前段化することになる。除去後・配置前でクラッシュすると新旧どちらのパスにも実体が無い瞬間が生じる（home-manager はこれを受容しているが、nput は成功時の挙動を home-manager と同一に保ちながらクラッシュ耐性は上位互換にしたい）。依存除去のみ前段化すれば、この窓は開かない。
- **現状維持 + 実 dir 例外のみアドホックに追加**: 移行パターン（祖先 symlink・実 dir target・method 変更）ごとに個別の特例コードを積み上げると、パターンが増えるたびに例外が累積し、保守不能になる。`RemoveAction.Kind` によるデータモデル拡張で一般化した方が、将来の対象拡張にも耐える。
- **自動作成した親 dir を manifest に記録する（schema v2）**: 「nput が作った dir かどうか」を manifest で追跡すれば実 dir migration の判定が楽になるが、schemaVersion v1 の契約を破る（MVP は v1 のみ・ADR-0015 §7）。空 dir は由来を問わず migration 対象にできる（rmdir の性質上データ損失ゼロ）ため、記録なしで v1 契約のまま解決できる。
- **foreign symlink 混在まで除去許容**: 他ツール管理の symlink を巻き込む可能性があり、ADR-0015 §4 の安全策が崩れる。
- **PreRemove drift を warn+skip（home-manager 型）にする、または再計画リトライを内蔵する**: warn+skip は drift 後の実体経由での store/foreign 汚染窓を開く（ADR-0046 §3 と同根）。再計画リトライの内蔵は複雑化する上、冪等再実行（ADR-0017）による収束と役割が重複する。
- **copy→symlink も自動移行する**: ユーザーが編集済みの copy データを黙って失う可能性があり、place-once（ADR-0020）の「ユーザー管理」思想と衝突する。
- **内容同一なら上書きを許す（`cmp -s` 型の判定）・per-entry の `--force` フラグ**: symlink 配置に「内容同一」という概念はほぼ意味を持たない（dest パスの一致で十分）。per-entry force は #169 の `--backup` と目的が重複する。実需が具体化したら別 ADR で検討する。
