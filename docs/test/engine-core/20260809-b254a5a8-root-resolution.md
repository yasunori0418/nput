---
id: "TC-b254a5a8-7fbf-4f31-8486-3e373d66bfa7"
type: test_condition
name: "root 解決と profile ディレクトリ準備の成功・失敗経路が仕様どおりであること"
mitigates:
  - "RISK-24e0805d-53cd-40dd-9e7a-b5c4bbf2a298"
---
# TC-b254a5a8: root 解決と profile 準備の経路

project mode の root が git toplevel から解決されること、git repo の外で実行したときに
engine 実行時に停止すること、および root 解決に失敗したときに配置へ進まないことを検証する。
`fixed` で root パスが無いときに engine が拒否することも同じ条件の射程に含む。
成功経路も同じ条件の下に置く。`rootKind=home` が `$HOME` を絶対 root として返すこと、
`--root` 上書きが rootKind によらず優先されることを見る。

profile ディレクトリの作成・backref 書き込みの失敗が握り潰されずエラーとして表面化することも
同じ条件の下に置く。これらが黙って失敗すると、配置は進むのに逆引きできない profile が残る。

root 供給元の層分けそのもの（HM モジュール経由と CLI 経由で同じ root へ落ちること）は
`integration` 対象の担当で、条件の射程から外れる。engine が受け取った rootKind を絶対 root へ
解決する分岐は、home mode 側も含めてここで見る。
