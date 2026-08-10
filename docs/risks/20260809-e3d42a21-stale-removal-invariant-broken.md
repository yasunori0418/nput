---
id: "RISK-e3d42a21-1f43-4ac6-835e-a5caf8d86363"
type: risk
name: "保守的 stale 除去の不変条件が破れ、nput が作っていない実体を消す"
threatens:
  - "REQ-16aef46b-7bb8-4ca1-b962-e9f3ed1fd1d2"
  - "REQ-fc64de4c-c82b-419c-8706-07d8d97daa37"
likelihood: medium
impact: high
level: high
---
# RISK-e3d42a21: 保守的 stale 除去の不変条件が破れ、nput が作っていない実体を消す

stale 除去は「前世代 manifest に記録されており、かつ実 FS 上のリンクが記録先を指したままである」
symlink だけを対象とする。この不変条件は plan 時に一度判定されるが、plan から unlink までの
間に実 FS が変化しうるため、除去の直前に再検証する必要がある。再検証を欠くか、条件のどれかを
緩めれば、ユーザーや他ツールが置いた実体を消す。

home-manager の readlink パターンによる cleanup がまさにこの取り違えを起こす方式であり、
nput は manifest 記録による分類でそれを避けている。分類が記録ベースから外れることが、この
脅威の主要な発現経路になる。

空の entries を「設定ミス」と解釈して除去を止める逆方向の失敗も含む。全クリアは正当な操作で
あり、警告やエラーで妨げれば撤去手段が失われる。

## 想定する失敗

- plan 後に drift した target（リンク先が変わった / 実ファイルに置換された / 消えた）を
  そのまま unlink する
- 記録に無い foreign symlink を stale として除去する
- copy entry を stale 除去の対象にし、ユーザー編集を含む実体を消す
- 空の entries をエラー・警告として扱い、全クリアができない

## 評価

- likelihood: medium — TC-d160e18b が除去対象の限定を、TC-8a1f4b19 が除去直前の再検証を
  覆っている。ただし同じ保守的不変条件を reset の teardown（RISK-bb54245e）と共有しており、
  どちらかの経路への変更がもう一方へ波及しうる
- impact: high — 除去はユーザーや他ツールが置いた実体に届き、unlink は不可逆。home-manager
  の readlink パターンがまさにこの取り違えを起こす方式で、記録ベースの分類から外れることが
  主要な発現経路になる

## 張り先の判断

2 本とも requirement へ張る。除去対象の限定も空 entries の扱いもユーザーが直接観測する
契約であり、除去アルゴリズムを差し替えても懸念は残る。
