---
id: "REQ-d41b1d0a-c6d5-41cc-93f9-e5cc7f152da4"
type: requirement
name: "孤児 profile は backref で逆引き可能なまま放置許容とし、MVP では cleanup コマンドを持たない"
specification: |
  When a clone is deleted, its profile SHALL remain as an orphan under
  `<state>/nix/profiles/nput/`. The store SHALL be freed by `nix-collect-garbage` but the
  profile directory SHALL remain; that SHALL be tolerated, or removed by hand, and SHALL
  be noted in the public documentation. Which root an orphan originated from SHALL be
  recoverable from the backref file `.root` at the `<roothash>` level. The MVP SHALL NOT
  hold a cleanup command, the actual harm being small; the presence of the backref SHALL,
  nonetheless, leave a seam at which a future `nput prune` — resolving orphan series that
  point at a root no longer in existence, and deleting them — can be implemented once
  demand for it arises.
specification_ja: |
  クローンを削除すると profile は `<state>/nix/profiles/nput/` 下に孤児として残る。store は
  `nix-collect-garbage` で解放されるが profile ディレクトリは残る。これは放置許容（または手動
  削除）とし、公開ドキュメントに注記しなければならない。どの root 由来の孤児かは `<roothash>`
  階層の backref ファイル `.root` で逆引きできるものとする。実害が小さいため MVP では cleanup
  コマンドを持ってはならない。ただし backref があることで、実在しない root を指す孤児系列を
  逆引きして削除する将来の `nput prune` を、消費側の要求が出た時点で実装できる seam を残す。
---
# REQ-d41b1d0a: 孤児 profile は backref で逆引き可能なまま放置許容とし、MVP では cleanup コマンドを持たない

## 仕様

**orphan profile**: クローンを削除すると profile が `<state>/nix/profiles/nput/` 下に孤児として
残る。store は `nix-collect-garbage` で解放されるが profile ディレクトリは残る。放置許容
（または手動削除）とし、公開ドキュメントに注記する。どの root 由来の孤児かは `<roothash>` 階層の
backref ファイル（`.root`）で逆引きできる。**cleanup コマンドは MVP では持たない**（実害が小さく
`.pending` も config あたり最大 1）。backref があるので**将来 `nput prune`（実在しない root を指す
孤児系列を逆引きして削除）を実装できる seam** を残す（消費側の要求が出た時点で追加）。

> **上は原文の写しで、規範は frontmatter が正**。backref `.root` を roothash 階層へ置くこと自体と
> `.pending` が config あたり最大 1 であることは REQ-2aa3abbc、store 解放に
> `nix-collect-garbage` を使うことは REQ-706de717、profile を解決済み root でキーすること
> （孤児が生じる前提）は REQ-46fccb80 の担当。

## 出典

`docs/spec.md`「世代管理仕様」→「project mode の世代」節の箇条書き最終項。

決定の実体は ADR-0005「project mode（プロジェクト相対配置）と ephemeral 配置原則」（孤児 profile の
放置許容）で、backref による逆引きは ADR-0013、cleanup を持たず seam に留める判断は ADR-0024 が
定めている。
