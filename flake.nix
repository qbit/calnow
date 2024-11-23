{
  description = "calnow: stuff and calnows";

  inputs.nixpkgs.url = "nixpkgs/nixos-24.11";

  outputs =
    { self
    , nixpkgs
    ,
    }:
    let
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {
      overlay = _: prev: { inherit (self.packages.${prev.system}) calnow; };

      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          calnow = pkgs.buildGoModule {
            pname = "calnow";
            version = "v0.0.0";
            src = ./.;

            vendorHash = "sha256-SWkQMF9bFAzqsmHWKNsTpx8HuXrXZe9+vIOI0mmAPrw=";
          };
        });

      defaultPackage = forAllSystems (system: self.packages.${system}.calnow);
      devShells = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            shellHook = ''
              PS1='\u@\h:\@; '
              nix run github:qbit/xin#flake-warn
              echo "Go `${pkgs.go}/bin/go version`"
            '';
            nativeBuildInputs = with pkgs; [ git go gopls go-tools ];
          };
        });
    };
}
