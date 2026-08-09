---
id: "RISK-bc7526c2-2a2d-427f-9490-92d7d262fce3"
type: risk
name: "コピー中の syscall 失敗が握り潰され、中途半端な配置物と誤った成功報告が残る"
threatens:
  - "REQ-07c3b735-3744-4778-a640-8c6fb66f4aa7"
  - "REQ-95e97d01-5c34-44b3-bc85-9ca53472bc3d"
likelihood: medium
impact: medium
level: medium
---
# RISK-bc7526c2: コピー中の syscall 失敗が握り潰され、中途半端な配置物と誤った成功報告が残る

copy は mkdir / open / read-write / readlink / lstat という複数の syscall の連なりで、
どれもが権限・ENOTDIR・容量などで失敗しうる。個々の失敗を握り潰すと、部分的にしか書かれて
いないツリーが残ったまま apply が成功したように見え、ユーザーは壊れた配置物を正常と誤認する。

結果レコードの汚染も同じ脅威に含む。失敗した entry を「コピー済み」として結果へ記録すれば、
`--json` の消費者や後続の世代管理が実体と食い違う情報を受け取る。foreign な実ファイルを
place-once の規約どおり skip した場合も、黙って飛ばせばユーザーは「反映されたはず」と誤解する
ため、warning での可視化がこの脅威への対処にあたる。

## 想定する失敗

- mkdir / open / copy / readlink の失敗が無視され、不完全なツリーが残る
- 非 ENOENT の lstat 失敗が「target 不在」と同一視され、誤った分岐へ進む
- 失敗した entry が結果へ「コピー済み」として記録される
- foreign な実ファイルの skip が無警告で行われ、未反映に気づけない

## 張り先の判断

REQ-07c3b735（skip の warning 可視化）と REQ-95e97d01（停止時に全件をガイダンス付きで報告）
へ張る。どちらも「失敗・非適用がユーザーへ届く」というプロダクトの振る舞いへの懸念で、
コピー実装の選択に依存しない。
