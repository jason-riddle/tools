{
  description = "Go CLI tools from github.com/jason-riddle/tools";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      # All four tier-1 platforms.
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      # Helper that maps a function over all supported systems.
      # Uses legacyPackages so nixpkgs is only imported once per system.
      forAllSystems = f:
        nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      # Derive a short version string from the flake's git metadata.
      # - rev/dirtyRev are the canonical full-hash attrs; we take 7 chars.
      # - Falls back to "dev" when the source is not in a git repo at all.
      version =
        if self ? rev then builtins.substring 0 7 self.rev
        else if self ? dirtyRev then builtins.substring 0 7 self.dirtyRev
        else "dev";
    in {
      # --- packages ---
      # Individual tools and a combined default that installs both.
      #
      #   nix build .#gob      — build the gob binary
      #   nix build .#uuid     — build the uuid binary
      #   nix build            — build both (default)
      #   nix profile add github:jason-riddle/tools        — install both
      #   nix profile add github:jason-riddle/tools#gob    — install gob only
      packages = forAllSystems (pkgs:
        let
          # Build a single CLI tool from cmd/<name>/main.go.
          # The module has no external deps (stdlib only), so vendorHash is null —
          # this tells buildGoModule to skip the vendor hash check entirely.
          mkTool = name:
            pkgs.buildGoModule {
              pname = name;
              inherit version;
              src = self;
              vendorHash = null; # no external deps; stdlib only
              subPackages = [ "cmd/${name}" ]; # only compile the requested cmd
              # No explicit `go` pin: Go is forward-compatible, so the nixpkgs
              # default toolchain (currently 1.26) builds a go 1.22 module fine.
              meta.mainProgram = name;         # tells `nix run .#<name>` which binary to exec
            };

          gob  = mkTool "gob";
          uuid = mkTool "uuid";
        in {
          inherit gob uuid;

          # Combined package: symlinks both binaries into a single store path.
          # This is what `nix profile add github:jason-riddle/tools` installs.
          default = pkgs.symlinkJoin {
            name  = "tools"; # stable name; version is already encoded in the store hash
            paths = [ gob uuid ];
            meta.description = "Go CLI tools: gob and uuid";
          };
        }
      );

      # --- devShells ---
      # Run `nix develop` to enter a shell with the Go toolchain and gopls.
      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go    # nixpkgs default toolchain; forward-compatible with go.mod's go 1.22
            gopls
          ];
        };
      });
    };
}
