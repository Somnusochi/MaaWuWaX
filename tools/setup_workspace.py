#!/usr/bin/env python3
"""MaaWuWaX workspace setup — downloads deps, builds agent, creates install/ directory.

Usage:
  python3 tools/setup_workspace.py              # full setup
  python3 tools/setup_workspace.py --skip-mxu   # skip MXU download
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
from pathlib import Path

PROJECT = Path(__file__).parent.parent.resolve()
MXU_REPO = "MistEO/MXU"
MFW_REPO = "MaaXYZ/MaaFramework"

OS_KEY = "macos"
_raw_arch = platform.machine()
ARCH_KEY = "aarch64" if _raw_arch == "arm64" else _raw_arch


def _http_get(url: str) -> dict:
    req = urllib.request.Request(url)
    req.add_header("Accept", "application/vnd.github+json")
    req.add_header("User-Agent", "MaaWuWaX-setup")
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read())


def _download(url: str, dest: Path):
    print(f"  downloading {url}")
    urllib.request.urlretrieve(url, dest)


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


def ensure_maafw():
    """Download MaaFramework runtime to install/maafw/."""
    maafw_dir = PROJECT / "install" / "maafw"
    if maafw_dir.exists() and any(maafw_dir.iterdir()):
        print(f"✅ MaaFramework: {maafw_dir}")
        return

    print("📦 Downloading MaaFramework runtime...")
    data = _http_get(f"https://api.github.com/repos/{MFW_REPO}/releases/latest")
    for asset in data["assets"]:
        name = asset["name"]
        if "macos" in name.lower() and ARCH_KEY in name.lower() and name.endswith(".zip"):
            with tempfile.TemporaryDirectory() as tmp:
                archive = Path(tmp) / name
                _download(asset["browser_download_url"], archive)
                subprocess.run(["unzip", "-q", str(archive), "-d", tmp], check=False)
                bin_dir = Path(tmp) / "bin"
                if bin_dir.exists():
                    maafw_dir.mkdir(parents=True, exist_ok=True)
                    for f in bin_dir.iterdir():
                        dest = maafw_dir / f.name
                        if f.is_dir():
                            shutil.copytree(f, dest, dirs_exist_ok=True)
                        else:
                            shutil.copy2(f, dest)
    if maafw_dir.exists():
        print(f"✅ MaaFramework installed: {maafw_dir}")
    else:
        print("❌ Failed to download MaaFramework")


def configure_ocr():
    """Copy OCR model from MaaCommonAssets submodule to resource."""
    ocr_src = PROJECT / "assets" / "MaaCommonAssets" / "OCR" / "ppocr_v5" / "zh_cn"
    ocr_dst = PROJECT / "assets" / "resource" / "model" / "ocr"
    if not ocr_src.exists():
        print("⚠️  MaaCommonAssets not found, run: git submodule update --init")
        return
    if not ocr_dst.exists():
        shutil.copytree(ocr_src, ocr_dst, dirs_exist_ok=True)
        print("✅ OCR model configured")
    else:
        print("✅ OCR model already configured")


def build_agent():
    """Build Go service agent."""
    agent_dir = PROJECT / "agent" / "go-service"
    if not (agent_dir / "go.mod").exists():
        print("⚠️  No Go agent found, skipping")
        return

    print("🔨 Building Go agent...")
    result = subprocess.run(
        ["go", "build", "-o", "go-service", "."],
        cwd=agent_dir,
        capture_output=True,
        text=True,
    )
    if result.returncode == 0:
        print("✅ Go agent built")
    else:
        print(f"❌ Go build failed:\n{result.stderr}")


def build_install():
    """Assemble the install/ directory."""
    install = PROJECT / "install"
    install.mkdir(exist_ok=True)

    # interface.json
    shutil.copy2(PROJECT / "assets" / "interface.json", install / "interface.json")

    # resource
    res_src = PROJECT / "assets" / "resource"
    res_dst = install / "resource"
    if res_dst.exists():
        shutil.rmtree(res_dst)
    shutil.copytree(res_src, res_dst, dirs_exist_ok=True)

    # agent (Go service)
    agent_src = PROJECT / "agent"
    agent_dst = install / "agent"
    if agent_dst.exists():
        shutil.rmtree(agent_dst)
    shutil.copytree(agent_src, agent_dst, dirs_exist_ok=True)

    # tasks (referenced by interface.json import paths)
    tasks_src = PROJECT / "assets" / "tasks"
    tasks_dst = install / "tasks"
    if tasks_dst.exists():
        shutil.rmtree(tasks_dst)
    if tasks_src.exists():
        shutil.copytree(tasks_src, tasks_dst, dirs_exist_ok=True)

    # locales
    locales_src = PROJECT / "assets" / "locales"
    locales_dst = install / "locales"
    if locales_dst.exists():
        shutil.rmtree(locales_dst)
    if locales_src.exists():
        shutil.copytree(locales_src, locales_dst, dirs_exist_ok=True)

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
    args = parser.parse_args()

    print(f"🚀 MaaWuWaX setup — {OS_KEY}/{ARCH_KEY}\n")

    if not args.skip_mxu:
        ensure_mxu()
    if not args.skip_maafw:
        ensure_maafw()
    configure_ocr()
    if not args.skip_agent:
        build_agent()
    build_install()
    validate()

    print(f"\n✨ Setup complete. Run: install/mxu")


if __name__ == "__main__":
    main()
