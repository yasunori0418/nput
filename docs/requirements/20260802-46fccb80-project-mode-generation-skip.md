---
id: "REQ-46fccb80-4bae-4d37-bc19-dded88e9a9c0"
type: requirement
derives_from:
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
name: "project mode は世代を非公開にし、derivation 同一なら世代を積まず lstat ドリフト修復だけ行う"
specification: |
  In project mode the profile SHALL be keyed on the resolved root, so that cloning the
  same entrypoint into several places does not make the profiles collide and the stale
  removal of one clone does not destroy the placement of another — a problem that does not
  arise in home mode, where there is one per user. The `<roothash>` SHALL be a shortened
  hex of the sha256 of the resolved absolute root path, being thereby of fixed length and
  safe for a filesystem. Generations SHALL NOT be exposed to the user in project mode:
  the profile SHALL be confined to being the internal mechanism for stale removal and the
  generation-skip decision, and neither `rollback` nor `list-generations` SHALL be offered,
  the placed objects being ephemeral and there being no host generation to return to when
  kicked from a devShell. When the new link farm derivation is identical to that of the
  previous generation, no new generation SHALL be stacked; this is mandatory, since under
  devShell / direnv the `shellHook` runs on every shell re-entry and stacking a generation
  each time would make them grow without bound. Home mode SHALL remain as it is, stacking
  a new generation on every application, the generation skip being confined to project
  mode. Even when the generation is skipped, a lightweight filesystem inspection SHALL
  still be performed rather than making it a complete no-op: the target of each entry
  SHALL be inspected with `lstat`, and any entry that is not as recorded — rewritten by a
  foreign tool, or gone — SHALL be re-placed, and that entry alone, with a warning where
  it is a foreign symlink. The implication "identical derivation, therefore identical
  filesystem" breaks down under a foreign rewrite, so the filesystem SHALL be made to
  converge without stacking a generation; the `lstat` comparison is cheap and withstands
  the high frequency at which a `shellHook` runs. This drift repair SHALL cover both
  symlinks and copies. A copy entry SHALL be re-materialized under place-once only while
  its target is absent, and SHALL NOT be touched where the target exists but its content
  differs through the user having edited it, matching the behaviour of place-once in home
  mode; following the source SHALL remain confined to `apply --recopy`, and no comparison
  of content hashes SHALL be performed.
specification_ja: |
  project mode では profile を解決済み root でキーしなければならない。同一 entrypoint を複数箇所へ
  クローンしても profile が衝突せず、stale 除去が互いのクローンの配置を破壊しないようにするため
  である（home mode（1 ユーザー 1 つ）では起きない問題）。`<roothash>` は解決後の絶対 root パスの
  sha256 を短縮した hex（固定長・FS 安全）とする。project mode では世代をユーザーに公開しては
  ならない。profile は stale 除去と世代スキップ判定の内部機構に留め、`rollback` /
  `list-generations` を出してはならない（配置物が ephemeral で rollback の意味が薄く、devShell
  キック時は戻し先 host 世代も無いため）。新 link farm derivation が前世代と同一なら新世代を
  積んではならない。これは必須であり、devShell / direnv 運用では `shellHook` がシェル再入のたびに
  走るため、毎回新世代を積むと世代が無限増殖するからである。home mode は従来通り「適用のたびに
  新世代」のままとし、世代スキップは project mode 限定とする。ただし世代スキップ時も完全 no-op に
  せず、FS 検査だけは軽量に行わなければならない。各 entry の target を `lstat` で検査し、記録通り
  でない（foreign tool に書き換えられた・消えた）entry があればその entry だけ再張りする
  （foreign symlink なら warning）。「derivation 同一 ⇒ FS 同一」は foreign 書き換えで崩れるため、
  新世代を積まずに FS だけ収束させる。lstat 比較は安価で `shellHook` の高頻度実行に耐える。この
  ドリフト修復は symlink と copy の両方を対象とする。copy entry は target が不在のときだけ
  place-once で再マテリアライズし、存在するが内容が異なる（ユーザー編集）場合は触ってはならない
  （home mode の place-once と振る舞いを一致させる）。src 追従は `apply --recopy` 限定とし、内容
  ハッシュ比較はしない。
---
# REQ-46fccb80: project mode は世代を非公開にし、derivation 同一なら世代を積まず lstat ドリフト修復だけ行う

## 仕様

- **profile は解決済み root でキーする**（例: `<state>/nix/profiles/nput/<roothash>/<name>`）。
  同一 entrypoint を複数箇所にクローンしても profile が衝突せず、stale 除去が互いのクローンの
  配置を破壊しない。home mode（1 ユーザー 1 つ）では起きない問題。`<roothash>` は **解決後の
  絶対 root パスの sha256 を短縮した hex**（固定長・FS 安全）
- **世代はユーザーに公開しない**。profile は stale 除去 + 世代スキップ判定の内部機構に留め、
  `rollback` / `list-generations` を出さない。配置物が ephemeral で rollback の意味が薄く、
  devShell キック時は戻し先 host 世代も無いため
- **世代スキップ短絡（必須）**: 新 link farm derivation が前世代と同一なら**新世代を積まない**。
  devShell / direnv 運用では `shellHook` がシェル再入のたびに走るため、毎回新世代を積むと世代が
  無限増殖する。home mode は従来通り「適用のたびに新世代」のまま（世代スキップは project mode 限定）
- **ただし世代スキップ時も FS 検査だけは軽量に行う**（完全 no-op にしない）。各 entry の target を
  **lstat で検査**し、記録通りでない（foreign tool に書き換えられた・消えた）entry があればその
  entry **だけ**再張りする（foreign symlink なら warning）。「derivation 同一 ⇒ FS 同一」は
  foreign 書き換えで崩れるため、新世代を積まずに FS だけ収束させる。lstat 比較は安価で
  `shellHook` 高頻度実行に耐える
- この lstat ドリフト修復は **symlink と copy の両方**を対象にする。**copy entry は target が
  不在のときだけ** place-once で再マテリアライズし、**存在するが内容が異なる（ユーザー編集）
  場合は触らない**（home mode の place-once と振る舞いを一致させる。src 追従は `apply --recopy`
  限定・内容ハッシュ比較はしない）

> **上は原文の写しで、規範は frontmatter が正**。原文が参照する次の規範は本 item の担当ではない。
>
> - `<roothash>` 階層に backref `.root` を置き孤児 profile を逆引きできるようにすること →
>   REQ-2aa3abbc（レイアウト）・REQ-d41b1d0a（孤児 profile の扱い）
> - flock のキーが解決後 profileDir であること → REQ-2aa3abbc / REQ-1c1526b1
> - `<state>` の定義 → REQ-2aa3abbc
> - cross-config で同一 target を狙ったときの振動がユーザー責任であること → REQ-fc1118b1
> - ドリフト修復経路でも `--backup` の退避段が効くこと → REQ-9b0046e0
> - `rollback` / `list-generations` が home mode 限定であること（CLI 側の規範）→ REQ-05abce3e
> - foreign symlink の warning そのもの → REQ-622787dc、place-once そのもの → REQ-d2277c7a

## 出典

`docs/spec.md`「世代管理仕様」→「project mode の世代」節の箇条書き第 1〜4 項。

決定の実体は ADR-0005「project mode（プロジェクト相対配置）と ephemeral 配置原則」（世代の
非公開・ephemeral）と ADR-0017「実装前レビュー第 3 巡」（世代スキップとドリフト修復）。
profile を解決済み root でキーすることと `<roothash>` の定義は ADR-0013、`<state>` の確定は
ADR-0022、copy のドリフト修復の限定は ADR-0022 が定めている。
