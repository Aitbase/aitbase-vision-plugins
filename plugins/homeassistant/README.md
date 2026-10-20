# Home Assistant

Controls a [Home Assistant](https://www.home-assistant.io/) (HA) on your network by voice, and four favorite devices from a screen.

Needs:
- device firmware with D37, so plugins reach the local network over HTTP;
- device firmware with D38 (plugin settings) and D36 (switches);
- the SDK after 0.11.1.

## Set up

1. **In Home Assistant:**
   1. Create a user for the device, for example `aitbase`.
   2. Sign in as that user.
   3. On the user's **Profile** page, under **Security**, create a **Long-lived access token**.
   4. Copy the token; HA shows it only once.

   Use a dedicated user: the token is kept on the device's SD card, and anyone with the card can read it.
2. **On the SD card:** copy `example.homeassistant-<version>.avp` into `aitbase/plugins/`.
3. **On the Device Console,** under **Plugins**, **Settings: Home Assistant**, fill in:
   - **Home Assistant address:** for example `http://192.168.1.10:8123`. Use the IP address: the device may not resolve `homeassistant.local`.
   - **Long-lived access token:** saving it asks for the Device Console password. It is never shown again.
   - **Favorites:** up to four entity ids, separated by commas, for example `light.kitchen, switch.fan, cover.garage, lock.front_door`.

## Voice

Ask the device's assistant. It calls the plugin's tools, and the plugin answers within 5 seconds:

| Say | Tool | What happens |
| --- | --- | --- |
| "Turn on the kitchen light", "Toggle the fan" | `control` | `homeassistant.turn_on`, `turn_off` or `toggle`; for a lock, `lock.lock` or `lock.unlock` |
| "Set the bedroom light to 30 percent" | `set_level` | brightness for a light, position for a cover, speed for a fan, volume for a media player |
| "Is the front door locked?", "What's the living room temperature?" | `state` | the entity's state, with its unit |
| "Run movie time" | `scene` | `scene.turn_on` or `script.turn_on` |

**How names are found:**
- The plugin asks HA to render a template (`POST /api/template`) that finds the entity by its friendly name or entity id: an exact match first, then a name that contains the words.
- The spoken name is passed as a template variable, never as template text.
- Answers stay small, under the device's 16 KiB receive limit, instead of `GET /api/states`.

## Screen

"Open Home Assistant", the home screen shortcut, or copying the package to the card opens the favorites screen:
- four rows, each with the name and a switch;
- LEFT moves between the switches, RIGHT switches one;
- the states refresh every 10 seconds;
- a LEFT long press leaves.

## Limits

- **One request at a time:** a second voice request while one is running gets "I'm still busy".
- **Four favorites** on the screen. The decision record planned up to 8; four fit the 240×320 screen with readable rows.
- **No brightness on the screen** yet; ask by voice.
