---
id: "TC-3b02ab58-5bfb-4ae7-9ac1-c69e2ece2722"
type: test_condition
name: "実 FS の故障注入で FS 変更段の途中を失敗させ、apply 前の状態が完全に復元される"
mitigates:
  - "RISK-68e810c5-4e68-4b25-9bc0-6b2613022b49"
---
# TC-3b02ab58-5bfb-4ae7-9ac1-c69e2ece2722: 実 FS の故障注入で FS 変更段の途中を失敗させ、apply 前の状態が完全に復元される

## テスト条件

`Apply` を end-to-end で走らせ、FS を変更する各段の途中で失敗させたときに、その run が
加えた変更が全て巻き戻り、apply 前の状態へ戻ることを検証する。故障は実 FS の条件で
誘発する（→ TP-deb05610-44bc-4962-8939-952392e5fbd0）。

故障注入の手法は、対象ディレクトリを mode 0o555 にして「plan 時の `Lstat` / `ReadDir` は
通るが execute 時の書き込みだけが EACCES で落ちる」状態を作ること。通常ファイルで
置き換える手法だと配下の `Lstat` が plan 時に ENOTDIR で落ち、「バッチ途中の失敗」に
ならない。root では権限チェックが迂回されるため、その条件を検出して skip する
（空虚に成功させない）。

失敗させる段と、巻き戻し対象の組み合わせを覆う。

| 失敗させる段 | 巻き戻し対象 |
|---|---|
| place（新規 symlink バッチの 2 本目）| 先行して置かれた symlink。後続は配置されていないこと |
| place（張替えの後の別 entry）| 張替え済み symlink が pre-apply の dest へ戻ること |
| removeStale | 先に materialize 済みの copy tree |
| place（PreRemove の migration 完了後）| dir symlink の除去・実ディレクトリと配下 leaf symlink の再作成 |
| place（`--recopy` の rename 退避後）| 退避ファイルの rename 戻しと新しい copy 内容の破棄 |

いずれの場合も、巻き戻し後に新世代が commit されていないこと（`nix-env --set` へ到達
しない）を併せて確認する。

## 対応する CASE

CASE-154af597-df3e-49fb-a96b-b4f371dfcc63（`internal/engine/undo_journal_test.go`）。
