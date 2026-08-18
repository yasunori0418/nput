---
id: "RISK-e916f742-59a3-4202-a316-603a908fd0db"
type: risk
name: "entrypoint の形ごとに root 解決と配置の意味論が食い違う"
threatens:
  - "REQ-9cb26ffd-071e-4c68-a6fc-faac6373b75e"
  - "REQ-6506bc82-d1e1-4dbf-8c57-d5d1babf218a"
  - "REQ-c890ce4a-6528-4ab3-ac86-23d7aebff7da"
  - "REQ-da253cab-34d4-4d6e-96f0-de99e012b376"
  - "DSG-92f54490-872a-42ac-bbd7-d06e9ee381c6"
likelihood: medium
impact: high
level: high
---
# RISK-e916f742-59a3-4202-a316-603a908fd0db: entrypoint の形ごとに root 解決と配置の意味論が食い違う

## リスク

nput は flake の entrypoint と legacy の非 flake entrypoint という 2 つの入口を持ち、
project mode ではさらに git から root を解決する。入口が増えるほど「同じ配置のはずなのに
入口によって結果が違う」という食い違いが入り込む。

- **root 解決の誤り** — project mode の root は git の toplevel であり、config の位置でも
  カレントディレクトリでもない。リポジトリ内のどこから呼んでも同じ場所へ解決されなければ、
  呼び出した場所によって配置先が変わる。逆にリポジトリの外で呼ばれたときは、曖昧に成功せず
  停止しなければならない。祖先がたまたまリポジトリであるだけで誤って成功するのが典型的な
  壊れ方である
- **入口ごとの意味論の分岐** — legacy 経路でも同じ config の指定・全件適用・出力契約が成り
  立つ必要がある。分岐が attr path の組み立てを越えて広がると、入口ごとに別実装を抱えること
  になる（この分岐の閉じ込め方は設計上の選択であり、別の設計を採れば消える risk なので
  design にも張る）
- **既存の利用体験の破壊** — legacy の入口へ nput のための属性を足したことで、素の nix-shell
  としての利用が壊れる
- **store 化の差の取り違え** — flake 経路では相対パスの配置元が store へ入るのに対し、
  legacy の impure な評価では作業木のパスのまま解決される。この差は意図されたものであり、
  同一の性質を両経路へ期待すると誤った検証をすることになる

## 影響

意図しない場所への配置、あるいは入口によって再現しない挙動。impact を high とするのは、
root 解決の誤りが「利用者の意図しないディレクトリへ書き込む」形で現れ、どこへ書いたかが
記録に残らない以上、原状回復の手立てが無いため。

likelihood を medium とするのは、project mode の root 解決は TC-ec0bded4-c4bc-4a61-a2b5-0b652134b223 が、legacy 入口の
配置と出力契約の同等性は TC-d9c78439-e478-4779-977c-9c4c02dd4e93 が覆っている一方、入口が flake / legacy の 2 つに
分かれ、そこへ git 解決が掛かるため、片方の入口だけを見た変更が他方との食い違いを作りうる
ため。engine 層で同じ root 解決を扱う RISK-24e0805d-53cd-40dd-9e7a-b5c4bbf2a298 が low なのに対して本 item が medium に
留まるのは、入口ごとの分岐という掛け算の側を持つぶん混入の余地が広いことによる。
