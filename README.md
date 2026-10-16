# AITBASE VISION Plugins

These are official example plugins for AITBASE VISION devices, built with the [AITBASE VISION Plugin SDK](https://github.com/Aitbase/aitbase-vision-plugin-sdk).

> **Status: preview.** Plugins run only on devices with development firmware. They use plugin API version 1. The same source builds natively (development bridge, official firmware plugins) or as sandboxed WebAssembly, which is how community plugins run.

| Plugin | What it shows |
|---|---|
| [counter](plugins/counter) | A screen, a bound label, and a button action driven by the RIGHT key. |

## Build a plugin for the device

Install the SDK's `aitbase-plugin` CLI and its WebAssembly toolchain once ([Windows and details](https://github.com/Aitbase/aitbase-vision-plugin-sdk/blob/main/docs/cli.md#install)):

```sh
curl -fsSL https://raw.githubusercontent.com/Aitbase/aitbase-vision-plugin-sdk/main/install.sh | sh
```

Then, from a clone with its submodule:

```sh
git clone --recursive https://github.com/Aitbase/aitbase-vision-plugins.git
cd aitbase-vision-plugins

aitbase-plugin build --wasm plugins/counter
aitbase-plugin pack --wasm plugins/counter --allow-reserved-id   # plugins/counter/dist/aitbase.counter-<version>-wasm.zip
```

Unzip the package at the root of the device's SD card. It creates `aitbase/plugins/<id>/`, and the device runs it while Developer mode is on, which it is by default. `pack --wasm` refuses a module the device would refuse, for example one that uses files or sockets.

- **The SDK:** the CLI uses this repository's `sdk/` submodule, found automatically.
- **`--allow-reserved-id`:** needed because these official plugins use the `aitbase.*` ids, which the CLI rejects for everyone else.
- **Interim format:** the zip is what devices run until they install signed `.avp` packages.

### Other builds

```sh
aitbase-plugin check plugins/counter --allow-reserved-id   # manifest, ids, file names and screens
cmake -S . -B build && cmake --build build                  # every plugin, natively: checks code and UI compile
```

CI builds and packages every plugin as WebAssembly with the same CLI commands, and uploads the zips.

## Releases

Pushing a version tag (`v0.1.0`) publishes a GitHub release with each plugin's `<id>-<version>-wasm.zip` and `SHA256SUMS` (`.github/workflows/release.yml`). Each plugin keeps its own version from `plugin.json`; bump it when the plugin changes, so the zip names stay unique. Tags with a suffix (`v0.1.0-rc.1`) become prereleases.

## Layout

```
sdk/                 SDK, as a git submodule pinned to a release (v0.4.0)
plugins/<name>/
  plugin.json        manifest (draft)
  ui/
    screens/*.xml    screens (LVGL 9.4 XML conventions)
    components/*.xml reusable components
    globals.xml      images
    assets/          RGB565 images
  src/               callbacks using aitbase_plugin.h
  CMakeLists.txt
```

## Adding a plugin

1. Copy `plugins/counter` to `plugins/<name>`. Keep the first lines of its `CMakeLists.txt`, which let `aitbase-plugin build` build the plugin on its own.
2. Change the target and screen names in its `CMakeLists.txt`.
3. Change `id` and `name` in `plugin.json`, and `AITBASE_PLUGIN_ID(...)` in `src/` to the same id.
4. Run `aitbase-plugin check plugins/<name> --allow-reserved-id`.

To start a plugin of your own outside this repository, use `aitbase-plugin init` instead. It creates a standalone project with agent instructions.

The top-level CMake picks up every `plugins/*` directory. The SDK docs list what UI XML may contain ([`docs/ui-xml.md`](https://github.com/Aitbase/aitbase-vision-plugin-sdk/blob/main/docs/ui-xml.md)).

## License

To be decided.
