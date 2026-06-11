# MaaWuWaX C++ Algo Agent

This directory is the Stage 2 home for MaaEnd-style OpenCV/navigation components:
`MapLocator`, `MapNavigator`, `MotionTracker`, and Navmesh helpers.

Current status:

- Source skeleton has been ported from MaaEnd.
- `MaaUtils` is required as a submodule at `agent/cpp-algo/MaaUtils`.
- Third-party MaaDeps are downloaded through `tools/maadeps-download.py`.
- `assets/interface.json` keeps the baseline Go agent; the install/build scripts append `agent/cpp-algo` automatically when the executable is actually present in `install/agent/`.
- `assets/tasks/FarmMap.json` now defaults to the cpp path (`MapLocateRecognition` + `FarmMapWalkStepCpp`) and keeps the Go path as an explicit fallback backend option.
- Local build outputs live under `agent/cpp-algo/build/` and must stay out of git.

Bootstrap:

```sh
git submodule update --init --recursive agent/cpp-algo/MaaUtils
python3 tools/maadeps-download.py
```

After `MaaUtils` and MaaDeps are available, the next migration step is to keep tightening cpp `FarmMap` behavior toward ok-ww parity and continue wiring additional MaaEnd-style navigation pieces only where the task pipeline actually consumes them.
