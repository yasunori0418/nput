---
id: "TC-b9d4ffaf-ac91-4bf1-9f27-5ea3964466ad"
type: test_condition
name: "配置前除去による移行の可否判定が、自己記録 stale と method 変更の規則どおりであること"
mitigates:
  - "RISK-3e8dd093-f21f-49c6-8d8c-7d62a6fad1e5"
---
# TC-b9d4ffaf-ac91-4bf1-9f27-5ea3964466ad: 移行可否の判定

配置 target を占有する実体を配置前除去して移行してよいのは、それが自己記録の stale である
（あるいは空ディレクトリである）場合に限られることを検証する。実ディレクトリが占有する場合は、
ツリー全体の leaf が全て「記録済み stale または空」であるときにのみ移行できる。

per-file 配置 ↔ dir symlink 配置の双方向の移行、多階層にわたる空サブツリー、root 直下の実 dir
target を含む。旧 target と新 target が leaf 名を共有するケース（readlink パターンによる
cleanup が誤判定する形）が、記録ベースの分類で正しく移行されることを含む。

method 変更の非対称性も同じ条件の下に置く。symlink → copy は自動移行し（symlink はユーザー
データを持たない）、copy → symlink は通常の上書き拒否 conflict に留める。記録済み symlink が
drift していた場合に移行ではなく foreign 扱いへ落ちることも含む。
