cmake_minimum_required(VERSION 3.13)
project(aitbase_vision_plugins LANGUAGES C)

if(NOT EXISTS ${CMAKE_CURRENT_SOURCE_DIR}/sdk/CMakeLists.txt)
  message(FATAL_ERROR "SDK submodule missing; run: git submodule update --init")
endif()
add_subdirectory(sdk)

file(GLOB plugin_dirs LIST_DIRECTORIES true ${CMAKE_CURRENT_SOURCE_DIR}/plugins/*)
foreach(dir ${plugin_dirs})
  if(EXISTS ${dir}/CMakeLists.txt)
    add_subdirectory(${dir})
  endif()
endforeach()
