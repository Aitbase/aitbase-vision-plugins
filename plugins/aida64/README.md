# AIDA64

A home screen face with a PC's RAM, CPU and disk: bars, a CPU arc and labels, drawn by the device. A client on the PC pushes the numbers over the local network. The plugin has no code to speak of: the device shows the face without running it.

It shows what an idle face can do with the display widgets of device decision D36 (`lv_bar`, `lv_arc`, flex rows, any color), fed by a face push token (D35).

| File | Purpose |
|---|---|
| `ui/screens/face.xml` | The face: three cards bound to `face.1` to `face.13`. 31 widgets, 5.3 KB compiled. |
| `src/main.c` | Stops at once if started: everything is in the face. |
| `plugin.json` | `idle_face: "face"`. |

## Needs

- Device firmware with D35 (face cycle, face push tokens) and D36 (display widgets). Older firmware refuses the face, and the home screen keeps its clock.
- The SDK at `SDK_VERSION` (D36's compiler is in the SDK after 0.11.1).

## Install

```sh
aitbase-plugin build --wasm plugins/aida64
aitbase-plugin pack --wasm plugins/aida64 --out dist
```

1. Copy `dist/example.aida64-<version>.avp` into `aitbase/plugins/` on the device's SD card.
2. On the Device Console, open **Display**, add **AIDA64** to the face cycle, and create a push token for it.
3. Run the AIDA64 client on the PC with that token. The face fills in at the next push.

The face shows the device's **Plugin** badge, like every face from the SD card.

## The face's values

A client writes these keys with the push token (`POST /api/v1/plugins/face-values`). Each is text, at most 64 bytes. A bar or arc takes the leading integer of its key (`"45%"` is 45).

| Key | Value | Example | Drawn as |
|---|---|---|---|
| `face.1` | RAM speed | `DDR5-5600` | label |
| `face.2` | RAM used | `14.2GB / 32GB` | label |
| `face.3` | RAM free | `17.8GB` | (not shown; kept for clients) |
| `face.4` | RAM load | `45%` | label and bar |
| `face.5` | CPU name | `Ryzen 7 7800X3D` | label |
| `face.6` | CPU load | `12%` | label and arc |
| `face.7` | CPU temperature | `62C` | label and bar (0–100 °C) |
| `face.8` | CPU clock | `4850 MHz` | label |
| `face.9` | CPU fan | `1200 RPM` | label |
| `face.10` | Disk name | `Samsung 990 Pro` | label |
| `face.11` | Disk usage | `1.2TB/2.0TB` | label |
| `face.12` | Disk temperature | `41C` | label |
| `face.13` | Disk used, percent | `60` | bar |

`face.1` to `face.12` are the 12 metrics of the AIDA64 push client, in its order. `face.13` is the disk bar's percentage, which the client computes from the usage.

To make it your own, change the id (for example to `yourname.aida64`) in `plugin.json` and in `AITBASE_PLUGIN_ID` in `src/main.c`, and point your client's token at it.
