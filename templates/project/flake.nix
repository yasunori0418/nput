{
  description = "nput project config: フェッチ済み git リポジトリを repo 配下へ symlink / copy 配置する";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    # The nput library / CLI. Make nixpkgs follow so that mkManifest's pkgs and
    # the build inputs of the nput CLI placed on the devShell are aligned (schemaVersion consistency).
    nput = {
      url = "github:yasunori0418/nput";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    # In practice, declare the "git repository you want to place" as a flake = false input,
    # and swap the src of the entries below to this input (add me):
    #   my-repo = {
    #     url = "github:you/your-repo";
    #     flake = false;
    #   };
  };

  outputs =
    {
      self,
      nixpkgs,
      nput,
      ...
    }:
    let
      # Helper to expand the same config over 4 systems (flake-parts is overkill for a starter, so unused).
      forAllSystems = nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
    in
    {
      # nput.<system>.<name> namespace. `nput apply <name>` builds and places this config.
      # Since it is not a standard flake output, `nix flake check` emits
      # `warning: unknown flake output 'nput'` (exit 0, harmless).
      nput = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          # The config name example signals "rename me". Place it with `nput apply example`.
          example = nput.lib.mkManifest {
            inherit pkgs;

            # Take the repo (git toplevel) tree as root (target is relative to the repo root).
            root = nput.lib.projectRoot;

            # Attribute key = placement target (root-relative target). Here, nput's own docs/ is shown as an example.
            # In practice, swap src to the my-repo input above and subpath to the subdirectory you want to place.
            #
            # The placed artifacts are ephemeral (assumed regenerated, not git-managed). An ignore pattern is already
            # added to the .gitignore below. If you change target, update it with the output of `nput gitignore example`.
            entries.".nput/docs" = {
              src = nput;
              subpath = "docs";
            };
          };
        }
      );

      # devShell. On `nix develop` / direnv entry, put the pinned nput CLI on PATH
      # and auto-place the config in shellHook. No .envrc is bundled (adopting direnv is the user's call・ADR-0018).
      # If you use direnv, run `echo 'use flake' > .envrc && direnv allow` in the repo.
      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = [ nput.packages.${system}.nput ];

            # Place example on entry. With --no-wait, on flock contention skip without waiting (concurrent entries do not hang).
            # To place multiple configs at once:
            #   nput apply --all --project-root --no-wait
            shellHook = ''
              nput apply example --no-wait
            '';
          };
        }
      );
    };
}
