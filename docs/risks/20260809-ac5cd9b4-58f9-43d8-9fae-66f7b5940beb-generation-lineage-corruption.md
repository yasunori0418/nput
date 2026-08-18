---
id: "RISK-ac5cd9b4-58f9-43d8-9fae-66f7b5940beb"
type: risk
name: "profile のキーイングか世代の読み取りが誤り、別 config の世代系列を取り違える"
threatens:
  - "REQ-d5a2e289-40bc-45a9-9d44-21b8dc561b81"
  - "REQ-2aa3abbc-90b2-486e-92de-d785554bdeb3"
  - "REQ-1be4d678-959c-44d7-a346-44bfd95af56e"
likelihood: medium
impact: high
level: high
---
# RISK-ac5cd9b4-58f9-43d8-9fae-66f7b5940beb: profile のキーイングか世代の読み取りが誤り、別 config の世代系列を取り違える

## リスク

`profileDir` の解決規則か、そこに並ぶ世代リンクの読み取りが誤ると、nput は「前世代」に
別 config の manifest を読む。前世代 manifest は stale 除去の入力そのものなので、
取り違えた瞬間に「別 config が置いた symlink を、この config の stale として消す」という
誤削除に直結する。

キーイング側の破れ方は 2 方向ある。**分離すべきものが混ざる** — fixed root や `--root`
上書きが `<roothash>` でキーされず `<name>` 直キーへ落ちると、別 root の同名 config が
同じ世代系列を共有する（silent orphan）。project mode で同一 entrypoint を複数箇所へ
クローンした場合も同様に衝突する。**分離すべきでないものが割れる** — 同じ root を指す
はずの呼び出しが毎回違う `<roothash>` を得ると（hash が決定的でない・長さが揺れる）、
apply のたびに新しい世代系列が生まれ、前世代が永久に見つからず stale 除去が効かなくなる。

レイアウト側では、profile リンク・`profile-N-link` 世代・`.pending` out-link を同一
ディレクトリの兄弟として並べる規約（→ REQ-2aa3abbc-90b2-486e-92de-d785554bdeb3）が崩れると、read-only な store パスを
貫通して書こうとする破綻を招く。`<state>` の解決（`$XDG_STATE_HOME` 優先・無ければ
`~/.local/state`・`$HOME` も解決できなければエラー）が誤れば、profile 一式が想定外の
場所に作られ、既存の世代が一切見えなくなる。

`nix-env --list-generations` の出力パースも同じ系統の脅威に属する。世代番号・日時・
`(current)` マーカーの読み取りを誤れば、rollback の戻り先も `list-generations` の表示も
狂う。数値でない行を黙って読み飛ばして「世代 0 件」と見なすのは、前世代が無いという
最も危険な結論を沈黙で作る。

## 実現性

**likelihood: medium** — キーイング規則はモード（home / project / fixed）と `--root` の
有無の掛け合わせで枝が多く、モード追加のたびに漏れが入りうる。

**impact: high** — 帰結が前世代 manifest の取り違えであり、ユーザーの実配置の誤削除に
つながる。消される symlink 自体は別 config の再 apply で張り直せるが、取り違えの原因である
世代系列の破損はそれでは直らない。系列が割れれば（`<roothash>` が呼び出しごとに揺れる側）
前世代が永久に見つからず rollback の戻り先も stale 除去の入力も失われ、系列が混ざれば
（`<name>` 直キーへ落ちる側）別 root の config どうしが互いの世代を上書きし続ける。
どちらも profile 一式を捨てて配置をやり直すまで回復せず、その時点で世代の履歴は戻らない。

## 緩和

TC-40b762fc-3b48-471c-b495-c5ffe1f584b5（`<state>` 解決・profileDir キーイング・世代リンク命名）と
TC-746cb5b9-fe27-4d09-a51d-eb03d56a93a7（世代一覧のパース）が緩和する。
