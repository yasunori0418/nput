{
  description = "nput development environment";

  inputs = {
    root.url = "path:../";
    nixpkgs.follows = "root/nixpkgs";
    flake-parts.follows = "root/flake-parts";

    # Claude Code 用スキル集（mattpocock/skills）。
    # 従来は vercel の skills コマンド + skills-lock.json で .claude/skills/ に展開していたが、
    # nput のドッグフーディングとして project mode の nput apply で配置する（flake.lock が rev を pin）。
    matt-skills = {
      url = "github:mattpocock/skills";
      flake = false;
    };

    # niface エンベロープの E2E 適合検証キット（niface-validate CLI + id-vectors）。
    # 規格由来の検証依存として許容する（→ issue #132・niface ADR-0021 / 0023 / 0025）。
    niface = {
      url = "github:yasunori0418/niface";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-parts.follows = "flake-parts";
    };

    # 個人 NUR。sara（アーキテクチャ文書・要求をナレッジグラフとして管理する CLI）を
    # devShell に載せるために引く。nixpkgs は follows で寄せない：NUR 側 CI がビルドして
    # yasunori0418.cachix.org に push した store path をそのまま引くためで、寄せると
    # store path が変わり devShell 構築のたびに Rust をローカルビルドすることになる。
    # sara は nput のビルド・ライブラリに一切絡まない独立した CLI なので nixpkgs を
    # 揃える整合性上の必要もない（代償は flake.lock に nixpkgs がもう一本増えること）。
    nur = {
      url = "github:yasunori0418/nur-packages";
    };
  };

  outputs =
    inputs@{ flake-parts, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
    in
    flake-parts.lib.mkFlake { inherit inputs; } {
      inherit systems;
      imports = [
        inputs.root.flakeModules.default
        # nput dogfood config（perSystem.nput.skills）を flake-parts module として切り出す。
        ./nput.nix
      ];
      perSystem =
        { inputs', pkgs, ... }:
        let
          # niface の Go 参照実装は subPackages 由来の bin/validate を生むため、
          # 規格上の CLI 名 niface-validate で PATH に載せる薄い wrapper を挟む。
          niface-validate = pkgs.writeShellScriptBin "niface-validate" ''
            exec ${inputs'.niface.packages.validate}/bin/validate "$@"
          '';

          # sara item の ID を採番する（UUIDv4 二層構成・→ ADR-0048、epic #203）。
          #
          #   sara-id <型名 | prefix> [slug]
          #
          # 正式 ID（frontmatter の id:）・ファイル名素材・散文中の参照の 3 つを出す。
          # sara init の自動採番（suggest_next_id）は連番前提で使えないため代替する。
          #
          # 乱数 ID は並列レーン（parallel-worktree）での採番衝突を構造的に回避する。
          # フル UUIDv4 は事実上衝突ゼロだが、人間が触る前方 8 文字は 120 item で
          # 約 10⁻⁶ の偶然重複がありうるので、採番時に docs/ を 1 回走査して
          # 既出なら生成し直す（ms オーダーで item 数が増えても実用上コスト増なし）。
          sara-id = pkgs.writeShellApplication {
            name = "sara-id";
            runtimeInputs = with pkgs; [
              # macOS 標準の uuidgen には -r が無いため、明示的に載せて移植性を担保する。
              util-linux
              ripgrep
              coreutils
            ];
            text = ''
              usage() {
                cat >&2 <<'EOF'
              usage: sara-id <type|prefix> [slug]

                type|prefix  sara の型名（requirement / test_condition …）または
                             prefix そのもの（REQ / TC …）
                slug         ファイル名に使う短い識別子（省略可・英数字とハイフン）

              例:
                sara-id requirement lock-ordering
                sara-id adr
              EOF
              }

              if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
                usage
                exit 2
              fi

              case "$1" in
                -h | --help)
                  usage
                  exit 0
                  ;;
              esac

              # 型名 → prefix。docs/model.yaml の item_types と対応させる。
              # prefix を直接渡された場合はそのまま大文字化して通す。
              case "$1" in
                solution | sol) prefix=SOL ;;
                use_case | use-case | uc) prefix=UC ;;
                requirement | req) prefix=REQ ;;
                design | dsg) prefix=DSG ;;
                infrastructure | inf) prefix=INF ;;
                adr) prefix=ADR ;;
                risk) prefix=RISK ;;
                test_condition | test-condition | tc) prefix=TC ;;
                test_case | test-case | case) prefix=CASE ;;
                defect | d) prefix=D ;;
                *) prefix=$(printf '%s' "$1" | tr '[:lower:]' '[:upper:]') ;;
              esac

              slug=''${2-}

              # ADR は連番を維持する（既存 47 本の相互参照・docs/adr/README.md の運用・
              # Issue 言及を壊さないため → ADR-0048）。UUID は採番しない。
              if [ "$prefix" = "ADR" ]; then
                echo "sara-id: ADR は連番を維持する（docs/adr/ の最大値 + 1 を手で採番する）" >&2
                exit 2
              fi

              # UUID 生成は seam 経由で呼ぶ（テストが決定論的に差し替えるため）。
              uuidgen_cmd=''${SARA_ID_UUIDGEN:-uuidgen}

              # 既出チェックの走査先。docs/ が無いリポジトリでも動くようにする。
              scan_dir=docs

              uuid=""
              # 8 文字 prefix の偶然重複は極めて稀なので、有限回で打ち切る。
              # 打ち切りに達するのは乱数源が壊れているとき（同じ値を返し続ける等）で、
              # 黙って重複 ID を返すより失敗させたほうがよい。
              for _ in $(seq 1 16); do
                candidate=$("$uuidgen_cmd" -r | tr -d '\n')
                short=''${candidate:0:8}
                if [ ! -d "$scan_dir" ]; then
                  uuid=$candidate
                  break
                fi
                # 省略形は正式 ID の前方一致なので、8 文字で引けば
                # 宣言側（frontmatter の id:）と参照側（散文・relation）の両方に当たる。
                if ! rg -q --fixed-strings "$short" "$scan_dir"; then
                  uuid=$candidate
                  break
                fi
              done

              if [ -z "$uuid" ]; then
                echo "sara-id: 8 文字 prefix の未使用な候補を 16 回で引けなかった" >&2
                exit 1
              fi

              short=''${uuid:0:8}
              date_prefix=$(date +%Y%m%d)

              if [ -n "$slug" ]; then
                filename="$date_prefix-$short-$slug.md"
              else
                filename="$date_prefix-$short.md"
              fi

              printf 'id: %s-%s\n' "$prefix" "$uuid"
              printf 'filename: %s\n' "$filename"
              printf 'ref: %s-%s\n' "$prefix" "$short"
            '';
          };
        in
        {
          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              statix
              nixd
              inputs'.root.formatter
              inputs'.root.packages.nput
              # アーキテクチャ文書・要求をナレッジグラフとして扱う CLI（NUR 由来）。
              # CONTEXT.md / docs/adr の設計文書運用を補助する開発時ツールで、
              # nput のビルド・テスト経路には関与しない。
              inputs'.nur.packages.sara
              # sara item の ID 採番（UUIDv4 二層構成）。定義は上の let 束縛を参照。
              sara-id
              # sara-id が使う uuidgen -r の供給元。macOS 標準 uuidgen には -r が
              # 無いため明示的に載せて移植性を担保する（→ Issue #207）。
              util-linux
              go
              gopls
              # ローカルでカバレッジ計測する coverage ツール（go test -coverprofile + go tool cover）。
              # func サマリを出し、HTML を見たい場合のコマンドを案内する（閾値ゲートは持たない）。
              (writeShellScriptBin "nput-coverage" ''
                set -euo pipefail
                profile="cover.out"
                go test -coverprofile="$profile" ./...
                go tool cover -func="$profile"
                echo "HTML レポート: go tool cover -html=$profile"
              '')
            ];
            shellHook = ''
              export REPO_ROOT=$(git rev-parse --show-superproject-working-tree --show-toplevel)
              # mattpocock/skills を .claude/skills/ に dogfood 配置する（project mode）。
              # 競合時は待たず skip（--no-wait）し、no-op
              nput apply skills -f "$REPO_ROOT/dev" --no-wait
            '';
          };

          # CI の sara check 専用シェル。default devShell は nput のビルドと
          # dogfood の shellHook（nput apply skills）を伴うため、docs 変更だけの PR で
          # それらを走らせないよう sara 単体に絞る。NUR 由来の store path を
          # yasunori0418.cachix.org から引くだけで済む。
          devShells.sara = pkgs.mkShell {
            packages = [ inputs'.nur.packages.sara ];
            env.TERM = "dumb";
          };

          # 非 NixOS E2E ハーネス（tests/e2e/run.sh）専用の最小 CI シェル（→ ADR-0012 §2）。
          # dev 専用ツール（statix / nixd / gopls 等）と dogfood の shellHook を持たず、
          # ハーネスが要する nput バイナリ + bash / git / jq / coreutils だけを提供する。
          # nix / nix-env は install-nix-action が入れた ambient nix を使う（pkgs.nix を載せて
          # 上書きしない）。TERM=dumb で対話 UI を抑える。
          devShells.ci = pkgs.mkShell {
            packages = with pkgs; [
              inputs'.root.packages.nput
              bash
              git
              jq
              coreutils
              # --json エンベロープの適合検証（schema〔format assertion 込み〕+ lint MUST・→ issue #132）。
              niface-validate
            ];
            env.TERM = "dumb";
            # E2E の id-vectors 整合チェックが参照する適合ベクタ（niface testdata の正本）。
            env.NIFACE_ID_VECTORS = "${inputs.niface}/testdata/v1/id-vectors.json";
          };
        };
    };
}
