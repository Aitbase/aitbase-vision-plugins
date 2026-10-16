# Builds inside the repository (top-level CMakeLists.txt) or on its own, which is what
# `aitbase-plugin build --wasm plugins/counter` does: then it uses the repository's sdk/.
cmake_minimum_required(VERSION 3.13)
if(CMAKE_SOURCE_DIR STREQUAL CMAKE_CURRENT_SOURCE_DIR)
  project(counter LANGUAGES C)
  add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR}/../../sdk ${CMAKE_BINARY_DIR}/sdk)
endif()

add_executable(counter src/main.c)
target_link_libraries(counter PRIVATE aitbase::plugin)
set_target_properties(counter PROPERTIES C_STANDARD 99 C_STANDARD_REQUIRED YES)
target_compile_options(counter PRIVATE -Wall -Wextra -Werror)

# ui/screens/counter.xml -> counter.ui.json next to the binary; copy both to the device.
aitbase_plugin_add_ui(counter SCREENS counter)
