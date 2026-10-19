# AITBASE VISION Plugins

These are example plugins for AITBASE VISION devices, built with the [AITBASE VISION Plugin SDK](https://github.com/Aitbase/aitbase-vision-plugin-sdk).

> **Status: preview.** Plugins run only on devices with development firmware. They use plugin API version 1. The same source builds natively (development bridge, official firmware plugins) or as sandboxed WebAssembly, which is how community plugins run.

| Plugin | What it shows |
|---|---|
| [counter](plugins/counter) | A screen, a bound label, and a button action driven by the RIGHT key. |
| [stopwatch](plugins/stopwatch) | A screen without buttons, driven by the device's buttons (`AITBASE_EVENT_KEY`), with a timer and the monotonic clock. Needs SDK 0.8.0. |
| [weather](plugins/weather) | No screen: answers the voice assistant through an assistant tool (`assistant_tools`, `AITBASE_EVENT_TOOL_CALL`) with an HTTP request. Needs SDK 0.10.0. |

## Build a plugin for the device

Install the SDK's `aitbase-plugin` CLI and its WebAssembly toolchain once. The CLI carries the SDK, so there is nothing else to fetch ([Windows and details](https://github.com/Aitbase/aitbase-vision-plugin-sdk/blob/main/docs/cli.md#install)):

```sh
curl -fsSL https://raw.githubusercontent.com/Aitbase/aitbase-vision-plugin-sdk/main/install.sh | sh
```

Then, from a clone:

```sh
git clone https://github.com/Aitbase/aitbase-vision-plugins.git
cd aitbase-vision-plugins

aitbase-plugin build --wasm plugins/counter
aitbase-plugin pack --wasm plugins/counter   # plugins/counter/dist/example.counter-<version>.avp
```

Copy the `.avp` into `aitbase/plugins/` on the device's SD card, and the device runs it. `pack --wasm` refuses a module the device would refuse, for example one that uses files or sockets, and signs the package with your key (see `aitbase-plugin key`).

- **The SDK:** the one built into the CLI ([Which SDK](https://github.com/Aitbase/aitbase-vision-plugin-sdk/blob/main/docs/cli.md#which-sdk)). CI pins it in `SDK_VERSION`; `aitbase-plugin version` shows yours.
- **Ids:** the examples use the `example.*` namespace. Devices refuse `aitbase.*` ids on the SD card; that namespace is for official plugins, which ship in the firmware.

### Other builds

```sh
aitbase-plugin check plugins/counter   # manifest, ids, file names and screens
cmake -S . -B build -DAITBASE_SDK_DIR="$(aitbase-plugin sdk)" && cmake --build build   # every plugin, natively
```

CI builds and packages every plugin as WebAssembly with the same CLI commands, and uploads the packages.

## Releases

Pushing a version tag (`v0.1.0`) publishes a GitHub release with each plugin's `<id>-<version>.avp` and `SHA256SUMS` (`.github/workflows/release.yml`). Each plugin keeps its own version from `plugin.json`; bump it when the plugin changes, so the package names stay unique. Tags with a suffix (`v0.1.0-rc.1`) become prereleases.

## Layout

```
SDK_VERSION          SDK (CLI) release that CI builds with
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
4. Run `aitbase-plugin check plugins/<name>`.

To start a plugin of your own outside this repository, use `aitbase-plugin init` instead. It creates a standalone project with agent instructions.

The top-level CMake picks up every `plugins/*` directory. The SDK docs list what UI XML may contain ([`docs/ui-xml.md`](https://github.com/Aitbase/aitbase-vision-plugin-sdk/blob/main/docs/ui-xml.md)).

## License

To be decided.
