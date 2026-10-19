# Builds inside the repository (top-level CMakeLists.txt) or on its own, which is what
# `aitbase-plugin build --wasm plugins/weather` does: then it uses the SDK the CLI passes in
# AITBASE_SDK_DIR.
cmake_minimum_required(VERSION 3.13)
if(CMAKE_SOURCE_DIR STREQUAL CMAKE_CURRENT_SOURCE_DIR)
  project(weather LANGUAGES C)
  if(NOT AITBASE_SDK_DIR)
    message(FATAL_ERROR "AITBASE_SDK_DIR is not set; build with: aitbase-plugin build plugins/weather")
  endif()
  add_subdirectory(${AITBASE_SDK_DIR} ${CMAKE_BINARY_DIR}/aitbase-sdk)
endif()

# No screen: the plugin answers the voice assistant (assistant_tools in plugin.json).
add_executable(weather src/main.c)
target_link_libraries(weather PRIVATE aitbase::plugin)
set_target_properties(weather PROPERTIES C_STANDARD 99 C_STANDARD_REQUIRED YES)
target_compile_options(weather PRIVATE -Wall -Wextra -Werror)
