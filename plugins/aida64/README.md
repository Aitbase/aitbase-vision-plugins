# AIDA64

A home screen face with a PC's CPU, memory and storage: bars, a CPU arc and labels, drawn by the device. It is the card the device firmware drew before AIDA64 became a plugin, pixel for pixel. A client on the PC pushes the numbers over the local network. The plugin has no code to speak of: the device shows the face without running it.

It shows what an idle face can do with the display widgets of device decision D36 (`lv_bar`, `lv_arc`, flex rows, any color, a label floating over the arc, scrolling labels, growing bars), fed by a face push token (D35).

| File | Purpose |
|---|---|
| `ui/screens/face.xml` | The face: CPU, Memory and Storage cards bound to `face.4` to `face.16`, and the old card's made-up numbers until the first push. 29 widgets, 7.4 KB compiled. |
| `ui/globals.xml` | The card's titles in English and Vietnamese. |
| `src/main.c` | Stops at once if started: everything is in the face. |
| `plugin.json` | `idle_face: "face"`. |
| `client/` | The AIDA64 push client for Windows and Linux (Go): reads the PC and pushes the face's values. See its [README](client/README.md). |

## Needs

- Device firmware with D35 (face cycle, face push tokens) and D36 (display widgets). Older firmware refuses the face, and the home screen keeps its clock.
- Device firmware whose renderer has `long_mode`, `floating`, `align`, `flex_grow` and `style_arc_color` (the release after SDK 0.12.1), and, for 0.3.0, translations and face samples (SDK 0.14.0). Older firmware refuses the face, and the home screen keeps its clock.
- The SDK at `SDK_VERSION`, whose compiler takes those attributes.

## Install

```sh
aitbase-plugin build --wasm plugins/aida64
aitbase-plugin pack --wasm plugins/aida64 --out dist
```

1. Copy `dist/example.aida64-<version>.avp` into `aitbase/plugins/` on the device's SD card, and delete any older `example.aida64-*.avp` there: the 0.1.0 face was a different design.
2. On the Device Console, open **Display**, add **AIDA64** to the face cycle, and create a push token for it.
3. Run the [AIDA64 client](client/README.md) on the PC with that token. The face fills in at the next push.

Device firmware from 2026-10-22 on shows the face with no badge, exactly like the old built-in card. Older firmware draws its **Plugin** badge in the top right corner, as it does on every face from the SD card.

## The face's values

A client writes these keys with the push token (`POST /api/v1/plugins/face-values`). Each is text, at most 64 bytes. A bar or arc takes the leading integer of its key (`"45%"` is 45).

| Key | Value | Example | Drawn as |
|---|---|---|---|
| `face.1` | RAM speed | `DDR5-5600` | (not shown) |
| `face.2` | RAM used | `14.2GB` | (in `face.14`) |
| `face.3` | RAM free | `17.8GB` | (in `face.14`) |
| `face.4` | RAM load | `45%` | Memory: Load bar and value |
| `face.5` | CPU name | `Ryzen 7 7800X3D` | (in `face.16`) |
| `face.6` | CPU load | `12%` | the arc and the value in it |
| `face.7` | CPU temperature | `62C` | bar (0–100 °C) and value |
| `face.8` | CPU clock | `4850 MHz` | label |
| `face.9` | CPU fan | `1200 RPM` | label |
| `face.10` | Disk name | `Samsung 990 Pro` | (not shown) |
| `face.11` | Disk usage | `1.2TB/2.0TB` | (in `face.13` and `face.15`) |
| `face.12` | Disk temperature | `41C` | (not shown) |
| `face.13` | Disk used, percent | `60%` | Storage bar and value |
| `face.14` | RAM "used / total" | `14.2 / 32.0 GB` | Memory: Usage |
| `face.15` | Disk total | `2.0TB` | Storage, before the bar |
| `face.16` | CPU header | `CPU - Ryzen 7 7800X3D` | the CPU card's title |

`face.1` to `face.12` are the 12 metrics of the AIDA64 push client, in its order. `face.13` to `face.16` are worked out from them as the old card did; the client sends them, and so does the device's old route (`POST /api/v1/idle-screen/aida64`, kept one release):
- `face.13`: used over total of `face.11`, rounded, with `%`.
- `face.14`: `face.2` and `face.2 + face.3` with their unit, as whole numbers when both are, else one decimal; `face.2` alone when the units differ.
- `face.15`: what follows the `/` in `face.11` (the whole text without one).
- `face.16`: `CPU - ` and `face.5` (`Unknown CPU` when empty).

The titles follow the device's language, as the old card's did: `Memory`, `Usage`, `Load` and `Storage` in English, `Bộ nhớ`, `Sử dụng`, `Tải` and `Lưu trữ` in Vietnamese (`ui/globals.xml`, `translation_tag` in the face).

Until the client's first push, the face shows the old card's made-up numbers, as the card did: eight samples, one a second (`<samples>` in `face.xml`). The first push replaces them for good; the device keeps the pushed values.

To make it your own, change the id (for example to `yourname.aida64`) in `plugin.json` and in `AITBASE_PLUGIN_ID` in `src/main.c`, and point your client's token at it.
