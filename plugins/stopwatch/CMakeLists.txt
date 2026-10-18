# Builds inside the repository (top-level CMakeLists.txt) or on its own, which is what
# `aitbase-plugin build --wasm plugins/stopwatch` does: then it uses the SDK the CLI passes in
# AITBASE_SDK_DIR.
cmake_minimum_required(VERSION 3.13)
if(CMAKE_SOURCE_DIR STREQUAL CMAKE_CURRENT_SOURCE_DIR)
  project(stopwatch LANGUAGES C)
  if(NOT AITBASE_SDK_DIR)
    message(FATAL_ERROR "AITBASE_SDK_DIR is not set; build with: aitbase-plugin build plugins/stopwatch")
  endif()
  add_subdirectory(${AITBASE_SDK_DIR} ${CMAKE_BINARY_DIR}/aitbase-sdk)
endif()

add_executable(stopwatch src/main.c)
target_link_libraries(stopwatch PRIVATE aitbase::plugin)
set_target_properties(stopwatch PROPERTIES C_STANDARD 99 C_STANDARD_REQUIRED YES)
target_compile_options(stopwatch PRIVATE -Wall -Wextra -Werror)

# ui/screens/stopwatch.xml -> stopwatch.ui.json next to the binary.
aitbase_plugin_add_ui(stopwatch SCREENS stopwatch)
