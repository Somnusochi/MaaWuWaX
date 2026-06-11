cmake_minimum_required(VERSION 3.28)

project(maawuwax-cpp-agent)

include(MaaUtils/MaaUtils.cmake)
add_library(HeaderOnlyLibraries INTERFACE)

# DEPS_DIR: Set via -DDEPS_DIR=/path/to/MaaFramework/install or defaults to ../../deps
if(NOT DEFINED DEPS_DIR)
    set(DEPS_DIR ${CMAKE_CURRENT_SOURCE_DIR}/../../deps)
endif()

if(NOT EXISTS ${DEPS_DIR})
    message(FATAL_ERROR "Dependencies directory not found: ${DEPS_DIR}. "
        "Build MaaFramework first or set -DDEPS_DIR=/path/to/MaaFramework/install")
endif()

# MaaFramework deps ship RelWithDebInfo exports on macOS, so map all configs there.
set(CMAKE_MAP_IMPORTED_CONFIG_RELWITHDEBINFO RelWithDebInfo)
set(CMAKE_MAP_IMPORTED_CONFIG_MINSIZEREL RelWithDebInfo)
set(CMAKE_MAP_IMPORTED_CONFIG_RELEASE RelWithDebInfo)
set(CMAKE_MAP_IMPORTED_CONFIG_DEBUG RelWithDebInfo)

set(MaaFramework_DIR ${DEPS_DIR}/share/cmake/MaaFramework)
find_package(MaaFramework REQUIRED)

set(MaaAgentServer_DIR ${DEPS_DIR}/share/cmake/MaaAgentServer)
find_package(MaaAgentServer REQUIRED)

# WebView2 SDK configuration (Windows-only; on Windows this include may fail
# configuration if the SDK is unavailable or WEBVIEW2_SDK_DIR is not set correctly)
include(${CMAKE_CURRENT_SOURCE_DIR}/cmake/WebView2.cmake)

add_subdirectory(source)
