#pragma once

#include <MaaFramework/MaaAPI.h>

namespace farmmap
{

MaaBool MAA_CALL FarmMapWalkStepCppRun(
    MaaContext* context,
    MaaTaskId task_id,
    const char* node_name,
    const char* custom_action_name,
    const char* custom_action_param,
    MaaRecoId reco_id,
    const MaaRect* box,
    void* trans_arg);

} // namespace farmmap
