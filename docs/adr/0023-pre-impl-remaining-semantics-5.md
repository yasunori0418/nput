# ADR-0023: 実装前残セマンティクス第5巡（実行フロー順序・出力/終了コード規約・--root キーイング・実装順序・gitignore モード）

- ステータス: 採用
- 日付: 2026-06-14
- 関連: ADR-0005, ADR-0006, ADR-0007, ADR-0011, ADR-0013, ADR-0017, ADR-0018, ADR-0022, `docs/spec.md`, `docs/design.md`, `docs/concept.md`, `CONTEXT.md`
- 改訂対象: ADR-0011 / ADR-0013 の apply 実行フロー順序を「eval 先行」に具体化（決定の反転なし）。ADR-0017 の `--root` profile キーイングを home / fixed mode へ拡張。
- 起点: 実装着手前（ドキュメントのみの計画段階を抜ける直前）の横断検査で、ADR-0022 までで潰し切れていない5点を洗い出した（ADR-0015/0016/0017/0022 と同系列の「実装前残セマンティクス」確定の第5巡）

## 背景

設計は 22 ADR で成熟し、実装前セマンティクスは4巡で高度に潰されている。垂直スライス着手の直前にもう一度横断検査したところ、次の5点に**揺れ / 考慮漏れ**が残っていた。いずれも engine / CLI の骨格に直結するか、コード着手時に恣意判断が混入する箇所で、着手前に確定する。

1. **apply 実行フローの順序に循環矛盾**。ADR-0011 は `nix build --out-link <profileDir>/.pending-<name>` で取得と indirect gcroot 登録を一手に行うと定める。ADR-0013 は flock のキーを**解決後 profileDir**（project mode は `<roothash>/<name>`）とする。ところが spec の実行フローは「out-link build(1) → flock(2a) → root 解決(2b)」の順で、(a) `<profileDir>` を知るには root 解決が必須なのに root 解決が後、(b) root 解決には `manifest.json` の `rootKind` が要るがそれはビルド成果物の中、(c) out-link build が flock の外にあり並行 apply が `.pending-<name>` を奪い合う、という三重の循環・競合があった。
2. **CLI の出力ストリーム / 終了コード / 冗長度の規約が断片的**。`--dryrun` conflict は非ゼロ、`--all` 部分失敗は非ゼロ、warning は stderr、とだけ散在し、終了コード体系・stdout/stderr 規律・機械可読出力・冗長度フラグが未定義。
3. **`--root` 上書き × home / fixed mode の profile キーイング footgun**。ADR-0017 は `--root` を全モードの解決 root の一律上書きとしたが、profileDir のキーイングは project mode のみ `<roothash>` 再計算で、home / fixed mode は `<name>`（roothash でない）のまま。`manifest.json` は `--root` 上書き値を記録しない（`rootKind="home"` のみ）ため、同一 config を `--root /a` → `--root /b` と適用すると `/a` の配置物が silent orphan として残り、engine は前回 root の違いを検知できない。
4. **実装の着手順序が未定義**。ドキュメントのみの段階を抜けてコードに入る際、lib / engine / CLI / module をどの順・どの粒度で組むかが決まっていない。
5. **二次的細目3点**。(5a) 単体 `gitignore <name>` の対象モード（`--all` は projectRoot 限定と既決だが単体は無制限で、home config 指定時のアンカー出力が無意味）。(5b) cross-config 同一 target に対する ADR-0017/0022 の lstat ドリフト修復が能動的オシレーション（A 配置→B 奪取→A 再奪取）になりうる扱い。(5c) `listFilesInSrc` の `src` に out-of-store marker を渡せるかの明記。

## 決定

### 1. apply 実行フローは「eval 先行 → flock → build」に確定

root 解決を**ビルドより前**に行い、flock を**ビルドより前**に取得する。これで profileDir 未確定の循環と、ロック外 build の out-link 競合が同時に解消する。

```
0. entrypoint 発見（CWD 既定 / -f 上書き）
1. nix eval で nput.<name> の root kind（+ fixed root のときは絶対パス文字列）を取得
   → root 解決（project=git toplevel / home=$HOME / system=/ / fixed=記録値、--root で上書き）
   → profileDir 確定（home: <name> / project: <roothash>/<name>、--root 時は §3）
2. flock(profileDir) 取得（明示=blocking / shellHook=try-lock skip・ADR-0013）
3. ロック内で nix build --out-link <profileDir>/.pending-<name>（legacy は nix-build -A）→ os.Readlink で store path
4. 前世代 diff → 新規/張替を配置 → 保守的 stale 除去（ネイティブ FS）
5. nix-env --profile <profileDir> --set <link-farm>（コミット点）
6. --set 成功後に .pending-<name> 削除
```

- **`mkManifest` の返り値 derivation に root kind を passthru で露出する**（例: `passthru.rootKind` / `fixed` のとき `passthru.root`）。CLI はこれを `nix eval <ep>#nput.<system>.<name>.rootKind`（legacy は `nix eval -f <ep> nput.<name>.rootKind`）で**ビルドせずに**読む。`rootKind` は eval 時に確定する（git toplevel / `$HOME` の実体解決は依然 engine 実行時）ため、安価な eval で取れる。
- build が**常にロック内**になり、同名・同 root への並行 apply（shellHook + 手動 apply 等）が `.pending-<name>` out-link を奪い合う競合が構造的に消える。ADR-0011 の「`.pending-<name>` 固定パス・apply ごと上書き・orphan 最大1」はこのロック内直列化で成立する。
- `--dryrun` は build するが配置しないため pending gcroot を張らない（ADR-0011 不変）。eval 先行は dryrun でも同じ（root 解決はするが flock は読み取り専用のため取らない）。

### 2. 出力ストリーム規律・終了コード表を確定し、`--json` は将来送り・`--quiet`/`--verbose` を MVP に入れる

- **ストリーム規律**: **stdout は機械可読出力専有**（`gitignore` の列挙・`--dryrun` のプラン）。**進捗 / 配置レポート（placed / replaced / removed / skipped）・warning・shellHook skip 通知はすべて stderr**。これにより `nput gitignore <name> >> .gitignore` や `nput apply <name> --dryrun | ...` が安全にパイプできる。
- **終了コード表**:

  | code | 意味 |
  |---|---|
  | 0 | 成功 / no-op / `--no-wait` の try-lock skip（正常スキップ）|
  | 1 | 一般エラー（eval エラー・engine 実行時エラー・`--all` の部分失敗）|
  | 2 | `--dryrun` で conflict 検出（CI の事前 gate に使える）|

- **`--json` は MVP では持たない**。テキスト + stdout/stderr 規律で当面十分とし、必要になった時点で追加する。
- **`--quiet` / `--verbose` は MVP で持つ**。`--quiet` は進捗 / レポートを抑制し warning / error は残す（shellHook 既定との相性が良い）。`--verbose` は内部 nix コマンド等の詳細を出す。

### 3. `--root` 明示時は home / fixed mode も roothash でキーイング

- `--root` が**明示されたとき**は、全モードで profileDir を**上書き後 root の `<roothash>`** でキーする。

  | 状況 | profileDir |
  |---|---|
  | home（`--root` なし）| `<state>/nix/profiles/nput/<name>` |
  | home / fixed（`--root /p`）| `<state>/nix/profiles/nput/<roothash(/p)>/<name>` |
  | project（`--root` 有無）| `<state>/nix/profiles/nput/<roothash>/<name>`（ADR-0013 既定）|

- 異なるオーバーライド root が独立した世代系列に分離され、§背景4 の silent orphan が構造的に消える。`<roothash>` 算出・backref ファイル（`.root`）は project mode と同一機構（ADR-0013）を流用する。
- この roothash キーイングは `apply` / `reset` / `rollback` / `list-generations` で一貫させる（`--root` を付けた状態の世代を操作するには再び同じ `--root` が要る）。`--root` なしの通常 home は従来どおり `<name>` キーのまま（「1 ユーザー 1 profile」の UX を保つ）。
- `--root` は debug / test / 特殊配置の脱出路（ADR-0017）なので、`--root $HOME` のように既定と同値を明示すると通常 home profile と別系列になる端ケースは許容し、注記に留める。

### 4. 実装は垂直トレーサー弾で進める

- **第一スライス = home mode / `method = "symlink"` / store link の `nput apply <name>` を lib → engine → CLI で end-to-end に通す**。manifest.json 契約・`nix eval`/`nix build`・`nix-env --set`・flock の骨格を最早期に検証する。
- 以降は横展開:
  - Slice2: `--dryrun` + 保守的 stale 除去の安全不変条件（table-driven テスト）
  - Slice3: `method = "copy"`（place-once / owner-write / symlink 複製）+ out-of-store symlink
  - Slice4: `apply --all`（フィルタ含む）/ `reset` / `apply --recopy`
  - Slice5: project mode（roothash / backref / 世代スキップ / lstat ドリフト修復）
  - Slice6: `rollback` / `list-generations` + home-manager module + templates / `nput init`
- レイヤーボトムアップ（lib 全完成 → engine 全完成 → CLI 全完成）は end-to-end 検証が終盤に遅れ契約のずれを終盤で発見しがちなため採らない。

### 5. 二次的細目

- **(5a) `gitignore` は project mode 限定**。単体 `gitignore <name>` も project mode の config のみ受理し、非 project config（home / fixed）を指定したらエラーで停止する。`gitignore` の出力は root = git toplevel = `.gitignore` 置き場所を前提にした先頭 `/` アンカー形式（ADR-0013）で、home / fixed では意味を成さないため。`rollback` / `list-generations` が home mode 限定なのと対称。
- **(5b) cross-config 同一 target の lstat 修復振動はユーザー責任**。別 config A / B が同一 target を狙うと、ADR-0017/0022 の lstat ドリフト修復が「A が置く → B が foreign 検知して奪う → A が再奪取」と shellHook 再入のたびに振動しうる（単発「後勝ち」ではなく能動的オシレーション）。これは「同一 target を複数 config で狙わない」というユーザー責任とし、foreign symlink warning（ADR-0015）で可視化する。検知して止める機構は持たない（document-only 注記）。
- **(5c) `listFilesInSrc` の `src` は path 限定**。`listFilesInSrc` は eval 時に `readDir` するため `src` は path（store パス）に限り、out-of-store marker は受け付けない（marker は実行時解決の入れ物で eval 時にパスへ展開できない）。lib API にこの制約を明記する。

## 根拠

- **eval 先行**は循環依存（profileDir ← root 解決 ← rootKind ← ビルド成果物）を、rootKind だけを安価 eval で先取りして断ち切る最小の解。build をロック内に閉じることで out-link 競合も同時に消え、別途の PID 一意化等のパッチが要らない。
- **stdout 専有 + 終了コード表**は CI gate（dryrun=2）と shellHook（skip=0）の振る舞いを呼び出し側が機械判別できる最小規約。`--json` を先送りするのは YAGNI、`--quiet`/`--verbose` を入れるのは shellHook の静音とデバッグ両立の実利があるため。
- **`--root` の全モード roothash キーイング**は project mode の既存機構をそのまま流用でき、silent orphan を「保守的・誤破壊しない」方針に沿って構造的に防ぐ。document-only より実装は増えるが脱出路の安全性が上がる。
- **垂直トレーサー弾**は nix↔Go 契約という最もリスクの高い境界を最早期に通し、以降の横展開を低リスク化する。
- **gitignore project 限定 / listFilesInSrc path 限定**は API の意味矛盾を eval / CLI 段で弾く（ADR-0010 の「未定義挙動を早期に弾く」と一貫）。**振動のユーザー責任化**は ADR-0013/0015 の「衝突させない前提・後勝ち」を lstat 修復文脈へ素直に延長したもの。

## 影響

- **`docs/spec.md`**:
  - 実行フロー節（`nput apply` のステップ）を「eval 先行 → flock → build」順に書き換え、`nix eval` での rootKind 先取りと「build はロック内」を明記。
  - `mkManifest` 返り値に root kind の passthru 露出を追記。
  - CLI 仕様にストリーム規律・終了コード表・`--quiet`/`--verbose` グローバルフラグを追加。`--json` 非対応（将来送り）を注記。
  - `--root` の解決節に「明示時は全モード roothash キーイング・apply/reset/rollback/list-generations で一貫」を追記。
  - `gitignore` を project mode 限定（非 project はエラー）と明記し、エラー仕様表に1行追加。
  - `listFilesInSrc` の `src` を path 限定（marker 不可）と明記。
  - cross-config 同一 target の lstat 修復振動のユーザー責任注記を世代スキップ / 配置動作節に追記。
- **`docs/design.md`**: 実行モデル節の手順を eval 先行に更新。CLI サブコマンド表に `--quiet`/`--verbose`・終了コード方針を反映。`mkManifest` passthru を outputs 設計 / コアロジック節に反映。
- **`CONTEXT.md`**: `engine` 定義の実行順を「eval 先行 → flock → build」に整合させ、`nput CLI` の `gitignore` を project mode 限定と注記。
- **`docs/concept.md`**: project mode の devShell 例の近辺に振る舞い差は無いが、必要なら gitignore の project 限定を軽く反映（語の整合のみ）。
- **実装フェーズ**: lib（`mkManifest` の rootKind passthru・`listFilesInSrc` の src 型ガード）、CLI（eval 先取り・flock 順序・終了コード・stdout/stderr 規律・`--quiet`/`--verbose`・`--root` 全モード roothash・gitignore project 限定）、engine（build ロック内化に伴う呼び出し順）。垂直トレーサー弾の Slice 順で着手する。

## 棄却した代替案

- **out-link を temp に build してから profileDir へ gcroot 再登録**: eval 追加を避けられるが、ロック前 build が残り gcroot 再登録の段が増える。eval 先行のほうが手順が単純で build がロック内に収まる。
- **`.pending-<name>-<pid>` で out-link を PID 一意化**: 並行 out-link 競合だけは避けられるが、profileDir 未確定の循環（root 解決順）は別途解消が必要で根本解決にならない。
- **`--json` を MVP に入れる**: 実装・テストが増えるが現時点で消費側の要求が無い。stdout 規律で当面足りる。
- **`--root` を home / fixed で document-only（ユーザー責任）にする**: 実装最小だが silent orphan が残り「誤破壊しない / 可観測」方針に劣る。
- **gitignore を全モードで許容**: home / fixed で無意味なアンカー出力を生み、ユーザーが誤って `.gitignore` に貼る footgun。
- **レイヤーボトムアップ実装**: 各層が完成形になるが end-to-end 検証と契約検証が終盤に偏る。
- **cross-config 振動を検知して停止 / 調停**: 実装が重く、そもそも「同一 target を複数 config で狙わない」前提（ADR-0013/0015）を破る使い方への過剰防衛。
