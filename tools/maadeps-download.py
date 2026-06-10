#!/usr/bin/env python3
import sys
from pathlib import Path


sub_path = Path(__file__).parent.parent / "agent" / "cpp-algo" / "MaaUtils" / "tools"
if not sub_path.exists():
    raise SystemExit(
        "MaaUtils tools not found. Initialize submodules first:\n"
        "  git submodule update --init --recursive agent/cpp-algo/MaaUtils"
    )

sys.path.append(str(sub_path))

from maadeps_download import detect_host_triplet, main as download_main


REPO = "MaaXYZ/MaaDeps"
VERSION = "v2.12.2"


if __name__ == "__main__":
    if len(sys.argv) == 2:
        target_triplet = sys.argv[1]
    else:
        target_triplet = detect_host_triplet()

    download_main(target_triplet, REPO, VERSION)
