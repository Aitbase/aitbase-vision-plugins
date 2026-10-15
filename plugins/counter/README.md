# Counter

This is the smallest AITBASE VISION plugin. It shows **Count: N** and adds 1 each time you press **RIGHT**. A **LEFT long press** returns to the home screen.

| File | Purpose |
|---|---|
| `ui/screens/counter.xml` | The screen: a `card` component, an image, and a button with the `plugin.increment` action. |
| `ui/components/card.xml` | A component with a `title` prop and a label bound to the `title` subject. |
| `ui/globals.xml` | Declares the `logo` image. |
| `src/main.c` | Callbacks: opens the screen on start, then sets the title on `opened` and on each `action`. |
| `plugin.json` | Manifest draft. The fields will change once packaging lands. |

## Run on a development device

The device must run firmware built with `CC_ENABLE_PLUGIN_UI=ON` and `CC_ENABLE_PLUGIN_DEV_BRIDGE=ON`.

1. Copy `counter` and `counter.ui.json` from your build directory (`build-arm/plugins/counter/`) to the same directory on the device.
2. With the device on its home screen, run:

```sh
./counter
```

The runtime loads `counter.ui.json` from the binary's directory. Set `AITBASE_PLUGIN_UI_DIR` to load it from somewhere else.

## What you should see

1. The screen shows `Count: 0`.
2. Each RIGHT press adds 1 to the count.
3. A LEFT long press, or the voice assistant starting, closes the screen. The plugin then prints `screen preempted` and exits.
4. Ctrl+C on the plugin closes the screen.

The plugin logs to stderr, for example `[aitbase.counter] info: increment -> 1`.
