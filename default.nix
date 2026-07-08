{
  pkgs ? (
    let
      inherit (builtins) fetchTree fromJSON readFile;
      inherit ((fromJSON (readFile ./flake.lock)).nodes) nixpkgs gomod2nix;
    in
    import (fetchTree nixpkgs.locked) {
      overlays = [
        (import "${fetchTree gomod2nix.locked}/overlay.nix")
      ];
    }
  ),
  buildGoApplication ? pkgs.buildGoApplication,
}:

buildGoApplication {
  pname = "git-wit";
  version = "0.0.1";
  pwd = ./.;
  src = pkgs.lib.fileset.toSource {
    root = ./.;
    fileset = pkgs.lib.fileset.fileFilter (
      file:
      (file.hasExt "go" && !pkgs.lib.hasSuffix "_test.go" file.name)
      || builtins.elem file.name [
        "go.mod"
        "go.sum"
      ]
    ) ./.;
  };

  nativeBuildInputs = [
    pkgs.git
    pkgs.installShellFiles
  ];

  postInstall = ''
    installShellCompletion --cmd git-wit \
      --bash <($out/bin/git-wit completion bash) \
      --fish <($out/bin/git-wit completion fish) \
      --zsh <($out/bin/git-wit completion zsh)
  '';
}
