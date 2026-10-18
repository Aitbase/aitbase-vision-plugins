/*
 * Stopwatch: a screen without buttons, driven by the device's buttons (AITBASE_EVENT_KEY, SDK 0.8).
 *
 * RIGHT press starts and stops. RIGHT long press resets to zero. LEFT press records a lap while
 * running. A LEFT long press leaves the plugin, as on every plugin screen; the plugin never sees it.
 *
 * A long press always arrives after a press of the same button: the device can't know at
 * press-down that the user will hold it. So holding RIGHT first starts or stops, then resets,
 * which leaves the stopwatch at zero and stopped either way.
 */
#define _POSIX_C_SOURCE 199309L /* clock_gettime */
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#include <time.h>

#include "aitbase_plugin.h"

AITBASE_PLUGIN_ID("example.stopwatch");

/* Names from ui/screens/stopwatch.xml. */
#define SCREEN "stopwatch"
#define SUBJECT "title"
#define TIME_NODE "time"
#define STATE_NODE "state"
#define LAP_NODE "lap"
#define REFRESH_MS 100

static int g_running = 0;
static uint64_t g_started_ms = 0;  /* when the current run started */
static uint64_t g_banked_ms = 0;   /* time of the runs before it */
static int g_laps = 0;

static uint64_t now_ms(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (uint64_t)ts.tv_sec * 1000u + (uint64_t)ts.tv_nsec / 1000000u;
}

static uint64_t elapsed_ms(void) {
    return g_banked_ms + (g_running ? now_ms() - g_started_ms : 0);
}

/* "mm:ss.t", or "h:mm:ss.t" from one hour. */
static void format_time(uint64_t ms, char *out, size_t cap) {
    const unsigned tenths = (unsigned)(ms / 100 % 10);
    const unsigned seconds = (unsigned)(ms / 1000 % 60);
    const unsigned minutes = (unsigned)(ms / 60000 % 60);
    const unsigned hours = (unsigned)(ms / 3600000);
    if (hours > 0) {
        snprintf(out, cap, "%u:%02u:%02u.%u", hours, minutes, seconds, tenths);
    } else {
        snprintf(out, cap, "%02u:%02u.%u", minutes, seconds, tenths);
    }
}

static void show_time(aitbase_plugin_t *plugin) {
    char text[24];
    format_time(elapsed_ms(), text, sizeof(text));
    aitbase_ui_set_text(plugin, TIME_NODE, SUBJECT, text);
}

static void show_state(aitbase_plugin_t *plugin) {
    aitbase_ui_set_text(plugin, STATE_NODE, SUBJECT, g_running ? "Running" : "Stopped");
}

static void start_stop(aitbase_plugin_t *plugin) {
    if (g_running) {
        g_banked_ms = elapsed_ms();
        g_running = 0;
        aitbase_timer_set(plugin, 0);
    } else {
        g_started_ms = now_ms();
        g_running = 1;
        aitbase_timer_set(plugin, REFRESH_MS);
    }
    show_time(plugin);
    show_state(plugin);
}

static void reset(aitbase_plugin_t *plugin) {
    g_running = 0;
    g_banked_ms = 0;
    g_laps = 0;
    aitbase_timer_set(plugin, 0);
    show_time(plugin);
    show_state(plugin);
    aitbase_ui_set_text(plugin, LAP_NODE, SUBJECT, "");
}

static void lap(aitbase_plugin_t *plugin) {
    if (!g_running) return;
    char time_text[24], text[48];
    format_time(elapsed_ms(), time_text, sizeof(time_text));
    snprintf(text, sizeof(text), "Lap %d: %s", ++g_laps, time_text);
    aitbase_log(plugin, AITBASE_LOG_INFO, "%s", text);
    aitbase_ui_set_text(plugin, LAP_NODE, SUBJECT, text);
}

int aitbase_on_start(aitbase_plugin_t *plugin) {
    /* Wait up to 30 s if the device is busy, e.g. the voice assistant is talking. */
    return aitbase_ui_open(plugin, SCREEN, AITBASE_UI_MAX_WAIT_MS);
}

void aitbase_on_event(aitbase_plugin_t *plugin, const aitbase_event_t *event) {
    switch (event->kind) {
        case AITBASE_EVENT_OPENED:
            aitbase_log(plugin, AITBASE_LOG_INFO, "screen opened");
            show_time(plugin);
            show_state(plugin);
            break;
        case AITBASE_EVENT_KEY:
            if (event->key == AITBASE_KEY_RIGHT && event->gesture == AITBASE_GESTURE_PRESS) {
                start_stop(plugin);
            } else if (event->key == AITBASE_KEY_RIGHT && event->gesture == AITBASE_GESTURE_LONG_PRESS) {
                reset(plugin);
            } else if (event->key == AITBASE_KEY_LEFT && event->gesture == AITBASE_GESTURE_PRESS) {
                lap(plugin);
            }
            break;
        case AITBASE_EVENT_TIMER:
            if (g_running) show_time(plugin);
            break;
        case AITBASE_EVENT_ERROR:
            aitbase_log(plugin, AITBASE_LOG_ERROR, "%s failed (%d %s)", event->op, event->status, event->detail);
            if (strcmp(event->op, "open") == 0) aitbase_plugin_stop(plugin);
            break;
        case AITBASE_EVENT_REJECTED:
        case AITBASE_EVENT_CLOSED:
        case AITBASE_EVENT_PREEMPTED:
            aitbase_log(plugin, AITBASE_LOG_INFO, "screen %s %s", aitbase_event_kind_name(event->kind), event->detail);
            aitbase_plugin_stop(plugin);
            break;
        default:
            break;
    }
}
