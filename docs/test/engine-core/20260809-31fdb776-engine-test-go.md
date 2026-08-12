---
id: "CASE-31fdb776-9650-4ff9-97f6-45d56b2e7177"
type: test_case
name: "internal/engine/engine_test.go — 実 tmpdir に対する apply の統合テスト"
target: "internal/engine/engine_test.go"
covers:
  - "TC-9df804ce-35ee-44a7-87b1-17935d53fab2"
  - "TC-405606f0-3ac8-4cc2-998e-d4759a62a171"
  - "TC-4da40ee8-877e-405b-98ca-6bf5de926ba4"
  - "TC-b254a5a8-7fbf-4f31-8486-3e373d66bfa7"
  - "TC-596d697f-4ba6-4ec1-b71e-8b5375806c08"
  - "TC-d1eb1814-ac5e-4576-b092-7db4929fba43"
  - "TC-d160e18b-4c0c-4531-a506-e7d00d88788a"
  - "TC-b9d4ffaf-ac91-4bf1-9f27-5ea3964466ad"
  - "TC-a5eb7de3-a1a7-41ae-8fb9-c2aa374ac894"
  - "TC-8435052a-5dcc-49e2-ac26-82f645cb6890"
---
# CASE-31fdb776: engine_test.go

## 対象

`internal/engine/engine_test.go`

nix を介さず実 FS（`t.TempDir()`）へ apply を回す統合テスト。link farm と manifest を
テスト側で組み立て、配置結果を実 FS のプローブで確認する。

`resolveRoot` を直接叩く純引数テストだけは例外で、分岐網羅を 1 ファイルで追えるよう
generations_test.go 側からもここへ集約した（→ issue #328）。cwd を壊して `os.Getwd` を
失敗させる固定 root の Abs 失敗のみ、FS 誘発型として後段のセクションに残る。

## 主な検証内容

- **初回配置と subpath**: project mode の初回配置、`subpath = "."` の配置
- **配置分類**: 自己記録 symlink の無警告張替え、foreign symlink の警告付き後勝ち
- **上書き拒否**: target の通常ファイル占有でエラー停止、祖先 symlink でエラー停止、
  祖先が自己記録 stale のときのみ移行（copy 子を含む）
- **stale 除去との接続**: 記録済み entry の除去、foreign を残すこと
- **root 解決**: `rootKind=home` が `$HOME` を返すこと、`--root` 上書きが rootKind に
  よらず優先されること、git repo 外でのエラー、固定絶対 root の解決失敗、`fixed` でパスを省いた
  ときの拒否（エラー本文一致。manifest 層が受理する組み合わせを engine が止める責務
  → CASE-4179dcb2 / TC-172548ea）、rootKind 未決（`""`）と未知の値の拒否（いずれも
  エラー本文一致。未知の値は `%q` のクォートまで含めて固定する。`systemRoot` は system
  mode 実装で差し替わるため対象外）、profile ディレクトリ作成失敗・backref 書き込み失敗の
  エラー化
- **out-of-store**: live symlink の配置、stale 除去、リンク先不一致時の保持、marker の
  リンク先が不在のときのエラー停止（dangling symlink を作らず target も残さないこと）
- **copy 経路の呼び出し**: place-once の初回・既存保持、foreign 実ファイルの skip 警告、
  ディレクトリ再帰コピーでの内部 symlink 保存、`--recopy` の上書き（foreign 含む）
- **conflict 報告**: 複数 conflict の全件列挙と件数を含む集約エラー、dryrun と本番 apply が
  同数の conflict を報告すること、
  自己矛盾する祖先・copy 構造不一致それぞれのガイダンス、種類混在時のガイダンス対応付け
- **その他**: `--no-wait` で lock 取得済みのときの skip、pending 除去失敗の警告化
