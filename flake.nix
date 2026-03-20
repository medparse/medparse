{
  inputs = {
    flake-schemas.url = "https://flakehub.com/f/DeterminateSystems/flake-schemas/*";
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/*";
  };

  outputs = { self, flake-schemas, nixpkgs }:
    let
      supportedSystems = [ "aarch64-darwin" "x86_64-darwin" "x86_64-linux" "aarch64-linux" ];
      forEachSupportedSystem = f: nixpkgs.lib.genAttrs supportedSystems (system: f {
        pkgs = import nixpkgs { inherit system; };
      });
    in {
      schemas = flake-schemas.schemas;

      devShells = forEachSupportedSystem ({ pkgs }: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go

            # Dev tools
            nixpkgs-fmt
          ];

          shellHook = ''
            echo "🩺 medparse dev shell"
            echo "  go: $(go version)"
          '';
        };
      });
    };
}
