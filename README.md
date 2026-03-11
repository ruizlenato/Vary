# Vary

Vary is a desktop patching UI for Morphe CLI, built with Gio (Go).

![Demo](./demo.gif)

## Requirements

- Go 1.25+
- Java installed and available in `PATH`

Optional:
- ADB in `PATH` (if not available, Vary can download Android platform-tools automatically)

## Build and Run

Prebuilt binaries are available in GitHub Releases.

### Run from source

```bash
go run ./cmd/vary
```

### Build binary

```bash
go build -o vary ./cmd/vary
./vary
```

### Check or apply app updates

The binary can check for newer GitHub Releases and update itself before the UI starts:

```bash
go run ./cmd/vary --version
go run ./cmd/vary --check-updates
go run ./cmd/vary --self-update
```

For release builds, the workflow embeds the Git tag into the binary so `--version` and `--self-update` compare against the published release correctly.

## AppData

Vary stores runtime state in AppData under `vary`.

Typical locations:
- Linux: `~/.local/share/vary`
- Windows: `%LOCALAPPDATA%\vary`

Stored data includes:
- downloaded Morphe assets,
- patch selection state per package,
- config (`config.json`),
- default/generated keystore files.

## License

This project is licensed under the GNU GPL v3. See `LICENSE` for details.
