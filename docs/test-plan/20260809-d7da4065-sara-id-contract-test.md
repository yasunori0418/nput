---
id: "TP-d7da4065-ce7c-4a0b-be49-5108256e177a"
type: test_plan
name: "sara-id はモデル定義との一致を含めて契約テストで検証する"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  The numbering command that the documentation graph depends on SHALL be verified by a
  contract test covering its output contract and its collision behaviour: that it emits the
  formal id, the filename material and the prose reference as three fields whose short form
  is a prefix of the full one and whose uuid is well-formed; that repeated invocations
  differ; that a short prefix already present in the documentation tree causes regeneration,
  bounded by a retry limit beyond which it fails rather than emitting a duplicate; that the
  tree it scans is resolved from the repository root and falls back to the working directory
  outside a repository; and that its exit codes distinguish misuse from the deliberate
  refusal to number the one type that keeps sequential ids. The mapping from type names to
  prefixes SHALL be asserted against `docs/model.yaml` in both directions, so that a type
  added to the model without a mapping, or a mapping left behind by a type removed from it,
  fails the test rather than surfacing as a mis-numbered item. The test SHALL report every
  failing assertion in one run rather than stopping at the first.
specification_ja: |
  ドキュメントグラフが依存する採番コマンドは、その出力契約と衝突時の挙動を覆う契約テストで
  検証しなければならない。正式 ID・ファイル名素材・散文参照の 3 形式を出力し、省略形が完全形の
  前方一致であり uuid が妥当であること。呼ぶたびに異なる ID を返すこと。ドキュメント木に既出の
  省略形は再生成を促し、その再試行には上限があり、上限を超えたら重複 ID を出すのではなく失敗
  すること。走査する木をリポジトリルートから解決し、リポジトリ外ではカレントディレクトリへ
  フォールバックすること。そして終了コードが、引数の誤用と、連番を維持する唯一の型に対する
  意図的な採番拒否とを区別すること。型名から prefix への写像は `docs/model.yaml` と双方向に
  一致することをアサートしなければならない（モデルに型を足して写像を足し忘れた場合や、モデル
  から消した型の写像が残った場合に、誤った採番として現れるのではなくテストが落ちるように
  するため）。テストは最初の失敗で止まらず、1 回の実行で全ての失敗アサーションを報告しなければ
  ならない。
---
# TP-d7da4065: sara-id はモデル定義との一致を含めて契約テストで検証する

## 仕様

`sara-id`（UUIDv4 二層 ID の採番コマンド）は開発ツールだがドキュメントグラフ全体が依存する
ため、契約テストで検証する。覆う契約は次のとおり。

| 契約 | 内容 |
|---|---|
| 出力形式 | 正式 ID / ファイル名素材 / 散文参照の 3 形式。省略形は完全形の前方一致・uuid が妥当（version 4・variant ∈ {8,9,a,b}）|
| 一意性 | 呼ぶたびに異なる ID を返す |
| 衝突回避 | 既出の 8 文字 prefix は再生成。再試行上限を超えたら重複 ID を出さず失敗する |
| 走査先の解決 | リポジトリルート基準。git 管理外ではカレント基準へフォールバック |
| prefix 写像 | 型名・別名から prefix を引く。`docs/model.yaml` と**双方向に一致**する |
| 終了コード | 引数不正と、連番維持のための ADR 採番拒否とを区別する |

写像を双方向で見るのは、片方向だと「モデルに型を足して写像を足し忘れた」場合にフォール
バックの大文字化が働いて誤った prefix で採番が通ってしまい、テストではなく item の側に
欠陥として現れるため。

テストは最初の失敗で止めず 1 回の実行で全失敗を報告する。退行時の診断が先頭 1 件で切れるのを
避けるため。

> **本 item は test_plan のみを起こし、test_condition / test_case へは展開しない**。
> `sara-id` は開発基盤であってプロダクトの振る舞いではなく、リスクを requirement / design へ
> `threatens` で張る構造を持たないため（→ Issue #273 の決定事項）。

## 出典

ADR-0048「ドキュメントは sara でグラフ構造化する — model.yaml の全面置換・UUIDv4 二層 ID・
CI 非必須開始」が定める UUIDv4 二層 ID と `sara-id` による採番。起点は Issue #207。

現況の実装は `dev/tests/sara-id.sh`（`nix flake check ./dev` の `checks.sara-id` 経由でも
走る）で、同ファイルのヘッダが検証対象を 9 項目に列挙し、全失敗集計方式を採る理由も
述べている。
