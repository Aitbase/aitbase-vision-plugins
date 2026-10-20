/*
 * Home Assistant: controls a Home Assistant (HA) on the local network.
 *
 * - Voice: the assistant calls the tools in plugin.json ("turn on the kitchen light"). The plugin
 *   finds the entity by name with HA's template API, then calls a service, and answers within 5 s.
 * - Screen: up to four favorite entities as switches. LEFT moves between them, RIGHT toggles one
 *   (AITBASE_EVENT_CHANGED); the states refresh every 10 s.
 *
 * The owner fills in the address, a long-lived access token and the favorites on the Device
 * Console (plugin settings, device decision D38); they are in storage as "settings.<key>". Plugins
 * reach the local network over HTTP on devices with D37.
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "aitbase_plugin.h"

AITBASE_PLUGIN_ID("example.homeassistant"); /* must match "id" in plugin.json */

#define MAX_FAVORITES 4
#define HTTP_TIMEOUT_MS 2000 /* two requests must fit in the assistant's 5 s */
#define REFRESH_MS 10000

/* Settings (D38). */
static char g_url[1025];
static char g_token[1025];
static char g_favorites[MAX_FAVORITES][96];
static int g_favorite_count = 0;
static int g_loads[3];   /* storage.get request ids: url, token, favorites; 0 once answered */
static int g_loaded = 0; /* all three answered */

/* The screen. */
static int g_screen = 0;
static int g_state_on[MAX_FAVORITES];
static int g_toggle_index = -1; /* a switch change waiting for the HTTP request in flight */
static int g_toggle_on = 0;

/* One HTTP request at a time (the device allows one in flight). */
enum job { JOB_NONE, JOB_RESOLVE, JOB_ACT, JOB_REFRESH, JOB_TOGGLE };
static enum job g_job = JOB_NONE;
static int g_request = 0;

/* The assistant's call being answered. */
static uint32_t g_call = 0;
static char g_tool[33];
static char g_name[61];
static char g_action[8];
static long g_percent = 0;
static char g_entity[96];
static char g_entity_name[96];

/* --- JSON and HTTP helpers --------------------------------------------------------------------- */

/* Appends text as a JSON string (with quotes) to out; returns 0 when it doesn't fit. */
static int json_string(char *out, size_t cap, const char *text) {
    size_t n = strlen(out);
    if (n + 2 >= cap) return 0;
    out[n++] = '"';
    for (const unsigned char *c = (const unsigned char *)text; *c; ++c) {
        char esc[8] = {0};
        if (*c == '"' || *c == '\\') {
            esc[0] = '\\';
            esc[1] = (char)*c;
        } else if (*c == '\n') {
            strcpy(esc, "\\n");
        } else if (*c < 0x20) {
            snprintf(esc, sizeof(esc), "\\u%04x", *c);
        } else {
            esc[0] = (char)*c;
        }
        const size_t len = strlen(esc);
        if (n + len + 2 >= cap) return 0;
        memcpy(out + n, esc, len);
        n += len;
    }
    out[n++] = '"';
    out[n] = '\0';
    return 1;
}

static int append(char *out, size_t cap, const char *text) {
    const size_t n = strlen(out), len = strlen(text);
    if (n + len >= cap) return 0;
    memcpy(out + n, text, len + 1);
    return 1;
}

/* POSTs body to the HA API path (for example "/api/template"); the request id, or < 0. */
static int ha_post(aitbase_plugin_t *plugin, const char *path, const char *body) {
    char url[1100], auth[1100];
    size_t base = strlen(g_url);
    while (base > 0 && g_url[base - 1] == '/') --base; /* "http://ha:8123/" works too */
    if ((size_t)snprintf(url, sizeof(url), "%.*s%s", (int)base, g_url, path) >= sizeof(url)) return -1;
    snprintf(auth, sizeof(auth), "Authorization: Bearer %s", g_token);
    const char *headers[] = {auth, "Content-Type: application/json"};
    aitbase_http_request_t request = {"POST", url, headers, 2, body, HTTP_TIMEOUT_MS};
    return aitbase_http_request(plugin, &request);
}

/* --- Finding an entity by name ------------------------------------------------------------------ */

/* Renders "entity_id|name|state [unit]" of the first entity in domains d (any when empty) whose
 * name or id is q, else whose name contains q. The query goes in as a variable, never into the
 * template text, so a spoken name can't change the template. */
static const char kResolveTemplate[] =
    "{%- set ql = q | lower | trim -%}{%- set ns = namespace(hit='') -%}"
    "{%- for s in states if not ns.hit and (not d or s.domain in d) and (s.entity_id == ql or (s.name | lower) == ql) -%}"
    "{%- set ns.hit = s.entity_id ~ '|' ~ s.name ~ '|' ~ s.state ~ (' ' ~ s.attributes.unit_of_measurement if s.attributes.unit_of_measurement is defined else '') -%}"
    "{%- endfor -%}"
    "{%- for s in states if not ns.hit and (not d or s.domain in d) and ql in (s.name | lower) -%}"
    "{%- set ns.hit = s.entity_id ~ '|' ~ s.name ~ '|' ~ s.state ~ (' ' ~ s.attributes.unit_of_measurement if s.attributes.unit_of_measurement is defined else '') -%}"
    "{%- endfor -%}{{ ns.hit }}";

static const char *domains_for(const char *tool) {
    if (strcmp(tool, "control") == 0) {
        return "[\"light\",\"switch\",\"fan\",\"cover\",\"lock\",\"input_boolean\",\"media_player\",\"climate\","
               "\"automation\",\"siren\",\"humidifier\",\"vacuum\",\"valve\"]";
    }
    if (strcmp(tool, "set_level") == 0) return "[\"light\",\"cover\",\"fan\",\"media_player\"]";
    if (strcmp(tool, "scene") == 0) return "[\"scene\",\"script\"]";
    return "[]"; /* state: anything */
}

static void answer(aitbase_plugin_t *plugin, const char *text) {
    if (g_call) aitbase_tool_reply(plugin, g_call, text);
    g_call = 0;
}

static void start_refresh(aitbase_plugin_t *plugin);
static void run_waiting(aitbase_plugin_t *plugin);

static void start_resolve(aitbase_plugin_t *plugin) {
    if (!g_url[0] || !g_token[0]) {
        answer(plugin, "Home Assistant isn't set up yet. Add its address and token in the Device Console, under Plugins.");
        return;
    }
    char body[2048] = "{\"template\":";
    if (!json_string(body, sizeof(body), kResolveTemplate) || !append(body, sizeof(body), ",\"variables\":{\"q\":") ||
        !json_string(body, sizeof(body), g_name) || !append(body, sizeof(body), ",\"d\":") ||
        !append(body, sizeof(body), domains_for(g_tool)) || !append(body, sizeof(body), "}}")) {
        answer(plugin, "That name is too long.");
        return;
    }
    g_request = ha_post(plugin, "/api/template", body);
    if (g_request < 0) {
        answer(plugin, "Home Assistant can't be reached now.");
        return;
    }
    g_job = JOB_RESOLVE;
}

/* The service for the call, as "<domain>/<service>" and its JSON data; 0 when there is none. */
static int service_for(const char *domain, char *service, size_t service_cap, char *data, size_t data_cap) {
    char id[128] = "";
    json_string(id, sizeof(id), g_entity);
    if (strcmp(g_tool, "control") == 0) {
        if (strcmp(domain, "lock") == 0) {
            if (strcmp(g_action, "toggle") == 0) return 0;
            snprintf(service, service_cap, "lock/%s", strcmp(g_action, "on") == 0 ? "lock" : "unlock");
        } else {
            snprintf(service, service_cap, "homeassistant/%s",
                     strcmp(g_action, "on") == 0 ? "turn_on" : strcmp(g_action, "off") == 0 ? "turn_off" : "toggle");
        }
        snprintf(data, data_cap, "{\"entity_id\":%s}", id);
        return 1;
    }
    if (strcmp(g_tool, "set_level") == 0) {
        if (strcmp(domain, "light") == 0) {
            snprintf(service, service_cap, "light/turn_on");
            snprintf(data, data_cap, "{\"entity_id\":%s,\"brightness_pct\":%ld}", id, g_percent);
        } else if (strcmp(domain, "cover") == 0) {
            snprintf(service, service_cap, "cover/set_cover_position");
            snprintf(data, data_cap, "{\"entity_id\":%s,\"position\":%ld}", id, g_percent);
        } else if (strcmp(domain, "fan") == 0) {
            snprintf(service, service_cap, "fan/set_percentage");
            snprintf(data, data_cap, "{\"entity_id\":%s,\"percentage\":%ld}", id, g_percent);
        } else {
            snprintf(service, service_cap, "media_player/volume_set");
            snprintf(data, data_cap, "{\"entity_id\":%s,\"volume_level\":%ld.%02ld}", id, g_percent / 100, g_percent % 100);
        }
        return 1;
    }
    snprintf(service, service_cap, "homeassistant/turn_on"); /* scene and script */
    snprintf(data, data_cap, "{\"entity_id\":%s}", id);
    return 1;
}

static void on_resolved(aitbase_plugin_t *plugin, const char *text) {
    /* "entity_id|name|state": the name may hold '|', the entity id and the state don't. */
    const char *first = strchr(text, '|');
    const char *last = strrchr(text, '|');
    if (!first || first == last) {
        char reply[160];
        snprintf(reply, sizeof(reply), "I can't find %s in Home Assistant.", g_name);
        answer(plugin, reply);
        return;
    }
    snprintf(g_entity, sizeof(g_entity), "%.*s", (int)(first - text), text);
    snprintf(g_entity_name, sizeof(g_entity_name), "%.*s", (int)(last - first - 1), first + 1);
    if (strcmp(g_tool, "state") == 0) {
        char reply[256];
        snprintf(reply, sizeof(reply), "%s is %s.", g_entity_name, last + 1);
        answer(plugin, reply);
        return;
    }
    char domain[32], service[64], data[256], path[96];
    const char *dot = strchr(g_entity, '.');
    snprintf(domain, sizeof(domain), "%.*s", dot ? (int)(dot - g_entity) : 0, g_entity);
    if (!service_for(domain, service, sizeof(service), data, sizeof(data))) {
        answer(plugin, "Say lock or unlock for a lock.");
        return;
    }
    snprintf(path, sizeof(path), "/api/services/%s", service);
    g_request = ha_post(plugin, path, data);
    if (g_request < 0) {
        answer(plugin, "Home Assistant can't be reached now.");
        return;
    }
    g_job = JOB_ACT;
}

static void on_acted(aitbase_plugin_t *plugin, int ok) {
    char reply[200];
    if (!ok) {
        snprintf(reply, sizeof(reply), "Home Assistant refused that for %s.", g_entity_name);
    } else if (strcmp(g_tool, "set_level") == 0) {
        snprintf(reply, sizeof(reply), "%s is set to %ld percent.", g_entity_name, g_percent);
    } else if (strcmp(g_tool, "scene") == 0) {
        snprintf(reply, sizeof(reply), "Done: %s.", g_entity_name);
    } else {
        snprintf(reply, sizeof(reply), "%s: %s.", g_entity_name,
                 strcmp(g_action, "on") == 0 ? "on" : strcmp(g_action, "off") == 0 ? "off" : "switched");
    }
    answer(plugin, reply);
}

/* --- The favorites screen ------------------------------------------------------------------------ */

static void show_status(aitbase_plugin_t *plugin, const char *text) {
    if (g_screen) aitbase_ui_set_text(plugin, "status", "title", text);
}

static void show_row(aitbase_plugin_t *plugin, int i, const char *name, int on) {
    char node[24], key[24];
    snprintf(node, sizeof(node), "name%d", i + 1);
    snprintf(key, sizeof(key), "value.%d", i + 1);
    aitbase_ui_set_text(plugin, node, key, name);
    snprintf(node, sizeof(node), "on%d", i + 1);
    snprintf(key, sizeof(key), "value.%d", i + 5);
    aitbase_ui_set_text(plugin, node, key, on ? "1" : "0");
}

static void start_refresh(aitbase_plugin_t *plugin) {
    if (!g_screen || g_job != JOB_NONE) return;
    if (!g_url[0] || !g_token[0]) {
        show_status(plugin, "Set up in Device Console");
        return;
    }
    if (g_favorite_count == 0) {
        show_status(plugin, "Add favorites in Device Console");
        return;
    }
    char body[1024] = "{\"template\":";
    char ids[512] = "[";
    for (int i = 0; i < g_favorite_count; ++i) {
        if (i) append(ids, sizeof(ids), ",");
        json_string(ids, sizeof(ids), g_favorites[i]);
    }
    append(ids, sizeof(ids), "]");
    json_string(body, sizeof(body),
                "{%- for e in ids -%}{{ state_attr(e, 'friendly_name') or e }}|{{ states(e) }}{{ '\\n' }}{%- endfor -%}");
    append(body, sizeof(body), ",\"variables\":{\"ids\":");
    append(body, sizeof(body), ids);
    append(body, sizeof(body), "}}");
    g_request = ha_post(plugin, "/api/template", body);
    if (g_request > 0) g_job = JOB_REFRESH;
}

static void on_refreshed(aitbase_plugin_t *plugin, const char *text) {
    const char *line = text;
    for (int i = 0; i < g_favorite_count; ++i) {
        const char *end = strchr(line, '\n');
        const size_t len = end ? (size_t)(end - line) : strlen(line);
        char row[160];
        snprintf(row, sizeof(row), "%.*s", (int)len, line);
        char *bar = strrchr(row, '|');
        const char *state = bar ? bar + 1 : "";
        if (bar) *bar = '\0';
        g_state_on[i] = strcmp(state, "on") == 0 || strcmp(state, "open") == 0 || strcmp(state, "locked") == 0 ||
                        strcmp(state, "playing") == 0 || strcmp(state, "heat") == 0 || strcmp(state, "cool") == 0;
        show_row(plugin, i, row[0] ? row : g_favorites[i], g_state_on[i]);
        line = end ? end + 1 : line + len;
    }
    show_status(plugin, "Home Assistant");
}

static void start_toggle(aitbase_plugin_t *plugin) {
    const int i = g_toggle_index;
    g_toggle_index = -1;
    char data[160] = "{\"entity_id\":";
    json_string(data, sizeof(data), g_favorites[i]);
    append(data, sizeof(data), "}");
    g_request = ha_post(plugin, g_toggle_on ? "/api/services/homeassistant/turn_on" : "/api/services/homeassistant/turn_off", data);
    if (g_request > 0) g_job = JOB_TOGGLE;
}

/* After a request ends: an assistant call first, then a waiting switch, then the refresh. */
static void run_waiting(aitbase_plugin_t *plugin) {
    if (g_job != JOB_NONE || !g_loaded) return;
    if (g_call) {
        start_resolve(plugin);
        if (g_job != JOB_NONE) return;
    }
    if (g_toggle_index >= 0) start_toggle(plugin);
}

/* --- Settings ------------------------------------------------------------------------------------ */

static void load_settings(aitbase_plugin_t *plugin) {
    g_loaded = 0;
    g_url[0] = g_token[0] = '\0';
    g_favorite_count = 0;
    g_loads[0] = aitbase_storage_get(plugin, "settings.ha_url");
    g_loads[1] = aitbase_storage_get(plugin, "settings.ha_token");
    g_loads[2] = aitbase_storage_get(plugin, "settings.favorites");
}

/* "light.kitchen, switch.fan" into g_favorites. */
static void parse_favorites(const char *text) {
    g_favorite_count = 0;
    const char *p = text;
    while (*p && g_favorite_count < MAX_FAVORITES) {
        while (*p == ' ' || *p == ',') ++p;
        const char *start = p;
        while (*p && *p != ',' && *p != ' ') ++p;
        if (p > start && (size_t)(p - start) < sizeof(g_favorites[0])) {
            snprintf(g_favorites[g_favorite_count++], sizeof(g_favorites[0]), "%.*s", (int)(p - start), start);
        }
    }
}

/* The settings are read: draw the rows, answer a waiting call, refresh. */
static void settings_ready(aitbase_plugin_t *plugin) {
    for (int i = 0; g_screen && i < MAX_FAVORITES; ++i) show_row(plugin, i, i < g_favorite_count ? g_favorites[i] : "-", 0);
    run_waiting(plugin);
    start_refresh(plugin);
}

static void on_setting(aitbase_plugin_t *plugin, const aitbase_event_t *event) {
    const char *value = event->code == 1 ? event->text : "";
    for (int i = 0; i < 3; ++i) {
        if (g_loads[i] <= 0 || event->request != (uint32_t)g_loads[i]) continue;
        if (i == 0) snprintf(g_url, sizeof(g_url), "%s", value);
        if (i == 1) snprintf(g_token, sizeof(g_token), "%s", value);
        if (i == 2) parse_favorites(value);
        g_loads[i] = 0;
    }
    if (g_loads[0] || g_loads[1] || g_loads[2]) return;
    g_loaded = 1;
    settings_ready(plugin);
}

/* --- Callbacks ----------------------------------------------------------------------------------- */

int aitbase_on_start(aitbase_plugin_t *plugin) {
    load_settings(plugin);
    /* Started to answer the assistant: no screen. Otherwise (voice "open Home Assistant", the
     * shortcut, the SD card), the favorites. */
    if (strcmp(aitbase_trigger(plugin), "assistant") == 0) return 0;
    return aitbase_ui_open(plugin, "main", AITBASE_UI_MAX_WAIT_MS);
}

void aitbase_on_event(aitbase_plugin_t *plugin, const aitbase_event_t *event) {
    switch (event->kind) {
        case AITBASE_EVENT_OPENED:
            g_screen = 1;
            show_status(plugin, g_loaded ? "Home Assistant" : "Loading...");
            if (g_loaded) settings_ready(plugin);
            aitbase_timer_set(plugin, REFRESH_MS);
            break;
        case AITBASE_EVENT_TOOL_CALL: {
            if (g_call) {
                aitbase_tool_reply(plugin, event->request, "I'm still busy with the last request.");
                break;
            }
            long percent = 0;
            if (aitbase_tool_arg_string(event, "name", g_name, sizeof(g_name)) < 0) {
                aitbase_tool_reply(plugin, event->request, "Which device?");
                break;
            }
            snprintf(g_tool, sizeof(g_tool), "%s", event->action);
            g_action[0] = '\0';
            if (strcmp(g_tool, "control") == 0) aitbase_tool_arg_string(event, "action", g_action, sizeof(g_action));
            if (strcmp(g_tool, "set_level") == 0) aitbase_tool_arg_int(event, "percent", &percent);
            g_percent = percent;
            g_call = event->request;
            run_waiting(plugin); /* or later, once the settings are read or the request in flight ends */
            break;
        }
        case AITBASE_EVENT_CHANGED: {
            const int i = event->node[0] == 'o' && event->node[1] == 'n' ? atoi(event->node + 2) - 1 : -1;
            if (i < 0 || i >= MAX_FAVORITES) break;
            if (i >= g_favorite_count) {
                show_row(plugin, i, "-", 0); /* an empty row has nothing to switch */
                break;
            }
            g_toggle_index = i;
            g_toggle_on = event->code;
            run_waiting(plugin);
            break;
        }
        case AITBASE_EVENT_TIMER:
            start_refresh(plugin);
            break;
        case AITBASE_EVENT_RESULT:
            if (strcmp(event->op, "storage.get") == 0) {
                on_setting(plugin, event);
                break;
            }
            if (event->request != (uint32_t)g_request || g_job == JOB_NONE) break;
            {
                const enum job done = g_job;
                const int ok = event->code >= 200 && event->code < 300;
                g_job = JOB_NONE;
                if (done == JOB_RESOLVE) {
                    if (ok) on_resolved(plugin, event->text);
                    else answer(plugin, event->code == 401 ? "Home Assistant refused the token." : "Home Assistant can't be reached now.");
                } else if (done == JOB_ACT) {
                    on_acted(plugin, ok);
                } else if (done == JOB_REFRESH) {
                    if (ok) on_refreshed(plugin, event->text);
                    else show_status(plugin, event->code == 401 ? "Token refused" : "HA error");
                }
                run_waiting(plugin);
                if (done == JOB_ACT || done == JOB_TOGGLE) start_refresh(plugin);
            }
            break;
        case AITBASE_EVENT_ERROR:
            aitbase_log(plugin, AITBASE_LOG_WARN, "%s failed (%d %s)", event->op, event->status, event->detail);
            if (strcmp(event->op, "open") == 0) break; /* a voice answer still works without the screen */
            if (strcmp(event->op, "storage.get") == 0) {
                on_setting(plugin, event); /* no SD card: no settings */
                break;
            }
            if (g_job != JOB_NONE && event->request == (uint32_t)g_request) {
                const enum job failed = g_job;
                g_job = JOB_NONE;
                if (failed == JOB_RESOLVE || failed == JOB_ACT) answer(plugin, "Home Assistant can't be reached now.");
                show_status(plugin, "Can't reach HA");
                run_waiting(plugin);
            }
            break;
        case AITBASE_EVENT_DEVICE:
            if (strcmp(event->op, "plugin.settings_changed") == 0) load_settings(plugin);
            break;
        case AITBASE_EVENT_REJECTED:
        case AITBASE_EVENT_CLOSED:
        case AITBASE_EVENT_PREEMPTED:
            g_screen = 0;
            aitbase_timer_set(plugin, 0);
            if (!g_call) aitbase_plugin_stop(plugin);
            break;
        default:
            break;
    }
}
