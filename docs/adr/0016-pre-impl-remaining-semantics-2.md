# ADR-0016: 実装前レビュー第 2 巡で surfaced した残セマンティクスを確定する（copy の編集可能性 / --all 適用順 / copy 内 symlink / home mode skew / farm アンカー名 / pending gcroot）

- ステータス: 採用
- 日付: 2026-06-14
- 関連: ADR-0002, ADR-0005, ADR-0006, ADR-0011, ADR-0013, ADR-0014, ADR-0015, `docs/spec.md`, `docs/design.md`, `docs/concept.md`, `CONTEXT.md`
- 改訂対象: **ADR-0011**「再帰コピーは mode 保存」を「mode 保存 + owner-write 付与」へ拡張。**ADR-0013**「`--all` は定義順」を「辞書順（キーソート）」へ訂正
- 起点: ADR-0015 に続く実装前レビュー第 2 巡（grill）で surfaced した「実装前に決めておくべき」残細目の束

## 背景

ADR-0015 が実装前残セマンティクスの第 1 巡を固めたが、第 2 巡のレビューで次の 5 点（+ 軽微 1 点）に
未定義 / 仕様の揺れ / 事実誤認が残っていた。

1. **copy の「mode 保存」と「編集用途」が矛盾**。concept は copy を「ファイルを直接編集したい場合（テーマ・設定の
   一時調整）」と位置づけるが、ADR-0011 / spec / design は「file mode 保存」でコピーする。store パスは read-only
   （0444 / 0555）なので、保存するとコピー先も read-only になり編集できない。
2. **`apply --all` の「定義順に適用」が Nix 的に成立しない**。`nput.*` は attrset で、`nix eval` / `builtins.attrNames`
   が返すキー順は **辞書順（ソート）**。定義順は保持されない。spec / ADR-0013 の「定義順」は事実と異なる。
3. **copy の再帰コピーが src ツリー内の symlink をどう扱うか未確定**。ADR-0011 は「symlink 対応」と書くが、store 内に
   よくある symlink を symlink のまま複製するか、deref して実体を写すかが未定義。
4. **home mode の version skew が未対処**。project mode は devShell 同梱で CLI と `nput.lib` を一致させた（ADR-0015）が、
   home mode（グローバル install + flake pin の `nput.lib`）は別入力で skew し得る。
5. **symlink farm の GC アンカー名衝突**。「target をサニタイズして名前に使う」（spec）が、`/` 除去等で別 target が
   同名に潰れうる（linkFarm はキー一意必須）。
6. **（軽微）`.pending-<name>` out-link の失敗時残留**。`--set` 前に apply が失敗すると pending gcroot が残り、
   ビルド済み未使用 link-farm を掴み続ける。

## 決定

### 1. copy は mode 保存 + owner-write 付与（編集可能にする）

- 再帰 / 単一コピーとも、コピー後に **owner write ビットを付与**する（例: `0444 → 0644` / `0555 → 0755`）。
  perm の相対構造（実行ビット・group/other ビット）は保ちつつ、所有者が編集できる状態にする。
- copy の意図は「store から切り離してユーザーが所有・編集する place-once スナップショット」（concept）であり、
  read-only コピーはこの意図に反する。store の read-only ビットをそのまま持ち込むと、ユーザーが手で `chmod` する
  まで編集できず、copy を選ぶ意味が薄れる。
- spec / design / ADR-0011 の「mode 保存」を「**mode 保存 + owner-write 付与**」へ修正する。

### 2. `apply --all` の適用順は辞書順（キーソート・決定的）

- `nput.*` は attrset で、適用順は `builtins.attrNames` が返す **辞書順（キーソート）**になる。Nix attrset は定義順を
  保持しないため「定義順」は実現不能。
- 各 config は独立 profile で atomic（ADR-0002）なため、適用順は結果に影響しない（表示・失敗集約のための
  決定的順序であればよい）。spec / ADR-0013 の「定義順」を「**辞書順（決定的）**」へ訂正する。
- cross-config の同一 target 後勝ち（ADR-0015）と組み合わせると、「辞書順で後ろの config 名が勝つ」決定的挙動になる。

### 3. copy 内の symlink は symlink のまま複製する

- 再帰コピーで src ツリー内の symlink に遭遇したら、**symlink のまま複製**する（deref して実体を写さない）。
  ツリー構造を忠実に写し、循環 symlink / サイズ膨張を避ける。
- ただし **store 内への絶対 symlink を複製すると store 依存（read-only / GC 後 dangling）が残る**ことを docs に明記する。
  copy で完全に store から切り離したい場合は、その性質を理解した上でユーザーが対処する。相対 symlink はそのまま保つ。

### 4. home mode の version skew は schemaVersion ガード + 明記で許容する

- engine は自身の対応版より新しい `schemaVersion` を拒否する（ADR-0006）。**MVP は v1 のみ**（ADR-0015）なので
  skew は実害化せず、skew しても明確な error になる。
- project mode の devShell 同梱（ADR-0015）に対し、home mode は「**CLI（グローバル install）と flake が pin する
  `nput.lib` を同一 input から揃える**」ことを推奨として docs に明記するに留める。強制はしない（PATH 常駐の
  エルゴノミクスを損なわないため）。

### 5. symlink farm の GC アンカー名は target のハッシュにする

- farm の GC アンカー名を **`target` のハッシュ（sha256 の短縮 hex 等・固定長・FS 安全）**にする。サニタイズ
  （`/` 除去）方式は別 target が同名に潰れ得て linkFarm のキー一意制約に反する。
- farm は **GC アンカー専用**で、engine が配置に使う値は `manifest.json` の解決済み `src` 文字列（ADR-0010）。
  アンカー名は可読である必要がないため、衝突不可能なハッシュで十分。spec の「target をサニタイズして用いる」を修正する。

### 6. `.pending-<name>` 残留は許容（同名上書きで有界）

- apply が `--set` 前に失敗すると `.pending-<name>` gcroot が残るが、次回 apply が **同名**（`.pending-<name>`）で
  上書きするため、config あたり最大 1 個に有界。store は次回成功時に解放経路へ戻る。許容とし、spec に一行注記する。

## 根拠

- **copy の owner-write 付与**は concept の「編集用途」と実装を一致させる。read-only コピーは「copy を選んだのに
  編集できない」驚き最小原則違反。perm 相対構造を保つことで実行ビット等は失わない。
- **辞書順への訂正**は Nix attrset の事実に合わせるだけ。各 config が独立 atomic なので順序の意味は表示・集約に限られ、
  決定的でありさえすればよい。「定義順」は実装すると即破綻する誤記だった。
- **copy 内 symlink を symlink 複製**は ADR-0011 の「symlink 対応」記述と一致し、循環・膨張を避ける素直な選択。
  store 依存の caveat を明記することで「完全切り離し」を期待する誤解を防ぐ。
- **home skew の許容**は MVP=v1 のみ（ADR-0015）から論理的に従う。skew は schemaVersion ガードで安全に失敗し、
  同一 input 推奨で運用回避できる。home mode に devShell 同梱を強制すると PATH 常駐の利便を捨てる。
- **farm アンカーのハッシュ化**は linkFarm のキー一意制約を満たす唯一の確実な方法。可読性は farm に不要（GC 専用）。
- **pending 残留の許容**は同名上書きで有界なため、回収処理を自前で背負うより単純。

## 影響

- **`docs/spec.md`**:
  - 配置動作仕様（copy モード）の「mode 保存」を「mode 保存 + owner-write 付与」へ。copy 内 symlink の扱い（symlink
    複製 + store 依存 caveat）を追記。
  - CLI 仕様（`apply --all`）の「定義順」を「辞書順（決定的）」へ。
  - CLI 仕様 / 再現性スタンスに home mode の version skew 注記（schemaVersion ガード + 同一 input 推奨）を追記。
  - manifest.json スキーマ節の「symlink farm の GC アンカー名 = target をサニタイズ」を「target のハッシュ」へ。
  - 実行フローに `.pending-<name>` 失敗時残留の有界性注記を追記。
- **`docs/design.md`**: subpath 判別ロジック / 設計上の判断（copy）の「file mode 保存」を「mode 保存 + owner-write 付与」へ。
- **`docs/concept.md`**: copy の世代外表と「copy はユーザー管理の副作用」節に owner-write 付与を反映。
- **`CONTEXT.md`**: 必要なら `method` / copy 周辺へ owner-write 付与を反映（glossary 粒度では任意）。
- **ADR-0011**: 改訂注記で「再帰コピーは mode 保存 → mode 保存 + owner-write 付与」「farm アンカー名は target ハッシュ」を追記。
- **ADR-0013**: 改訂注記で「`--all` は定義順 → 辞書順（キーソート）」を追記。
- **実装フェーズ**: `internal/`（copy 後 owner-write chmod・symlink 複製・farm アンカー名のハッシュ化）、`lib/`（farm 構築の
  アンカー名生成）。

## 棄却した代替案

- **copy で厳密に mode 保存（read-only 受容）**: 「編集用途」の concept と矛盾。ユーザーが毎回 chmod を強いられる。
- **`method = copy` に `writable` オプション追加**: 柔軟だが entry スキーマが膨らみ MVP のシンプルさに反する。owner-write は
  copy の意図上ほぼ常に望ましく、オプション化の利得が薄い。
- **copy 内 symlink を deref して実体コピー**: store から完全に切り離せるが、循環 symlink / サイズ膨張のリスク。
- **home mode も devShell / `nix run` 一元化を強制**: skew を根絶できるが PATH 常駐の home mode エルゴノミクスを損なう。
- **farm アンカー名にサニタイズ + index 接尾**: 可読だが index が辞書順依存で安定名にならず、衝突回避もハッシュより脆い。
- **farm アンカー名を store パスごとに集約**: GC 目的には足りるが、target との対応が失われ将来の用途で不便。
- **pending gcroot の失敗時クリーンアップを自前実装**: 同名上書きで有界なため過剰。回収処理の複雑性に見合わない。
