# AITBASE VISION Plugins

These are official example plugins for AITBASE VISION devices, built with the [AITBASE VISION Plugin SDK](https://github.com/Aitbase/aitbase-vision-plugin-sdk).

> **Status: preview.** Plugins run only on devices with development firmware. They use plugin API version 1. The same source builds natively (development bridge, official firmware plugins) or as sandboxed WebAssembly, which is how community plugins run.

| Plugin | What it shows |
|---|---|
| [counter](plugins/counter) | A screen, a bound label, and a button action driven by the RIGHT key. |

## Build

```sh
git clone --recursive https://github.com/Aitbase/aitbase-vision-plugins.git
cd aitbase-vision-plugins

# Host build. It checks that code and UI compile, but can't run without a device.
cmake -S . -B build && cmake --build build

# Device build: pass the cross toolchain for your device.
cmake -S . -B build-arm -DCMAKE_TOOLCHAIN_FILE=<toolchain.cmake> && cmake --build build-arm
```

WebAssembly build, with [wasi-sdk](https://github.com/WebAssembly/wasi-sdk) 34 or later:

```sh
cmake -S . -B build-wasm -DCMAKE_TOOLCHAIN_FILE=<wasi-sdk>/share/cmake/wasi-sdk-p1.cmake \
  -DWASI_SDK_PREFIX=<wasi-sdk> -DCMAKE_BUILD_TYPE=MinSizeRel
cmake --build build-wasm

# Writes dist/<id>-<version>-wasm.zip; unzip it at the root of the device's SD card.
python3 tools/package_wasm.py plugins/counter build-wasm/plugins/counter dist --strip <wasi-sdk>/bin/llvm-strip
```

`package_wasm.py` refuses a module the device would refuse, for example one that imports files or sockets through WASI. The zip holds `aitbase/plugins/<id>/` with `<entry>.wasm`, the compiled screens and `plugin.json`, the directory the device runs community plugins from. This is an interim format until devices install signed `.avp` packages.

Each plugin builds:
- its program;
- one `<screen>.ui.json` per screen, which the device validates and renders.

## Check and package

Use the SDK's [`aitbase-plugin`](https://github.com/Aitbase/aitbase-vision-plugin-sdk/blob/main/docs/cli.md) CLI. Download it from the [SDK releases](https://github.com/Aitbase/aitbase-vision-plugin-sdk/releases), or build the copy matching the pinned SDK with `go -C sdk build -o aitbase-plugin ./cmd/aitbase-plugin`.

```sh
# Before building: manifest, ids, file names and screens.
aitbase-plugin check plugins/counter --sdk sdk --allow-reserved-id

# After building: writes dist/aitbase.counter-<version>.avp.
aitbase-plugin pack plugins/counter --sdk sdk --allow-reserved-id \
  --build-dir build/plugins/counter --out dist
```

`--allow-reserved-id` is needed because these official plugins use the `aitbase.*` ids, which `check` rejects for everyone else. CI runs both commands on every plugin and uploads the packages as an artifact. It also builds every plugin as WebAssembly and uploads the SD card zips.

## Releases

Pushing a version tag (`v0.1.0`) publishes a GitHub release with each plugin's `<id>-<version>-wasm.zip` and `SHA256SUMS` (`.github/workflows/release.yml`). Each plugin keeps its own version from `plugin.json`; bump it when the plugin changes, so the zip names stay unique. Tags with a suffix (`v0.1.0-rc.1`) become prereleases.

## Layout

```
sdk/                 SDK, as a git submodule pinned to a release or merged commit
tools/package_wasm.py  WebAssembly packaging for the SD card
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

1. Copy `plugins/counter` to `plugins/<name>`.
2. Change the target and screen names in its `CMakeLists.txt`.
3. Change `id` and `name` in `plugin.json`, and `AITBASE_PLUGIN_ID(...)` in `src/` to the same id.
4. Run `aitbase-plugin check plugins/<name> --sdk sdk --allow-reserved-id`.

To start a plugin of your own outside this repository, use `aitbase-plugin init` instead. It creates a standalone project with agent instructions.

The top-level CMake picks up every `plugins/*` directory. The SDK docs list what UI XML may contain ([`docs/ui-xml.md`](https://github.com/Aitbase/aitbase-vision-plugin-sdk/blob/main/docs/ui-xml.md)).

## License

To be decided.
