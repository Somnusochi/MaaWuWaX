from pathlib import Path

import shutil
import sys

try:
    import jsonc
except ModuleNotFoundError as e:
    raise ImportError(
        "Missing dependency 'json-with-comments' (imported as 'jsonc').\n"
        f"Install it with:\n  {sys.executable} -m pip install json-with-comments\n"
        "Or add it to your project's requirements."
    ) from e

from configure import configure_ocr_model


working_dir = Path(__file__).parent.parent.resolve()
install_path = working_dir / Path("install")
version = len(sys.argv) > 1 and sys.argv[1] or "v0.0.1"

# the first parameter is self name
if sys.argv.__len__() < 4:
    print("Usage: python install.py <version> <os> <arch>")
    print("Example: python install.py v1.0.0 win x86_64")
    sys.exit(1)

os_name = sys.argv[2]
arch = sys.argv[3]


def sync_interface_agents():
    interface_path = install_path / "interface.json"
    with open(interface_path, "r", encoding="utf-8") as f:
        interface = jsonc.load(f)

    binary_candidates = {
        "agent/go-service": (
            install_path / "agent" / "go-service",
            install_path / "agent" / "go-service.exe",
        ),
        "agent/cpp-algo": (
            install_path / "agent" / "cpp-algo",
            install_path / "agent" / "cpp-algo.exe",
        ),
    }
    existing_agents = {
        agent.get("child_exec"): agent
        for agent in interface.get("agent", [])
        if agent.get("child_exec")
    }

    agents = []
    for child_exec in ("agent/go-service", "agent/cpp-algo"):
        if any(path.is_file() for path in binary_candidates[child_exec]):
            agent = existing_agents.get(
                child_exec,
                {
                    "child_exec": child_exec,
                    "child_args": [],
                },
            )
            agents.append(agent)

    interface["agent"] = agents

    with open(interface_path, "w", encoding="utf-8") as f:
        jsonc.dump(interface, f, ensure_ascii=False, indent=4)


def remove_path(path: Path):
    if path.is_symlink() or path.is_file():
        path.unlink(missing_ok=True)
    elif path.is_dir():
        shutil.rmtree(path)


def get_dotnet_platform_tag():
    """自动检测当前平台并返回对应的dotnet平台标签"""
    if os_name == "win" and arch == "x86_64":
        platform_tag = "win-x64"
    elif os_name == "win" and arch == "aarch64":
        platform_tag = "win-arm64"
    elif os_name == "macos" and arch == "x86_64":
        platform_tag = "osx-x64"
    elif os_name == "macos" and arch == "aarch64":
        platform_tag = "osx-arm64"
    elif os_name == "linux" and arch == "x86_64":
        platform_tag = "linux-x64"
    elif os_name == "linux" and arch == "aarch64":
        platform_tag = "linux-arm64"
    else:
        print("Unsupported OS or architecture.")
        print("available parameters:")
        print("version: e.g., v1.0.0")
        print("os: [win, macos, linux, android]")
        print("arch: [aarch64, x86_64]")
        sys.exit(1)

    return platform_tag


def install_deps():
    if not (working_dir / "deps" / "bin").exists():
        print('Please download the MaaFramework to "deps" first.')
        print('请先下载 MaaFramework 到 "deps"。')
        sys.exit(1)

    if os_name == "android":
        shutil.copytree(
            working_dir / "deps" / "bin",
            install_path,
            dirs_exist_ok=True,
        )
        shutil.copytree(
            working_dir / "deps" / "share" / "MaaAgentBinary",
            install_path / "MaaAgentBinary",
            dirs_exist_ok=True,
        )
    else:
        shutil.copytree(
            working_dir / "deps" / "bin",
            install_path / "runtimes" / get_dotnet_platform_tag() / "native",
            ignore=shutil.ignore_patterns(
                "*MaaDbgControlUnit*",
                "*MaaThriftControlUnit*",
                "*MaaRpc*",
                "*MaaHttp*",
                "plugins",
                "*.node",
                "*MaaPiCli*",
            ),
            dirs_exist_ok=True,
        )
        shutil.copytree(
            working_dir / "deps" / "share" / "MaaAgentBinary",
            install_path / "libs" / "MaaAgentBinary",
            dirs_exist_ok=True,
        )
        shutil.copytree(
            working_dir / "deps" / "bin" / "plugins",
            install_path / "plugins" / get_dotnet_platform_tag(),
            dirs_exist_ok=True,
        )



def install_resource():

    configure_ocr_model()

    resource_path = install_path / "resource"
    interface_path = install_path / "interface.json"
    if resource_path.is_symlink():
        resource_path.unlink()
    if interface_path.is_symlink():
        interface_path.unlink()

    shutil.copytree(
        working_dir / "assets" / "resource",
        resource_path,
        dirs_exist_ok=True,
    )
    shutil.copy2(
        working_dir / "assets" / "interface.json",
        interface_path,
    )

    with open(interface_path, "r", encoding="utf-8") as f:
        interface = jsonc.load(f)

    interface["version"] = version

    with open(interface_path, "w", encoding="utf-8") as f:
        jsonc.dump(interface, f, ensure_ascii=False, indent=4)


def install_chores():
    for name in ("README.md", "LICENSE", "MaaWuWaX.icns"):
        remove_path(install_path / name)

    shutil.copy2(
        working_dir / "README.md",
        install_path,
    )
    shutil.copy2(
        working_dir / "LICENSE",
        install_path,
    )
    icon_path = working_dir / "assets" / "icon" / "MaaWuWaX.icns"
    if icon_path.is_file():
        shutil.copy2(
            icon_path,
            install_path / "MaaWuWaX.icns",
        )


def copy_into_app_bundle(source: Path, destination: Path):
    if source.name == "MaaWuWaX.app":
        return

    if source.is_dir():
        shutil.copytree(source, destination, dirs_exist_ok=True)
    else:
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, destination)


def create_macos_app_bundle():
    if os_name != "macos":
        return

    app_dir = install_path / "MaaWuWaX.app"
    contents_dir = app_dir / "Contents"
    macos_dir = contents_dir / "MacOS"
    resources_dir = contents_dir / "Resources"

    if app_dir.exists():
        shutil.rmtree(app_dir)

    macos_dir.mkdir(parents=True, exist_ok=True)
    resources_dir.mkdir(parents=True, exist_ok=True)

    launcher_path = macos_dir / "MaaWuWaX"
    launcher_path.write_text(
        """#!/bin/sh
set -eu
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
cd "${SCRIPT_DIR}"
exec "${SCRIPT_DIR}/mxu" "$@"
""",
        encoding="utf-8",
    )
    launcher_path.chmod(0o755)

    info_plist = contents_dir / "Info.plist"
    info_plist.write_text(
        """<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "https://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>zh_CN</string>
    <key>CFBundleDisplayName</key>
    <string>MaaWuWaX</string>
    <key>CFBundleExecutable</key>
    <string>MaaWuWaX</string>
    <key>CFBundleIconFile</key>
    <string>MaaWuWaX.icns</string>
    <key>CFBundleIdentifier</key>
    <string>com.maawuwax.mxu</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>MaaWuWaX</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>"""
        + version
        + """</string>
    <key>CFBundleVersion</key>
    <string>"""
        + version
        + """</string>
    <key>LSMinimumSystemVersion</key>
    <string>12.0</string>
</dict>
</plist>
""",
        encoding="utf-8",
    )

    icon_source = working_dir / "assets" / "icon" / "MaaWuWaX.icns"
    if icon_source.is_file():
        shutil.copy2(icon_source, resources_dir / "MaaWuWaX.icns")

    for item in install_path.iterdir():
        if item == app_dir:
            continue
        copy_into_app_bundle(item, macos_dir / item.name)


def install_agent():
    agent_dir = install_path / "agent"
    agent_dir.mkdir(parents=True, exist_ok=True)

    nested_source_dirs = (
        agent_dir / "go-service",
        agent_dir / "cpp-algo",
    )
    for path in nested_source_dirs:
        if path.is_dir():
            shutil.rmtree(path)

    expected_binaries = {
        "android": (),
        "win": ("go-service.exe",),
        "macos": ("go-service",),
        "linux": ("go-service",),
    }
    missing = [name for name in expected_binaries.get(os_name, ()) if not (agent_dir / name).is_file()]
    if missing:
        print(f"Missing built agent binaries: {', '.join(missing)}")
        sys.exit(1)

    # cpp-algo is optional during development, but when present it should be a
    # direct binary artifact rather than a copied source directory.
    for nested_source_dir in ("cpp-algo", "go-service"):
        nested = agent_dir / nested_source_dir
        if nested.is_dir():
            shutil.rmtree(nested)


if __name__ == "__main__":
    install_deps()
    install_resource()
    install_chores()
    install_agent()
    create_macos_app_bundle()
    sync_interface_agents()

    print(f"Install to {install_path} successfully.")
