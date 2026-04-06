{
  description = "Argus Flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            poetry
            zizmor
            stdenv.cc.cc.lib
          ];

          shellHook = ''
            export LD_LIBRARY_PATH="${pkgs.lib.makeLibraryPath [ pkgs.stdenv.cc.cc.lib ]}:$LD_LIBRARY_PATH"

            echo "Argus dev shell"
            echo "Go $(go version | awk '{print substr($3, 3)}')"
            echo "Poetry $(poetry --version | awk -F '[ ()]' '{print $4}')"
            echo "Zizmor $(zizmor --version | awk '{print $2}')"
          '';
        };
      });
}
