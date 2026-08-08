---
id: "QA-0949183b-7ef0-4cae-b88f-3ad361576b63"
type: quality
name: "リリースはバージョンの一次情報の変更で駆動し、手作業の工程を挟まない"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  The version of nput SHALL have a single source of truth in the repository, from which
  every other place that states the version SHALL be derived rather than maintained by
  hand. Releasing SHALL be driven by a change to that source of truth reaching `main`, and
  the tag, the release notes and the GitHub Release SHALL be produced automatically from
  it; no manual tagging or hand-written changelog SHALL be part of the release path. The
  change that drives a release SHALL itself go through the merge gate that every other
  change to `main` goes through.
specification_ja: |
  nput のバージョンはリポジトリ内に唯一の一次情報を持たなければならず、バージョンを述べる
  他のすべての箇所はそこから導出されなければならない（手で二重管理してはならない）。
  リリースはその一次情報の変更が main へ到達することで駆動されなければならず、タグ・
  リリースノート・GitHub Release はそこから自動的に生成されなければならない。手動での
  タグ打ちや手書きの CHANGELOG をリリース経路に含めてはならない。リリースを駆動する変更
  自体も、main へのその他の変更と同じマージゲートを通らなければならない。
---
# QA-0949183b: リリースはバージョンの一次情報の変更で駆動し、手作業の工程を挟まない

## 方針

バージョンは flake の `version` と Go バイナリに埋め込む文字列の両方が述べるが、それぞれを
手で書き換える運用は片方だけが古くなる。一次情報を 1 箇所に置き、他はそこから導出する。

リリースの駆動も同じ考え方で、**一次情報の変更が main へ入ること**がリリースそのものになる。
人が別途タグを打つ工程を挟むと「マージしたがタグを忘れた」状態が生じ、CHANGELOG を手書き
すればコミット履歴との二重管理になる。

リリースを駆動する変更も PR としてマージゲート（QA-a5f7f088）を通る。「main 直接コミット
禁止」と「リリースの自動化」がここで整合する。

## 実現する基盤

- INF-9878e9f5（リリース自動化）— `VERSION` を一次情報とする bump PR・自動タグ・自動リリースノート

## 出典

ADR-0042（リリースを bump PR 起点で自動化する）が置いた方針を、基盤（INF）から分離して
quality item として立てたもの。一次情報のファイル名・workflow の分割・成果物を添付しない
判断は INF-9878e9f5 が持つ。
