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

# Map all build configs to Release (matching MaaFramework's installed config)
set(CMAKE_MAP_IMPORTED_CONFIG_RELWITHDEBINFO Release)
set(CMAKE_MAP_IMPORTED_CONFIG_MINSIZEREL Release)
set(CMAKE_MAP_IMPORTED_CONFIG_RELEASE Release)
set(CMAKE_MAP_IMPORTED_CONFIG_DEBUG Release)

set(MaaFramework_DIR ${DEPS_DIR}/share/cmake/MaaFramework)
find_package(MaaFramework REQUIRED)

set(MaaAgentServer_DIR ${DEPS_DIR}/share/cmake/MaaAgentServer)
find_package(MaaAgentServer REQUIRED)

# WebView2 SDK configuration (Windows-only; on Windows this include may fail
# configuration if the SDK is unavailable or WEBVIEW2_SDK_DIR is not set correctly)
include(${CMAKE_CURRENT_SOURCE_DIR}/cmake/WebView2.cmake)

add_subdirectory(source)
