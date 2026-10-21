/*
 * AIDA64: a home screen face with a PC's RAM, CPU and disk.
 *
 * The face (ui/screens/face.xml) is all there is. The device draws it without running this
 * plugin, and a client on the PC writes its values (face.1 to face.16) with a face push token
 * the owner creates on the Device Console. So when something starts the plugin, it has nothing
 * to do and stops at once.
 */
#include "aitbase_plugin.h"

AITBASE_PLUGIN_ID("example.aida64");

int aitbase_on_start(aitbase_plugin_t *plugin) {
    aitbase_log(plugin, AITBASE_LOG_INFO, "a face only: the AIDA64 client fills it in");
    aitbase_plugin_stop(plugin);
    return 0;
}

void aitbase_on_event(aitbase_plugin_t *plugin, const aitbase_event_t *event) {
    (void)plugin;
    (void)event;
}
