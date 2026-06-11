#!/usr/bin/env python3
"""MaaWuWaX workspace setup — downloads deps, builds agents, creates install/ directory.

Usage:
  python3 tools/setup_workspace.py                # full setup
  python3 tools/setup_workspace.py --skip-mxu     # skip MXU download
  python3 tools/setup_workspace.py --cpp-algo     # also build cpp-algo
"""

import argparse
import json
import os
import platform
import shutil
import subprocess
import sys
import tarfile
import tempfile
import urllib.request
import zipfile
from pathlib import Path

from configure import configure_ocr_model

PROJECT = Path(__file__).parent.parent.resolve()
MXU_REPO = "MistEO/MXU"
MFW_REPO = "MaaXYZ/MaaFramework"

OS_KEY = {"darwin": "macos", "windows": "win"}.get(platform.system().lower(), "linux")
_raw_arch = platform.machine()
ARCH_KEY = "aarch64" if _raw_arch == "arm64" else _raw_arch

MFW_OS_TAGS = {
    "macos": ("macos", "osx"),
    "win": ("win", "windows"),
    "linux": ("linux",),
}


def _http_get(url: str) -> dict:
    req = urllib.request.Request(url)
    req.add_header("Accept", "application/vnd.github+json")
    req.add_header("User-Agent", "MaaWuWaX-setup")
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read())


def _download(url: str, dest: Path):
    print(f"  downloading {url}")
    urllib.request.urlretrieve(url, dest)


def workspace_go_env() -> dict[str, str]:
    go_cache = PROJECT / ".cache" / "go-build"
    go_mod_cache = PROJECT / ".cache" / "go-mod"
    go_tmp = PROJECT / ".tmp" / "go-tmp"
    go_cache.mkdir(parents=True, exist_ok=True)
    go_mod_cache.mkdir(parents=True, exist_ok=True)
    go_tmp.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ)
    env["GOCACHE"] = str(go_cache)
    env["GOMODCACHE"] = str(go_mod_cache)
    env["GOTMPDIR"] = str(go_tmp)
    return env


def remove_path(path: Path):
    if path.is_symlink() or path.is_file():
        path.unlink(missing_ok=True)
    elif path.is_dir():
        shutil.rmtree(path)


def ensure_mxu():
    """Download MXU to install/ directory."""
    mxu_path = PROJECT / "install" / "mxu"
    if mxu_path.exists():
        print(f"✅ MXU: {mxu_path}")
        return

    print("📦 Downloading MXU...")
    data = _http_get(f"https://api.github.com/repos/{MXU_REPO}/releases/latest")
    for asset in data["assets"]:
        name = asset["name"]
        if OS_KEY in name.lower() and ARCH_KEY in name.lower() and name.endswith(".tar.gz"):
            with tempfile.TemporaryDirectory() as tmp:
                archive = Path(tmp) / name
                _download(asset["browser_download_url"], archive)
                _extract_tar(archive, Path(tmp))
                extracted = list(Path(tmp).glob("mxu*"))
                if extracted:
                    mxu_path.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(extracted[0], mxu_path)
                    mxu_path.chmod(0o755)
    if mxu_path.exists():
        print(f"✅ MXU installed: {mxu_path}")
    else:
        print("❌ Failed to download MXU")


def _extract_tar(archive: Path, dest: Path):
    dest.mkdir(parents=True, exist_ok=True)
    with tarfile.open(archive) as tf:
        tf.extractall(dest)


def _extract_zip(archive: Path, dest: Path):
    dest.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(archive) as zf:
        zf.extractall(dest)


def ensure_maafw():
    """Download MaaFramework runtime to install/maafw/."""
    maafw_dir = PROJECT / "install" / "maafw"
    if (
        (maafw_dir / "bin").exists()
        and any((maafw_dir / "bin").iterdir())
        and (maafw_dir / "share").exists()
        and any((maafw_dir / "share").iterdir())
    ):
        print(f"✅ MaaFramework: {maafw_dir}")
        return

    print("📦 Downloading MaaFramework runtime...")
    data = _http_get(f"https://api.github.com/repos/{MFW_REPO}/releases/latest")
    os_tags = MFW_OS_TAGS.get(OS_KEY, (OS_KEY,))
    for asset in data["assets"]:
        name = asset["name"]
        lower_name = name.lower()
        if any(tag in lower_name for tag in os_tags) and ARCH_KEY in lower_name and name.endswith(".zip"):
            with tempfile.TemporaryDirectory() as tmp:
                archive = Path(tmp) / name
                _download(asset["browser_download_url"], archive)
                _extract_zip(archive, Path(tmp))
                bin_dir = Path(tmp) / "bin"
                share_dir = Path(tmp) / "share"
                if bin_dir.exists():
                    maafw_dir.mkdir(parents=True, exist_ok=True)
                    (maafw_dir / "bin").mkdir(parents=True, exist_ok=True)
                    for f in bin_dir.iterdir():
                        flat_dest = maafw_dir / f.name
                        bin_dest = maafw_dir / "bin" / f.name
                        if f.is_dir():
                            shutil.copytree(f, flat_dest, dirs_exist_ok=True)
                            shutil.copytree(f, bin_dest, dirs_exist_ok=True)
                        else:
                            shutil.copy2(f, flat_dest)
                            shutil.copy2(f, bin_dest)
                if share_dir.exists():
                    shutil.copytree(share_dir, maafw_dir / "share", dirs_exist_ok=True)
    if maafw_dir.exists():
        print(f"✅ MaaFramework installed: {maafw_dir}")
    else:
        print("❌ Failed to download MaaFramework")


def ensure_deps_layout():
    """Mirror MaaFramework runtime into deps/ for tools that expect DEPS_DIR."""
    maafw_dir = PROJECT / "install" / "maafw"
    deps_dir = PROJECT / "deps"
    if not maafw_dir.exists():
        return

    bin_src = maafw_dir / "bin"
    share_src = maafw_dir / "share"
    if bin_src.exists():
        shutil.copytree(bin_src, deps_dir / "bin", dirs_exist_ok=True)
    else:
        # Backward compatibility: older setup scripts flattened MaaFramework
        # runtime files directly under install/maafw/.
        flat_bin_dst = deps_dir / "bin"
        flat_bin_dst.mkdir(parents=True, exist_ok=True)
        for item in maafw_dir.iterdir():
            if item.name in {"bin", "share"}:
                continue
            if item.is_dir():
                if item.name == "plugins":
                    shutil.copytree(item, flat_bin_dst / "plugins", dirs_exist_ok=True)
                continue
            shutil.copy2(item, flat_bin_dst / item.name)

    if not share_src.exists():
        print("⚠️  install/maafw/share not found; rerun setup to refresh MaaFramework package layout")
        return

    shutil.copytree(share_src, deps_dir / "share", dirs_exist_ok=True)
    print(f"✅ Synced MaaFramework runtime into {deps_dir}")


def configure_ocr():
    """Provision OCR model using the shared configure.py logic."""
    configure_ocr_model()


def build_agent():
    """Build Go service agent."""
    agent_dir = PROJECT / "agent" / "go-service"
    if not (agent_dir / "go.mod").exists():
        print("⚠️  No Go agent found, skipping")
        return True

    print("🔨 Building Go agent...")
    output_dir = PROJECT / "install" / "agent"
    output_dir.mkdir(parents=True, exist_ok=True)
    output_path = output_dir / "go-service"
    if output_path.is_dir():
        shutil.rmtree(output_path)
    elif output_path.exists():
        output_path.unlink()
    result = subprocess.run(
        ["go", "build", "-o", str(output_path), "."],
        cwd=agent_dir,
        capture_output=True,
        text=True,
        env=workspace_go_env(),
    )
    if result.returncode == 0:
        print(f"✅ Go agent built: {output_path}")
        return True
    else:
        print(f"❌ Go build failed:\n{result.stderr}")
        return False


def build_cpp_algo():
    """Build cpp-algo agent into install/agent/."""
    cpp_dir = PROJECT / "agent" / "cpp-algo"
    if not cpp_dir.exists():
        print("⚠️  No cpp-algo source found, skipping")
        return True

    maautils_cmake = cpp_dir / "MaaUtils" / "MaaUtils.cmake"
    if not maautils_cmake.exists():
        print("⚠️  cpp-algo MaaUtils submodule not initialized, skipping")
        return True

    deps_dir = PROJECT / "deps"
    if not (deps_dir / "share").exists():
        print("⚠️  MaaFramework deps/share not found, skipping cpp-algo build")
        return True

    print("🔨 Building cpp-algo...")
    build_dir = cpp_dir / "build"
    agent_dir = PROJECT / "install" / "agent"
    agent_dir.mkdir(parents=True, exist_ok=True)
    resolved_os = {"windows": "win", "darwin": "macos"}.get(platform.system().lower(), "linux")
    ext = ".exe" if resolved_os == "win" else ""
    machine = platform.machine().lower()
    resolved_arch = "x86_64" if machine in ("x86_64", "amd64") else "aarch64" if machine in ("aarch64", "arm64") else machine
    arch_part = "x64" if resolved_arch == "x86_64" else "arm64"
    os_part = {"win": "windows", "macos": "osx", "linux": "linux"}.get(resolved_os, resolved_os)
    maadeps_triplet = f"maa-{arch_part}-{os_part}"
    configure_cmd = [
        "cmake",
        "-S",
        str(cpp_dir),
        "-B",
        str(build_dir),
        "-DCMAKE_BUILD_TYPE=Release",
        f"-DMAADEPS_TRIPLET={maadeps_triplet}",
        f"-DDEPS_DIR={deps_dir}",
    ]
    if resolved_os == "macos":
        osx_arch = "x86_64" if resolved_arch == "x86_64" else "arm64"
        configure_cmd.append(f"-DCMAKE_OSX_ARCHITECTURES={osx_arch}")
    build_cmd = ["cmake", "--build", str(build_dir), "--config", "Release", "--parallel"]

    configure = subprocess.run(configure_cmd, capture_output=True, text=True)
    if configure.returncode != 0:
        print(f"❌ CMake configure failed:\n{configure.stderr}")
        return False

    build = subprocess.run(build_cmd, capture_output=True, text=True)
    if build.returncode != 0:
        print(f"❌ CMake build failed:\n{build.stderr}")
        return False

    src = build_dir / "bin" / f"cpp-algo{ext}"
    dst = agent_dir / f"cpp-algo{ext}"
    if not src.exists():
        print(f"❌ cpp-algo binary missing: {src}")
        return False
    if dst.is_dir():
        shutil.rmtree(dst)
    elif dst.exists():
        dst.unlink()
    shutil.copy2(src, dst)
    print(f"✅ cpp-algo built: {dst}")
    return True


def sync_interface_agents():
    install = PROJECT / "install"
    interface_path = install / "interface.json"
    if not interface_path.exists():
        return

    with open(interface_path, encoding="utf-8") as f:
        interface = json.load(f)

    binary_candidates = {
        "agent/go-service": (
            install / "agent" / "go-service",
            install / "agent" / "go-service.exe",
        ),
        "agent/cpp-algo": (
            install / "agent" / "cpp-algo",
            install / "agent" / "cpp-algo.exe",
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
            agents.append(
                existing_agents.get(
                    child_exec,
                    {
                        "child_exec": child_exec,
                        "child_args": [],
                    },
                )
            )
    interface["agent"] = agents

    with open(interface_path, "w", encoding="utf-8") as f:
        json.dump(interface, f, ensure_ascii=False, indent=4)


def build_install():
    """Assemble the install/ directory."""
    install = PROJECT / "install"
    install.mkdir(exist_ok=True)

    # interface.json
    interface_dst = install / "interface.json"
    remove_path(interface_dst)
    shutil.copy2(PROJECT / "assets" / "interface.json", interface_dst)

    # resource
    res_src = PROJECT / "assets" / "resource"
    res_dst = install / "resource"
    remove_path(res_dst)
    shutil.copytree(res_src, res_dst, dirs_exist_ok=True)

    agent_dst = install / "agent"
    agent_dst.mkdir(parents=True, exist_ok=True)
    for nested_source_dir in ("go-service", "cpp-algo"):
        nested = agent_dst / nested_source_dir
        if nested.is_dir():
            shutil.rmtree(nested)

    # tasks (referenced by interface.json import paths)
    tasks_src = PROJECT / "assets" / "tasks"
    tasks_dst = install / "tasks"
    remove_path(tasks_dst)
    if tasks_src.exists():
        shutil.copytree(tasks_src, tasks_dst, dirs_exist_ok=True)

    # locales
    locales_src = PROJECT / "assets" / "locales"
    locales_dst = install / "locales"
    remove_path(locales_dst)
    if locales_src.exists():
        shutil.copytree(locales_src, locales_dst, dirs_exist_ok=True)

    sync_interface_agents()
    print("✅ install/ assembled")


def validate():
    """Run basic validation."""
    print("🔍 Validating...")
    iface = PROJECT / "assets" / "interface.json"
    with open(iface) as f:
        data = json.load(f)
    imports = data.get("import", [])
    missing = [imp for imp in imports if not (PROJECT / "assets" / imp).exists()]
    if missing:
        print(f"⚠️  Missing task files: {missing}")
    else:
        print(f"✅ {len(imports)} tasks, {len(data.get('group',[]))} groups")


def main():
    parser = argparse.ArgumentParser(description="MaaWuWaX workspace setup")
    parser.add_argument("--skip-mxu", action="store_true")
    parser.add_argument("--skip-maafw", action="store_true")
    parser.add_argument("--skip-agent", action="store_true")
    parser.add_argument("--cpp-algo", action="store_true")
    args = parser.parse_args()

    print(f"🚀 MaaWuWaX setup — {OS_KEY}/{ARCH_KEY}\n")

    if not args.skip_mxu:
        ensure_mxu()
    if not args.skip_maafw:
        ensure_maafw()
    ensure_deps_layout()
    configure_ocr()
    if not args.skip_agent:
        if not build_agent():
            sys.exit(1)
        if args.cpp_algo:
            if not build_cpp_algo():
                sys.exit(1)
    build_install()
    validate()

    print(f"\n✨ Setup complete. Run: install/mxu")


if __name__ == "__main__":
    main()
