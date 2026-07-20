{ inputs, ... }:
{
  imports = [
    # keep-sorted start
    inputs.devshell.flakeModule
    inputs.git-hooks.flakeModule
    inputs.treefmt-nix.flakeModule
    # keep-sorted end
  ];
  perSystem =
    {
      self',
      config,
      pkgs,
      system,
      ...
    }:
    let
      goEnv = (pkgs.mkGoEnv { pwd = ../../.; });
    in
    {
      _module.args.pkgs = import inputs.nixpkgs {
        inherit system;
        overlays = [
          inputs.gomod2nix.overlays.default
        ];
      };

      checks = {
        go-test = pkgs.stdenv.mkDerivation {
          inherit (self'.packages.default) goVendorDir goCacheDir;
          name = "go-test";
          src = ../../.;
          doCheck = true;
          nativeBuildInputs = with pkgs; [
            hooks.goConfigHook
            hooks.goBuildHook

            git
            goEnv
            writableTmpDirAsHomeHook
          ];
          checkPhase = ''
            go test -v ./...
          '';
          installPhase = ''
            mkdir "$out"
          '';
        };
        go-lint = pkgs.stdenv.mkDerivation {
          inherit (self'.packages.default) goVendorDir goCacheDir;
          name = "go-lint";
          src = ../../.;
          doCheck = true;
          nativeBuildInputs = with pkgs; [
            hooks.goConfigHook
            hooks.goBuildHook

            goEnv
            golangci-lint
            writableTmpDirAsHomeHook
          ];
          checkPhase = ''
            golangci-lint run
          '';
          installPhase = ''
            mkdir "$out"
          '';
        };
      };

      pre-commit = {
        check.enable = false;
        settings = {
          src = ../../.;
          hooks = {
            # keep-sorted start block=yes
            golangci-lint.enable = true;
            gotest = {
              enable = true;
              # nixpkgs' default `go` still trails go.mod's `go 1.26.1`, which makes
              # `go test` try (and, offline, fail) to fetch a newer toolchain.
              package = pkgs.go_1_26;
            };
            actionlint.enable = true;
            treefmt.enable = true;
            # keep-sorted end
          };
        };
      };

      treefmt = {
        projectRootFile = "go.mod";
        programs = {
          nixfmt.enable = true;
          gofumpt.enable = true;
        };
      };

      devshells.default = {
        devshell = {
          packages =
            with pkgs;
            [
              nix-update
              golangci-lint
              gomod2nix
            ]
            ++ lib.optional pkgs.stdenv.isLinux pkgs.gcc;
          packagesFrom = [
            goEnv
          ];
          startup = {
            pre-commit = {
              text = config.pre-commit.shellHook;
            };
          };
        };
      };
    };
}
