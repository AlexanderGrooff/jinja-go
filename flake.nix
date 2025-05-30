{
  description = "Spage Go project with Temporal";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        
        # Create Python environment with required packages
        pythonEnv = pkgs.python3.withPackages (ps: with ps; [
          jinja2
          tabulate
        ]);
      in
      {
        devShells.default = pkgs.mkShell {
          nativeBuildInputs = [
            pkgs.go_1_23 # Or your preferred Go version
            pkgs.pre-commit
            pkgs.gotools # Contains benchstat
            pythonEnv    # Python with dependencies
            # Add other Go tools or dependencies here if needed, e.g.:
            # pkgs.gopls
            # pkgs.delve
          ];

          # Set environment variables if necessary
          # shellHook = '''
          #   export GOPATH=$(pwd)/.go
          #   export GOBIN=$(pwd)/.go/bin
          #   # You might not need to set GOROOT if using a nix-shell provided Go
          # ''';
        };
      });
} 
