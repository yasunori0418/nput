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
              # 既出チェックの走査先をリポジトリルート基準で解決するため。
              git
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
                sara-id design

              注: ADR は連番を維持するため採番しない（docs/adr/ の最大値 + 1 を手で採る）
              EOF
              }

              # help は引数個数に先立って処理する（`sara-id --help extra` も help になる）。
              case "''${1-}" in
                -h | --help)
                  usage
                  exit 0
                  ;;
              esac

              if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
                usage
                exit 2
              fi

              # 型名 → prefix。docs/model.yaml の item_types と対応させる。
              # 別名は sara init のサブコマンド別名に揃える。
              # 未知の入力は prefix そのものを渡されたとみなして大文字化して通す。
              case "$1" in
                solution | sol) prefix=SOL ;;
                use_case | use-case | uc) prefix=UC ;;
                requirement | req) prefix=REQ ;;
                design | dsg) prefix=DSG ;;
                infrastructure | inf) prefix=INF ;;
                quality | qa) prefix=QA ;;
                test_plan | test-plan | tp) prefix=TP ;;
                adr) prefix=ADR ;;
                risk) prefix=RISK ;;
                test_condition | test-condition | tc) prefix=TC ;;
                test_case | test-case | case) prefix=CASE ;;
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

              # 既出チェックの走査先。リポジトリルート基準で解決する（カレント相対だと
              # dev/ 等から叩いたときに docs/ を見つけられず、重複チェックが黙って
              # 外れる）。git 管理外ならカレント基準へフォールバックする。
              repo_root=$(git rev-parse --show-toplevel 2>/dev/null || printf '.')
              scan_dir="$repo_root/docs"

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

          # sara ドキュメントグラフの未カバーを 3 段で列挙する決定論コマンド。
          #
          #   sara-gap [--json]
          #
          #   ① threatens されていない requirement / design（リスク識別が未着手の仕様）
          #   ② mitigates されていない risk（テスト条件が無いリスク）
          #   ③ covers されていない test_condition（テストケースが無いテスト条件）
          #
          # `sara check --format json` は宣言辺（threatens 等の primary relation）だけを
          # 出力し逆辺を持たないため、items 配列から jq で逆引きインデックスを自前構築する。
          # exit code は 0 = ギャップなし / 1 = ギャップあり / 2 = sara check 失敗・JSON 形状異常。
          #
          # CI ゲートにはしない。現グラフは threatens されていない REQ / DSG を多数残して
          # おり（リスク識別は epic #283 の逆算で起こした範囲しか覆っていない）最初から
          # 赤のためゲートにならず、仮に赤を許容基準にしても、実証で item を起こすたびに
          # 件数が動いて工程の進行順序を CI が強制してしまう（ゲート化は forward 運用の
          # 感触を得てから判断する）。
          sara-gap = pkgs.writeShellApplication {
            name = "sara-gap";
            runtimeInputs = [
              inputs'.nur.packages.sara
              pkgs.jq
              pkgs.coreutils
              # sara.toml のある走査基点をリポジトリルート基準で解決するため。
              pkgs.git
            ];
            text = ''
              usage() {
                cat >&2 <<'EOF'
              usage: sara-gap [--json]

              sara ドキュメントグラフの未カバーを 3 段で列挙する:
                1. threatens されていない requirement / design
                2. mitigates されていない risk
                3. covers されていない test_condition

              exit code:
                0  ギャップなし
                1  ギャップあり
                2  sara check の失敗・JSON 形状の異常
              EOF
              }

              json_out=0
              case "''${1-}" in
                -h | --help)
                  usage
                  exit 0
                  ;;
                --json) json_out=1 ;;
                "") ;;
                *)
                  usage
                  exit 2
                  ;;
              esac
              if [ "$#" -gt 1 ]; then
                usage
                exit 2
              fi

              # sara.toml のあるリポジトリルートで sara を実行する（sara-id と同じ解決）。
              # SARA_GAP_ROOT はサンドボックスの契約テストが fixture を指すための seam。
              repo_root=''${SARA_GAP_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null || printf '.')}
              cd "$repo_root" || {
                echo "sara-gap: リポジトリルートへ移動できない: $repo_root" >&2
                exit 2
              }

              # sara 呼び出しの seam（テストが JSON 形状異常を決定論的に再現するため）。
              sara_cmd=''${SARA_GAP_SARA:-sara}

              # グラフが invalid の間はギャップを算出しない（壊れたグラフから作る一覧は
              # 信用できないため。stderr は sara の診断をそのまま通す）。
              if ! graph=$("$sara_cmd" check --format json); then
                echo "sara-gap: sara check が失敗した（先にグラフを valid にすること）" >&2
                exit 2
              fi

              # .items 配列と valid を確認する。sara のバージョン更新で JSON 形状が変わった
              # とき、黙って空の結果（＝ギャップなし）を返す事故を防ぐガード。
              if ! printf '%s' "$graph" | jq -e '(.items | type) == "array"' >/dev/null; then
                echo "sara-gap: sara check の JSON に .items 配列が無い（sara の出力形状が変わった可能性）" >&2
                exit 2
              fi
              if ! printf '%s' "$graph" | jq -e '.valid == true' >/dev/null; then
                echo "sara-gap: sara check が invalid を報告した（先にグラフを valid にすること）" >&2
                exit 2
              fi

              # 宣言辺から張り先（to）の集合を作り、各段の未カバーを逆引きする 1 パス。
              # source.file_path は repositories.paths（./docs）相対なので docs/ を前置する。
              gaps=$(printf '%s' "$graph" | jq '
                def targets(t): [.items[].relationships[]? | select(.relationship_type == t) | .to] | unique;
                def row: {ref: (.id | split("-") | .[0] + "-" + .[1][0:8]), name, file: ("docs/" + .source.file_path)};
                targets("threatens") as $threatened
                | targets("mitigates") as $mitigated
                | targets("covers") as $covered
                | {
                    unthreatened: ([.items[]
                      | select(.item_type == "requirement" or .item_type == "design")
                      | select(.id as $i | ($threatened | index($i)) | not) | row] | sort_by(.ref)),
                    unmitigated: ([.items[]
                      | select(.item_type == "risk")
                      | select(.id as $i | ($mitigated | index($i)) | not) | row] | sort_by(.ref)),
                    uncovered: ([.items[]
                      | select(.item_type == "test_condition")
                      | select(.id as $i | ($covered | index($i)) | not) | row] | sort_by(.ref))
                  }')

              if [ "$json_out" -eq 1 ]; then
                printf '%s\n' "$gaps"
              else
                printf '%s' "$gaps" | jq -r '
                  def section(title; rows):
                    ["## " + title]
                    + (if (rows | length) == 0 then ["なし"] else [rows[] | "\(.ref)\t\(.name)\t\(.file)"] end);
                  section("threatens されていない requirement / design"; .unthreatened)
                  + [""]
                  + section("mitigates されていない risk"; .unmitigated)
                  + [""]
                  + section("covers されていない test_condition"; .uncovered)
                  | .[]'
              fi

              total=$(printf '%s' "$gaps" | jq '[.[] | length] | add')
              if [ "$total" -gt 0 ]; then
                exit 1
              fi
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
              # uuidgen -r の供給元（util-linux）は sara-id の runtimeInputs で
              # wrapper の PATH に前置されるため、devShell 側には載せない。
              sara-id
              # グラフ未カバー 3 段の列挙。定義は上の let 束縛を参照。sara / jq は
              # runtimeInputs で wrapper の PATH に前置される。
              sara-gap
              # dev/tests/test-doc-map.sh・dev/scripts/test-doc-matrix.sh が CASE
              # frontmatter を読むのに使う yq-go（mikefarah/yq v4）。載せないと
              # ambient PATH の python-yq（別実装・別構文）を拾って黙って空を返す。
              yq-go
              # test-doc-matrix.sh が sara report matrix --format json を整形するのに使う。
              jq
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

          # sara-id の採番契約を固定するテスト（dev/tests/sara-id.sh）。同じスクリプトを
          # 2 経路から走らせる意図的な二重化で、役割が違う:
          #
          # - この checks 派生: ローカルの `nix flake check ./dev`（CLAUDE.md の標準検証
          #   手順）に載せ、dev flake を触ったときに手を動かさず走るようにする。
          #   CI の flake-check job はルート flake を対象にするため、ここは CI では回らない。
          # - CI: .github/workflows/test.yml の sara job が devShells.sara 経由で実行する。
          #   PR での退行検知はこちらが担保する。
          #
          # サンドボックス（runCommandLocal）と devShell では実行条件が違うので、
          # 両経路とも緑であることを確認してから変更を入れること。
          checks.sara-id =
            pkgs.runCommandLocal "sara-id-test"
              {
                # prefix マップの突合（テスト §6b）が読む正本 2 つ。サンドボックスには
                # リポジトリの作業ツリーが無く、テスト側の git ルート解決も効かないため
                # nix から store path を渡す。devShell / CI 経路は cwd がリポジトリ
                # ルートなのでテスト側の解決に任せる。
                SARA_MODEL_YAML = ../docs/model.yaml;
                SARA_DEV_FLAKE = ./flake.nix;
                nativeBuildInputs = [
                  sara-id
                  pkgs.ripgrep
                  pkgs.coreutils
                  # テストがルート解決経路を検証するため一時 repo を git init する。
                  pkgs.git
                  # テストが sara-id の出力から id / filename / ref を抜くのに使う。
                  # stdenv 既定でも PATH に載るが、暗黙依存にすると実行条件の違う
                  # サンドボックスで踏み抜く（→ 0a18d87 の exit 126）ため明示する。
                  pkgs.gnused
                ];
              }
              ''
                bash ${./tests/sara-id.sh}
                touch "$out"
              '';

          # テストコード ⇔ CASE 対応の契約テスト（dev/tests/test-doc-map.sh）。
          # checks.sara-id と同じ二重化の意図で 2 経路から走らせる:
          #
          # - この checks 派生: ローカルの `nix flake check ./dev` に載せる
          # - CI: .github/workflows/test.yml の sara job が devShells.sara 経由で実行し、
          #   PR での退行検知を担保する（flake-check job はルート flake が対象なので
          #   この派生は CI では回らない）
          #
          # テストは docs/ と実際のテスト資産（cmd/ internal/ tests/）の両方を走査する。
          # サンドボックスに作業ツリーは無く、テスト側の git ルート解決も効かないため、
          # ルート flake の store path（inputs.root。dev/flake.nix の path:../ 入力）を
          # 書き込み可能な場所へ複製し、その中で走らせる。dev/ 配下のスクリプト・
          # データファイルは store の dev flake 側から重ねる（ルート flake の store path は
          # dev/ を含むが、そちらは編集中の内容と一致しない可能性がある）。
          checks.test-doc-map =
            pkgs.runCommandLocal "test-doc-map"
              {
                nativeBuildInputs = [
                  pkgs.yq-go
                  pkgs.git
                  pkgs.coreutils
                  pkgs.gnused
                  pkgs.gnugrep
                  pkgs.diffutils
                  pkgs.findutils
                ];
              }
              ''
                cp -r ${inputs.root} repo
                chmod -R u+w repo
                rm -rf repo/dev/scripts repo/dev/tests
                mkdir -p repo/dev
                cp -r ${./scripts} repo/dev/scripts
                cp -r ${./tests} repo/dev/tests
                chmod -R u+w repo/dev
                cd repo
                # テスト側の走査基点はカレントへフォールバックする（git 管理外のため）。
                bash dev/tests/test-doc-map.sh
                touch "$out"
              '';

          # risk の level 導出マトリクス整合の契約テスト（dev/tests/risk-matrix.sh）。
          # checks.sara-id と同じ二重化の意図で 2 経路から走らせる:
          #
          # - この checks 派生: ローカルの `nix flake check ./dev` に載せる
          # - CI: .github/workflows/test.yml の sara job が devShells.sara 経由で実行し、
          #   PR での退行検知を担保する（flake-check job はルート flake が対象なので
          #   この派生は CI では回らない）
          checks.risk-matrix =
            pkgs.runCommandLocal "risk-matrix"
              {
                # テストが走査する risk item の在り処。サンドボックスにはリポジトリの
                # 作業ツリーが無く、テスト側の git ルート解決も効かないため nix から
                # store path を渡す（checks.sara-id の SARA_MODEL_YAML と同じ手法）。
                # devShell / CI 経路は cwd がリポジトリルートなのでテスト側の解決に任せる。
                # マトリクスの正本（dev/tests/risk-matrix.tsv）は下で dev/ の木ごと
                # 配置するので、テストがスクリプト基準で解決する。
                RISK_DOCS_DIR = ../docs/risks;
                nativeBuildInputs = [
                  pkgs.coreutils
                  pkgs.findutils
                  # 走査基点の解決に使う。この経路では RISK_DOCS_DIR が先に解決するので
                  # 実際には使われないが、`git` が無いと `git rev-parse` が
                  # command not found となり診断が濁る。
                  pkgs.git
                  # frontmatter の読み取りに使う（mikefarah/yq v4）。テスト側も
                  # require_yq_go で実装を確認して落とす。
                  pkgs.yq-go
                  # lib-testdoc.sh が使う（read_tsv のコメント除去・require_yq_go の
                  # yq --version 判定）。
                  pkgs.gnugrep
                ];
              }
              ''
                # テストは dev/scripts/lib-testdoc.sh を自身からの相対パスで source する
                # （checks.test-doc-map と同じ配置前提）。store の単体ファイルを直接
                # 実行すると解決できないので、dev/ の木の形を作ってから走らせる。
                mkdir -p dev
                cp -r ${./scripts} dev/scripts
                cp -r ${./tests} dev/tests
                chmod -R u+w dev
                bash dev/tests/risk-matrix.sh
                touch "$out"
              '';

          # sara-gap の検出契約を固定するテスト（dev/tests/sara-gap.sh）。
          # checks.sara-id と同じ二重化の意図で 2 経路から走らせる:
          #
          # - この checks 派生: ローカルの `nix flake check ./dev` に載せる
          # - CI: .github/workflows/test.yml の sara job が devShells.sara 経由で実行し、
          #   PR での退行検知を担保する（flake-check job はルート flake が対象なので
          #   この派生は CI では回らない）
          #
          # テストは fixture（dev/tests/fixtures/sara-gap/）を自身からの相対パスで解決する
          # ため、checks.risk-matrix と同じく dev/ の木の形を作ってから走らせる。
          checks.sara-gap =
            pkgs.runCommandLocal "sara-gap-test"
              {
                nativeBuildInputs = [
                  sara-gap
                  # テスト自身のアサーション用（sara / jq は sara-gap の runtimeInputs
                  # から wrapper 経由で解決されるが、テストは jq を直接も使う）。
                  pkgs.jq
                  pkgs.coreutils
                  pkgs.gnugrep
                  # ルート解決経路（SARA_GAP_ROOT 無し）の検証で一時 repo を git init する。
                  pkgs.git
                ];
              }
              ''
                mkdir -p dev
                cp -r ${./tests} dev/tests
                chmod -R u+w dev
                bash dev/tests/sara-gap.sh
                touch "$out"
              '';

          # CI の sara check 専用シェル。default devShell は nput のビルドと
          # dogfood の shellHook（nput apply skills）を伴うため、docs 変更だけの PR で
          # それらを走らせないよう sara 単体に絞る。NUR 由来の store path を
          # yasunori0418.cachix.org から引くだけで済む。
          # CI からは sara check・dev/tests/sara-id.sh・dev/tests/test-doc-map.sh・
          # dev/tests/risk-matrix.sh・dev/tests/sara-gap.sh をこのシェルで実行する。
          devShells.sara = pkgs.mkShell {
            packages = [
              inputs'.nur.packages.sara
              sara-id
              sara-gap
              # 以下は dev/tests/sara-id.sh が使う。stdenv 既定や runner の system
              # PATH でも引けるが、checks.sara-id と揃えて明示する。
              pkgs.ripgrep
              pkgs.git
              pkgs.gnused
              pkgs.coreutils
              # dev/tests/test-doc-map.sh が CASE frontmatter の target を読むのに使う。
              # yq-go（mikefarah/yq v4）。nixpkgs の `yq` は python-yq（別実装・別構文）
              # なので取り違えないこと。テスト側も実装を確認して落とす。
              pkgs.yq-go
              # dev/tests/sara-gap.sh が --json 出力のアサーションに使う。
              pkgs.jq
            ];
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
