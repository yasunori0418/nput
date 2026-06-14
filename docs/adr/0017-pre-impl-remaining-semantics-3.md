# ADR-0017: 実装前レビュー第 3 巡で surfaced した残セマンティクスを確定する（--all の root モードフィルタ / --root の適用範囲 / 世代スキップとドリフト修復 / 張替えの atomic 性）

- ステータス: 採用
- 日付: 2026-06-14
- 関連: ADR-0002, ADR-0005, ADR-0006, ADR-0013, ADR-0015, ADR-0016, `docs/spec.md`, `docs/design.md`, `docs/concept.md`, `CONTEXT.md`
- 改訂対象: **ADR-0013**「`--all` は全 config を適用」に root モードフィルタを追加。**ADR-0005 / ADR-0006**「project mode の世代スキップ = 完全 no-op」を「lstat 検査 + 必要時のみ再張り」へ精緻化
- 起点: ADR-0016 に続く実装前レビュー第 3 巡（grill）で surfaced した残細目の束

## 背景

ADR-0015 / ADR-0016 で実装前残セマンティクスの大半が固まったが、第 3 巡のレビューで次の 4 点に
未定義 / footgun / 仕様の揺れが残っていた。

1. **`apply --all` が root モードを区別せず全 config を適用する footgun**。`nput.*` に home mode と project mode の
   config が混在する entrypoint で、devShell の `shellHook` から `--all` を打つと **home mode config も `$HOME` に配置**される。
2. **`--root <path>` の適用範囲が曖昧**。spec は「project mode の root を上書き」と書くが、実行フローは汎用的に「--root 上書き」。
   home mode / fixed root の config に `--root` を渡したときの挙動が未定義。
3. **project mode の世代スキップとドリフト修復の関係が未定義**。新 link-farm が前世代と同一なら no-op（世代スキップ）だが、
   間に foreign tool が target を書き換えても derivation 不変なら skip され、**ドリフトが修復されない**。
4. **単一 entry の symlink 張替えの atomic 性**。replace = unlink + symlink の 2 操作で、間でクラッシュすると target が
   一時消失する（再実行で収束）。

## 決定

### 1. `apply --all` に root モードフィルタ（`--project-root` / `--home-root` / `--system-root`）を追加する

- `--all` に **root マーカー名に揃えたフィルタフラグ**を追加する: `--project-root` / `--home-root` / `--system-root`。
  指定すると `nput.*` のうち該当 root モードの config **のみ**を適用する。
- **素の `--all`（フィルタなし）は従来通り全 config を辞書順に適用**する（ADR-0016）。フィルタは opt-in。
- devShell の `shellHook` は config を絞るため **`nput apply --all --project-root`**（または名指し `nput apply skills`）を
  使うよう docs にガイドする。これで home mode config が devShell 入室で誤って `$HOME` に配置される footgun を回避できる。
- フィルタは `--all` と併用する修飾で、名指し apply（`nput apply <name>`）では `<name>` が 1 config を pin するため無意味。
- `systemRoot` は eval 時に未実装拒否（ADR-0013）のため、`--system-root` は当面マッチする config が無い将来 seam。

### 2. `--root <path>` は全モードで解決 root を上書きする

- `--root` は manifest の `rootKind`（project / home / system / fixed）に関わらず、**engine が解決する絶対 root を一律に上書き**する。
  デバッグ・テスト・特殊配置の脱出路として project mode 以外でも使える。
- project mode の `<roothash>` profile キーも **上書き後の root** で計算する（上書き root ごとに profile が分かれる）。
- spec の「project mode の root を上書き」という限定記述を「**全モードの解決 root を上書き**」へ一般化する。

### 3. 世代スキップでも FS 検査だけは軽量に行い、ドリフト時はその entry のみ修復する

- project mode の世代スキップを「**完全 no-op**」から「**lstat 検査 + 必要時のみ再張り**」へ精緻化する。
- 新 link-farm derivation が前世代と同一なら **新世代は積まない**（世代無限増殖の回避は維持・ADR-0005）。
  ただし各 entry の target を **lstat で軽量検査**し、記録通りでない（foreign に書き換えられた・消えた）entry が
  あればその entry **だけ**再張りする（foreign symlink なら warning・ADR-0015）。
- lstat 比較は安価で `shellHook` 高頻度実行にも耐える。「derivation 同一 = FS も同一」という暗黙前提を捨て、
  ephemeral 配置でも devShell 再入室でドリフトが収束するようにする。

### 4. symlink 張替えは unlink + symlink のまま（atomic swap にしない）

- replace は **unlink + symlink の 2 操作のまま**とし、rename ベースの atomic swap は採らない。
- クラッシュ窓（target 一時消失）は **冪等な再実行で収束**する（ADR-0006「積まれる世代は常に完全適用済み」と整合）。
  atomic swap の実装コストに見合う安全性向上が薄いため、シンプルさを優先する。docs に「張替えは非 atomic・再実行で収束」と
  一行注記するに留める（仕様変更なし）。

## 根拠

- **root モードフィルタ**は混在 entrypoint の footgun を opt-in で塞ぐ。素の `--all` を全適用のまま残すことで既存の
  単一モード運用を壊さず、devShell には `--project-root` ガイドで安全な既定を与える。フラグ名をマーカー名に揃えることで
  「`projectRoot` の config を適用」という対応が自明になる。
- **`--root` 全モード上書き**は project mode 限定にする技術的理由が無い。テスト時に home mode config を一時ディレクトリへ
  逃がす等の正当な用途があり、限定はむしろ脱出路を塞ぐ。
- **世代スキップの lstat 検査**は「derivation 同一 ⇒ FS 同一」が foreign 書き換えで崩れる現実に対応する。完全 no-op は
  最速だがドリフトを放置し、毎回完全再アサートは世代増殖（project mode で禁忌・ADR-0005）を招く。lstat 検査は両者の中庸で、
  世代を積まずに FS だけ収束させられる。
- **張替えを非 atomic のまま**にするのは ADR-0006 の冪等収束方針と一貫。atomic swap は窓を消すが、クラッシュ自体が稀で
  再実行が収束させるため、複雑性に見合わない。

## 影響

- **`docs/spec.md`**:
  - CLI 仕様 / グローバルフラグに `--project-root` / `--home-root` / `--system-root`（`--all` 修飾）を追加。`apply --all` 節に
    フィルタ説明と devShell ガイドを追記。
  - グローバルフラグ `--root` の「project mode の root を上書き」を「全モードの解決 root を上書き」へ。root 解決節の
    `--root` 記述も一般化。
  - 世代管理仕様 / devShell 節 / 実行フローの「世代スキップ = no-op」を「lstat 検査 + 必要時のみ再張り（新世代は積まない）」へ。
  - 配置動作仕様（symlink モード）に「張替えは unlink + symlink で非 atomic・再実行で収束」を一行注記。
- **`docs/design.md`**: 使用パターン 1 の devShell `shellHook` を `nput apply skills`（名指し）または `--all --project-root` に
  揃える注記。世代スキップの精緻化を反映。
- **`docs/concept.md`**: project mode 説明 / 使用例 devShell の `shellHook` ガイドを反映。
- **`CONTEXT.md`**: 必要なら `generation` / project mode 周辺へ世代スキップ精緻化を反映（glossary 粒度では任意）。
- **ADR-0013**: 改訂注記で「`--all` に root モードフィルタ（`--project-root` 等）を追加・素の `--all` は全適用維持」。
- **ADR-0005**: 改訂注記で「世代スキップ = 完全 no-op → lstat 検査 + 必要時のみ再張り（新世代は積まない）」。
- **実装フェーズ**: `cmd/nput`（`--all` の root モードフィルタ・`--root` 全モード上書き）、`internal/`（世代スキップ時の
  lstat 検査 + ドリフト entry の再張り）、`templates/project`（shellHook を `--all --project-root` か名指しに）。

## 棄却した代替案

- **`--all` を完全モード非依存のまま（ドキュメント明記のみ）**: シンプルだが混在 entrypoint の footgun を docs 頼みにする。
- **`--all` を混在時 error 停止**: footgun は確実に塞げるが、home + project を 1 レポで管理する正当な混在運用を阻害する。
- **`--all=<mode>` の値付きフラグ**: フラグ数は減るが bool フラグ（`--all`）と値付きの混在で cobra 表現が複雑。
- **`--root` を project mode 限定**: home mode の root 不変性は守れるが、テスト等の正当な脱出路を塞ぐ。
- **`--root` を他モードで無視 + warning**: 「指定したのに効かない」驚きを生む。
- **世代スキップを完全 no-op のまま**: 最速だが foreign ドリフトを放置する。
- **世代スキップを廃止し毎回完全再アサート**: ドリフトは確実に収束するが世代無限増殖（ADR-0005 禁忌）。
- **張替えを rename ベース atomic swap**: 窓は消えるが実装コストに見合わず、ADR-0006 の冪等収束で十分。
