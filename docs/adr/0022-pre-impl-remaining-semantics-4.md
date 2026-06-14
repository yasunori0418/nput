# ADR-0022: 実装前残セマンティクス第4巡（schema 互換・copy ドリフト修復・profile パス・copy foreign 衝突・shellHook skip 通知）

- ステータス: 採用
- 日付: 2026-06-14
- 関連: ADR-0002, ADR-0005, ADR-0006, ADR-0013, ADR-0016, ADR-0017, ADR-0019, ADR-0020, `docs/spec.md`, `docs/concept.md`, `docs/design.md`
- 改訂対象: なし（既存決定の細目補完・反転なし）
- 起点: 実装着手前のドキュメント検査で、各文書が沈黙していた5つの細目を洗い出した（ADR-0015/0016/0017 と同系列の「実装前残セマンティクス」確定）

## 背景

設計は 21 ADR で成熟したが、実装着手前の検査で次の5点が**どの文書にも明示されていない**ことが判明した。
いずれも既存決定の隙間にある二次的細目で、決めないと実装で恣意判断が混入する。

1. **schemaVersion の後方互換ポリシー**。engine は「自分の対応版より新しい `schemaVersion` を拒否」とだけ規定済み（ADR-0006）だが、
   stale 除去 / rollback で読む**前世代 manifest が将来は古い版になりうる**ことへの方針が未定義。
2. **project mode 世代スキップ時の copy ドリフト修復範囲**。ADR-0017 の lstat ドリフト修復は「再張り」と symlink 前提の表現で、
   copy entry が対象に入るかが曖昧。
3. **profile の具体パス**。spec で「推奨・未確定」のまま。`<roothash>` / backref / flock キーがこのパスに依存する。
4. **copy の place-once が foreign 実ファイルに当たったときの挙動**。symlink は foreign 実ファイルでエラー停止（spec）だが、
   copy は「target 存在 → 触らない」（ADR-0020）で、両モードの非対称が未整理。
5. **shellHook の try-lock skip が無言**（ADR-0013）。skip 時にユーザーへ通知するか未定義。

## 決定

### 1. マイグレーション（schemaVersion 後方互換）は現時点では考慮しない

- MVP は `schemaVersion = 1` のみを発行・読取りし、**古い版の manifest を読むマイグレーション機構を最初から作らない**
  （ADR-0006・ADR-0015 と一貫）。v1 単一の現状では互換問題が発生しないため。
- **最初のリリース後、フィールド追加で v2 が必要になった時点で**、後方互換ポリシー（engine が古い版の manifest を
  stale 除去 / rollback のために読めること）を改めて検討する。spec にはこの方針を明示するに留める。
- これにより「contract をいま過剰設計しない」一方、後方互換が必須要件であること（アップグレード直後の stale 除去が前世代 manifest を読む）
  を忘れない seam を残す。複数セッションでの確認の結論として、マイグレーションは v2 が現実に必要になるまで着手しない。

### 2. project mode 世代スキップ時、copy は「不在のみ復帰・内容は不可触」

- 世代スキップ時の lstat ドリフト修復は **symlink と copy の両方**を対象にする。
- copy entry は **target が不在のときだけ place-once で再マテリアライズ**する。**存在するが内容が異なる（ユーザー編集）場合は触らない**。
- これにより home mode（毎回 place-once 再実行で消えた copy が復活）と project mode の振る舞いが一致し、
  src 追従は引き続き `apply --recopy` 限定（ADR-0020）に保たれる。内容ハッシュ比較はしない（place-once 哲学・shellHook 高頻度実行のコスト）。

### 3. profile パスは `$XDG_STATE_HOME` 尊重で確定

- profile の基底を **`$XDG_STATE_HOME` があればそれ、無ければ `~/.local/state`** とする。
  - home mode: `$XDG_STATE_HOME/nix/profiles/nput/<name>`（既定 `~/.local/state/nix/profiles/nput/<name>`）
  - project mode: `$XDG_STATE_HOME/nix/profiles/nput/<roothash>/<name>`（既定 `~/.local/state/nix/profiles/nput/<roothash>/<name>`）
- `nix` 本体の profile 既定（`~/.local/state/nix/profiles`）と整合し、XDG をカスタムするユーザーでも配置が散らばらない。
- spec の「推奨・未確定」注記を外し確定値とする。`<roothash>` / backref / flock キーはこの確定パスを基準にする（ADR-0005, ADR-0013）。

### 4. copy が foreign 実ファイルに当たったら warning を出して skip

- copy entry の target に**前世代 manifest に無い実ファイル**（= nput が置いていない foreign ファイル）が既存のとき、
  **上書きせず place-once skip し、warning を出す**（「target に既存ファイルがあり copy をスキップした」）。
- symlink の foreign 警告（記録に無い symlink は warning で後勝ち・ADR-0015）と振る舞いを対称化し、
  「nput が中身を置いた」とユーザーが誤認する masking を防ぐ。copy はユーザーデータを上書きしない哲学（ADR-0019）を保ち、
  apply 全体は止めない（project mode の shellHook での全停止 footgun を避ける）。
- 「自分が置いたか」は前世代 manifest に entry があるかで判別する(内容は判別しない)。

### 5. shellHook の try-lock skip は stderr に1行通知

- shellHook が try-lock 失敗で apply を skip したとき、**stderr に1行**通知する
  （例: `nput: another apply in progress, skipped (run \`nput apply\` manually)`）。
- シェル入室はブロックしない（ADR-0013）まま、「なぜ config が反映されないか」が見えるようにする。
  direnv / シェル再入の衝突は低頻度なのでノイズにならない。

## 根拠

- **schema 互換の先送り**は YAGNI と seam の両立。v1 単一の今は不要だが、後方互換が必須要件である事実を spec に固定して将来の見落としを防ぐ。
- **copy 不在のみ復帰**は home / project のモード間で place-once セマンティクスを統一する最小の規則。内容比較を持ち込むと place-once の
  「マテリアライズ後は触らない」が崩れる。
- **XDG 尊重の profile パス**は nix 本体・XDG 規約の双方に整合する素直な既定。
- **copy foreign warning skip**は symlink の foreign 扱いとの対称性を回復し、データ保護と非対話安全性（apply 非停止）を両立する。
- **shellHook skip 通知**は無言 skip のデバッグ不能性を解消する最小コストの可観測性。

## 影響

- **`docs/spec.md`**:
  - manifest 契約節に「v1 固定。v2 導入 ADR で後方互換ポリシーを確定」を1行追記。
  - 世代スキップ節（lstat ドリフト修復）の「再張り」を symlink + copy 両対象と明記し、copy は不在のみ復帰・内容不可触を追記。
  - profile パス表の「推奨・未確定」注記を外し、`$XDG_STATE_HOME` 尊重の確定値に書き換え。
  - copy 配置 / エラー仕様に「foreign 実ファイル既存時は warning + skip」を追記。
  - flock / shellHook 記述に「try-lock skip 時は stderr に1行通知」を追記。
- **`docs/concept.md`**: copy のユーザー管理副作用節に foreign skip warning、ephemeral 節に copy 不在復帰のニュアンスを軽く反映。
- **`docs/design.md`**: profile パスの確定値、shellHook の skip 通知を反映。
- **実装フェーズ**: engine（lstat 修復の copy 対応・foreign copy 検出と warning・skip 通知）、profile パス解決（XDG）、manifest reader（v1 のみ受理）。

## 棄却した代替案

- **schema 後方互換をいま全面設計**: v1 単一では検証不能な投機設計になり、実 v2 で要件が変わるリスク。seam の明示で十分。
- **copy ドリフトを symlink のみ修復（copy 放置）**: 実装は単純だが home / project で copy 復活挙動が齟齬し、ユーザーの予測を裏切る。
- **copy 内容まで検査して src へ戻す**: place-once 哲学に反しユーザー編集を破壊、shellHook 高頻度実行でコスト高。
- **profile パスを `~/.local/state` 決め打ち**: XDG カスタム環境で nix 本体と配置がずれる。
- **copy foreign をエラー停止**: symlink と対称だが、copy は本来上書きしない方針で、既存ファイル1つで apply 全体が止まると shellHook footgun。
- **shellHook skip を無言のまま**: 実装は最小だが未反映原因が不可視でデバッグ困難。
