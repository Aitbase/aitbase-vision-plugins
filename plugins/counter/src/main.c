/*
 * Counter: the smallest AITBASE VISION plugin.
 *
 * Shows "Count: N" on the plugin screen and increments N each time the user activates the button
 * (RIGHT press). Exits when the device closes or takes back the screen (for example a LEFT long
 * press), or on Ctrl+C.
 *
 * Usage: counter [counter_ui.json]
 */
#include <signal.h>
#include <stdio.h>
#include <string.h>

#include "aitbase_plugin.h"

/* Node and binding key declared in ui/ui.xml (<use component="card" id="status"/>). */
#define TITLE_NODE "status.title"
#define TITLE_KEY "title"
#define INCREMENT_ACTION "plugin.increment"

static volatile sig_atomic_t g_stop = 0;

static void on_signal(int sig) {
    (void)sig;
    g_stop = 1;
}

static int show_count(aitbase_plugin_t *plugin, int count) {
    char text[32];
    snprintf(text, sizeof(text), "Count: %d", count);
    return aitbase_ui_set_text(plugin, TITLE_NODE, TITLE_KEY, text);
}

int main(int argc, char **argv) {
    const char *ui_path = argc > 1 ? argv[1] : "counter_ui.json";
    signal(SIGINT, on_signal);
    signal(SIGTERM, on_signal);

    aitbase_plugin_t *plugin = aitbase_plugin_connect(NULL);
    if (!plugin) {
        fprintf(stderr, "counter: cannot reach the device plugin endpoint\n");
        return 1;
    }
    /* Wait up to 30 s if the device is busy (e.g. the voice assistant is talking). */
    int rc = aitbase_ui_open_file(plugin, ui_path, AITBASE_UI_MAX_WAIT_MS);
    if (rc != AITBASE_OK) {
        fprintf(stderr, "counter: cannot open %s (%d)\n", ui_path, rc);
        aitbase_plugin_disconnect(plugin);
        return 1;
    }

    int count = 0;
    int status = 0;
    int running = 1;
    while (running && !g_stop) {
        aitbase_event_t ev;
        rc = aitbase_plugin_next_event(plugin, &ev, 200);
        if (rc == 0) continue;
        if (rc < 0) {
            fprintf(stderr, "counter: lost the device (%d)\n", rc);
            status = 1;
            break;
        }
        switch (ev.kind) {
            case AITBASE_EVENT_OPENED:
                printf("counter: screen opened\n");
                rc = show_count(plugin, count);
                break;
            case AITBASE_EVENT_ACTION:
                if (strcmp(ev.action, INCREMENT_ACTION) == 0) {
                    printf("counter: %s -> %d\n", ev.node, ++count);
                    rc = show_count(plugin, count);
                }
                break;
            case AITBASE_EVENT_ERROR:
                fprintf(stderr, "counter: %s failed (%d %s)\n", ev.op, ev.status, ev.detail);
                if (strcmp(ev.op, "open") == 0) {
                    status = 1;
                    running = 0;
                }
                break;
            case AITBASE_EVENT_UPDATE_REJECTED:
                fprintf(stderr, "counter: update to %s rejected (%s)\n", ev.node, ev.detail);
                break;
            case AITBASE_EVENT_REJECTED:
            case AITBASE_EVENT_CLOSED:
            case AITBASE_EVENT_PREEMPTED:
                printf("counter: screen %s %s\n", aitbase_event_kind_name(ev.kind), ev.detail);
                running = 0;
                break;
            default:
                break;
        }
        if (rc < 0) {
            fprintf(stderr, "counter: lost the device (%d)\n", rc);
            status = 1;
            running = 0;
        }
        fflush(stdout);
    }
    if (g_stop) (void)aitbase_ui_close(plugin);
    aitbase_plugin_disconnect(plugin);
    return status;
}
