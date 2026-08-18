---
id: "RISK-ffb5100d-c553-4dd8-937a-1263ad84a8c7"
type: risk
name: "コピー結果が src と構造・属性で食い違い、配置物が使えなくなる"
threatens:
  - "REQ-84e3c717-adf5-4ff3-b0db-d039b82ef19c"
  - "REQ-0bd55dfc-b28c-4ad2-ad50-bb1f76b2246c"
  - "REQ-a8a923ad-07fb-4582-b90a-07a6e0c41baa"
likelihood: low
impact: medium
level: low
---
# RISK-ffb5100d-c553-4dd8-937a-1263ad84a8c7: コピー結果が src と構造・属性で食い違い、配置物が使えなくなる

copy の src は nix store の read-only なツリーであり、そのまま複製すると書き込めない配置物に
なる。逆に mode を捨てて一律のパーミッションにすれば、実行ビットを持つスクリプトが実行できなく
なる。「元の mode を保存しつつ owner-write を付与する」はこの両方を同時に満たすための規範で、
どちらかへ倒れると配置物が使いものにならない。

ツリー構造の再現にも同種の脅威がある。src 内の symlink を deref すればサイズと意味が変わり、
空ディレクトリを落とせばツールが期待するレイアウトが崩れる。out-of-store marker が live symlink
ではなくコピーとして落ちる場合も、「編集が即座に反映される」という利用者の期待を裏切る。

## 想定する失敗

- read-only な mode をそのまま複製し、配置後のファイルを編集できない
- mode を捨てて実行ビットが落ち、スクリプトが実行できなくなる
- src 内の symlink を deref し、リンク構造を失う
- ネストしたディレクトリ・空ディレクトリが再現されない
- out-of-store marker が live symlink にならず、src 側の変更が反映されない

## 評価

- likelihood: low — 忠実性の規範は copytree の 1 箇所に閉じており、TC-b1b8c163-9d37-47ee-9838-7168569df03a（構造・mode・
  symlink の再現と owner-write 付与）と TC-cf1b44ec-2ec4-4ee1-9a37-378e41ccb01e（out-of-store marker の live 性）が
  機械的に覆っている
- impact: medium — 配置物が使えない状態は目に見える形で現れ、規範どおりに直したうえでの
  再 apply で回復できる

## 張り先の判断

3 本とも requirement へ張る。mode の扱い・symlink 非 deref・out-of-store の live 性は
いずれも配置物としてユーザーが直接観測する性質であり、コピー実装を差し替えても懸念は残る。
