# ADR-0032: legacy entrypoint（shell.nix / default.nix）は mkShell passthru を canonical とし、addressing を `nix build -f` に統一する

- ステータス: 採用
- 日付: 2026-07-03
- 関連: ADR-0007, ADR-0002, ADR-0011, ADR-0023, ADR-0024, ADR-0025, `docs/spec.md`, `docs/design.md`
- 改訂対象: ADR-0007 §4 の legacy entrypoint addressing 記述（`nix-build -A nput.<name>`）を `nix build -f <ep> nput.<name>` へ改訂。§5（best-effort・再現性はユーザー責任）の decision 自体は不変
- 起点 Issue: #112（親 epic #107）。ADR-0007 §5 で「採用済み」としつつ実装（`cmd/nput/nix.go`）が明示拒否していた乖離を解消する

## 背景

ADR-0007 §4/§5 は shell.nix / default.nix を entrypoint として受理する方針を採用済みとしていたが、実装（`discoverEntrypoint`）は
「この slice では未対応」として明示的にエラーで拒否していた。2026-07-03 のセッションで「実装するか将来送りか」を再検討し、
以下の理由で **A: 実装する** に確定した。

- 将来送りは ADR-0007 §4/§5 という**採用済み decision の巻き戻し**になり、起点 Issue #4 のクローズ済み解決を覆す
- #81（cmd orchestration 分離・DI seam 拡張）が先行してマージされ、単体テスト可能な形で実装できる土台が整った

再検討にあたり、ADR-0007 §4 が置いていた 2 つの前提を精査した。

1. **entrypoint の作り**: ADR-0007 §4 は「default.nix / shell.nix はトップレベル `{ nput.<name> = mkManifest { ... }; }`」とだけ書いていたが、
   これは nixpkgs の `mkShell` 同梱と両立しない。同じ `shell.nix` が「project の devShell」と「nput entrypoint」を兼ねる典型ユースケース
   （project mode の canonical・→ ADR-0015）では、`{ shell = ...; nput = ...; }` のようなトップレベル分離を強いるか、
   素の `nix-shell` が壊れるかの二択を迫られていた。
2. **nix コマンド形**: ADR-0007 §4 は legacy build に旧 CLI `nix-build -A nput.<name>` を挙げていたが、flake 側の eval は
   ADR-0025 前提（`nix-command` experimental-features）に基づき新 CLI `nix eval` を使っており、legacy だけ旧 CLI を混在させる理由がない。

## 決定

### 1. entrypoint の作り = nixpkgs `mkShell` と共存できる passthru 形を canonical にする

- **canonical**: `shell.nix` は**素の `mkShell` derivation を返し**、`passthru.nput.<name>` に named manifest を載せる。

  ```nix
  { pkgs ? import <nixpkgs> {}, nput ? import (fetchTarball "https://github.com/yasunori0418/nput/archive/....tar.gz") }:
  pkgs.mkShell {
    packages = [ ];
    shellHook = "nput apply skills --no-wait";
    passthru.nput = {
      skills = nput.lib.mkManifest { inherit pkgs; root = nput.lib.projectRoot; entries = { }; };
    };
  }
  ```

- これにより **素の `nix-shell`（`-A` 不要）が壊れない**。`nix-shell` は返り値が derivation でありさえすれば入室でき、`passthru` は
  無視される。project mode の devShell 同梱（→ ADR-0015）と legacy entrypoint 対応が同じファイルで両立する。
- **トップレベル attrset 形（`{ shell = ...; nput = {...}; }`）も引き続き有効**（廃止しない）。`mkShell` を使わない・
  devShell を兼ねない最小構成ではこちらの方が単純なため、docs に両形を併記する。

### 2. CLI の attr path はどちらの形でも同一（実装分岐なし）

- Nix の属性選択は `passthru` に透過的に届く（derivation は attrset であり、`passthru` の各フィールドはその attrset に
  マージされる）。したがって passthru 形でもトップレベル attrset 形でも、**CLI が叩く attr path は同一の `nput.<name>`**。
- CLI 実装は「entrypoint の**種別**（flake / legacy）」でのみ分岐し、legacy のうち「passthru か素の attrset か」では
  分岐しない。ユーザーがどちらの形で書いても同じコードパスを通る。

### 3. 複数 manifest 一括処理は既存の `apply --all` で充足する

- `apply --all` の一括 eval（`nix eval -f <ep> nput --apply 'cs: ...' --json`）は passthru 形・トップレベル attrset 形の
  どちらでも同一に動く（`nput` 直下 attrset を読むだけで、`mkShell` 本体の評価は要らない）。
- **`mkShell` の `inputsFrom` で複数 manifest を合成するヘルパは採らない**。1 config = 1 profile の atomic 性（→ ADR-0002）と
  衝突する（合成すると世代の単位が曖昧になる）。役割ごとの分離は既存どおり config 分割（`nput.<name>` を複数持つ）で担保する。

### 4. nix コマンド形 = 新 CLI `-f` に統一する

- legacy の eval / build は ADR-0007 §4 が挙げていた旧 CLI `nix-build -A` ではなく、**新 CLI の `-f` 形**を使う。

  | 用途 | flake | legacy |
  |---|---|---|
  | rootKind 先取り eval | `nix eval <ep>#nput.<system>.<name>.rootKind --raw` | `nix eval -f <ep> nput.<name>.rootKind --raw` |
  | build（apply 内） | `nix build <ep>#nput.<system>.<name> --out-link <pending>` | `nix build -f <ep> nput.<name> --out-link <pending>` |
  | store path 取得（gitignore 等） | `nix build <ep>#nput.<system>.<name> --no-link --print-out-paths` | `nix build -f <ep> nput.<name> --no-link --print-out-paths` |
  | 一括 rootKind eval（`--all`） | `nix eval <ep>#nput.<system> --apply '...' --json` | `nix eval -f <ep> nput --apply '...' --json` |

- legacy には per-system 次元（`nput.<system>.<name>`）が無く、フラットな `nput.<name>` を直接叩く。
- `runNixCapture` / `runNixStream` / `nixError`（experimental-features 未有効時の案内含む）は**そのまま再利用**する。
  legacy 固有の追加フラグ（`--impure` 等）は要らない：`-f`/`--file` 評価は新 CLI でも pure-eval を強制しない
  （pure-eval はデフォルト無効で、flake 評価だけが暗黙に強制する）ため、`<nixpkgs>` の NIX_PATH 参照はそのまま解決できる
  （→ ADR-0007 §5 の「impure eval を許容」はこの意味）。

### 5. entrypoint 発見順序に shell.nix / default.nix を追加する

- 既定探索（CWD）: `flake.nix` → `shell.nix` → `default.nix` の優先順（ADR-0007 §3 の記述どおり）。
- `-f` / `--file` はファイル直接指定（`flake.nix` / `shell.nix` / `default.nix` のいずれも受理）に加え、ディレクトリ指定時も
  同じ優先順で探索する。

## 根拠

- **passthru を canonical にする理由**: トップレベル attrset を canonical にすると、`mkShell` を返さない前提になり、
  project mode の devShell 同梱（→ ADR-0015）と同じファイルで両立しない。passthru は「nput の追加情報を乗せるだけ」で
  ホスト側の derivation の型・振る舞いを変えない。素の `nix-shell` 互換を壊さない。
- **属性選択の透過性で実装分岐を避ける**: `passthru` はホスト言語（Nix）機構としてトップレベル属性と区別なく参照できるため、
  CLI 側に「passthru か直下か」の判定ロジックを持ち込む必要がない。実装をシンプルに保てる。
- **`inputsFrom` 型合成を採らない理由**: 世代管理の単位を「1 profile = 1 config」（→ ADR-0002）に保つほうが、
  rollback / stale 除去の意味論を単純に保てる。複数 manifest を 1 profile に混ぜると、どの manifest 由来の stale かが曖昧になる。
- **新 CLI `-f` に統一する理由**: flake 側は ADR-0025 で `nix-command` experimental-features を前提にしており、
  legacy だけ旧 CLI（`nix-build`）を使うと同一プロセスの二重海域（新旧 CLI 混在）になり、エラーハンドリング
  （`nixError` の experimental-features 未有効判定等）を両対応させる必要が生じる。新 CLI `-f` に統一すれば
  既存の `runNixCapture` / `runNixStream` / `nixError` をそのまま再利用でき、実装・テストの両方が単純になる。

## 影響

- **ADR-0007 改訂**: §4 の「legacy は `nix-build -A nput.<name>`」を本 ADR の記述へ改訂する旨のヘッダ注記を追加する。
  §5（best-effort・再現性はユーザー責任）の decision 自体は不変。
- **`docs/spec.md`**: アドレッシング表（`default.nix` / `shell.nix` 行）に passthru 形を追記し、`nix-build -A` の記述を
  `nix build -f` へ改める。実行フロー節の legacy 行（rootKind 先取り eval / build / 一括 eval）を同様に改める。
- **`docs/design.md`**: 該当するアドレッシング節・CLI 実行モデル節を spec.md と整合させる。
- **実装（`cmd/nput/nix.go`）**: `entrypoint` を flake / legacy の 2 種別に拡張し、`discoverEntrypoint` に shell.nix /
  default.nix の受理を追加、`evalRoot` / `evalAllRoots` / `buildFunc` / `dryBuildFunc` / `buildManifestStorePath` に
  legacy 形（`nix eval -f` / `nix build -f`）を追加する。
- **e2e**: legacy シナリオ（passthru 形 fixture + `NIX_PATH` を flake.lock の nixpkgs pin に固定）を追加する。

## 棄却した代替案

- **トップレベル attrset 形を canonical にし、passthru は補助的な代替案に留める**: 素の `nix-shell` が devShell 用途で
  壊れうる（`{ nput = ...; }` だけを返すと `mkShell` にならない）。project mode の devShell 同梱という主要ユースケースと
  相性が悪いため不採用。ただし引き続き**有効な代替形**として docs に残す（廃止ではない）。
- **`mkShell` の `inputsFrom` で複数 manifest を合成するヘルパを提供する**: 1 profile = 1 config の atomic 性
  （→ ADR-0002）と衝突する。役割分離は config 分割で担保する既存決定と一貫させるため不採用。
- **legacy build を旧 CLI `nix-build -A` のまま存続させる**（ADR-0007 §4 原案）: flake 側は新 CLI 前提（→ ADR-0025）であり、
  legacy だけ旧 CLI を混在させる利点がない。エラーハンドリング・experimental-features 前提の二重化を避けるため、
  新 CLI `-f` 形に統一する。
- **legacy eval/build に `--impure` を自動付与する**: `-f` 評価は pure-eval をデフォルトで強制しないため不要。
  ユーザー環境の pure-eval 設定を黙って上書きしない（ADR-0025 §1 の `--extra-experimental-features` 自動付与拒否と同じ姿勢）。
- **将来送り（本 issue を設計判断のまま据え置く）**: ADR-0007 §4/§5 という採用済み decision の巻き戻しになり、
  起点 Issue #4 のクローズ済み解決を覆すため不採用。
