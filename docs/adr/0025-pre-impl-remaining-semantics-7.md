# ADR-0025: 実装前残セマンティクス第7巡（nix experimental-features 前提・HM profile 制約明記と seam・nput init 固定 ref・profile 専用ディレクトリレイアウト・reset 非TTY・Go fmt/lint・main 直コミット撤廃トリガ）

- ステータス: 採用
- 日付: 2026-06-14
- 関連: ADR-0002, ADR-0005, ADR-0007, ADR-0011, ADR-0012, ADR-0013, ADR-0016, ADR-0020, ADR-0021, ADR-0022, ADR-0023, ADR-0024, `docs/spec.md`, `docs/design.md`, `CONTEXT.md`, `CLAUDE.md`
- 改訂対象: ADR-0022/0023 の `--out-link <profileDir>/.pending-<name>` 命名と ADR-0023 §3・ADR-0024 §1 の profileDir 表が暗黙に持っていた「profileDir = profile リンクそのもの」という用語の二義性を、専用ディレクトリレイアウトへ確定（pending out-link 名 `.pending-<name>` → `.pending`）。ADR-0024 §2 の HM profile 粒度に「role 分離不可」のユーザー視点制約と将来 seam を明示追加（決定の反転なし）。
- 起点: ADR-0024 までの第6巡で設計は実装着手の閾値に達したが、`/grill-me` による横断検査で骨格に直結する未決・揺れ・考慮漏れが7点残っていた。いずれも第一スライス（ADR-0023 §4）着手で即踏むか、docs の穴で、着手前に確定する（実装前残セマンティクス確定の第7巡）。

## 背景

ADR-0024 で fixed root キーイング・HM profile 粒度・非 build コマンド eval 先行等が確定した。直後の横断検査（`/grill-me`）で、なお次の7点に未決・揺れ・考慮漏れが残っていた。大半は既決方針の素直な延長か docs の明文化で、新規の方針反転は §4（profile レイアウト）の用語確定を除き無い。

1. **nix の experimental-features 依存が docs に一切無い**。第一スライスの実行フローは `nix eval` / `nix build`（新 CLI）を使い、これらは `experimental-features = nix-command`（flake entrypoint はさらに `flakes`）が無効だと動かない。未有効環境では第一スライスが cryptic な nix エラーで落ちるが、前提条件・検出・エラー UX の記述がエラー仕様にもどこにも無い。
2. **HM モジュールの単一 manifest 制約がユーザー視点で明示されていない**。ADR-0024 §2 は「MVP は固定名 `default` profile 1個」を決めたが、「standalone は `nput.<name>` で複数 profile = role 分離できるのに HM ではできない」という表現力の非対称が spec/CONTEXT に明記されておらず、将来拡張の seam も検討事項として残っていない。
3. **`nput init` のテンプレート参照 `<nput>` の解決方法が未定義**。`nput init <t>` は `nix flake init -t <nput>#<t>` のラッパーだが、バイナリが `<nput>` をどう解決するか（registry / 固定文字列 / ビルド時注入）が未記述。
4. **`profileDir` が用語として二義**。世代管理仕様は profile パスを `<state>/nix/profiles/nput/<name>`（= profile *シンボリックリンク*）と書く一方、実行フローは `nix build --out-link <profileDir>/.pending-<name>`（= profileDir が*ディレクトリ*である前提）と書く。`nix-env --profile P` は P を profile リンクとして扱い世代を兄弟 `P-N-link` として `dirname(P)` に作るため、P が store 世代を指す symlink なら `<profileDir>/.pending-<name>` は read-only な store 内を貫通して破綻する。さらに project mode の `<roothash>/<name>` 階層に backref `.root` と複数 `<name>` が同居し、flock キー「解決後 profileDir 単位」が roothash 共有 dir 単位か (roothash,name) 単位か曖昧。
5. **`reset` の確認プロンプトが非 TTY（CI / direnv / パイプ）のときの挙動が未定義**。`--yes` 無しでプロンプトを出すとハングするか空入力で誤動作する。
6. **Go の formatter / linter の採否・導入タイミングが未確定**。`treefmt` は現状 nixfmt のみで、第一スライスで入る Go コードの整形・静的解析方針が ADR-0012（CI 構成）でも未記述。
7. **CLAUDE.md「ドキュメントのみ計画段階・main 直コミット許容」の撤廃トリガが曖昧**。撤廃条件（コード実装開始）は明記されているが、第一スライス着手のどの時点で branch/PR 運用へ切替え該当セクションを削除するかが未確定。

## 決定

### 1. nix experimental-features は「ユーザー前提条件」とし、未有効時はエラー停止する

`nix-command`（flake entrypoint はさらに `flakes`）の experimental-features は **ユーザー環境で有効化済みであることを前提条件**とする。CLI 側で `--extra-experimental-features` を自動付与しない。

- 未有効で `nix eval` / `nix build` が機能未有効エラーを返したら、CLI は**前提条件を案内する分かりやすいエラーで停止**する（生の nix エラーを握り潰さず、有効化方法を1行添える）。
- 自動付与しないのは、nix.conf / Determinate Nix / 組織ポリシーで experimental-features を制御している環境の設定を CLI が黙って上書きするのを避け、「余計な前提・state を持ち込まない」方針（ADR-0006）に揃えるため。前提条件は docs に明記する。

### 2. HM モジュールは MVP 単一 `nput.entries` 据え置き・role 分離不可を明記し将来 seam を残す

ADR-0024 §2（固定名 `default` profile 1個）を維持しつつ、**ユーザー視点の制約**を docs に明示する：

- standalone / CLI は entrypoint の `nput.<name>` で**複数の独立 profile（role 分離・個別 rollback）**を持てるが、**HM モジュール経由は単一 `nput.entries` = 1 profile（`default`）に限られ、role 分離はできない**。役割分離が要るユーザーは standalone CLI 経路を使う。
- 将来 `nput.configs.<name>.entries`（attrsOf）形で HM の複数 profile 化を行える **seam を ADR の検討事項として残す**（消費側の要求が出た時点で追加・YAGNI）。HM の低い positioning（ADR-0007）に鑑み MVP では背負わない。

### 3. `nput init` は固定文字列 `github:yasunori0418/nput` を参照する

`nput init <t>` は `nix flake init -t github:yasunori0418/nput#<t>` を呼ぶ。テンプレート参照は**バイナリにハードコードした固定 flake ref**とする。

- registry 依存（`nput#<t>`）はユーザー環境の登録に依存して脆く、ビルド時 `self` 注入は実装が重い。固定文字列は最小実装で、`init` は新規プロジェクトの bootstrap 用途なので **CLI バージョンと template バージョンのズレ（常に latest main 参照）は許容**する。
- apply 時の schemaVersion 整合は project mode の devShell 同梱 pin（ADR-0015）が担うため、init 時点の版ズレは実害が小さい。docs / examples 中の `<owner>` プレースホルダは実 owner `yasunori0418` に統一する。

### 4. profile は config 専用ディレクトリレイアウトにし、`profileDir` の二義性を解消する

各 config に**専用ディレクトリ**を与え、その中に profile リンク・世代・pending out-link を置く。`profileDir` = この専用ディレクトリと定義する。

```
<state>/nix/profiles/nput/<roothash>/.root              # backref（roothash 階層・複数 <name> で共有・ADR-0013）
<state>/nix/profiles/nput/<roothash>/<name>/             # ← profileDir（flock キー）
<state>/nix/profiles/nput/<roothash>/<name>/profile       # profile リンク（nix-env --profile の対象）
<state>/nix/profiles/nput/<roothash>/<name>/profile-N-link # 世代（nix-env が profile の兄弟に作成）
<state>/nix/profiles/nput/<roothash>/<name>/.pending      # nix build --out-link（profile を貫通しない兄弟）
# home（--root なし）: <state>/nix/profiles/nput/<name>/{profile, profile-N-link, .pending}
```

- `nix-env --profile <profileDir>/profile` / `nix build --out-link <profileDir>/.pending` となり、世代兄弟 `profile-N-link` と pending が profile リンクを貫通せず兄弟として並ぶ。read-only store 貫通の破綻が構造的に起きない。
- **flock キー = profileDir（専用ディレクトリ）**。同一 config の apply / reset / rollback を直列化し、別 config（同 roothash でも別 `<name>` dir）とは独立する。
- pending out-link は専用ディレクトリ内に1個なので **名は `.pending` で十分**（ADR-0022/0023 の `.pending-<name>` を改訂。`<name>` はディレクトリ階層が表す）。`--set` 前失敗で残る pending は config あたり最大1（ADR-0016 の有界性は不変）。
- backref `.root` は roothash 階層（`<name>` dir の親）に置き、複数 `<name>` で共有する（ADR-0013 の逆引き seam は不変）。
- profileDir のキー（`<name>` / `<roothash>/<name>`）の決定は ADR-0023 §3・ADR-0024 §1 のまま不変。本決定は「キーで指す先がディレクトリで、profile リンクはその中の `profile`」という**物理レイアウトの確定**であって、キーイング規則の変更ではない。

### 5. `reset` は非 TTY かつ `--yes` 無しならエラー停止する

`reset` の確認プロンプトは、**stdin が TTY のときのみ**出す。**非 TTY（CI / direnv / パイプ）かつ `-y/--yes` 未指定なら、プロンプトを出さず即エラー停止（exit 1）**する（例: `refusing destructive reset without --yes in non-interactive context`）。

- CI でのハングと空入力での誤削除を同時に防ぎ、破壊的操作への明示同意を非対話環境でも強制する。`apply` / `gitignore` は非破壊なので影響しない。

### 6. Go fmt/lint は gofmt + go vet/golangci-lint を採用・実体は第一スライスで追加する

- **整形**: `treefmt` に `gofmt` を追加する（既存 nixfmt と併存）。
- **静的解析**: `nix flake check` に `go vet` + `golangci-lint` の check derivation を追加する。stdlib-only 厳守（ADR-0011）なので依存検出は軽い。nix 側の `deadnix` / `statix` は任意（採用は実装時判断）。
- **タイミング**: 採否は本 ADR で確定するが、**実体の設定追加は第一スライス PR で行う**（整形対象の Go コードが無い段階で空設定を先置きしない）。ADR-0012 の CI 構成（flake check + E2E 分離）にこの check を載せる。

### 7. main 直コミット許容は docs 反映までで終え、第一スライスから branch/PR 運用へ切替える

- 本 ADR を含む grill 第7巡決定の docs 反映（ADR-0025 作成・spec/CONTEXT/design 更新）までは「ドキュメントのみ計画段階」なので **main 直コミットを継続**する。
- **最初の Go コードが入る第一スライス PR から branch/PR 運用へ切替え、同時に CLAUDE.md の当該暫定セクションを削除**する。壊れ得るコードが生じる瞬間で規律を入れるのが撤廃条件の趣旨に最も忠実。

## 根拠

- §1 は第一スライスが即踏む前提条件の明文化で、ADR-0006 の「余計な state/前提を持ち込まない」を experimental-features にも適用したもの。
- §2 は ADR-0024 §2 の決定を維持したまま欠けていたユーザー視点の制約と将来 seam を補う明文化で、新規トレードオフ無し。
- §3 は ADR-0007 の「設定を生成しない／薄いラッパー」を保ちつつ最小実装を選ぶもので、版整合は既決の pin 機構（ADR-0015）に委ねる。
- §4 は ADR-0023/0024 のキーイング規則を変えず、`nix-env`/`nix build` の実ファイルレイアウト制約に整合する物理形を確定するもの。用語の二義性という docs の穴を塞ぐ。
- §5 は ADR-0020/0021 の「誤破壊しない」方針を非対話環境へ延長したもの。
- §6 は ADR-0011/0012 の技術スタック・CI 方針の延長で、実装量を第一スライスへ寄せる。
- §7 は CLAUDE.md が自ら定めた撤廃条件の発火点を具体化したプロセス決定。

## 影響

- **`docs/spec.md`**:
  - CLI 仕様 / 再現性スタンス近辺に experimental-features 前提条件を追記し、エラー仕様表に「`nix-command` / `flakes` 未有効」行を追加（§1）。
  - モジュールオプション仕様 / 世代管理仕様に HM = 単一 `nput.entries` = 1 profile・role 分離不可・将来 seam を明記（§2）。
  - `nput init` 節の `<nput>` を `github:yasunori0418/nput` に確定し固定 ref の注記を追加。examples の `<owner>` を実 owner に統一（§3）。
  - 世代管理仕様の profile dir 表・実行フロー・`root の解決` 表・配置動作仕様の `nix-env --profile` / `--out-link` / 世代 / backref 記述を専用ディレクトリレイアウト（profile リンク = `profileDir/profile`・pending = `profileDir/.pending`）へ全面整合（§4）。
  - `reset` サブコマンド説明・配置動作仕様（recopy/reset）・エラー仕様表に非 TTY エラー停止を追記（§5）。
- **`docs/design.md`**: 実行モデルに experimental-features 前提を1文、プロジェクト構成 / レイヤー記述の profile パスを §4 レイアウトへ整合、テスト戦略 / CI 記述に Go fmt/lint 採用を反映（§1・§4・§6）。モジュール統合表の HM 行に role 分離不可注記（§2）。
- **`CONTEXT.md`**: `engine` 定義の `.pending-<name>` を `.pending` に、profile レイアウト言及を §4 に整合（§4）。`module` / `nput CLI` 定義に HM の単一 profile 制約を軽く反映（§2）。
- **`CLAUDE.md`**: §7 により当該暫定セクションの削除は**第一スライス PR で**行う（本 ADR 反映時点では削除しない）。
- **実装フェーズ**: CLI（experimental-features エラーハンドリング・固定 init ref・専用ディレクトリレイアウトでの profile/pending/flock 操作・reset 非 TTY ガード）、ビルド（treefmt gofmt・flake check の go vet/golangci-lint）。ADR-0023 §4 の Slice 順で着手。

## 棄却した代替案

- **§1 で `--extra-experimental-features` を engine が自動付与**: 未設定環境でも追加要求ゼロで動くが、ユーザー / 組織の experimental-features 制御を CLI が黙って上書きし、設定差を隠す。前提条件の明示の方が透明。
- **§2 で MVP から HM を `nput.configs.<name>` の attrsOf に**: 粒度は standalone と揃うが options 設計と activation の複数 profile swap が重く、HM の低い positioning（ADR-0007）に見合わない（ADR-0024 §棄却と一貫）。
- **§3 でビルド時 `self` ref を ldflags 注入**: init が CLI と完全同一版の template を引けて schemaVersion が自動整合するが、buildGoModule への ref 注入と legacy/flake 両対応の実装が重い。版整合は devShell pin が担うため固定文字列で足りる。
- **§3 で nix registry の `nput` エントリ依存**: 最も短いが未登録環境で壊れ脆い。
- **§4 で共有 dir + name-prefix 兄弟（`profile=<name>` / `<name>-N-link` / `.pending-<name>`）**: 階層が浅いが flock を `<name>` 単位の別 lock ファイルで別途持つ必要があり、profile リンクと世代・pending の所属が一目で分からない。専用ディレクトリの方が所属が自明で flock キーが dir に一致する。
- **§5 で非 TTY を `--yes` 扱い（自動承諾）**: スクリプトから楽だが非対話での誤削除リスクが高い。
- **§6 で今すぐ treefmt/check に空設定を先置き / 着手時にその場判断**: 前者は整形対象不在で無意味、後者は恣意判断が混入。採否を今確定し実体を第一スライスへ寄せるのが両者の中間で最適。
- **§7 で今すぐ branch/PR へ切替 / 据え置き**: 前者は docs 反映の軽量運用利得を早く捨て、後者は撤廃条件の発火点を曖昧なまま残す。
