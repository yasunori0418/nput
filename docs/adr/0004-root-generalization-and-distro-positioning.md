# ADR-0004: root 一般化と「純粋関数プリミティブ」としての positioning

- ステータス: 採用
- 日付: 2026-06-07
- 関連: ADR-0002, ADR-0003, `docs/concept.md`, `docs/design.md`（mkActivationScript シグネチャ）

## 背景

nput の north-star として、「nixpkgs のパッケージ群（＝ストアパス）を活かしつつ、配置だけをユーザーに操作させ、
Arch / Gentoo 的なミニマル Linux ディストリビューションの基盤を作る」という構想がある。
これを踏まえ、コアの中心抽象と、既存ツール（特に numtide/system-manager）との positioning を定める。

## 決定

- **コア = 「nix store のパスを root 相対の target に配置する」プリミティブ**とする。
  `home.file` 相当（root = `$HOME`）はその一適用に過ぎない。
- **root は内部でパラメータ化する**が、公開 API（`mkActivationScript`）は当面 root = `$HOME` 固定とする。
  シグネチャは `{ pkgs, name, entries }`。root 差し替え（将来の system 配置, root = `/`）は
  design.md に「将来の拡張 seam」として記し、安定公開引数にはまだしない。
- **今回の実装スコープは standalone + home-manager をコアとする。** NixOS / nix-darwin モジュールは将来拡張。
- **nput はフレームワークではなく、テスト可能な純粋関数群（配置プリミティブ）である。**
  distro は nput の関数をユーザーが合成して組むものとし、モジュール抽象は意図的に避ける。
  これが home-manager / NixOS / system-manager との差別化点そのものである。

## system-manager との positioning

調査により system-manager（numtide）の境界を一次情報で確認した。

- `lib.evalModules` による NixOS 風モジュールシステムを流用し、`environment.systemPackages`
  （→ `/run/system-manager/sw` profile + PATH）/ systemd unit / `environment.etc`（`/etc/nix/nix.conf` 含む）を
  **宣言的に**作る。
- ブートローダ・カーネル・init・FS・パーティション・stage1 には一切関与しない。既存 distro に乗る augment 層。

含意:

- ブート / init / FS 部分は system-manager にも空白であり、ミニマル distro 基盤のその層は誰も埋めていない。
- パッケージ / systemd / `/etc` の **モジュールによる宣言管理**は system-manager と完全に被る。
  これを nput が **モジュールとして**再実装するのは再発明であり、避ける。
- ただし nput の thesis は「モジュールで隠さず、テスト可能な純粋関数でユーザーに握らせる」ことであり、
  同じドメインでも**アプローチが思想レベルで異なる**ため競合しない。
- 比較軸は「機能の有無」ではなく **「モジュール抽象で隠す（NixOS / HM / system-manager） vs 純粋関数でユーザーが握る（nput）」**。

## 根拠

- root を `$HOME` にハードコードすると distro ビジョンが作り直しになる。root 一般化なら引数差で到達できる。
- 「関数ベースのパッケージ導入・PATH 追加」の具体機構は未定義であり、今回の spec スコープに含めない
  （north-star に留める）ことで、スコープの肥大を防ぐ。

## 影響

- concept.md に north-star 節と、「モジュール抽象 vs 純粋関数」軸での既存ツール比較を追加する。
- design.md の mkActivationScript シグネチャに root 内部パラメータと将来 seam を反映する。

## 棄却した代替案

- **nput を distro フレームワークとして root=/ の systemd / `/etc` 統合まで担う**:
  system-manager と大きく重複し再発明リスク。
- **今は `$HOME` 固定で distro は将来別設計**: system 層拡張時に root 抽象を後付けするリファクタが発生する。
- **distro ビジョンを docs に書かない**: 設計判断の文脈（root 一般化の動機）が失われる。
