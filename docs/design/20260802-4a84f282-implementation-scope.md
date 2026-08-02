---
id: "DSG-4a84f282-76ea-47c9-aede-deac12ff5257"
type: design
name: "実装スコープを standalone CLI + project mode + home mode に限り、system mode とモジュール 2 層は将来拡張に置く"
satisfies:
  - "REQ-14f0aec9-abae-4621-82f3-40536a1ad904"
  - "REQ-690f2730-2628-420d-8e72-ed1ce747ac1e"
  - "REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9"
---
# DSG-4a84f282: 実装スコープを standalone CLI + project mode + home mode に限り、system mode とモジュール 2 層は将来拡張に置く

## 設計

| 対象 | スコープ |
|---|---|
| standalone CLI（home mode）| 実装する |
| project mode（devShell / `projectRoot`）| 実装する（コア）|
| home-manager モジュール | 実装する |
| NixOS / nix-darwin モジュール | 将来拡張（スタブ公開のみ・→ DSG-0e186e89）|
| system mode（`systemRoot`）| 将来拡張 |

REQ-14f0aec9 が nput CLI を一次 UX に据え、project mode を devShell 同梱で canonical と
する以上、**最初に完成させるべきは CLI と project mode**になる。home mode を同時に含めるのは、
world mode（`homeRoot`）が rollback・世代管理という nput 固有機構を通る唯一の経路であり、
ここを外すと世代機構が実装されないまま残るため。

将来拡張へ置く 2 つは、いずれも「engine 側の追加を必要としない」ことを条件に選んでいる。

- **NixOS / nix-darwin モジュール**: REQ-c1b3ca5f の通り配線に過ぎず、
  home-manager モジュールで確立する engine kick の形（DSG-98d7fa5d の
  ビルド済み manifest クラス）をそのまま流用できる。実装が後になっても
  engine の設計が変わらない。実 activate を E2E で検証しないことは
  REQ-690f2730 が定めている
- **system mode**: root マーカーとしては `systemRoot` を最初から union に含めるため
  （root マーカーの型は requirement 側の担当）、engine の root 解決に
  分岐を後から足すだけで済む

**モジュール対応の位置づけ**も同じ線引きから来る。モジュール層は他のモジュールシステムの
switch と一括で動いてほしいユースケースを拾うために存在するのであって、各モジュールの
内部事情を nput の設計へ持ち込まない。持ち込むと REQ-c1b3ca5f の「ネイティブ機構へ
翻訳しない」が層ごとに崩れる。

## 出典

`docs/design.md`「プロジェクト構成」末尾の実装スコープ段落（L64）と、
「モジュール統合設計」→「各統合層の動作」末尾のモジュール対応の位置づけ（L318-319）。
