{
  description = "e64ec personal wiki/blog static site";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            go-tools
            golangci-lint
            templ
            air
            gnumake
            go-task
          ];

          shellHook = ''
            export GOPATH="$HOME/go"
            export PATH="$GOPATH/bin:$PATH"
            echo "e64ec dev shell: go $(go version | awk '{print $3}'), templ $(templ version 2>/dev/null || echo '?')"
          '';
        };
      });
}
