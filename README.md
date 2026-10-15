# AITBASE VISION Plugins

These are official example plugins for AITBASE VISION devices, built with the [AITBASE VISION Plugin SDK](https://github.com/Aitbase/aitbase-vision-plugin-sdk).

> **Status: preview.** Plugins currently run only on devices with development firmware. They use the SDK's v0 API and are native binaries. They will move to sandboxed WebAssembly packages as the platform matures, with the same SDK calls.

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

Each plugin builds:
- its program;
- `<plugin>_ui.json`, the compiled screen the device validates and renders.

## Layout

```
sdk/                 SDK, as a git submodule pinned to a release
plugins/<name>/
  plugin.json        manifest (draft)
  ui/ui.xml          screen definition; assets live in ui/assets/
  src/               plugin code using aitbase_plugin.h
  CMakeLists.txt
```

## Adding a plugin

1. Copy `plugins/counter` to `plugins/<name>`.
2. Change the target name in its `CMakeLists.txt`.
3. Change `id` and `name` in `plugin.json`.

The top-level CMake picks up every `plugins/*` directory. The SDK docs list what UI XML may contain ([`docs/ui-xml.md`](https://github.com/Aitbase/aitbase-vision-plugin-sdk/blob/main/docs/ui-xml.md)).

## License

To be decided.
