---
id: "ADR-0036"
type: adr
name: "system mode（`root = systemRoot`）を実装する（root 権限チェック・profile は `/nix/var/nix/profiles` 配下・世代公開）"
status: 採用
origin: "次期マイルストーン計画の grilling（2026-07-04）。NixOS / nix-darwin モジュール実装（将来・別マイルストーン）に先行して、lib / engine / CLI の system mode 対応だけを縦に通す"
revises:
  - "ADR-0013"
  - "ADR-0015"
  - "ADR-0018"
  - "ADR-0025"
references:
  - "ADR-0004"
  - "ADR-0005"
  - "ADR-0007"
  - "ADR-0023"
  - "ADR-0024"
  - "ADR-0034"
---
# ADR-0036: system mode（`root = systemRoot`）を実装する（root 権限チェック・profile は `/nix/var/nix/profiles` 配下・世代公開）

- ステータス: 採用
- 日付: 2026-07-04
- 関連: ADR-0004, ADR-0005, ADR-0007, ADR-0023, ADR-0024, ADR-0034, `docs/spec.md`, `docs/design.md`
- 改訂対象: ADR-0013 §5 の「`root = systemRoot` は eval 時エラー（未実装拒否）」を撤回し実装へ。ADR-0015 §5 の「rollback は home mode 限定」・ADR-0018 §4 の「`list-generations --all` は homeRoot の config を対象」を home / system の両モードへ拡張。ADR-0025 §4 の profile 専用ディレクトリレイアウトに system mode の state 基底（`/nix/var/nix/profiles`）を追加（ユーザー state 基底のレイアウト自体は不変）
- 起点: 次期マイルストーン計画の grilling（2026-07-04）。NixOS / nix-darwin モジュール実装（将来・別マイルストーン）に先行して、lib / engine / CLI の system mode 対応だけを縦に通す

## 背景

`systemRoot` マーカーは ADR-0004/0007 で distro 構想の seam として正式 API 化されたが、ADR-0013 §5 により `normalizeManifest` が「system mode は未実装（予定）」で eval 時 throw する据え置き状態だった。NixOS / nix-darwin モジュール（将来拡張）の前提となる engine / CLI 側の system mode を、モジュール実装に先行して standalone CLI から使える形で実装する。

決めるべきは (1) 拒否解除後の root 解決、(2) 権限モデル、(3) profile 状態の置き場所、(4) 世代操作（`rollback` / `list-generations`）の公開範囲、の 4 点。

## 決定

### 1. eval 時拒否を解除し `rootKind = "system"` を通す

- `normalizeManifest` の systemRoot throwIf（ADR-0013 §5）を撤去し、`rootKind = "system"` を manifest v1 スキーマの正式な値にする（スキーマは enum 追加のみで `schemaVersion` は据え置き。既存 engine は未知 rootKind を拒否するため、旧 CLI × 新 manifest の組み合わせは既存の検査で安全に停止する）。
- engine の root 解決に system 分岐を追加する: `rootKind = "system"` → root = `/`（実行時解決・`--root` で上書き可能な点は他マーカーと同じ・ADR-0017）。

### 2. engine は実行時に root 権限（euid 0）を検査する

- `rootKind = "system"` の manifest に対する**変更系操作（`apply` / `reset` / `rollback`）**では、engine が FS 変更・flock 取得に入る前に **euid = 0 を検査**し（`os.Geteuid()`・stdlib-only 制約内）、非 root なら「system mode requires root privileges (run with sudo)」の趣旨のエラーで即停止する。
- sudo / doas / root ログインの区別はしない（euid 0 であることだけを見る。`SUDO_USER` 等の環境変数には依存しない）。
- **読み取り専用の経路は検査しない**: `list-generations` は profile dir の読み取りだけで動き、`apply --dryrun` は配置を行わない（eval / build は nix daemon 経由で非 root でも可能）。非 root でも計画確認・世代閲覧はできる状態を保つ。
- 権限があっても個別 target の書き込みが失敗するケース（read-only mount 等）は、既存のエラー wrap 規約（op + 対象パス）で自然に報告される。euid 検査は「確実に全滅する実行を早期に・明確な案内付きで止める」ための前段ゲート。

### 3. profile 状態は `/nix/var/nix/profiles/nput` 配下に置く

- `rootKind = "system"` のとき、profileDir の state 基底をユーザー state dir から **`/nix/var/nix/profiles/nput`** に切り替える。nix の system profile 慣習（`/nix/var/nix/profiles`）に一致し、将来の NixOS / nix-darwin モジュールが同じ場所をそのまま使える。
- キーイングは home mode と同型: root = `/` は固定なので **`<name>` 直キー**（`/nix/var/nix/profiles/nput/<name>`）。`--root` 明示時は全モード共通ルール（ADR-0023）通り `<roothash>/<name>` キー + backref `.root`。ディレクトリ内部レイアウト（`profile` リンク・`profile-N-link`・`.pending`・flock）は ADR-0025 §4 と同一。
- **home / project / fixed の profileDir は一切変わらない**（切り替えは rootKind = system のときだけ）。
- `nput prune`（ADR-0034）の走査対象にこの system 基底を加える。root = `/` の `<name>` 直キー系列は構造的に孤児化しないが、`--root` 上書きの roothash 系列は孤児化し得る。削除には root 権限が要るため、非 root 実行の prune は権限エラーの系列を warning 付きで skip する。

### 4. `rollback` / `list-generations` の公開基準を「配置の永続性」に整理し、system mode に公開する

- 世代操作のユーザー公開の判断基準を「起動形態」ではなく**配置の永続性**として明文化する: project mode は ephemeral placement（ADR-0005）だから非公開、home / system mode は永続配置だから公開。
- これに伴い `rollback` / `list-generations` のゲーティングを「home mode 限定」から **「home / system mode」**へ拡張する。`list-generations --all` の対象（ADR-0018 §4「homeRoot の config」）も homeRoot / systemRoot の config へ広がる。`rollback --all` 非対応（名指し必須・ADR-0018 §4）は不変。
- `apply --all` の `--system-root` 修飾（spec 予約済み・当面マッチなし）が実際に機能し始める。
- 将来 NixOS モジュールが rollback を host 世代（`nixos-rebuild`）へ一本化するのはモジュール側の配線判断（HM と同型・ADR-0002）であり、CLI の能力を先回りして削らない。

## 根拠

- **euid を engine で検査する理由**: 権限不足の system apply は最初の mkdir / symlink で必ず失敗するが、その時点では flock 取得や部分的な検査が進んでおり、エラーも「たまたま最初に触った target の EACCES」になる。前段の euid ゲートは失敗を決定的・説明的にする。検査を CLI でなく engine（`internal/engine`）に置くのは、`apply --manifest` 経由（将来のモジュール activation）でも同じゲートを通すため。
- **`/nix/var/nix/profiles` を選ぶ理由**: 世代は `nix-env --profile <dir>` に乗る設計（ADR-0002）であり、system スコープの profile を `/nix/var/nix/profiles` に置くのは NixOS / nix-darwin が system 世代で使っている標準動線。root ユーザーの home 配下（`/root/.local/state`）に置くと「システムの状態が root の個人領域に住む」ことになり、将来モジュールが場所を移す際に世代の孤児化（移行痛）を生む。
- **世代を公開する理由**: `/` 配下への配置は project mode と違い永続的で、「戻したい」需要は home mode と同質。engine の世代機構は既にモード非依存で動いており、ゲーティングの条件変更だけで公開できる。

## 影響

- **`lib/manifest.nix`**: systemRoot throwIf 撤去・`rootKind = "system"` 変換。nix-unit / namaka テスト更新（拒否テスト → 変換テスト）。
- **`internal/engine` / `internal/paths`**: root 解決の system 分岐・profileDir の state 基底切替・euid ゲート（変更系のみ）。
- **`cmd/nput`**: `rollback` / `list-generations` のモードゲート拡張・`--system-root` 修飾の実効化。
- **`docs/spec.md`**: エラー仕様表の「systemRoot 未実装」を削除し、権限エラーを追加。root 解決表・profile dir 表・世代管理仕様に system mode を追記。「rollback / list-generations は home mode 限定」の記述を home / system へ更新。
- **`docs/design.md` / `CONTEXT.md` / `docs/glossary.md`**: system mode を「将来拡張」から実装済みへ更新（NixOS / nix-darwin モジュールは引き続き将来拡張のまま）。
- **ADR-0013 / ADR-0015 / ADR-0018 / ADR-0025**: 各改訂対象への改訂注記を同一 PR で追記。
- **テスト**: euid ゲートは非 root 実行での拒否を単体テスト（`Geteuid` の seam 注入または実 euid 判定）で、system 配置の e2e は fakeroot / VM が要るため NixOS モジュール実装時の `runNixOSTest`（design.md 既定・将来）に送る。本マイルストーンでは root 解決・profileDir 切替・ゲーティングの単体/統合テストまでを範囲とする。

## 棄却した代替案

- **euid 事前チェックなし（書き込み失敗を自然に返す）**: 失敗が「最初に触った target の EACCES」という偶発的な形になり、sudo への誘導も無い。決定的な前段ゲートの方がユーザー体験・テスト容易性ともに優る（grilling 中の当初案から反転）。
- **profile をユーザー state dir のまま（root 実行 → `/root/.local/state`）**: 実装差分ゼロだが、システム状態が root の home に住み、NixOS モジュール導入時に移行痛（世代孤児化）が確定する。正しい場所を先に選ぶ。
- **profile を `/var/lib/nput` に置く**: FHS 的には自然だが、nix profile の慣習（`/nix/var/nix/profiles`）から外れ、nix エコシステム内での発見可能性が下がる。
- **世代操作を非公開のまま据え置く**: standalone の system 利用者に戻す手段が `nix-env` 直叩きしか無くなる。「配置の永続性」基準の方が ADR-0005（project = ephemeral だから非公開）とも一貫する。
- **NixOS / nix-darwin モジュールまで同時に実装する**: 縦割りとしては自然だが、モジュール実装は VM テスト（`runNixOSTest`）・activation 配線・host 世代統合と規模が大きい。engine / CLI の system mode を先に安定させ、モジュールは次マイルストーンとする（grilling で確定）。
