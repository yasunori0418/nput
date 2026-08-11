# テストドキュメント対応の 3 スクリプト（test-inventory.sh / test-doc-map.sh /
# test-doc-matrix.sh）が共有する処理（→ Issue #304、epic #283）。
#
# source 専用で、直接実行しない。呼び出し側は自分の位置から解決する:
#
#   . "$(dirname "$0")/lib-testdoc.sh"          # dev/scripts/ から
#   . "$(dirname "$0")/../scripts/lib-testdoc.sh"  # dev/tests/ から
#
# ここに置くのは「同一の知識」を表現する処理だけ（データファイルのコメント様式・
# 走査基点の解決規則・yq 実装の前提）。スクリプト固有のロジックは各々が持つ。
#
# 注: dev/flake.nix の checks.test-doc-map はサンドボックスへ dev/scripts と dev/tests を
# 重ねて配置する。このファイルが dev/scripts/ に居るのはその複製範囲に収めるため。

# データファイル（TSV）の実データだけを読む。行頭 # のコメントと空行を落とす。
read_tsv() { grep -v '^[[:space:]]*#' "$1" | grep -v '^[[:space:]]*$'; }

# 走査基点をリポジトリルートへ解決する。git 管理外ではカレント基準へフォールバックする
# （checks 派生のサンドボックスは作業ツリーを持たないためこの経路を通る）。
testdoc_repo_root() { git rev-parse --show-toplevel 2>/dev/null || printf '.'; }

# yq が mikefarah/yq（Go 実装・v4）であることを確かめる。nixpkgs の `yq` と ambient PATH の
# python-yq（v2/v3 系）は別実装・別構文で、取り違えると eval が黙って空を返す
# （CASE 0 件の対応表が出る事故を実際に踏んだ）。第 1 引数は診断メッセージ用のスクリプト名。
require_yq_go() {
  local caller=${1:-yq}
  if ! yq --version 2>&1 | grep -q 'mikefarah\|version v4'; then
    echo "$caller: mikefarah/yq v4 が要る（実際: $(yq --version 2>&1)）" >&2
    return 1
  fi
}

# 必須コマンドの存在確認。第 1 引数は診断メッセージ用のスクリプト名、以降が必須コマンド。
require_commands() {
  local caller=$1
  shift
  local cmd missing=0
  for cmd in "$@"; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      echo "$caller: $cmd が要る" >&2
      missing=1
    fi
  done
  return "$missing"
}
