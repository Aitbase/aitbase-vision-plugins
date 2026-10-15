add_executable(counter src/main.c)
target_link_libraries(counter PRIVATE aitbase::plugin)
set_target_properties(counter PROPERTIES C_STANDARD 99 C_STANDARD_REQUIRED YES)
target_compile_options(counter PRIVATE -Wall -Wextra -Werror)

# Produces counter_ui.json next to the binary; copy both to the device.
aitbase_plugin_add_ui(counter XML ${CMAKE_CURRENT_SOURCE_DIR}/ui/ui.xml OUTPUT COUNTER_UI)
