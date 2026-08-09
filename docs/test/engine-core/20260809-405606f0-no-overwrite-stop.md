---
id: "TC-405606f0-3ac8-4cc2-998e-d4759a62a171"
type: test_condition
name: "実体が占有する target と祖先 symlink を上書きせず停止すること"
mitigates:
  - "RISK-e8449214-7794-4d0d-b584-a3a152e2c1f9"
---
# TC-405606f0: 実体が占有する target と祖先 symlink での停止

target に通常ファイル・実ディレクトリが在るとき、および target の祖先コンポーネントが
symlink であるとき、engine が配置を行わず conflict として停止することを検証する。
上書きは回復不能な破壊になるため、「停止する」ことに加えて「配置物を一切残さない」ことまでを
条件に含む。

祖先が自己記録の stale symlink である場合だけは移行（配置前除去）が許されるので、その分岐が
foreign 祖先の停止と取り違えられていないことも併せて検証する。
