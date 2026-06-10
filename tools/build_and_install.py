#!/usr/bin/env python3
"""MaaWuWaX 构建与安装脚本

用法:
  python tools/build_and_install.py              # 开发模式（symlink）
  python tools/build_and_install.py --ci         # CI 模式（copy + go build）
  python tools/build_and_install.py --cpp-algo   # 同时构建 cpp-algo
  python tools/build_and_install.py --cpp-algo --ci --os macos --arch x86_64 --version v0.0.3
"""

import argparse
import os
import platform
import shutil
import subprocess
import sys
from pathlib import Path


def info(msg: str) -> str:
    return f"\033[36m{msg}\033[0m"


def ok(msg: str) -> str:
    return f"\033[32m{msg}\033[0m"


def warn(msg: str) -> str:
    return f"\033[33m{msg}\033[0m"


def err(msg: str) -> str:
    return f"\033[31m{msg}\033[0m"


def step(msg: str) -> str:
    return f"\n{info('==>')} {msg}"


def create_directory_link(src: Path, dst: Path) -> bool:
    if dst.exists() or dst.is_symlink():
        if dst.is_dir() and not dst.is_symlink():
            try:
                dst.rmdir()
            except OSError:
                shutil.rmtree(dst)
        else:
            dst.unlink(missing_ok=True)
    dst.parent.mkdir(parents=True, exist_ok=True)
    if platform.system() == "Windows":
        result = subprocess.run(
            ["cmd", "/c", "mklink", "/J", str(dst), str(src)],
            capture_output=True, text=True,
        )
        return result.returncode == 0
    else:
        dst.symlink_to(src)
    return True


def copy_directory(src: Path, dst: Path) -> bool:
    if dst.exists():
        shutil.rmtree(dst)
    shutil.copytree(src, dst)
    return True


def build_go_agent(
    root_dir: Path,
    install_dir: Path,
    target_os: str | None = None,
    target_arch: str | None = None,
    version: str | None = None,
    ci_mode: bool = False,
) -> bool:
    """构建 Go Agent"""
    go_service_dir = root_dir / "agent" / "go-service"
    if not go_service_dir.exists():
        print(f"  {err('错误')}: Go 源码目录不存在: {go_service_dir}")
        return False

    if target_os:
        goos = {"win": "windows", "macos": "darwin", "linux": "linux"}.get(target_os, target_os)
    else:
        system = platform.system().lower()
        goos = {"windows": "windows", "darwin": "darwin"}.get(system, "linux")

    if target_arch:
        goarch = {"x86_64": "amd64", "aarch64": "arm64"}.get(target_arch, target_arch)
    else:
        machine = platform.machine().lower()
        goarch = "amd64" if machine in ("x86_64", "amd64") else "arm64" if machine in ("aarch64", "arm64") else machine

    ext = ".exe" if goos == "windows" else ""
    agent_dir = install_dir / "agent"
    agent_dir.mkdir(parents=True, exist_ok=True)
    output_path = agent_dir / f"go-service{ext}"
    print(f"  目标平台: {goos}/{goarch}")
    print(f"  输出路径: {output_path}")

    env = {**os.environ, "GOOS": goos, "GOARCH": goarch, "CGO_ENABLED": "0"}

    ldflags = ""
    if version:
        ldflags += f" -X main.Version={version}"
    ldflags = ldflags.strip()

    build_cmd = ["go", "build"]
    if ci_mode:
        build_cmd.append("-trimpath")
    if ldflags:
        build_cmd.append(f"-ldflags={ldflags}")
    build_cmd.extend(["-o", str(output_path), "."])

    print(f"  构建命令: {' '.join(build_cmd)}")
    result = subprocess.run(build_cmd, cwd=go_service_dir, capture_output=True, text=True)
    if result.stdout:
        print(result.stdout)
    if result.returncode != 0:
        print(f"  {err('错误')}: Go 构建失败")
        if result.stderr:
            print(result.stderr)
        return False
    print(f"  {ok('->')} {output_path}")
    return True


def check_cmake_environment() -> bool:
    try:
        result = subprocess.run(["cmake", "--version"], capture_output=True, text=True)
        if result.returncode == 0:
            print(f"  CMake: {result.stdout.strip().splitlines()[0]}")
            return True
    except FileNotFoundError:
        pass
    print(f"  {err('错误')}: 未找到 CMake")
    return False


def build_cpp_algo(
    root_dir: Path,
    install_dir: Path,
    target_os: str | None = None,
    target_arch: str | None = None,
    ci_mode: bool = False,
) -> bool:
    """构建 C++ Algo Agent。

    CI 模式下 DEPS_DIR 使用 root_dir/deps（CI workflow 下载 MaaFramework 到此），
    开发模式使用 install_dir（开发者自行 build + install MaaFramework 到此）。
    """
    if not check_cmake_environment():
        return False

    cpp_algo_dir = root_dir / "agent" / "cpp-algo"
    if not cpp_algo_dir.exists():
        print(f"  {err('错误')}: cpp-algo 目录不存在: {cpp_algo_dir}")
        return False

    if target_os:
        resolved_os = target_os
    else:
        system = platform.system().lower()
        resolved_os = {"windows": "win", "darwin": "macos"}.get(system, "linux")

    if target_arch:
        resolved_arch = target_arch
    else:
        machine = platform.machine().lower()
        resolved_arch = "x86_64" if machine in ("x86_64", "amd64") else "aarch64" if machine in ("aarch64", "arm64") else machine

    arch_part = "x64" if resolved_arch == "x86_64" else "arm64"
    os_part = {"win": "windows", "macos": "osx", "linux": "linux"}.get(resolved_os, resolved_os)
    maadeps_triplet = f"maa-{arch_part}-{os_part}"

    build_dir = cpp_algo_dir / "build"
    build_type = "Release"

    print(f"  目标平台: {resolved_os}/{resolved_arch}")
    print(f"  MaaDeps triplet: {maadeps_triplet}")

    # CMake configure
    deps_dir = root_dir / "deps" if ci_mode else install_dir

    configure_cmd = [
        "cmake", "-S", str(cpp_algo_dir), "-B", str(build_dir),
        f"-DCMAKE_BUILD_TYPE={build_type}",
        f"-DMAADEPS_TRIPLET={maadeps_triplet}",
        f"-DDEPS_DIR={deps_dir}",
    ]

    if resolved_os == "macos":
        osx_arch = "x86_64" if resolved_arch == "x86_64" else "arm64"
        configure_cmd.extend([
            f"-DCMAKE_OSX_ARCHITECTURES={osx_arch}",
        ])

    print(f"  配置: {' '.join(configure_cmd)}")
    result = subprocess.run(configure_cmd, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"  {err('错误')}: CMake 配置失败")
        if result.stderr:
            print(result.stderr)
        return False

    # CMake build
    build_cmd = ["cmake", "--build", str(build_dir), "--config", build_type, "--parallel"]
    print(f"  构建: {' '.join(build_cmd)}")
    result = subprocess.run(build_cmd, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"  {err('错误')}: CMake 构建失败")
        if result.stderr:
            print(result.stderr)
        return False

    # Copy cpp-algo binary to install/agent/
    agent_dir = install_dir / "agent"
    agent_dir.mkdir(parents=True, exist_ok=True)
    ext = ".exe" if resolved_os == "win" else ""
    src = build_dir / "bin" / f"cpp-algo{ext}"
    if src.exists():
        shutil.copy2(src, agent_dir / f"cpp-algo{ext}")
        print(f"  {ok('->')} {agent_dir / f'cpp-algo{ext}'}")
    else:
        print(f"  {warn('警告')}: 二进制文件未找到: {src}")
        return False

    return True


def main():
    parser = argparse.ArgumentParser(description="MaaWuWaX 构建与安装脚本")
    parser.add_argument("--ci", action="store_true", help="CI 模式（复制文件而非 symlink）")
    parser.add_argument("--os", dest="target_os", help="目标操作系统 (win/macos/linux)")
    parser.add_argument("--arch", dest="target_arch", help="目标架构 (x86_64/aarch64)")
    parser.add_argument("--version", help="版本号")
    parser.add_argument("--cpp-algo", action="store_true", help="同时构建 cpp-algo")
    args = parser.parse_args()

    use_copy = args.ci
    root_dir = Path(__file__).parent.parent.resolve()
    assets_dir = root_dir / "assets"
    install_dir = root_dir / "install"

    mode_text = "CI (复制)" if use_copy else "开发 (symlink)"
    print(f"{info('根目录')}: {root_dir}")
    print(f"{info('安装目录')}: {install_dir}")
    print(f"{warn('模式')}: {mode_text}")

    install_dir.mkdir(parents=True, exist_ok=True)

    # 1. Assets → install
    print(step("1/4 安装 assets"))
    link_or_copy_dir = copy_directory if use_copy else create_directory_link
    for item in assets_dir.iterdir():
        dst = install_dir / item.name
        if item.is_dir():
            if link_or_copy_dir(item, dst):
                print(f"  {ok('->')} {dst}")
        elif item.is_file():
            if dst.exists():
                dst.unlink()
            if use_copy:
                shutil.copy2(item, dst)
            else:
                dst.symlink_to(item)
            print(f"  {ok('->')} {dst}")

    # 2. Go Agent
    print(step("2/4 构建 Go Agent"))
    if not build_go_agent(root_dir, install_dir, args.target_os, args.target_arch, args.version, use_copy):
        sys.exit(1)

    # 3. cpp-algo (可选)
    if args.cpp_algo:
        print(step("3/4 构建 C++ Algo Agent"))
        if not build_cpp_algo(root_dir, install_dir, args.target_os, args.target_arch, use_copy):
            print(f"  {err('错误')}: cpp-algo 构建失败")
            sys.exit(1)
    else:
        print(step("3/4 跳过 cpp-algo"))

    # 4. 项目文件
    print(step("4/4 准备项目文件"))
    for filename in ["README.md", "LICENSE"]:
        src = root_dir / filename
        dst = install_dir / filename
        if src.exists():
            if use_copy:
                shutil.copy2(src, dst)
            elif not dst.exists():
                dst.symlink_to(src)
            print(f"  {ok('->')} {dst}")

    maafw_dir = install_dir / "maafw"
    maafw_dir.mkdir(parents=True, exist_ok=True)
    print(f"  {ok('->')} {maafw_dir} (空目录，运行时由 MaaFramework 填充)")

    print(f"\n{ok('===== 构建完成 =====')}")
    if not use_copy:
        print(f"  {warn('提示')}: 请确保 install/maafw/ 中有 MaaFramework 动态库")


if __name__ == "__main__":
    main()
