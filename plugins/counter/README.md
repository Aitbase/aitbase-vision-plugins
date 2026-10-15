# Counter

This is the smallest AITBASE VISION plugin. It shows **Count: N** and adds 1 each time you press **RIGHT**. A **LEFT long press** returns to the home screen.

| File | Purpose |
|---|---|
| `ui/ui.xml` | The screen, a bound title label, and a button with the `plugin.increment` action. |
| `src/main.c` | Opens the screen, then sets the title on `opened` and on each `action`. |
| `plugin.json` | Manifest draft. The fields will change once packaging lands. |

## Run on a development device

The device must run firmware built with `CC_ENABLE_PLUGIN_UI=ON` and `CC_ENABLE_PLUGIN_DEV_BRIDGE=ON`.

1. Copy `counter` and `counter_ui.json` from your build directory (`build-arm/plugins/counter/`) to the device.
2. With the device on its home screen, run:

```sh
./counter ./counter_ui.json
```

## What you should see

1. The screen shows `Count: 0`.
2. Each RIGHT press adds 1 to the count.
3. A LEFT long press, or the voice assistant starting, closes the screen. The plugin then prints `screen preempted` and exits.
4. Ctrl+C on the plugin closes the screen.
