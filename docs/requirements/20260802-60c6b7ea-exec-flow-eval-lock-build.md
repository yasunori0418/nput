---
id: "REQ-60c6b7ea-e936-4ce8-bd75-ad35e9c693b9"
type: requirement
name: "実行フローの順序は eval 先行 → flock → build とし build をロック内に閉じる"
specification: |
  The order of the execution flow SHALL be eval first, then flock, then build. Because
  `profileDir` is not determined until after root resolution, and root resolution requires
  the `rootKind` of `manifest.json`, the CLI SHALL obtain `rootKind` in advance with a
  cheap `nix eval` (of the root kind only) *before* building, SHALL determine `profileDir`,
  SHALL then acquire the flock, and SHALL enclose the build inside the lock. This
  simultaneously resolves the circularity of an undetermined `profileDir` and the
  contention over the `.pending` out-link by a build outside the lock.

  The flow SHALL be: (0) entrypoint discovery, overridable by `-f`; (1) a preliminary eval
  of the root kind, followed by root resolution (project = git rev-parse, home = `$HOME`,
  system = `/`, or a fixed path, overridable by `--root`) and determination of
  `profileDir`, `rootKind` being fixed at evaluation time as a passthru of `mkManifest`
  while the actual resolution of the git toplevel / `$HOME` happens at engine runtime;
  then (2) driving the engine: (a) acquire the flock keyed on the determined `profileDir`,
  an explicit apply / rollback blocking (LOCK_EX, waiting until acquisition and displaying
  that it waits for another apply) while the shellHook path (`--no-wait`) uses a try-lock
  (LOCK_NB) that skips when the lock is held and prints a one-line notice on stderr
  without blocking shell entry, concurrent execution against the same `profileDir` being
  the user's responsibility and last-writer-wins on collision; (b) inside the lock, run
  `nix build` with `--out-link <profileDir>/.pending` and obtain the link-farm store path
  via `os.Readlink`, the out-link establishing an indirect gcroot that closes the GC
  window between placement and `--set`, and the build being inside the lock so that
  out-link contention cannot structurally occur; (c) read the previous generation
  `manifest.json` of `profileDir`, an absent one meaning the first run with zero removal
  targets; (d) in project mode, skip stacking a new generation when the new link-farm is
  identical to the previous one, while still lstat-checking each target and re-linking
  only the drifted entries rather than making it a complete no-op; (e) diff the old and
  new `manifest.json`, remove before placement the self-recorded stale entries that block
  placement (ancestor symlink, real directory target, method change), place the new and
  re-linked ones, and then perform conservative stale removal with native filesystem
  operations, any mid-way failure among these four stages unwinding every filesystem
  change made by this run via the in-memory undo journal back to the pre-apply state;
  (f) run `nix-env --profile <profileDir>/profile --set <link-farm>` as a subprocess, the
  commit point; and (g) delete `<profileDir>/.pending` after `--set` succeeds, the
  generation link taking over the gcroot.

  `apply --manifest <link-farm>` SHALL skip stages 0 to 1 and stage 2b, passing the
  pre-built link-farm directly to the engine and having the engine read `rootKind` from
  the `manifest.json` inside it. Stage 2a onwards through 2g SHALL be identical to an
  ordinary apply, and no pending out-link SHALL be established since nothing is built.
specification_ja: |
  実行フローの順序は「eval 先行 → flock → build」でなければならない。profileDir は
  root 解決後にしか確定せず、root 解決には `manifest.json` の `rootKind` が要るため、
  これを安価な `nix eval`（root kind のみ）でビルド前に先取りし、profileDir を確定してから
  flock を取り、build をロック内に閉じる。これで profileDir 未確定の循環と、ロック外
  build の `.pending` out-link 競合が同時に解消する。

  フローは次のとおりとする。(0) entrypoint 発見（`-f` 上書き）。(1) root kind の先取り
  eval → root 解決（project = git rev-parse / home = `$HOME` / system = `/` / 固定パス、
  `--root` 上書き）→ profileDir 確定。rootKind は mkManifest の passthru として eval 時に
  確定し、git toplevel / `$HOME` の実体解決は engine 実行時に行う。(2) engine を駆動する:
  (a) 確定 profileDir をキーに flock を取得する。明示 apply / rollback は blocking
  （LOCK_EX・取得まで待ち「他の apply 完了待ち」を表示）、shellHook 経路（`--no-wait`）は
  try-lock（LOCK_NB）で保持中ならスキップし stderr に 1 行通知する（シェル入室は
  ブロックしない）。同一 profileDir への同時実行はユーザー責任で衝突時は後勝ちとする。
  (b) ロック内で `nix build` を `--out-link <profileDir>/.pending` 付きで実行し、
  `os.Readlink` で link-farm store path を得る。out-link が indirect gcroot を張り、
  配置〜`--set` の GC 窓を塞ぐ。build がロック内なので out-link 競合は構造的に起きない。
  (c) profileDir の前世代 `manifest.json` を読む（無ければ初回 = 削除対象ゼロ）。
  (d) project mode かつ新 link-farm が前世代と同一なら新世代は積まない。ただし各 target を
  lstat 検査し、ドリフトした entry だけ再張りする（完全 no-op にしない）。(e)
  `manifest.json` を新旧 diff し、配置を塞ぐ自己記録 stale（祖先 symlink・実 dir target・
  method 変更）を配置前除去し、新規 / 張替を配置し、保守的 stale 除去（ネイティブ FS）を
  行う。この 4 段のいずれかが途中失敗すると、この run が行った FS 変更を全てインメモリ
  undo ジャーナルで巻き戻し pre-apply 状態へ戻す。(f)
  `nix-env --profile <profileDir>/profile --set <link-farm>`（サブプロセス・コミット点）。
  (g) `--set` 成功後に `<profileDir>/.pending` を削除する（世代リンクが gcroot を引き継ぐ）。

  `apply --manifest <link-farm>` は 0〜1 と 2b を skip しなければならない。ビルド済み
  link-farm を engine へ直接渡し、rootKind は link-farm 内 `manifest.json` から engine が
  読む。2a 以降〜2g は通常 apply と同一とし、pending out-link は build しないため張らない。
---
# REQ-60c6b7ea: 実行フローの順序は eval 先行 → flock → build とし build をロック内に閉じる

## 仕様

**順序は「eval 先行 → flock → build」**。profileDir は root 解決後にしか確定せず
（project / `--root` 時は `<roothash>`）、root 解決には `manifest.json` の `rootKind` が
要る。これを安価な `nix eval`（root kind のみ）で**ビルド前に**先取りし、profileDir を
確定してから flock を取り、**build をロック内**に閉じる。これで profileDir 未確定の循環と、
ロック外 build の `.pending` out-link 競合（並行 apply の奪い合い）が同時に解消する。
`profileDir` は config 専用ディレクトリで profile リンクは `<profileDir>/profile`・
pending out-link は `<profileDir>/.pending`。

```
nput apply <name> [-f <ep>] [--root <p>]
  0. entrypoint 発見（-f 上書き）
  1. root kind を先取り eval:
     nix eval <ep>#nput.<system>.<name>.rootKind（legacy は per-system 次元なし: nix eval -f <ep> nput.<name>.rootKind）
     → root 解決（kind: project=git rev-parse / home=$HOME / system=/ / 固定パス、--root 上書き）
     → profileDir 確定（home: <name> / project: <roothash>/<name>。--root 明示時は全モード <roothash>/<name>）
       ※ rootKind は mkManifest の passthru として eval 時に確定（git toplevel / $HOME の実体解決は engine 実行時）
  2. engine を駆動:
     a. flock を取得（キー = 確定 profileDir）。
        明示 apply / rollback は blocking（LOCK_EX・取得まで待ち「他の apply 完了待ち」を表示）。
        shellHook 経路（--no-wait）は try-lock（LOCK_NB）で保持中ならスキップし、stderr に1行通知する
        （例: `nput: another apply in progress, skipped (run \`nput apply\` manually)`・シェル入室はブロックしない）。
        同一 profileDir への同時実行はユーザー責任で衝突時は後勝ち
     b. ロック内で nix build <ep>#nput.<system>.<name> --out-link <profileDir>/.pending
        （legacy は nix build -f <ep> nput.<name> --out-link <profileDir>/.pending）
        → os.Readlink で link-farm store path を得る。out-link が indirect gcroot を張り
          配置〜--set の GC 窓を塞ぐ。build がロック内なので out-link 競合は構造的に起きない
     c. profileDir の前世代 manifest.json を読む（無ければ初回 = 削除対象ゼロ）
     d. project mode かつ新 link-farm が前世代と同一なら新世代は積まない（世代スキップ）。ただし各 target を lstat 検査し、
        ドリフトした entry だけ再張りする（完全 no-op にしない）
     e. manifest.json を新旧 diff → 配置を塞ぐ自己記録 stale（祖先 symlink・実 dir target・method 変更）を配置前除去
        （PreRemove・migration）→ 新規/張替を配置 → 保守的 stale 除去（ネイティブ FS）
        ※ e の 4 段（PreRemove / 配置 / copy 反映 / stale 除去）のいずれかが途中失敗すると、この run が行った
           FS 変更を全てインメモリ undo ジャーナルで巻き戻し pre-apply 状態へ戻す
     f. nix-env --profile <profileDir>/profile --set <link-farm>（サブプロセス・コミット点）
     g. --set 成功後に <profileDir>/.pending を削除（世代リンクが gcroot を引き継ぐ）
```

**`apply --manifest <link-farm>` は 0〜1（entrypoint 発見・rootKind 先取り eval）と 2b
（ロック内 `nix build`）を skip する**。ビルド済み link-farm を engine へ直接渡し、
rootKind は link-farm 内 `manifest.json` から engine が読む。2a（flock 取得）以降〜2g は
通常 apply と同一。pending out-link は build しないため張らない。

> **上のフロー図は原文の写しで、規範は frontmatter が正**。図が触れる次の規範は本 item の
> 担当ではなく、本 item はそれらがフロー上のどの段に位置するかだけを規定する。
>
> - profileDir のレイアウト（`<roothash>/<name>` 等）と世代スキップ / ドリフト修復の
>   詳細 → `docs/spec.md`「世代管理仕様」節の担当（#209-PR4）
> - 配置前除去（PreRemove）・保守的 stale 除去・undo ジャーナルの規範 →
>   「配置動作仕様」節の担当（#209-PR4）
> - root 解決の各モードの規範 → 「root の解決」節の担当（#209-PR5）
> - `apply --manifest` そのものの契約（引数・併用エラー）→ REQ-dec58330

## 出典

`docs/spec.md`「CLI 仕様」→「実行フロー」の本文・コードブロックと、
`apply --manifest` の skip を述べる最終箇条書き。

決定の実体は ADR-0023「実行フロー順序」（eval 先行 → flock → build）で、out-link に
よる indirect gcroot は ADR-0011、try-lock による shellHook skip は ADR-0022、
同一 profileDir 同時実行の後勝ちは ADR-0013、`apply --manifest` の skip は ADR-0026。
