# ADR-0024: 実装前残セマンティクス第6巡（fixed root キーイング・HM profile 粒度・非 build コマンドの eval 先行・listFilesInSrc 型ガード・target 衝突 eval 検出・既存 flake 組込・終了コード優先・--all 一括 eval・--quiet 規律・振動 warning・cleanup seam）

- ステータス: 採用
- 日付: 2026-06-14
- 関連: ADR-0005, ADR-0007, ADR-0010, ADR-0011, ADR-0013, ADR-0015, ADR-0017, ADR-0019, ADR-0023, `docs/spec.md`, `docs/design.md`, `docs/concept.md`, `CONTEXT.md`
- 改訂対象: ADR-0023 §3 の profileDir キーイング表に fixed（`--root` なし）行を補う。ADR-0023 §1 の eval 先行フローを非 build コマンドへ拡張。ADR-0010 の早期エラー方針へ target 衝突検出を回収（決定の反転なし）。
- 起点: ADR-0023 までの第5巡で潰し切れていない11点を、垂直トレーサー弾（ADR-0023 §4）着手の直前に再度の横断検査で洗い出した。いずれも実装時に恣意判断が混入するか、ドキュメントの穴で、着手前に確定する（実装前残セマンティクス確定の第6巡）。

## 背景

ADR-0023 で実行フロー順序・出力規約・`--root` キーイング・実装順序が確定し、設計は実装着手の閾値に達した。着手直前の横断検査で、骨格に直結する揺れ・考慮漏れが11点残っていた。第5巡（ADR-0023）の決定を素直に延長して埋められるものが大半で、新規の方針反転は無い。

1. **fixed root mode（`--root` なし）の profileDir キーイングが未定義**。ADR-0023 §3 の表は home（`--root` なし）= `<name>`、home/fixed（`--root /p`）= `<roothash>`、project = `<roothash>` の3行だが、`root` に絶対パス文字列を渡し `--root` を付けない fixed mode が抜けている。`<name>` で素朴にキーすると別 root の同名 config が世代系列を共有し ADR-0023 §背景4 と同型の silent orphan が起きうる。
2. **HM モジュール経由の profile `<name>` キーが未定義・複数 config 不可**。standalone は entrypoint の `nput.<name>` で複数 profile を持つが、`modules/common.nix` は `nput.entries` 単一 attrset で `<name>` 次元が無い。profile dir の `<name>` に何が入るか未記述で、「役割ごとに別 profile」の中心思想が HM 利用者に届かない。
3. **`reset` / `rollback` / `list-generations` の実行フローに eval 先行が未明記**。ADR-0023 §1 は apply にのみ「rootKind 先取り eval → root 解決 → profileDir 確定」を書いたが、これら非 build コマンドも profileDir 単位の flock / 前世代 manifest 読みのため profileDir 確定（= rootKind eval）が前提になる。
4. **`listFilesInSrc` の `src` に `set`（derivation）を渡したときの挙動が未定義**。ADR-0023 5c は path 限定・marker 不可と決めたが、entries の `src` は `path | set | marker` の3種で、`set`（`fetchFromGitHub` の生 derivation）の扱いが書かれていない。`set` を渡すと IFD（import-from-derivation）が発生し flake pure eval で破綻する。
5. **同一 manifest 内 target 衝突の検出が engine 実行時とされ ADR-0010 とずれる**。別キー A/B が `target` を同値に明示上書きしたケースを spec は「engine 実行時に conflict 検出」とするが、これは正規化後 target 文字列の静的衝突で eval 時に判定可能。ADR-0010 の「単一検査ゲート・早期エラー」とずれている。
6. **既存プロジェクトへの組み込み経路が docs に無い**。project-first（ADR-0007）を標榜するのに導入は `nput init`（= `nix flake init -t`・新規作成向け・既存ファイル非上書き）のみで、既に `flake.nix` がある repo への後付け手順が通しで示されていない。
7. **`apply --all --dryrun` の終了コード混在優先が未定義**。conflict（2）と error（1）が同居したときの最終 exit code が未定義で、単純な最大値では 2 が 1 を隠す。
8. **`apply --all` の rootKind eval 回数が未定義**。N config の profileDir 確定に rootKind が要るが、個別 eval（N 回）か一括 eval（1 回）か未記述。
9. **`--quiet` と stdout 機械可読出力の相互作用が未定義**。`--quiet` は「進捗/レポート抑制・warning/error 残す」だが、stdout 専有の dryrun plan / gitignore 列挙を抑制するかが書かれていない。
10. **cross-config 振動の foreign warning が shellHook 高頻度実行で出続けるノイズの扱いが未定義**（ADR-0023 5b の延長）。
11. **orphan profile dir / `.pending-<name>` の cleanup コマンドの要否が未定義**。

## 決定

### 1. fixed root mode（`--root` なし）も常に roothash でキーする

`root` に絶対パス文字列（fixed root）を渡したときは、`--root` の有無に依らず profileDir を **`<roothash>/<name>`** でキーする。ADR-0023 §3 の表を次に拡張する。

| 状況 | profileDir |
|---|---|
| home（`--root` なし）| `<state>/nix/profiles/nput/<name>` |
| home / fixed（`--root /p`）| `<state>/nix/profiles/nput/<roothash(/p)>/<name>` |
| **fixed（`--root` なし・`root = "/abs"`）** | **`<state>/nix/profiles/nput/<roothash(/abs)>/<name>`** |
| project（`--root` 有無）| `<state>/nix/profiles/nput/<roothash>/<name>` |

- fixed root は評価時確定の任意絶対パスなので、project / `--root` 上書きと同じく root ごとに独立系列へ分離するのが一貫し、silent orphan を構造的に防ぐ。`<name>` 直キーは「1 ユーザー 1 profile」UX が成立する home（`--root` なし）に限る。
- `<roothash>` 算出・backref（`.root`）は project mode と同一機構（ADR-0013）を流用。`apply` / `reset` / `rollback` / `list-generations` で一貫する。

### 2. HM モジュール経由は MVP で固定名 1 profile（`default`）、複数化は将来

- HM モジュール（`modules/common.nix` の `nput.entries`）経由の nput profile は **`<name>` = `"default"` 固定の 1 profile** とする（profile dir = `<state>/nix/profiles/nput/default`、HM の root は `homeRoot` を pin する home mode なので `<name>` 直キー）。
- 「役割ごとに別 profile」を使いたいユーザーは **standalone CLI 経路**（entrypoint の `nput.<name>`）を使う。HM モジュールの複数 profile 化（`nput.configs.<name>.entries` 等）は将来拡張とする。
- ADR-0007 の「HM 対応は他モジュールの switch と一括で動くユースケースを拾うだけ」positioning と整合。MVP で options を attrsOf へ広げて activation の複数 profile swap を背負わない。

### 3. 非 build コマンドも eval 先行を共通前段にする

`reset` / `rollback` / `list-generations` も、profileDir 確定のため **rootKind 先取り eval → root 解決 → profileDir 確定**を実行フローの共通前段として持つ（build はしない）。`--root` 上書き時は ADR-0023 §3・本 ADR §1 と同じ roothash キーで profileDir を引く（`--root` を付けた世代を操作するには同じ `--root` が要る）。`reset` はさらに entries 読みのため entrypoint eval も行う。

### 4. `listFilesInSrc` の `src` は path 限定を型で明示し `set` も eval エラー

`listFilesInSrc` の `src` は **path（store パス）限定**とし、`set`（derivation）も marker も **eval 時にエラー**で弾く（型で拒否）。

- 理由の非対称：`listFilesInSrc` は eval 時に `builtins.readDir` するため未 realise の derivation を渡すと IFD を誘発し flake pure eval で破綻する。一方 entries の `src` は engine 実行時解決なので `set` を許容できる。この「eval 時 readDir / 実行時解決」の差を spec に明記する。
- 典型ユースケース（`claude-skills.flake = false` 等の flake input = 既 realise の store path）は path として問題なく通る。

### 5. 同一 manifest 内 target 衝突は eval 時に throwIf で弾く

別キー A/B が `target` フィールドを同値に明示上書きした衝突は、`normalizeManifest` が正規化後 target の重複を **eval 時に `lib.throwIf` で検出・停止**する（engine 実行時ではない）。ADR-0010 の「`mkManifest` を単一検査ゲートに・未定義挙動を早期に弾く」と一貫させる。

- **cross-config（別 profile・別 manifest）の同一 target 衝突は eval では検出不可**で、これは引き続き engine 実行時の後勝ち + foreign symlink warning（ADR-0015）。両者を区別して記述する。

### 6. 既存 flake への組み込み手順を docs に追加

`nput init`（新規作成向け）とは別に、**既に `flake.nix` がある既存 repo への後付け手順**を docs（使用パターン）に通しで追加する：(1) input に `nput` 追加 → (2) `outputs.nput.<system>.<name>` に `mkManifest` 公開 → (3) devShell の `packages` に pin 版 `nput` 同梱 → (4) `shellHook` に名指し apply。CLI に flake 自動マージ機構（`init --merge` 等）は持たない（「設定を生成しない」thesis を維持）。

### 7. `--all --dryrun` の終了コードは error(1) 最優先 → conflict(2) → 0

`apply --all --dryrun` で複数 config が混在したときの最終 exit code は **いずれかが error なら 1、error が無く conflict があれば 2、どちらも無ければ 0**。単純な最大値（2 > 1 で conflict 優先）は採らない（より深刻な eval/engine エラーを CI で隠すため）。非 dryrun の `--all` は conflict 概念が無く従来どおり 0/1。

### 8. `apply --all` は rootKind を 1 回の一括 eval で取る

`apply --all` は `nix eval <ep>#nput.<system> --apply 'cs: builtins.mapAttrs (_: c: c.rootKind) cs' --json`（legacy 経路は対応する `-f` 形）で **config 名 → rootKind マップを 1 回の eval で取得**する。`--project-root` 等のフィルタもこの結果で振り分ける。build だけは atomic 性のため config ごと N 回。eval プロセス起動コストを N→1 に固定。

### 9. `--quiet` は stderr の進捗/レポートのみ抑制し、stdout 機械可読出力は不可触

`--quiet` が抑制するのは **stderr の進捗 / 配置レポートのみ**（warning / error は残す・ADR-0023）。**stdout 専有の機械可読出力（`apply --dryrun` の plan・`gitignore` の列挙）は `--quiet` 下でも抑制しない**。stdout 専有原則を貫き、`--dryrun --quiet` / `gitignore --quiet` のパイプを壊さない。

### 10. cross-config 振動の foreign warning は MVP で document-only

cross-config 同一 target の lstat 修復振動中に foreign warning が `shellHook` 高頻度実行で出続けるのは、「同一 target を複数 config で狙わない」前提（ADR-0013/0015）違反の**設定ミスのシグナル**であり、出続けるのが正しい。warning は `--quiet` 対象外（§9）なので `--quiet` でも消えない。抑制 / 集約機構は MVP で持たず、config の同一 target 重複を解消して直す旨を docs 注記に留める（ADR-0023 5b「検知して止める機構は持たない」と一貫）。

### 11. cleanup コマンドは MVP 非対応・将来 `nput prune` の seam を残す

orphan profile dir（クローン削除で残る `<roothash>/<name>`）・`.pending-<name>`（`--set` 前失敗で config あたり最大1）の cleanup コマンドは **MVP では持たない**。実害が小さい（store は `nix-collect-garbage` で解放され、残るのは小さな symlink dir）ため放置許容 + 手動削除 + docs 注記。backref（`.root`・ADR-0013）があるので**将来 `nput prune`（実在しない root を指す孤児系列を逆引きして削除）を実装できる seam** を残す。消費側の要求が出た時点で追加（YAGNI）。

## 根拠

- §1〜§3・§5・§8 はいずれも ADR-0023 / ADR-0013 / ADR-0010 の既決方針を該当文脈へ素直に延長したもので、新規トレードオフを生まない穴埋め。
- §4 の型ガードは IFD 破綻を eval 段で構造的に回避し、ADR-0010 の早期エラー思想と一貫。
- §6 は project-first（ADR-0007）の主経路（既存 repo 後付け）を docs で埋めるもので、CLI 機構を増やさず thesis を維持。
- §7・§9 は CI gate と shellHook 静音の両立を呼び出し側が機械判別 / 安全にパイプできる規律の補完。
- §10・§11 は「誤破壊しない・問題を隠さない・YAGNI」の各方針の延長で、MVP 実装量を最小に保つ。

## 影響

- **`docs/spec.md`**:
  - `root の解決` 節の profileDir キーイングに fixed（`--root` なし）= roothash 行を追記（§1）。実行フロー §1 の profileDir 確定にも反映。
  - 世代管理仕様 / profile dir 表に HM 経由 = `<name>` = `"default"` 固定 1 profile を明記（§2）。モジュールオプション仕様にも注記。
  - 実行フロー節に reset / rollback / list-generations の eval 先行共通前段を追記。`reset` / `rollback` のサブコマンド説明・配置動作仕様にも反映（§3）。
  - `lib.listFilesInSrc` 節の `src` を path 限定（`set` も eval エラー）と明記し非対称の理由を1文追加。エラー仕様表に `set` 行を追加（§4）。
  - エラー仕様表の「別キー target 明示上書き衝突」を eval 時 throwIf に修正。`normalizeManifest` 節に target 重複検出を追記。cross-config 衝突との区別を明記（§5）。
  - 出力ストリーム / 終了コード節に `--all --dryrun` の優先順位（error > conflict > 0）と `--quiet` × stdout 不可触を追記（§7・§9）。
  - 実行フローに `--all` の rootKind 一括 eval を明記（§8）。
  - project mode 世代節に振動 warning が `--quiet` でも出る注記（§10）、orphan profile 節に将来 `nput prune` seam 注記（§11）。
- **`docs/design.md`**: 実行モデル節に非 build コマンドの eval 先行・`--all` 一括 eval を反映。モジュール統合表の home-manager 行に profile = `default` 固定を注記。使用パターンに「既存 flake への組み込み」セクションを追加（§6）。
- **`docs/concept.md`**: 使用パターン / project mode 近辺に既存 repo 後付けの導線を軽く反映（§6・語の整合のみ）。
- **`CONTEXT.md`**: `engine` / `nput CLI` 定義の実行順記述に非 build コマンドの eval 先行を整合。profile キーイングの fixed 行に触れる場合は roothash を反映。
- **実装フェーズ**: lib（`normalizeManifest` の target 重複 throwIf・`listFilesInSrc` の src 型ガード）、CLI（fixed/`--root` の roothash 解決・非 build コマンドの eval 先行・`--all` 一括 eval・終了コード優先・`--quiet` 規律・HM profile = default）。ADR-0023 §4 の Slice 順で着手する。

## 棄却した代替案

- **fixed root を `<name>` 直キー**: 実装最小だが別 root の同名 config が世代系列を共有し silent orphan が残る（§背景1）。
- **fixed root を MVP で eval エラー禁止**: seam を削るが任意固定 root の利用余地を失い、ADR-0004 の root 一般化方針に逆行。
- **HM モジュールを MVP から `nput.configs.<name>` の attrsOf にする**: 粒度は揃うが options 設計と activation の複数 profile swap が重く、HM の低い positioning（ADR-0007）に見合わない。
- **target 衝突を engine 実行時のまま**: 検査経路は1つで済むが eval で弾けるエラーを実行時へ遅らせ ADR-0010 とずれる。
- **`listFilesInSrc` で `set` を許容し IFD はユーザー責任**: entries と型が揃うが flake pure eval での破綻を招き、典型外ユースケースの footgun。
- **`--all --dryrun` の終了コードを最大値**: 実装最小だが conflict(2) が error(1) を隠し CI で深刻エラーを見落とす。
- **`nput init --merge` で既存 flake を自動マージ**: 後付けは楽だが「設定を生成しない」thesis を崩す。
- **MVP に `nput prune` / warning 抑制を入れる**: 運用は楽だが実害の小さい問題に実装・テストを増やし、問題（振動の設定ミス）を隠す。
