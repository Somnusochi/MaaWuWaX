#pragma once

#include "MaaFramework/MaaAPI.h"

// EchoDetectRecognition — YOLOv8-based echo orb detection.
// Uses assets/echo_model/echo.onnx for inference.
// Ported from ok-ww OnnxYolo8Detect / OpenVinoYolo8Detect.
extern MaaBool EchoDetectRecognitionRun(
    MaaContext* context,
    MaaTaskId task_id,
    const char* node_name,
    const char* custom_recognition_name,
    const char* custom_recognition_param,
    const MaaImageBuffer* image,
    const MaaRect* roi,
    void* trans_arg,
    /* out */ MaaRect* out_box,
    /* out */ MaaStringBuffer* out_detail);
