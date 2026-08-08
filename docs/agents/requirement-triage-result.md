# requirement 全件の仕分け結果（作業ファイル）

`docs/agents/sara-graph.md`「Where a norm belongs: `requirement`, `quality` or `test_plan`」
の基準で `docs/requirements/` の残存全件を分類した結果。**Issue #240 の成果物**で、
T3b（Issue #241）の移設完了後に削除する一時ファイル。

## 集計

対象は移設（#238 / #239）後に残る **134 件**（2026-08-08 時点）。表の行は手作業の判定結果で
あり自動生成物ではないので、`docs/requirements/` が増減しても追従しない。表が実体と一致して
いることは件数ではなく **ID 集合の差分**で確かめる（差分が出たら、その時点で表は陳腐化して
いる）。**リポジトリルートで bash / zsh から実行する**（プロセス置換を使うため sh / dash では
動かない。両辺ともルート起点の相対パスなので、別ディレクトリで実行すると両辺とも空集合に
なり、`sed` / `rg` がエラーを吐きながら `diff` 自体は差分ゼロで成功する。差分ゼロを合格と
読む前にエラー出力が無いことを確かめる）:

```bash
diff <(sed -n 's/^| `\(REQ-[0-9a-f]\{8\}\)` .*/\1/p' docs/agents/requirement-triage-result.md | sort) \
     <(rg --no-filename -o '^id: "(REQ-[0-9a-f]{8})' -r '$1' docs/requirements/*.md | sort)
```

| 判定 | 件数 |
|---|---|
| requirement 据え置き | 134 |
| quality へ移設 | 0 |
| test_plan へ移設 | 0 |

**移設対象は 0 件**。「品質や規約を守ったりプロダクトの方向性に関わるドキュメントが
requirement には多い」という Issue #240 背景の想定は、全件を通した結果 **成立しなかった**。
理由は #209 の分割元が `docs/spec.md`（プロダクト仕様）1 本であり、開発プロセスの規約は
そもそも `docs/agents/` と `CLAUDE.md` に置かれていて requirement 化されていないため。
テスト計画に相当する 4 件は #238 / #239 で既に `docs/test-plan/` へ抜けている。

したがって **T3b（#241）に移設作業は発生しない**。#241 に残るのは「移設 0 件」の確認と
本ファイルの削除のみになる。本ファイルの削除トリガは #241 のクローズで、この結論が
レビューで否認された場合は #241 ではなく本 issue（#240）へ差し戻し、否認された判定を
基準に照らして再判定してから改めて #241 へ渡す。

**申し送り（境界外・#241 での対応）**: いずれも本レーンの境界（`docs/agents/**`）の外なので
#241 側で直す。

1. `docs/quality/` が 0 件のまま残る帰結は、`CLAUDE.md`「ドキュメント」節の「quality /
   test_plan は既存 item の移設で作られる」という記述と食い違う。同節は `docs/test-plan/` を
   「item もディレクトリもまだ無い」と書いたままでもある（#238 / #239 で作成済み）
2. `docs/model.yaml` の `quality` の `specification` フィールドに付いた「RFC2119 キーワード
   検証は requirement 型にハードコードされており quality / test_plan の specification には
   効かない」というコメント（`test_plan` 側はこれを参照している）は**事実に反する**。
   sara 0.9.4 の `validate_item_metadata` は `item_type` ではなく `specification` フィールド
   の有無だけで分岐しており、実地でも `test_plan` item から SHALL を外すと
   `Test Plan specification must contain at least one RFC2119 keyword` で `sara check` が
   落ちることを確認した。`requirement` の同フィールドのコメント（「sara は requirement の
   `specification` に … ハードコードで検証する」）も、型限定と読める書き方なので併せて直す

## quality item が今後生まれる先

現状 `docs/quality/` は空のままになる。requirement からの移設では埋まらないので、
quality を作るなら requirement 以外を出典に新規起票することになる。候補（本 issue の
スコープ外・観測の記録のみ）:

- `CLAUDE.md`「規約」節: main ブランチへの直接コミット禁止・PR 経由でマージする
- `docs/agents/sara-graph.md`: `specification` は英語で書く・SHALL 系のみを使う
- `docs/dev/definition-of-done.md`: E2E と `nix flake check`（3 プラットフォーム）の
  green をマージの必須条件とする

`docs/infrastructure/` の 6 件（CI パイプライン・merge gate・リリース自動化など）は
「規範を支える機械」側であり、quality ではなく infrastructure のまま（→ 基準
「`quality` versus `infrastructure`」）。

## 判定を分けた主な類型

グレーゾーンに当たった 19 件は、基準の「Grey zones」節が定めた先例で判定した。

| 類型 | 件数 | 基準の対応節 | 該当 |
|---|---|---|---|
| アーキテクチャ境界（`bound`）| 6 | A norm about how nput is built | `REQ-637599dc` / `REQ-6c4e174a` / `REQ-b74a118a` / `REQ-d85f0cef` / `REQ-2bd0d35f` / `REQ-f4d7d4ab` |
| エラー・警告の出力（`msg`）| 5 | A norm about error and warning output | `REQ-07c3b735` / `REQ-8ef34101` / `REQ-95e97d01` / `REQ-9fca28c9` / `REQ-fea038de` |
| nix ワークフロー（`nixflow`）| 3 | A norm about a nix-level workflow | `REQ-67095391` / `REQ-d0aef5af` / `REQ-f9920c87` |
| 提供しないことの宣言（`scope`）| 3 | A declaration that something is out of scope | `REQ-4fc98fa6` / `REQ-d41b1d0a` / `REQ-fc1118b1` |
| バージョン・互換方針（`compat`）| 2 | A version or compatibility policy | `REQ-250d936c` / `REQ-79ce0a09` |

いずれも「落とせば nput の使い方が変わるか」の signature test で requirement に落ちた。
`scope` の 3 件は、テスト範囲のスコープ外宣言（`TP-b7f1dc79`）と違って**プロダクトが機能を
提供しないこと**の宣言であり、利用者が使える操作の範囲を規定するため requirement 側になる。

`REQ-2b0c2bb8`（`mkManifest` は nix-unit / namaka の単体対象でもある）と `REQ-b232ec98`
（`normalizeManifest` について同様）は、いずれも specification にテストへ言及する SHALL 文を
持つが requirement のまま。基準の先例（`TP-403c55c7`）が test_plan と裁く条件は「その面が
存在する唯一の動機が検証であること」かつ「契約としての保証を自ら放棄していること」の 2 つで、
この 2 件はどちらも満たさない。`mkManifest` / `normalizeManifest` はテストが無くてもプロダクト
の動作に必要な関数であり、シグネチャと純粋性という契約を放棄せず宣言している。

## 全件

判定はすべて requirement 据え置きのため、判定列は全行同一。

| item ID | name | 現状の親 | 判定 | 判定理由 |
|---|---|---|---|---|
| `REQ-02a33511` | apply --dryrun は読み取り専用で conflict 検出時に非ゼロ終了する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-053cfed2` | target に通常ファイル・ディレクトリが在れば上書きせずエラーで停止する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-059eb4d5` | --all は config ごとの SubjectResult を単一実行と同一形状で積む | `UC-1c280dce` / `UC-0b6f60cb` | requirement 据え置き | --json 契約（niface エンベロープ）の形を規定する。機械 consumer が読む契約 |
| `REQ-05abce3e` | rollback と list-generations は home mode 限定にする | `UC-0b6f60cb` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-07c3b735` | copy が foreign 実ファイルを skip したときは warning で可視化する | `UC-403fbe32` | requirement 据え置き | エラー・警告の出力内容と経路を規定する。利用者が直接受け取る振る舞い（→ 基準「A norm about error and warning output」） |
| `REQ-0a123b89` | 冗長度は -v、デバッグは --debug に分離し --json と直交させる | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-0b0cd1e3` | manifest.json の entries は attrset を配列へ正規化し 5 フィールドを記録する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | Nix ↔ engine の契約（manifest.json）の形を規定する。consumer が読む契約 |
| `REQ-0bd55dfc` | copy は src ツリー内の symlink を deref せず symlink のまま複製する | `UC-403fbe32` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-0e341430` | rollback は FS を先に収束させてから profile ポインタを最後に移す | `UC-0b6f60cb` | requirement 据え置き | 世代・profile の振る舞いを規定する。rollback / GC として利用者から観測できる |
| `REQ-14f0aec9` | nput CLI は PATH 常駐の一次 UX で、project mode は devShell 同梱を canonical とする | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-16aef46b` | stale 除去は前世代の記録通りを指す symlink のみに限り、copy は消さず orphan を警告する | `UC-0b6f60cb` / `UC-19a90989` / `UC-403fbe32` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-16faf428` | 意図が矛盾する組み合わせをクロスフィールドチェックで評価時に拒否する | `UC-f2436d68` / `UC-19a90989` / `UC-403fbe32` / `UC-01b896b4` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-1767b250` | config は Nix で書き nix build で評価する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-196ddabf` | template は動く example を 1 config だけ置きバリエーションはコメントで示す | `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-1be4d678` | 世代は link farm derivation を nput 自前 profile へコミットして積み、前世代 manifest から stale を除去する | `UC-0b6f60cb` / `UC-19a90989` / `UC-d39c1994` | requirement 据え置き | 世代・profile の振る舞いを規定する。rollback / GC として利用者から観測できる |
| `REQ-1c1526b1` | flock の取得は既定 blocking とし --no-wait のときだけ try-lock でスキップする | `UC-19a90989` / `UC-1c280dce` / `UC-0b6f60cb` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-1cc080f6` | entrypoint は CWD で flake.nix → shell.nix → default.nix の順に探し -f で上書きする | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-1dcc9a33` | marker は判別タグで識別し manifest.json には漏らさない | `UC-01b896b4` / `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | Nix ↔ engine の契約（manifest.json）の形を規定する。consumer が読む契約 |
| `REQ-1f128917` | gitignore --all は projectRoot の全 config の target をソート + 重複除去して出力する | `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-205d744d` | config 名 default を慣例の解決先とし専用 nput 名前空間で packages を汚さない | `UC-1c280dce` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-2353259f` | --json 指定時は行指向 stdout を出さずエンベロープが stdout を専有する | `UC-f2436d68` / `UC-19a90989` / `UC-403fbe32` / `UC-0b6f60cb` | requirement 据え置き | --json 契約（niface エンベロープ）の形を規定する。機械 consumer が読む契約 |
| `REQ-250d936c` | MVP は manifest v1 のみを発行・受理しマイグレーション機構を持たない | `UC-f2436d68` / `UC-19a90989` / `UC-0b6f60cb` | requirement 据え置き | 契約のバージョン方針。consumer が読む manifest.json の受理範囲を縛る（→ 基準「A version or compatibility policy」） |
| `REQ-27b75fe6` | subpath は src 内の相対パスとし、リポジトリ全体は省略で表して専用トークンを設けない | `UC-f2436d68` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-2a613337` | reset --json は --yes を必須とし無ければ fail fast する | `UC-403fbe32` / `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | --json 契約（niface エンベロープ）の形を規定する。機械 consumer が読む契約 |
| `REQ-2aa3abbc` | profileDir は config 専用ディレクトリとし、profile リンク・世代・pending out-link をその中に並べる | `UC-0b6f60cb` / `UC-19a90989` / `UC-403fbe32` | requirement 据え置き | 世代・profile の振る舞いを規定する。rollback / GC として利用者から観測できる |
| `REQ-2b0c2bb8` | mkManifest は配置データを生成する純粋関数である | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-2b48620a` | method 変更は symlink→copy のみ配置前除去で移行し、copy→symlink は移行しない | `UC-403fbe32` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-2bd0d35f` | modules/common.nix は nixpkgs.lib のみに依存する | `UC-d39c1994` | requirement 据え置き | アーキテクチャ境界の宣言。lib / engine を任意環境から取り込める性質は利用者が観測できる（→ 基準「A norm about how nput is built」） |
| `REQ-2c5a10d8` | 終了コードは 0 = 成功 / 1 = 一般エラー / 2 = dryrun の conflict とする | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-2ea19863` | 変更系の JSON ペイロードは engine 結果からフルインベントリと実差分を導く | `UC-f2436d68` / `UC-19a90989` / `UC-403fbe32` / `UC-0b6f60cb` | requirement 据え置き | --json 契約（niface エンベロープ）の形を規定する。機械 consumer が読む契約 |
| `REQ-2f9205ee` | mkManifest の返り値は passthru で root kind を露出する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-31dae599` | reset の確認プロンプトは stdin が TTY のときだけ出し、非 TTY で同意が無ければ即エラー停止する | `UC-403fbe32` / `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-31f2882e` | reset は profile を触らない FS-only teardown で配置物を無い状態へ戻す | `UC-403fbe32` / `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-37b56673` | root は 3 マーカーと絶対パス文字列の union を取る | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-3e446ad9` | entry submodule は strict とし未知キーと旧名を評価時エラーにする | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-3f541d39` | root マーカーは kind を運ぶ入れ物でパス解決は engine が行う | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-46fccb80` | project mode は世代を非公開にし、derivation 同一なら世代を積まず lstat ドリフト修復だけ行う | `UC-19a90989` | requirement 据え置き | 世代・profile の振る舞いを規定する。rollback / GC として利用者から観測できる |
| `REQ-496b1a07` | entrypoint は nput.<name> に named manifest を公開し CLI は形ごとの attr path で build する | `UC-1c280dce` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-4cbd9a0d` | apply --all は辞書順に適用し部分失敗でも続行して最後に集約する | `UC-1c280dce` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-4ec3accc` | root は明示必須で暗黙デフォルトを持たない | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-4fc98fa6` | 一部 entry だけを適用する --only は提供しない | `UC-1c280dce` | requirement 据え置き | 機能を提供しないことの宣言。利用者が使える操作の範囲を規定する（→ 基準「A declaration that something is out of scope」）|
| `REQ-4ffda99a` | 内部実行する nix コマンドを開示し世代の切替と GC は標準の nix コマンドへ委譲する | `UC-0b6f60cb` / `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-535b811d` | apply --all は rootKind を 1 回の一括 eval で取る | `UC-1c280dce` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-57137302` | item id は identity の JCS を SHA-256 した小文字 hex とする | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | --json 契約（niface エンベロープ）の形を規定する。機械 consumer が読む契約 |
| `REQ-5923ac79` | 単一 HM config 内の configs 間 target 衝突は eval 時 assertion で停止する | `UC-1c280dce` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-5c2e64c3` | エンベロープはコマンド完了時に 1 回だけ出し成立条件を満たさない実行では出さない | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | --json 契約（niface エンベロープ）の形を規定する。機械 consumer が読む契約 |
| `REQ-5c6b07da` | target 衝突の検出経路を同一 manifest 内と cross-config で分ける | `UC-1c280dce` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-5dd5a4e9` | apply --backup は配置を塞ぐ記録外実体を rename 退避してから配置する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-5e75aabc` | 途中失敗した apply / rollback はインメモリ undo ジャーナルで全 FS 変更を巻き戻す | `UC-0b6f60cb` / `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-60787ed2` | gitignore は method を区別せず copy target も含めて全 target を列挙する | `UC-19a90989` / `UC-403fbe32` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-60c6b7ea` | 実行フローの順序は eval 先行 → flock → build とし build をロック内に閉じる | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-60e6b49c` | mkManifest は manifest.json と symlink farm を含む store オブジェクトを返す | `UC-f2436d68` / `UC-19a90989` / `UC-0b6f60cb` | requirement 据え置き | Nix ↔ engine の契約（manifest.json）の形を規定する。consumer が読む契約 |
| `REQ-61856da1` | 既存 symlink の張替えは unlink + symlink の 2 操作で行い冪等な再実行で収束させる | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-61c05e09` | --root は全モード共通で解決 root を明示上書きする | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-622787dc` | symlink 配置は親 dir を作り配置元/subpath を指すリンクを張り、foreign symlink は警告して後勝ちする | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-62eda895` | symlink farm の GC アンカー名は target のハッシュとする | `UC-0b6f60cb` | requirement 据え置き | 世代・profile の振る舞いを規定する。rollback / GC として利用者から観測できる |
| `REQ-637599dc` | CLI が持ち込む依存は許可した第三者ライブラリと pin した Go に限り、いずれも固定する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | アーキテクチャ境界の宣言。lib / engine を任意環境から取り込める性質は利用者が観測できる（→ 基準「A norm about how nput is built」） |
| `REQ-6506bc82` | project mode で git から root を解決できないときは engine 実行時に停止する | `UC-19a90989` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-67095391` | flake は pure eval で flake.lock が固定し legacy は impure を許容しユーザー責任とする | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | consumer の flake / nix ワークフローに現れる性質を規定する（→ 基準「A norm about a nix-level workflow」） |
| `REQ-687e225f` | apply 修飾フラグは --all と合成できる | `UC-1c280dce` / `UC-403fbe32` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-6911eab6` | target / subpath のパス安全性を評価時に検査する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-6a950d6d` | reset --dryrun は副作用ゼロで削除対象を表示して終了する | `UC-403fbe32` / `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-6be1cbf1` | nput init は nix flake init -t への透明なラッパーとしファイルを生成しない | `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-6c4e174a` | engine が叩く外部コマンドは nix と git のみに限る | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | アーキテクチャ境界の宣言。lib / engine を任意環境から取り込める性質は利用者が観測できる（→ 基準「A norm about how nput is built」） |
| `REQ-706de717` | 世代操作は nix-env --profile 系で統一し、GC root の間引きと store 解放を分けて行う | `UC-0b6f60cb` | requirement 据え置き | 世代・profile の振る舞いを規定する。rollback / GC として利用者から観測できる |
| `REQ-774cef80` | 要求された操作が成立しないときは CLI がエラーで停止し、暗黙のフォールバックを採らない | `UC-f2436d68` / `UC-19a90989` / `UC-0b6f60cb` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-77689c68` | method は配置方法を選び symlink は世代管理下・copy は世代管理外になる | `UC-f2436d68` / `UC-403fbe32` / `UC-01b896b4` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-79ce0a09` | manifest.json が Nix と engine の唯一の安定契約であり schemaVersion は 1 に固定する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | 契約のバージョン方針。consumer が読む manifest.json の受理範囲を縛る（→ 基準「A version or compatibility policy」） |
| `REQ-7a71a049` | --dryrun は root を解決するが flock も pending gcroot も取らない | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-7cc32a2b` | apply --recopy は config 内の全 copy target を src から無条件に上書き再コピーする | `UC-403fbe32` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-7cee95dd` | 実 dir の target は全 leaf が除去可能なときだけ全体を配置前除去して symlink 化する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-8085f194` | home-manager モジュールの engine kick 1 回は activation からビルド済み link-farm を渡し、失敗で switch を止める | `UC-d39c1994` | requirement 据え置き | モジュール統合層が利用者へ公開する option と activation の振る舞いを規定する |
| `REQ-81249072` | out-of-store のローカルパスは評価時に確定し、target の root 解決には影響しない | `UC-01b896b4` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-8409db86` | target 除去後は空の親ディレクトリチェーンを root 境界まで保守的に剪定する | `UC-f2436d68` / `UC-19a90989` / `UC-403fbe32` / `UC-0b6f60cb` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-840b3641` | 失敗時に残る pending gcroot は config あたり最大 1 個に有界とし回収処理を持たない | `UC-0b6f60cb` | requirement 据え置き | 世代・profile の振る舞いを規定する。rollback / GC として利用者から観測できる |
| `REQ-844ee375` | module 時は rollback を host へ一本化し、nput profile は前進のみで追従する | `UC-d39c1994` | requirement 据え置き | モジュール統合層が利用者へ公開する option と activation の振る舞いを規定する |
| `REQ-84e3c717` | copy は元の mode を保存しつつ owner-write を付与する | `UC-403fbe32` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-89c7baf9` | rollback は名指し必須で --all に対応しない | `UC-0b6f60cb` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-8d965ca2` | home mode の root は層ごとに定まった供給元から解決する | `UC-f2436d68` / `UC-d39c1994` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-8ef34101` | 成功時はデフォルト沈黙とし warning と error は常時 stderr へ出す | `UC-f2436d68` / `UC-19a90989` / `UC-0b6f60cb` / `UC-403fbe32` | requirement 据え置き | エラー・警告の出力内容と経路を規定する。利用者が直接受け取る振る舞い（→ 基準「A norm about error and warning output」） |
| `REQ-9341fa5d` | エンベロープのエラーは主体の有無で層を分けコードを分類する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | --json 契約（niface エンベロープ）の形を規定する。機械 consumer が読む契約 |
| `REQ-95e97d01` | conflict で停止するときは全件を対処ガイダンス付きで列挙してから 1 本の集約エラーを返す | `UC-f2436d68` / `UC-19a90989` / `UC-0b6f60cb` | requirement 据え置き | エラー・警告の出力内容と経路を規定する。利用者が直接受け取る振る舞い（→ 基準「A norm about error and warning output」） |
| `REQ-97c1e088` | mkManifest の引数は pkgs / entries / root の 3 つとする | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-99ca5381` | src は path / set / marker の 3 種を取り store link を既定として out-of-store は marker で opt-in する | `UC-f2436d68` / `UC-01b896b4` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-9b0046e0` | backup 退避は配置前除去の後・配置の前に置き、drift 修復経路でも同じく実施する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-9c111c32` | 非 build コマンドも eval 先行を共通前段に持つ | `UC-f2436d68` / `UC-19a90989` / `UC-0b6f60cb` / `UC-403fbe32` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-9cb26ffd` | project mode の root は git toplevel から解決し、config 相対も CWD 相対も採らない | `UC-19a90989` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-9dc7dac7` | 配置元の実在は判定できる層で検査し、いずれの層でも停止する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-9ed6b500` | --version は埋め込みバージョンを cobra 既定書式で表示して終了し短縮形を持たない | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-9fca28c9` | 巻き戻し自体の失敗は best-effort で続行し、全件を stderr へ報告して停止する | `UC-0b6f60cb` / `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | エラー・警告の出力内容と経路を規定する。利用者が直接受け取る振る舞い（→ 基準「A norm about error and warning output」） |
| `REQ-a0bdf6db` | devShell は shellHook から engine を起動する配線で、シェル入室のたびに project mode で配置する | `UC-19a90989` | requirement 据え置き | モジュール統合層が利用者へ公開する option と activation の振る舞いを規定する |
| `REQ-a33a11e3` | entry submodule のフィールドは src / subpath / target / method の 4 つとする | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-a480c183` | gitignore は配置 target を stdout へ列挙するだけでファイルを書き込まない | `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-a5053191` | --json は niface 規約準拠のエンベロープを出す第 2 契約とする | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | --json 契約（niface エンベロープ）の形を規定する。機械 consumer が読む契約 |
| `REQ-a8a923ad` | out-of-store symlink は marker の絶対パスを指し、版管理はリンク先マッピングのみとする | `UC-01b896b4` | requirement 据え置き | 世代・profile の振る舞いを規定する。rollback / GC として利用者から観測できる |
| `REQ-a8edc58f` | reset は名指し必須で profileDir 単位の blocking flock を取る | `UC-403fbe32` / `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-b12fc3c0` | symlink farm は GC アンカー専用でアンカーは store-backed な symlink entry に限る | `UC-0b6f60cb` | requirement 据え置き | 世代・profile の振る舞いを規定する。rollback / GC として利用者から観測できる |
| `REQ-b232ec98` | normalizeManifest が検査・デフォルト適用・marker 変換を行い mkManifest が derivation を組む | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-b4e4b65d` | recopy の上書きは削除ではなく同一親内への rename 退避で行う | `UC-403fbe32` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-b74a118a` | engine は第三者依存ゼロの stdlib-only とし内部層に閉じる | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | アーキテクチャ境界の宣言。lib / engine を任意環境から取り込める性質は利用者が観測できる（→ 基準「A norm about how nput is built」） |
| `REQ-b7bb09d6` | apply --all --dryrun の終了コードは error を conflict より優先する | `UC-1c280dce` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-c1b3ca5f` | 全モジュールと devShell は engine をキックするだけの配線とし、ネイティブ機構へ翻訳しない | `UC-d39c1994` | requirement 据え置き | モジュール統合層が利用者へ公開する option と activation の振る舞いを規定する |
| `REQ-c2654ca5` | NixOS / nix-darwin モジュールは配置先ユーザーを特定する user オプションを必須で取る | `UC-d39c1994` | requirement 据え置き | モジュール統合層が利用者へ公開する option と activation の振る舞いを規定する |
| `REQ-c2d44626` | apply の config 選択は name 省略で default・明示で単一・--all で全件 | `UC-1c280dce` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-c50df875` | flake-parts 経路は直書きと同一の derivation を生み CLI のアドレッシングを変えない | `UC-d39c1994` | requirement 据え置き | モジュール統合層が利用者へ公開する option と activation の振る舞いを規定する |
| `REQ-c5dfcae6` | 設定の誤りは評価時に、実体の不整合は engine 実行時に検出する層分けを守る | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-c6891aeb` | HM モジュール経由でも名前つき config ごとに役割分離した独立 profile を取れる | `UC-d39c1994` / `UC-1c280dce` | requirement 据え置き | モジュール統合層が利用者へ公開する option と activation の振る舞いを規定する |
| `REQ-c847d1af` | HM の activation は configs を辞書順に走査して profile ごとに engine を起動し、部分失敗を最後に集約する | `UC-d39c1994` / `UC-1c280dce` | requirement 据え置き | モジュール統合層が利用者へ公開する option と activation の振る舞いを規定する |
| `REQ-c890ce4a` | legacy entrypoint は mkShell passthru 形を canonical とし CLI の attr path を分岐させない | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-c9ab91c1` | 祖先 symlink は自己記録 stale のみ配置前除去し、それ以外はエラーで停止する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-cb77ea05` | entry の識別子は属性キー = target とし一意性は Nix が担保する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-cbd61281` | init のテンプレート参照はバイナリにハードコードした固定 flake ref とする | `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-d0aef5af` | nput カスタム output は nix flake check の unknown 警告を許容し主検証は nix build で行う | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | consumer の flake / nix ワークフローに現れる性質を規定する（→ 基準「A norm about a nix-level workflow」） |
| `REQ-d1b5b3f5` | mkManifest 自身が evalModules で入力を検査する単一ゲートになる | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-d2277c7a` | copy は target 不在のときだけマテリアライズする place-once で世代管理の対象外とする | `UC-403fbe32` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-d41b1d0a` | 孤児 profile は backref で逆引き可能なまま放置許容とし、MVP では cleanup コマンドを持たない | `UC-19a90989` | requirement 据え置き | 機能を提供しないことの宣言。利用者が使える操作の範囲を規定する（→ 基準「A declaration that something is out of scope」）|
| `REQ-d5a2e289` | profileDir は home のみ name 直キーとし、fixed root と --root 上書きは roothash でキーする | `UC-0b6f60cb` / `UC-19a90989` / `UC-403fbe32` | requirement 据え置き | 世代・profile の振る舞いを規定する。rollback / GC として利用者から観測できる |
| `REQ-d85f0cef` | lib は nixpkgs.lib のみに依存する純データ生成器である | `UC-f2436d68` / `UC-19a90989` / `UC-d39c1994` | requirement 据え置き | アーキテクチャ境界の宣言。lib / engine を任意環境から取り込める性質は利用者が観測できる（→ 基準「A norm about how nput is built」） |
| `REQ-d95b814f` | --all は root モードフィルタで対象 config を絞れる | `UC-1c280dce` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-da253cab` | legacy entrypoint では相対 path の src が自動で store 化されない | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-dd10d820` | manifest.json の root は rootKind を持ち fixed のときだけ絶対パスを併記する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | Nix ↔ engine の契約（manifest.json）の形を規定する。consumer が読む契約 |
| `REQ-dec58330` | apply --manifest はビルド済み link-farm を engine へ直接適用する | `UC-d39c1994` | requirement 据え置き | モジュール統合層が利用者へ公開する option と activation の振る舞いを規定する |
| `REQ-dedd2c28` | manifest.json のトップレベルは schemaVersion / root / entries の 3 フィールドとする | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | Nix ↔ engine の契約（manifest.json）の形を規定する。consumer が読む契約 |
| `REQ-e1e1114b` | nput.backup は engine 起動の配線レイヤーのオプションで manifest には影響しない | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | モジュール統合層が利用者へ公開する option と activation の振る舞いを規定する |
| `REQ-e79178f5` | project mode の配置物は ephemeral とし、activation は git 状態に干渉しない | `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-eaa8c0df` | gitignore は project mode 限定で非 project config を指定したらエラーで停止する | `UC-19a90989` | requirement 据え置き | CLI が利用者へ見せる振る舞い。落とせば nput の使い方が変わる |
| `REQ-eb363122` | mkOutOfStoreSymlink は out-of-store symlink を表すマーカーを返す | `UC-01b896b4` | requirement 据え置き | lib の公開 API の形と意味を規定する。利用者が entrypoint で直接触る面 |
| `REQ-f4d7d4ab` | nput は CLI とエンジンの 2 層で構成する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | アーキテクチャ境界の宣言。lib / engine を任意環境から取り込める性質は利用者が観測できる（→ 基準「A norm about how nput is built」） |
| `REQ-f9920c87` | nix experimental-features は前提条件とし、CLI は自動付与せず案内エラーで停止する | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | consumer の flake / nix ワークフローに現れる性質を規定する（→ 基準「A norm about a nix-level workflow」） |
| `REQ-fa181aa6` | 読み取り系の JSON ペイロードは dryRun パリティと info インベントリで表す | `UC-f2436d68` / `UC-19a90989` / `UC-0b6f60cb` | requirement 据え置き | --json 契約（niface エンベロープ）の形を規定する。機械 consumer が読む契約 |
| `REQ-fc1118b1` | 同一 target を複数 config で狙うことによる振動はユーザー責任とし warning で可視化するに留める | `UC-1c280dce` | requirement 据え置き | 機能を提供しないことの宣言。利用者が使える操作の範囲を規定する（→ 基準「A declaration that something is out of scope」）|
| `REQ-fc1c7ce6` | 全モジュールは共通オプションの同一集合を公開し、entries は configs 経由・root はモジュールが pin する | `UC-d39c1994` | requirement 据え置き | モジュール統合層が利用者へ公開する option と activation の振る舞いを規定する |
| `REQ-fc64de4c` | 空の entries は正当な全クリアとして扱い、エラーにも警告にもしない | `UC-1c280dce` | requirement 据え置き | engine の実行時の振る舞い（配置・除去・失敗時の収束）を規定する |
| `REQ-fea038de` | stdout は機械可読出力を専有しレポートと warning は stderr へ出す | `UC-f2436d68` / `UC-19a90989` | requirement 据え置き | エラー・警告の出力内容と経路を規定する。利用者が直接受け取る振る舞い（→ 基準「A norm about error and warning output」） |
