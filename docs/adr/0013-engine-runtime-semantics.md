# ADR-0013: engine 実行時セマンティクスの細目を確定する（manifest v1 / flock / --all / copy+marker / systemRoot / gitignore / roothash）

- ステータス: 採用（2026-06-14 拡張: 配置セマンティクスに foreign symlink warning・祖先 symlink error を追加 → ADR-0015）
- 日付: 2026-06-13
- 関連: ADR-0002, ADR-0005, ADR-0006, ADR-0007, ADR-0010, ADR-0011, ADR-0015, `docs/spec.md`, `docs/design.md`, `CONTEXT.md`
- 起点: ドキュメント計画段階のレビューで surfaced した「実装前に決めておくべき」細目の束

> **2026-06-14 拡張（ADR-0015）**: 本 ADR の配置 / flock セマンティクスに次を追加した。(a) cross-config（別 profile）の同一 target は
> 後勝ち維持だが、engine が **自身の前世代 manifest に記録の無い foreign symlink を上書きするときは warning** を出す。
> (b) target の **祖先 component が symlink なら配置前に lstat walk で検出し error 停止**（`--dryrun` も conflict 扱い）。
> 同一 profileDir 後勝ちポリシー（本 ADR §2）の cross-config / cross-tool への一般化（→ ADR-0015）。
>
> **2026-06-14 改訂注記（ADR-0016）**: 本 ADR §3 の「`apply --all` は **定義順**」を **辞書順（キーソート・決定的）**へ訂正した。
> `nput.*` は attrset で `builtins.attrNames` が辞書順を返すため定義順は保持されない。各 config は独立 profile で atomic なので
> 適用順は結果に影響せず、決定的でありさえすればよい（→ ADR-0016）。

## 背景

ADR-0006〜0012 で言語・層分離・契約方式・技術スタック・CI まで大枠は決まったが、
**engine 実行時の具体セマンティクス**に未定義の穴が残っていた。実装に入る前にこれらを固める。

1. `manifest.json` は「唯一の安定契約」（ADR-0011）なのに、`schemaVersion` の実値・全フィールド・symlink farm との対応が未 pin（ADR-0010 に部分マッピングがあるのみ）。
2. `profileDir` の flock は「保持中ならスキップ」一律（spec）。明示 `apply` で silent skip されるとユーザーが「適用済み」と誤認する。
3. `apply --all` で一部 config が失敗したときの挙動（stop / continue / 順序 / 終了コード）が未定義。
4. `mode = "copy"` + out-of-store marker は spec の表で「非推奨」とだけ書かれ、engine 実挙動が未定義。
5. `systemRoot` は lib API に露出している（ADR-0004/0007 で seam として正式マーカー化）が engine 未実装。使ったときの挙動が未定義。
6. `nput gitignore` は stdout 出力のみと決まっているが、行フォーマットが未定義。
7. project mode profileDir を `<roothash>/<name>` でキーする（ADR-0005）が、roothash の算出方式が未定義。孤児 profile の逆引き可否がこれで変わる。

## 決定

### 1. `manifest.json` v1 を `docs/spec.md` に全文 pin する

- `schemaVersion = 1` を実値として確定する。engine は自身の対応版より新しい `schemaVersion` を拒否する（ADR-0006）。
- 全フィールド（`root.rootKind` / `root.root`、`entries[].name` / `srcKind` / `src` / `subpath` / `target` / `mode`）と
  symlink farm との対応を spec に表 + JSON 例で固定する。lib / engine / テストが同一の一次ソース（spec）を見る。
- 内部タグ `_nputMarker` は manifest に漏らさず clean enum（`srcKind` / `rootKind`）で写す（ADR-0010 を踏襲）。
- symlink farm は **GC アンカー専用**。engine が配置に使う値は manifest が持つ解決済み store パス文字列（ADR-0010）であり、
  farm は store-backed src への GC root を張るためだけに併存する。out-of-store src は store 外なので farm アンカーを持たない。

### 2. flock は起動経路で振る舞いを分ける（明示=wait / shellHook=skip）、キーは解決後 profileDir

- **明示 `apply` / `rollback` は blocking lock（`LOCK_EX`、取得まで待つ）**。待機中は「他の apply 完了待ち」を表示する。
- **devShell の shellHook 経路は try-lock（`LOCK_NB`）で skip**。高頻度実行で待ち行列を作らないため。判別はテンプレートが
  shellHook に `--no-wait`（仮称）相当を埋めることで行う。
- **ロックキーは解決後の `profileDir`**（home: `<name>` / project: `<roothash>/<name>`）。素の name だけでロックすると、
  同一 project を 2 箇所にクローンして両 devShell に入ったとき別 root なのに name 衝突で互いをブロックする。profileDir
  単位なら ADR-0005 の「クローン間衝突回避」と整合する。
- **同一 profileDir への同時実行（衝突）はユーザー責任**。基本は衝突しない前提とし、衝突時は **後勝ち**（明示 apply が待って
  直列化し、後に lock を取った側が前世代を読み直して適用する）。shellHook は skip するため衝突時は既存適用を尊重する。

### 3. `apply --all` は continue + 集約レポート + 非ゼロ終了

- 各 config は独立 profile で atomic（ADR-0002）。1 つの失敗で残りを止める理由がないため、失敗した config は
  スキップして残りを適用し、**最後に成功 / 失敗を集約表示**する。**1 つでも失敗なら非ゼロ終了**。
- **適用順序は entrypoint の `nput.*` 定義順**（決定的順序）。`--all` 自体を atomic（全成功か全 rollback）にはしない
  （project mode は rollback 非公開で意味論が崩れるため）。

### 4. `copy` + out-of-store marker は eval 時エラー

- `mode = "copy"` と out-of-store marker（`mkOutOfStoreSymlink`）の同時指定は **`normalizeManifest` で eval 時に
  `lib.throwIf` で弾く**（name 一意性チェックと同層のクロスフィールド検査）。
- out-of-store は「ライブ symlink」、copy は「place-once スナップショット」で意図が矛盾するため、暗黙降格ではなく
  早期エラーにする。「型 / 評価時に契約を強制する」（ADR-0010）と一貫させる。

### 5. `systemRoot` は API に露出を残し、使うと eval 時エラー

- `lib.systemRoot` は API に残す（north-star = distro の seam・ADR-0004/0007 を文書として保持）が、
  `root = systemRoot` を実際に使うと `normalizeManifest` が **「system mode は未実装（予定）」で eval 時 throw** する。
- API 面を将来に向け安定させつつ、eval は通って engine 実行時に未定義挙動になる罠を防ぐ。

### 6. `nput gitignore` は先頭 `/` アンカー・1 行 1 target

- 出力は **root 相対 target に先頭 `/` を付けた gitignore アンカー形式**（例: `/.claude/skills/nix`）を 1 行 1 件で stdout に出す。
- project mode の root = git toplevel = `.gitignore` の置き場所なので、先頭 `/` が正しくアンカーし、別階層の同名パスを
  誤って無視しない。ディレクトリ / ファイルの区別なく末尾 `/` は付けない。書き込みはしない（更新責務は管理者・ADR-0005）。

### 7. roothash = 短縮 sha256 hex + backref ファイル

- project mode profileDir のキー roothash は **解決後の絶対 root パスの sha256 を短縮した hex**（固定長・FS 安全）。
- profileDir の親（`<roothash>` 階層）に **元 root の絶対パスを記録した backref ファイル**（例: `.root`）を置く。
- これにより「クローン削除で残る孤児 profile がどの root 由来か」を人間が逆引きでき（ADR-0005 の孤児手動削除を現実的にする）、
  固定長キーと逆引き可能性を両立する。可逆エンコードは深いパスで profileDir 名が FS 制限を超えうるため採らない。

## 根拠

- **manifest v1 を spec に pin** することで、lib・engine・テストの 3 ワークストリームが実装開始前に同一契約を共有でき、
  「契約を最後に固める」ことによる手戻りを避ける（ADR-0011 の「安定面は manifest.json 1 点」を具体化）。
- **flock の経路分岐** は「明示操作は確実に適用したい / 高頻度 hook は待ち行列を作りたくない」という相反要求を、
  起動経路で素直に分けて両立させる。profileDir キーは ADR-0005 のクローン分離と一直線。
- **`--all` continue** は独立 profile の atomic 性を活かし、1 つの失敗で他を巻き込まない。集約 + 非ゼロで CI gate にも使える。
- **copy+marker / systemRoot を eval 時エラー** にするのは、ADR-0010 の「型・評価時に契約を強制」「未定義挙動を早期に弾く」と一貫。
- **gitignore アンカー** は project mode の root 定義（git toplevel）から自然に導かれる正しいアンカー位置。
- **roothash + backref** は ADR-0005 が「放置許容（または手動削除）」とした孤児 profile の手動削除を、逆引き手段を与えて現実的にする。

## 影響

- **`docs/spec.md`**: manifest.json v1 スキーマ節を新設。実行フローの flock を wait/skip 分岐 + profileDir キーに更新。
  `apply --all` 失敗の挙動を追記。エラー仕様表に「copy + out-of-store marker」「systemRoot 未実装」を追加。
  `gitignore` 出力形式を明記。project mode の世代節に roothash + backref を追記。
- **`CONTEXT.md`**: flock キー（profileDir）・後勝ちポリシーを `engine` / `generation` 周辺に反映。
  Flagged ambiguities に「standalone = CLI 起動形態であって配置モードではない」を追加。
- **実装フェーズ**: `lib/manifest.nix`（copy+marker / systemRoot の throwIf）、`internal/`（flock 経路分岐・--all 集約・roothash + backref・
  gitignore 整形）、`cmd/nput`（--no-wait フラグ・--all レポート）。

## 棄却した代替案

- **manifest スキーマを実装時に Go struct と擦り合わせて確定**: 契約を最後に固めることになり lib / engine / テストの並行着手で手戻りが出る。
- **manifest を別ファイル JSON Schema 一次ソース化**: doc/コードの二重管理は減るが、計画段階で JSON Schema 記法の重さが増す。spec の表 + 例で足りる。
- **flock を全経路一律 skip**: 明示 apply の silent skip でユーザーが誤認する。
- **flock を全経路一律 wait**: shellHook が高頻度で待ち行列を作り devShell 入室が詰まる。
- **flock キーを素の name だけにする**: 同一 project の複数クローンが別 root なのに互いをブロックする。
- **`apply --all` を fail-fast**: 独立 profile なのに 1 失敗で残りを止め、適用済み / 未適用が中途半端に混在する。
- **`apply --all` を全体 atomic**: project mode の rollback 非公開と衝突し意味論が崩れる。実装も過剰。
- **copy + marker を warn + symlink 降格**: 暗黙の振る舞い変更を生み「型で拒否」方針に逆行する。
- **systemRoot を今 API から外す**: north-star の seam を文書から消すことになり ADR-0004/0007 の意図が薄れる。
- **roothash を短縮 hash のみ**: 孤児 profile の逆引きができず手動削除が困難。
- **roothash を可逆エンコード**: 深いパスで profileDir 名が長大化し FS 制限に触れうる。
