{
  description = "nput development environment";

  inputs = {
    root.url = "path:../";
    nixpkgs.follows = "root/nixpkgs";
    flake-parts.follows = "root/flake-parts";
  };

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
      perSystem =
        { inputs', pkgs, ... }:
        {
          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              statix
              nixd
              inputs'.root.formatter
              gopls
            ];
            shellHook = ''
              export REPO_ROOT=$(git rev-parse --show-superproject-working-tree --show-toplevel)
            '';
          };
        };
    };
}
