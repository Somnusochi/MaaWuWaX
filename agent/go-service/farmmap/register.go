package farmmap

import maa "github.com/MaaXYZ/maa-framework-go/v4"

func Register() {
	maa.AgentServerRegisterCustomAction("FarmMapLoadPath", &LoadPathAction{})
	maa.AgentServerRegisterCustomAction("FarmMapWalkStep", &WalkStepAction{})
	maa.AgentServerRegisterCustomAction("FarmMapResetTracker", &ResetTrackerAction{})
	maa.AgentServerRegisterCustomRecognition("FarmMapPathDone", &PathDoneRecognition{})
	maa.AgentServerRegisterCustomRecognition("MapLocateRecognition", &LocateRecognition{})
}
