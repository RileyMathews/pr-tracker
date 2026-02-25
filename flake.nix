{
  description = "GH Tracker tooling and Home Manager module";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };

        gh-tracker = pkgs.buildGoModule {
          pname = "gh-tracker";
          version = "0.1.0";
          src = self;
          vendorHash = "sha256-flPqLHi3dSUaG1ZwIlaJtKf1oifotD6ae3iZwZm+cJg=";
          subPackages = [
            "cmd/ght"
            "cmd/ght-ui"
            "cmd/ghtd"
          ];
        };
      in {
        packages = {
          default = gh-tracker;
          gh-tracker = gh-tracker;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            golangci-lint
            delve
            gotools
            go-tools
          ];

          shellHook = ''
            export GOPATH="$PWD/.gopath"
            export GOBIN="$GOPATH/bin"
            export PATH="$GOBIN:$PATH"
            mkdir -p "$GOPATH" "$GOBIN"
            echo "Go dev shell ready (go, gopls, golangci-lint, delve)."
          '';
        };
      });
}
