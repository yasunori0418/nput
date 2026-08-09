---
id: "RISK-24e0805d-53cd-40dd-9e7a-b5c4bbf2a298"
type: risk
name: "root 解決を誤り、意図しないディレクトリツリーへ配置する"
threatens:
  - "REQ-9cb26ffd-071e-4c68-a6fc-faac6373b75e"
  - "REQ-6506bc82-d1e1-4dbf-8c57-d5d1babf218a"
  - "REQ-8d965ca2-f8fd-44a4-87f3-94e850e9f85b"
likelihood: medium
impact: high
level: high
---
# RISK-24e0805d: root 解決を誤り、意図しないディレクトリツリーへ配置する

全ての配置先は `<root>/<target>` として組み立てられるため、root の解決を誤ると entry 単位の
正しさに関わらず配置全体が別のツリーへ流れ込む。project mode の root を CWD 相対や config
相対で採ってしまえば、リポジトリ外へ配置物を撒く。home mode の root を層ごとに定まった供給元
以外から採れば、HM activation と CLI 直叩きで配置先が食い違う。

git から root を解決できないときに engine 実行時ではなく評価時に落ちる／逆に無言で fallback
する、という層分けの崩れも同じ脅威に含む。fallback すると「git repo の外で apply した」ことに
ユーザーが気づけない。

## 想定する失敗

- project mode で git toplevel ではなく CWD を root に採り、リポジトリ外へ配置する
- git repo 外での apply が停止せず、任意のディレクトリを root に採る
- home mode の root 供給元が層ごとに揃わず、HM 経由と CLI 経由で配置先が分岐する
- profile ディレクトリの作成・backref 書き込みの失敗が握り潰され、逆引き不能な状態で配置が進む

## 張り先の判断

3 本とも requirement へ張る。root をどこから解決するかはユーザーが直接触れる契約であり、
解決経路の実装を差し替えても「意図しないツリーへ配置される」懸念は消えない。
