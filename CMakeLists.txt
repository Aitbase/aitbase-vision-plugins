cmake_minimum_required(VERSION 3.13)
project(aitbase_vision_plugins LANGUAGES C)

# The SDK comes with the aitbase-plugin CLI: cmake -S . -B build -DAITBASE_SDK_DIR="$(aitbase-plugin sdk)"
if(NOT AITBASE_SDK_DIR)
  message(FATAL_ERROR "AITBASE_SDK_DIR is not set; run: cmake -S . -B build -DAITBASE_SDK_DIR=\"$(aitbase-plugin sdk)\"")
endif()
add_subdirectory(${AITBASE_SDK_DIR} ${CMAKE_BINARY_DIR}/aitbase-sdk)

file(GLOB plugin_dirs LIST_DIRECTORIES true ${CMAKE_CURRENT_SOURCE_DIR}/plugins/*)
foreach(dir ${plugin_dirs})
  if(EXISTS ${dir}/CMakeLists.txt)
    add_subdirectory(${dir})
  endif()
endforeach()
