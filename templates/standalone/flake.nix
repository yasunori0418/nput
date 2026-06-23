{
  description = "nput standalone config: symlink / copy fetched git repositories under home";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    # The nput library / CLI. Make nixpkgs follow to align the pkgs that mkManifest uses.
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

            # Take the home tree as root (target is relative to $HOME).
            root = nput.lib.homeRoot;

            # Attribute key = placement target (root-relative target). Here, nput's own docs/ is shown as an example.
            # In practice, swap src to the my-repo input above and subpath to the subdirectory you want to place.
            entries.".config/nput-docs" = {
              src = nput;
              subpath = "docs";
            };

            # ---- Variation examples ------------------------------------------------
            #
            # Omitting subpath = place the whole repository:
            #   entries.".config/my-repo" = { src = my-repo; };
            #
            # method = "copy" = place as a regular file instead of a symlink (writable, place-once):
            #   entries.".config/editable" = {
            #     src = my-repo;
            #     subpath = "config";
            #     method = "copy";
            #   };
            #
            # out-of-store symlink = symlink directly to a local absolute path instead of the store:
            #   entries.".config/live" = {
            #     src = nput.lib.mkOutOfStoreSymlink "/abs/path/to/dir";
            #   };
            #
            # Multiple entries = just add more attributes:
            #   entries.".config/a" = { src = my-repo; subpath = "a"; };
            #   entries.".config/b" = { src = my-repo; subpath = "b"; };
            #
            # Dynamic entries = scan an already-realised store path / flake input with builtins.readDir
            # to build entries (a raw derivation / out-of-store marker cannot be readDir'd, to avoid IFD):
            #   entries = builtins.mapAttrs
            #     (name: _: { src = my-repo; subpath = "skills/${name}"; })
            #     (builtins.readDir "${my-repo}/skills");
          };
        }
      );
    };
}
