# ADR-0046: 自己記録の祖先 symlink 配下ネストを許可する — 前世代 manifest 判定の祖先拡張 + 配置前除去（PreRemove）

- ステータス: 採用
- 日付: 2026-07-12
- 関連: ADR-0002, ADR-0003, ADR-0006, ADR-0013, ADR-0015, ADR-0017, ADR-0031, ADR-0038, `docs/spec.md`, GitHub Issue #170
- 改訂対象: **ADR-0015 §4**「target の祖先 component が symlink なら配置前に一律 error 停止」を「**foreign 祖先のみ error / 自己記録 stale 祖先は migration（配置前除去）**」へ改訂。**ADR-0015 §2**「前世代 manifest 記録の symlink は置換可（silent・後勝ち）」の判定を祖先へ拡張。
- 起点: dotfiles 側での実遭遇（#170）と、次期マイルストーンの grilling セッション（2026-07-11）で設計確定

## 背景

`.claude/skills` のような**全体 symlink** entry を、配下に子 entry をネストする形（`.claude/skills/foo`, `.claude/skills/bar`, …）へ移行しようとすると、apply が丸ごと失敗する。原因は 2 段に分かれる。

1. **判定**: `ancestorSymlink`（`internal/planner/planner.go`）が祖先 symlink を検出すると、それが**自 profile の前世代 manifest が記録した symlink**であっても `Compute` が一律 conflict を積む（ADR-0015 §4）。
2. **収束不能**: `len(plan.Conflicts) > 0` で engine の `Apply()` が早期 return し、`place` も `removeStale` も走らない。結果、旧世代の全体 symlink も除去されず、新旧どちらの状態にも収束しない（`nixos-rebuild switch` しても新 entries が 1 つも配置されない）。

実運用では、親 entry → 子 entry 群への定義変更のたびに apply 前に対象 symlink を手動 `rm` する回避策が要り、「設定を宣言的に管理する」という nput の思想と相性が悪い。

一方で ADR-0015 §4 の一律停止は「foreign symlink 配下へネストすると `os.MkdirAll` が symlink-to-store を既存 dir と見なし read-only store へ書く / dangling を作る」汚染を防ぐための安全策であり、**未知の祖先に対しては維持したい**。緩和したいのは「自分が前世代に置いた symlink を、自分が次世代で消して子へ張り替える」既知・安全なケースだけである。

## 決定

### 1. 緩和条件は recorded ∧ stale の AND

祖先 symlink の conflict を外して migration へ振るのは、下記を**すべて**満たすときに限る。

1. **recorded**: 前世代 manifest がその祖先 target を entry として記録し、on-disk symlink が記録 dest と一致（`recordedLink` 相当の保守的不変条件・ADR-0002 / ADR-0015 §2）。
2. **stale**: その祖先 target が**次世代 manifest に無い**（＝除去予定）。
3. **帰結**: 次世代にも祖先 entry が残る**自己矛盾 manifest**（`.claude/skills` symlink と `.claude/skills/foo` を同時に持つ）は緩和せず **true conflict のまま維持**する。親 symlink を消せない以上ネストできないため。

foreign（他 nput profile / 他ツール / 手動作成 = 記録 dest と不一致 or 前世代に無い）・`prev == nil`・初回 apply は**従来通り error 停止**を維持する。**既知（自己記録 stale）のみ silent 移行・未知は安全停止**という非対称を保ち、store 汚染の担保を損なわない。

### 2. 子 entry は lstat を経ず無条件に absent 配置する

祖先 symlink が緩和対象のとき、配下子 entry は **lstat 結果を使わず「target 不在」として配置**する（symlink → `PlaceNew` / copy → place-once の `CopyAction`）。

理由: OS の lstat は中間 component（緩和対象の祖先 symlink）を辿るため、`root/.claude/skills/foo` は前世代 farm の中身 `store/…/skills/foo` を指して**存在するように見える**。通常分類に落とすと `PlaceForeign` / 構造衝突と誤認して **store 内へ書き込もうとする**（ADR-0015 §4 が防いでいた汚染そのもの）。祖先を配置前に除去する前提なので、子は不在として新規配置してよい。

### 3. 順序例外は `Plan.PreRemove` で planner に閉じる

「自己記録 stale 祖先 symlink を配置**前**に除去」を純粋関数 planner の中でデータとして表現する。

- `Plan` に **`PreRemove []RemoveAction`** を新設。planner が緩和対象の祖先を PreRemove に載せる（複数の子が同一祖先を検出しても **dedup** して 1 件）。
- 既存 stale ループは、同 target が PreRemove 済みなら**スキップ**（二重除去回避）。
- engine は **PreRemove → place → copies → removeStale** の順で流す。ADR-0006 の本流（新を先に置いてから旧を消す）は不変で、**局所例外のみ**を明示的に前段へ置く。順序決定が planner に閉じるため純粋・テスト可能（ADR-0003 の役割分担と一致）。
- 除去は removeStale と同じ**保守的不変条件を配置直前に再検証**してから unlink する（planning と実行の間のドリフトに対する安全確認は共通）。

### 4. 報告は既定 silent・`-v` で可視・warning にはしない

自己記録祖先の移行は**意図された安全操作**なので、除去した祖先を warning としては出さず、通常の配置レポートと同じ経路（`result.Removed`）に畳む。既定沈黙・`-v` で可視というレポート規律（ADR-0031・#52 の -v opt-in）と一貫させる。`--dryrun` では自己記録 stale 祖先を**非 conflict の removal** として、foreign 祖先を conflict として報告する。

### 5. 失敗時の意味論（#168 と独立）

PreRemove で親 symlink を除去した後に子配置が途中失敗すると世代未コミットのまま中間状態が残るが、**冪等再実行で FS は正しい最終状態へ収束する**（ADR-0017 と一貫）。再実行時、親は実 dir 化して `ancestorSymlink` は offender を返さず、既配置子は張替え、旧 stale 記録は放置される。#168（undo ジャーナル・ADR-0044）マージ後は中間失敗が atomic に巻き戻るが、**#168 に hard 依存させず本 ADR 単独で出荷**する（PreRemove 段を #168 の undo 対象に含める seam だけ用意する）。

### 6. 前提（前世代 manifest の取得）は全 mode 共通で成立する

緩和は「前世代 manifest が祖先 target を記録している」ことを前提とする。これは特定 mode の性質ではなく**全 mode 共通**に成立する。緩和ロジックは mode 非依存の純粋関数 `planner.Compute`（rootKind を受け取らない）に閉じ、前世代 manifest 読み込みは mode 固有分岐（project の世代スキップ等）**より前**に走る。profile + manifest は store でも ephemeral でもなく state dir に**永続**するため、前世代が同一 profile に commit 済みなら HM / standalone / devShells（project）いずれでも読める。親→子は derivation が変わるため project でも**世代スキップされず**通常 apply（PreRemove + place）が走る。config 名の改名・repo の別パス移動・初回 apply では `prev == nil` → foreign 扱いで**従来通り conflict 停止**（安全側フォールバック）。これは本 ADR で新設する挙動ではなく既存の世代モデルの帰結。

## 根拠

- **recorded ∧ stale の AND** は「自分が置いて自分が消す」ケースだけを最小の条件で切り出す。recorded だけだと自己矛盾 manifest まで緩和してしまい親を消せないのに子を置く矛盾に陥る。stale だけだと foreign を緩和して store 汚染が復活する。
- **子の無条件 absent 配置**は ADR-0015 §4 の汚染防止を維持したまま migration を成立させる唯一の手段。除去前の lstat は祖先 symlink を辿るため信用できない、という事実に対する正しい対処。
- **PreRemove を planner に閉じる**のは ADR-0003（planner=純粋にプラン決定 / engine=FS 反映）の役割分担に沿う。engine 側で暗黙に順序を解決すると純粋性・テスト可能性が崩れる。
- **warning にしない**のは ADR-0031 の成功時沈黙と一貫。意図された安全操作を warning にすると本当に注意すべき foreign 上書きの警告が埋もれる。
- **#168 への非依存**は本 epic を独立に出荷可能にする。冪等再実行で収束する（ADR-0017）ため、atomic 巻き戻しは品質改善であって前提条件ではない。

## 影響

- **`internal/planner/planner.go`**: `Plan.PreRemove` 追加。`Compute` の祖先 conflict 生成部を緩和条件で分岐（recorded ∧ stale ∧ ¬kept → PreRemove + 子 absent 配置 / それ以外 → conflict）。`ancestorSymlink` が offender の root 相対 target も返す。stale ループが PreRemove 済み target を skip。
- **`internal/engine/`**: PreRemove を place の前段で実行。除去祖先を `result.Removed` へ畳む。dryrun も PreRemove を Removed としてパック。gen-skip 経路では PreRemove が構造的に空になる旨を明記。
- **`docs/adr/0015-*.md`**: §4 に本 ADR への改訂注記（back-ref）を追記。本文は当時のまま（ADR-0008 の慣例）。
- **`docs/spec.md`**: 配置動作（祖先 symlink 非対称）・エラー仕様表（祖先 symlink を foreign 限定の error へ・自己記録 stale は migration）・実行フロー（PreRemove 段）を反映。
- **cross-epic**: #168（undo・ADR-0044）は PreRemove 段を undo 対象に含める。#167 / #149 は `planner.go` / `engine.go` を共有するため同時着手時は rebase 調整（hard 依存なし）。#126（--json）/ #131（変更系 niface）は PreRemove を出力サーフェスへどう表すかで接触（先行側が seam を用意）。

## 棄却した代替案

- **運用回避（apply 前の手動 `rm`）を維持**: 移行のたびに手作業が要り宣言的管理の思想に反する。
- **engine 側で暗黙に順序解決 / グローバル順序反転**: ADR-0006 の本流（新を先に・旧を最後に）を壊し、純粋 planner の外に順序判断が漏れる。局所例外を PreRemove として明示する方が安全。
- **祖先 symlink 検出時に常時 warning を出して置換**: foreign symlink 配下への誤 nest（store 汚染）まで通してしまう。ADR-0015 §4 の安全策が失われる。
- **#170 で自前 undo を実装 / #168 へ hard 依存**: 冪等再実行で収束する（ADR-0017）ため自前 undo は不要。#168 へ hard 依存させると独立出荷できない。
- **recorded のみ（stale 条件を外す）で緩和**: 自己矛盾 manifest（次世代に祖先が残る）まで緩和し、親を消せないのに子を置く矛盾を生む。
