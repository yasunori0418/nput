# ADR-0044: apply 途中失敗の完全巻き戻し — インメモリ undo ジャーナル

- ステータス: 採用
- 日付: 2026-07-12
- 関連: ADR-0002, ADR-0006, ADR-0015, ADR-0017, ADR-0020, ADR-0031, ADR-0046, ADR-0047, `docs/spec.md`, GitHub Issue #168, #172
- 改訂対象: **ADR-0046 §5**「#168 に hard 依存させず本 ADR 単独で出荷する」/ **ADR-0047 影響節**「#168（undo ジャーナル）は本 ADR の Unlink/Rmdir 両方を undo 対象に含める前提で設計する」を実装として満たす。両 ADR の PreRemove（祖先 symlink・実 dir target・method 変更）の意味論自体は不変。
- 起点: 旧 epic #167 から統合された epic #172 の grilling セッション（2026-07-11・2026-07-12）で確定

## 背景

現状、apply が途中で失敗すると世代は未コミットのまま（`nix-env --set` に到達しない）だが、**FS には部分的な配置が残る**。ADR-0006 は「積まれる世代は常に完全適用済み」を保証するが、これは「未コミット世代の内容を信じない」という世代側の保証であり、FS 上の部分状態そのものは復元されない。

これが実運用で問題になる 2 点:

1. **失敗 run が新規に置いた symlink / copy は前世代 manifest に記録が無い**。次の apply は前世代 manifest との diff で stale 除去するため、失敗 run が置いた「今の新世代にも無いはずの」ゴミを拾えない。
2. **初回 apply（`prev == nil`）が途中失敗すると `reset` が完全 no-op になる**。`reset` は entries 定義から対象を決めるため巻き戻しには使えず、entries に無い旧世代の残骸は永久に残る。

「失敗した apply は FS に痕跡を残さない」を apply の意味論として明文化し、途中失敗時に **その run が行った FS 変更を全て巻き戻し、pre-apply 状態へ復元する** ことを常時有効（フラグなし）の既定動作にする。

## 決定

### 1. インメモリ undo ジャーナル: 各 FS 変更を旧状態付きで記録し、逆順 unwind する

`applier`（`internal/engine/engine.go`）に journal フィールドを追加する。PreRemove → place → materializeCopies（copy 部）→ removeStale の各実行段が FS へ 1 回書き込むたびに、その逆操作を journal へ 1 件 push する。いずれかの段でエラーが発生したら、**その時点までに push された journal を逆順（LIFO）に unwind** し、pre-apply 状態へ戻してからエラーを返す。

対象と逆操作の対応（issue #168 の設計どおり）:

| 順操作 | 逆操作 |
|---|---|
| 新規配置 symlink / copy | 削除 |
| 張替え（`PlaceReplace` / `PlaceForeign`。unlink 前に readlink で旧リンク先を捕捉） | 旧リンク先で symlink を再作成 |
| stale 除去したリンク（`removeStale`） | 前世代 manifest の記録 dest で symlink を再作成 |
| `--recopy` の上書き（`recopyAll`） | 同一親ディレクトリ内の一時名へ rename 退避 → 成功で退避物削除 / 失敗（後続 unwind 含む）で rename back |
| PreRemove の Unlink（祖先 symlink・実 dir 配下 leaf・method 変更 symlink → ADR-0046, ADR-0047） | 記録 dest で symlink を再作成 |
| PreRemove の Rmdir（空 dir 除去 → ADR-0047） | `os.Mkdir` で再作成 |

journal のエントリは「何を」「どこへ」だけで逆操作が定まる十分な情報（対象パス・旧リンク先・一時退避パスなど）を持つ。unwind は journal スライスの末尾から先頭へ辿ることで、生成順と厳密に逆の順序で実行される（例: PreRemove で消した祖先 symlink の下に子を新規配置していた場合、子の削除 → 祖先の再作成、という正しい順序になる）。

### 2. unwind の発火範囲は PreRemove〜removeStale（+ 世代スキップの repairDrift）。commit 成功後は対象外

journal を破棄せず持ち越すのは **FS 変更を行う 4 段 + 世代スキップの drift 修復**、つまり `preRemove` / `place` / `materializeCopies`（`placeCopies` / `recopyAll`）/ `removeStale` / `repairDrift` の呼び出し中に発生したエラーに限る。これらのいずれかが `error` を返したら、その `Apply` / `Rollback` 呼び出しは即座に unwind してから元エラーを返す。

`nix-env --set`（commit）が成功した後は journal を破棄する。commit 成功は「この run の FS 変更が正式に新世代として確定した」ことを意味し、以後 FS を戻すのは巻き戻しではなく別世代への変更になるため、undo ジャーナルの責務ではない（rollback コマンドが担う領域）。commit 自体が失敗するケース（`--set` が失敗する）は対象外とする: この時点で FS 側の変更は全て成功済みであり、戻すべき理由がない。世代が進んでいないだけで、ADR-0006/0017 の冪等再実行で収束する既存の意味論のままでよい。

### 3. undo 自体の失敗は best-effort 続行。失敗項目は元エラーと共に全報告

unwind の実行中、個々の逆操作が失敗しても（例: 再作成しようとした親ディレクトリが別プロセスに消された）unwind 全体を中断しない。失敗した逆操作はスキップし、残りの journal エントリの unwind を続行する。すべての unwind 試行が終わった時点で、**元の apply エラー**と**巻き戻せなかった項目の一覧**を両方 stderr に全報告し、exit 1 で停止する。

これは `reportConflicts`（→ ADR-0047 D6、grilling 2026-07-12）と同じ「1 回の実行で全件を可視化する」姿勢を踏襲する。ユーザーは未復元項目を手動で確認・修復できる。

### 4. クラッシュ（SIGKILL・電源断）は対象外。永続 WAL は持たない

undo ジャーナルはプロセスのメモリ上にのみ存在する。プロセスが `error` を返せずに終了する（SIGKILL・電源断・panic 未回収）ケースは、ジャーナルも失われて何も巻き戻せない。これは意図的なスコープ外である。

- **エラー return 時の巻き戻し**（本 ADR の対象）と**クラッシュ耐性**（対象外）は異なる保証である。前者は「apply が失敗したと自分で報告できたケース全て」を完全に FS 未変更へ戻す。後者は永続 WAL・fsync 規律・クラッシュリカバリ手順を要し、コストに対して既存のバックストップ（ADR-0006 の「世代未コミット」+ ADR-0017 の「冪等再実行で FS が収束する」）で十分に運用できている。
- 本 ADR 後も、クラッシュ時は変わらず「世代未コミット + 冪等再実行で最終的に正しい状態へ収束する」が保証全体を担う。undo ジャーナルはその上に「プロセスが生きている間の失敗は即座に完全復元する」という追加の品質を積む。

### 5. PreRemove の Unlink/Rmdir も undo 対象に含める（一般化後の段構成に対応）

ADR-0046 §5・ADR-0047 影響節が seam として残した「PreRemove を undo 対象に含める」を実装する。ADR-0047 で一般化された PreRemove は 1 バッチ内に複数の `RemoveUnlink`（子から）と `RemoveRmdir`（深い方から）を bottom-up 順で実行するため、journal もその実行順のまま記録し、unwind は当然その逆順（親の Rmdir を先に取り消して mkdir → 子の Unlink を取り消して symlink 再作成、という浅い方から深い方への順）になる。これは `classifyDirMigration` が返す実行順と、journal の LIFO 特性が自然に噛み合う（PreRemove 内の順序自体を undo 側で並べ替える必要はない）。

`repairDrift`（世代スキップ経路）は ADR-0046/0047 の不変条件により PreRemove を構造的に受け取らない（`len(plan.PreRemove) > 0` は internal invariant violation として fail closed）。したがって drift 修復側で journal に積まれるのは `place` の再張り分のみであり、本 ADR はこの経路にも同じ journal/unwind 機構をそのまま適用する（専用分岐は増やさない）。

## 根拠

- **メモリ内 journal + 逆順 unwind** は、既に FS へ書き込んだ順操作の情報（旧リンク先・退避パス）を実行時に持っているため、追加の永続化コストなしに実装できる。プロセス生存中の失敗は 100% カバーし、クラッシュは既存のバックストップに委ねる、という線引きが ADR-0006/0017 の既存不変条件と整合する。
- **commit 成功後は対象外**とするのは、「apply の途中失敗」と「コミット済み世代からの巻き戻し」を混同しないため。後者は `rollback` の責務であり、undo ジャーナルが世代境界を越えて FS を戻す機能を持つと、両者の意味論が曖昧になる。
- **best-effort 続行 + 全件報告**は、1 個の unwind 失敗で残りの巻き戻しを止めると被害が拡大するため。個々の unwind 失敗は「巻き戻せなかった 1 項目」に閉じ込め、残りは正常に戻す方が pre-apply 状態に近づく。
- **PreRemove を undo 対象に含める**のは、ADR-0046/0047 が一般化した除去（祖先 symlink・実 dir 配下 leaf・method 変更）も他の FS 変更と同じく「apply 中に行った変更」であり、undo ジャーナルの対象から除外する理由が無いため。除外すると、PreRemove を経由する移行（dotfiles でよく起きるレイアウト変更）だけ巻き戻しの粒度が粗いという非対称が生まれる。

## 影響

- **`internal/engine/engine.go`**: `applier` に journal フィールドを追加。`Apply` の 8. 節（PreRemove → place → materializeCopies → removeStale）と `repairDrift` 呼び出しをジャーナル対象としてラップし、エラー時に unwind してから return する。
- **`internal/engine/place.go`**: `place` の各 symlink 作成・張替え直後に journal へ push。
- **`internal/engine/copy.go`**: `placeCopies` の新規コピー、`recopyAll` の rename 退避/上書きを journal 化する（rename 退避は成功時に退避物削除、失敗時に rename back の往復操作として記録）。
- **`internal/engine/staleremove.go`**: `removeStale` の unlink、`preRemove` の Unlink/Rmdir を journal 化する。
- **`internal/engine/generations.go`**: `Rollback` も同じ journal/unwind 機構を通す（Rollback は独自に `preRemove` / `place` / `removeStale` を呼ぶため、Apply と同じラップを適用する）。
- **`docs/spec.md`**: 実行フロー・エラー仕様表・張替え意味論に「途中失敗時は全巻き戻し」を追記する。
- **`docs/adr/0046-*.md`**: §5 の「#168 に hard 依存させず出荷する」への改訂注記（本 ADR による実装完了の back-reference）。
- **`docs/adr/0047-*.md`**: 影響節の「#168（undo ジャーナル）は…前提で設計する」への改訂注記（同上）。
- **cross-epic**: #169（`--backup`）は本 ADR に完全直列 stack。`--backup` の退避操作も journal 化対象になる（別 PR）。

## 棄却した代替案

- **永続 WAL（ディスク上の undo ログ）**: クラッシュ耐性まで含めて保証できるが、fsync 規律・リカバリ手順・WAL 自体の破損時フォールバックが要り、実装・保守コストが「apply 途中失敗を完全に戻す」という目的に対して過大。ADR-0006/0017 の既存バックストップ（冪等再実行での収束）で電源断ケースは十分にカバーされている。
- **rename ベースの atomic スナップショット（apply 前に対象ツリー全体を退避）**: target ごとに一時コピーを作る必要があり、大きなツリーではコストが跳ねる。undo ジャーナルは「変更した分だけ」逆操作を記録するため、変更量に比例したコストで済む。
- **unwind 失敗時に即中断**: 1 個の失敗で残りの巻き戻しを諦めると、まだ戻せたはずの項目まで未復元のまま放置される。best-effort 続行の方が pre-apply 状態への近さを最大化する。
- **PreRemove を undo 対象外にする（ADR-0046/0047 の移行は冪等再実行の収束のみに委ねる）**: ADR-0046 §5 が容認した選択だが、本 ADR で undo ジャーナルを実装する以上、PreRemove だけ例外的に粗い保証のままにする理由がない。一般化する方が意味論として一貫する。
