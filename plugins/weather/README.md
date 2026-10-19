# Weather

Answers the device's voice assistant, with no screen of its own (assistant tools, D31). `plugin.json` declares one tool, `today`, with a `city` argument. When you ask "what's the weather in Hanoi?", the assistant calls it through the device's `plugins.call`. The device starts the plugin if it isn't running, and the plugin replies within 5 s with one line from [wttr.in](https://wttr.in), such as `Hanoi: ☀️ +31°C`.

| File | Purpose |
|---|---|
| `plugin.json` | Manifest: `assistant_tools` declares `today {city: string, at most 40 bytes}`; capabilities `http` and `assistant.tools`. |
| `src/main.c` | Callbacks: on `AITBASE_EVENT_TOOL_CALL` it reads `city` with `aitbase_tool_arg_string`, requests `https://wttr.in/<city>?format=3`, and answers with `aitbase_tool_reply` when the `RESULT` (or an `ERROR`) arrives. Started any other way, it stops at once. |

Needs SDK 0.10.0 or later, and device firmware with assistant tools (D31). Older firmware ignores `assistant_tools`, so the plugin never gets called.

## Run on a device

From the repository root (see the top-level README to install the CLI):

```sh
aitbase-plugin build --wasm plugins/weather
aitbase-plugin pack --wasm plugins/weather --out dist
```

Copy `dist/example.weather-<version>.avp` into `aitbase/plugins/` on the device's SD card, or, with a paired device, run `aitbase-plugin dev plugins/weather`. Its log is `aitbase/logs/example.weather.log` on the card, or `aitbase-plugin logs plugins/weather`.

## What you should see

1. Right after the copy, the device starts the plugin once (`autorun`), and it stops at once: there is nothing to show.
2. The Device Console's Plugins page lists it with "Answers the assistant: today".
3. Ask the assistant "what's the weather in Hanoi?" (or "ask the weather plugin about Hanoi"). It answers with the plugin's line. The log shows the run started with trigger `assistant`, and it ends about 2 s after the reply.
4. The device needs internet access. If wttr.in doesn't answer within the 5 s the device allows, the assistant says the plugin didn't answer in time.

To make it your own, change the id, for example to `yourname.weather`, in `plugin.json` and in `AITBASE_PLUGIN_ID` in `src/main.c`.
