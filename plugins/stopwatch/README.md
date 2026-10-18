# Stopwatch

A screen without buttons, driven by the device's buttons themselves (`AITBASE_EVENT_KEY`). It shows the elapsed time to a tenth of a second.

| Button | What it does |
|---|---|
| RIGHT press | Start or stop. |
| RIGHT long press | Reset to `00:00.0`, stopped. |
| LEFT press | Record a lap while running. |
| LEFT long press | Leave the plugin. The device handles it; the plugin never sees it. |

A long press always comes after a press of the same button, because the device can't know at press-down that you'll hold it. So holding RIGHT first starts or stops, then resets: you end at zero, stopped, either way.

| File | Purpose |
|---|---|
| `ui/screens/stopwatch.xml` | The screen: the time, the state and the last lap (labels bound to the `title` subject), and help text. No buttons, so nothing takes focus. |
| `src/main.c` | Callbacks: reacts to `AITBASE_EVENT_KEY`, refreshes the time every 100 ms with a timer while running, and measures with the monotonic clock. |
| `plugin.json` | Manifest: only the `ui` capability. |

Needs SDK 0.8.0 or later and device firmware with button events (D20). On older firmware the screen opens but never reacts.

## Run on a device

From the repository root (see the top-level README to install the CLI):

```sh
aitbase-plugin build --wasm plugins/stopwatch
aitbase-plugin pack --wasm plugins/stopwatch --out dist
```

Copy `dist/example.stopwatch-<version>.avp` into `aitbase/plugins/` on the device's SD card, or, with a paired device, run `aitbase-plugin dev plugins/stopwatch`. Its log is `aitbase/logs/example.stopwatch.log` on the card, or `aitbase-plugin logs plugins/stopwatch`.

To make it your own, change the id, for example to `yourname.stopwatch`, in `plugin.json` and in `AITBASE_PLUGIN_ID` in `src/main.c`.

## What you should see

1. The screen shows `00:00.0` and `Stopped`.
2. RIGHT: the time runs and the state says `Running`. RIGHT again stops it.
3. LEFT while running: `Lap 1: 00:04.2` appears, and the log has `[example.stopwatch] info: Lap 1: 00:04.2`.
4. Hold RIGHT: back to `00:00.0`, `Stopped`, no lap.
5. A LEFT long press, or the voice assistant starting, closes the screen. The plugin logs `screen preempted` and exits.

With a paired device, the same check without touching it:

```sh
aitbase-plugin press right && sleep 2 && aitbase-plugin screenshot -o running.png
aitbase-plugin press left --long   # leave
```
