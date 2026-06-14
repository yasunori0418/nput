# ADR-0020: copy の明示上書き（`apply --recopy`）と配置物のリセット（`nput reset`）を追加する

- ステータス: 採用
- 日付: 2026-06-14
- 関連: ADR-0002, ADR-0005, ADR-0006, ADR-0013, ADR-0017, ADR-0019, `docs/spec.md`, `docs/design.md`, `docs/concept.md`
- 改訂対象: **ADR-0002**（copy = place-once・世代外）に「明示フラグでの上書き経路」を追加。**ADR-0006**（サブコマンド体系）に `reset` を追加
- 起点: 「copy で配置した物を参照元更新に追従させたい」「entry の配置物を無い状態に戻したい（リセット）」という 2 つのユースケース未対応

## 背景

copy は place-once（初回マテリアライズ後は触らない・ADR-0002）で、参照元（`src`）の更新を反映する手段が
「target を手で削除して再実行」しか docs に無かった（spec の旧記述）。また、配置した物を**明示的に無い状態へ戻す**
（symlink も copy も撤去する）手段が無かった。stale 除去は config から entry が消えたときに symlink のみ保守的に消すが、
copy は決して自動除去しないため、copy を消す明示手段が存在しなかった。

この 2 つは関連する。**reset が「配置物を消す」プリミティブ**で、**recopy はその応用**（copy target を消してから
置き直す = reset + apply を 1 操作に畳んだ形）と捉えられる。

## 決定

### 1. `nput apply <name> --recopy` で copy target を上書き再コピーする

- `apply` に `--recopy` フラグを追加する。通常の apply（symlink 配置 + stale 除去 + 世代コミット）に加え、
  **config 内の全 copy entry の target を現 `src`/`subpath` から無条件に再コピー（上書き）**する。
- nput は copy 内容の hash を追跡しない（世代外・ADR-0002）ため、差分判定はせず**無条件上書き**とする。ユーザー責任モデルに整合。
- **上書きした target をレポート表示**する。`--recopy` フラグ自体が opt-in なので確認プロンプトは出さない。
  ローカルの copy 編集は破棄され src 内容へ戻る（= 「upstream に追従したい」という flag の意図そのもの）旨を docs に明記する。
- copy は世代外のまま（ADR-0002）。`--recopy` は世代を増やさず copy target を再マテリアライズするだけで、symlink 部の
  通常 apply 挙動（世代コミット）は不変。

### 2. `nput reset <name> [target...]` で配置物を無い状態へ戻す

- `reset` サブコマンドを追加する。`target` 省略で config の**全 entry**、`target...` 指定で**その entry のみ**を撤去する。
- 撤去対象:
  - **symlink**: stale 除去と同じ**保守的不変条件**（nput 管理・記録通りを指す symlink のみ。foreign symlink は warning を出して
    残す・実ファイルは触らない・ADR-0002）。
  - **copy target**: stale 除去が決して触らない領域だが、reset は **copy を消す唯一の明示手段**として copy target も削除する。
    place-once は配置実績を追跡しないため、reset は manifest が宣言する copy target を削除対象とする（事前存在ファイルを消す
    リスクがあるため確認で守る）。
- **データ損失リスク**（copy のユーザー編集・事前存在ファイル）があるため、**確認プロンプトを出す**か `--yes` で明示同意を要求する。
  削除した target をレポート表示する。
- **profile / 世代は触らない FS-only teardown**。宣言的な真実（config → 世代）は不変のまま。config が entry を残している限り
  **次の apply で再配置される**（= reset は transient。project mode は ADR-0017 の lstat 検査で次の devShell 入室時に復帰する）。
  恒久除去は config から entry を消して apply する。profile / 世代の完全除去は `nix-env --profile <dir> --delete-generations` で行う。
- reset は home / project 両モードで使える（rollback と違いモード限定しない）。manifest の解決は apply と同じく entrypoint 発見 + `nix build`。

### 3. `--yes` / `-y` グローバルフラグ（破壊的操作の確認スキップ）

- `reset` の確認プロンプトを `--yes` でスキップできる（スクリプト / CI 用）。`reset` の既定は対話確認。
- `--recopy` は flag 自体が opt-in なので `--yes` 不要（確認なしで上書き + レポート）。

## 根拠

- **`--recopy` を apply フラグに**するのは、symlink 更新と copy 追従を 1 コマンドで済ませられ、reset + apply の 2 ステップを
  畳めるため。無条件上書きは copy が世代外で hash 追跡しない事実から必然で、ユーザー責任モデルと整合する。
- **`reset` をサブコマンドに**するのは、teardown が apply（配置）と意味論が逆で、per-entry 指定をフラグに載せるより自然なため。
- **reset の copy 削除 + 確認**は「copy を無い状態に戻す唯一の手段」を提供しつつ、データ損失を確認で防ぐ。symlink は従来の
  保守的不変条件を維持して誤破壊を避ける。
- **profile 不変の FS-only teardown**は per-entry リセットと整合する（世代から一部 entry だけ抜くのは不可能）。reset を宣言的真実から
  切り離し「今 FS を消す」操作に限定することで、apply による復帰経路（ADR-0017）と素直に両立する。

## 影響

- **`docs/spec.md`**:
  - サブコマンド体系に `reset <name> [target...]` を追加。グローバルフラグに `--recopy`（apply 修飾）/ `--yes`（`-y`）を追加。
  - copy モード節の「ストア更新の反映は target 削除後に再実行」を「`--recopy` で上書き再コピー（または `reset` 後 apply）」へ更新。
  - 新規「reset 仕様」節（撤去対象・保守的 symlink 除去 + copy 削除 + 確認・profile 不変・transient・両モード可）。
  - エラー仕様表に reset 関連（前世代/配置物なし時の挙動・foreign symlink は残す）を追記。
- **`docs/design.md`**: copy の使用パターン / 「copy を place-once にする理由」に `--recopy` 追従経路を反映。CLI フローに reset を追記。
- **`docs/concept.md`**: copy の「ユーザー管理の副作用」節に「src 追従は `--recopy`、撤去は `reset`」を反映。
- **ADR-0002**: 改訂注記で「copy = place-once は既定。`apply --recopy` で明示上書き、`reset` で撤去（copy 削除）を追加」。
- **ADR-0006**: 改訂注記で「サブコマンド体系に `reset` を追加。`apply` に `--recopy`、グローバルに `--yes`」。
- **実装フェーズ**: `cmd/nput`（`apply --recopy` / `reset` サブコマンド / `--yes` / 確認プロンプト）、`internal/`（copy 上書き再コピー・
  reset の保守的 symlink 除去 + copy 削除）。

## 棄却した代替案

- **`--recopy` を専用サブコマンド `nput recopy`**: symlink も含む一括更新が 2 コマンドになる。apply フラグなら 1 コマンド。
- **`--recopy` を差分判定（変わった target のみ）**: 編集を不必要に潰さないが、mode 保存 + symlink 複製下での diff が非自明で
  実装複雑。「ユーザー責任で上書き」意図ともずれる。
- **recopy 専用機能を設けず reset + apply で代用**: 表面積最小だが 2 ステップで、symlink と同時更新できない。
- **`reset` を `apply --reset` フラグ**: 「配置しない apply」は意味矛盾。per-entry 指定の表現も複雑。
- **reset で copy を残す**: データ安全だが「copy を無い状態に戻す」要望を満たせない。
- **reset で profile / 世代も削除（uninstall）**: reset と uninstall を混同し per-entry リセットと矛盾。profile 完全除去は
  `nix-env --delete-generations` に委ねる。
- **reset を確認なしで実行**: 高速だが copy のユーザー編集 / 事前存在ファイルを黙って消す。確認 + `--yes` が安全。
