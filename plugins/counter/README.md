# Counter

This is the smallest AITBASE VISION plugin. It shows **Count: N** and adds 1 each time you press **RIGHT**. A **LEFT long press** returns to the home screen.

| File | Purpose |
|---|---|
| `ui/screens/counter.xml` | The screen: a `card` component, an image, and a button with the `plugin.increment` action. |
| `ui/components/card.xml` | A component with a `title` prop and a label bound to the `title` subject. |
| `ui/globals.xml` | Declares the `logo` image. |
| `src/main.c` | Callbacks: opens the screen on start, then sets the title on `opened` and on each `action`. |
| `plugin.json` | Manifest draft. The fields will change once packaging lands. |

## Run on a device

From the repository root (see the top-level README to install the CLI):

```sh
aitbase-plugin build --wasm plugins/counter
aitbase-plugin pack --wasm plugins/counter --out dist
```

Copy `dist/example.counter-<version>.avp` into `aitbase/plugins/` on the device's SD card. The device checks the signature and starts the plugin after the card mounts. Its log is `aitbase/logs/example.counter.log` on the card.

To make it your own, change the id, for example to `yourname.counter`, in `plugin.json` and in `AITBASE_PLUGIN_ID` in `src/main.c`. Devices refuse `aitbase.*` ids on the SD card: that namespace is for official plugins, which ship in the firmware.

## What you should see

1. The screen shows `Count: 0`.
2. Each RIGHT press adds 1 to the count.
3. A LEFT long press, or the voice assistant starting, closes the screen. The plugin then logs `screen preempted` and exits.

The plugin's log lines look like `[example.counter] info: increment -> 1`.
