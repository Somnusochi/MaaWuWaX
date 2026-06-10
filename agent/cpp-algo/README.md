# MaaWuWaX C++ Algo Agent

This directory is the Stage 2 home for MaaEnd-style OpenCV/navigation components:
`MapLocator`, `MapNavigator`, `MotionTracker`, and Navmesh helpers.

Current status:

- Source skeleton has been ported from MaaEnd.
- `MaaUtils` is required as a submodule at `agent/cpp-algo/MaaUtils`.
- Third-party MaaDeps are downloaded through `tools/maadeps-download.py`.
- `assets/interface.json` intentionally keeps only the Go agent until the release/install workflow builds and installs the `cpp-algo` executable.
- Local build outputs live under `agent/cpp-algo/build/` and must stay out of git.

Bootstrap:

```sh
git submodule update --init --recursive agent/cpp-algo/MaaUtils
python3 tools/maadeps-download.py
```

After `MaaUtils` and MaaDeps are available, the next migration step is to wire a CMake build into the install workflow and switch FarmMap pipeline nodes to the C++ custom components.
