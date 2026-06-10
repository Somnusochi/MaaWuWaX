#include "echo_detect.h"

#include <fstream>
#include <vector>
#include <algorithm>

#include <MaaUtils/Logger.h>
#include <MaaUtils/NoWarningCV.hpp>
#include <onnxruntime/onnxruntime_cxx_api.h>

#include "../utils.h"

namespace {

constexpr int kModelSize = 640;
constexpr float kConfThreshold = 0.5f;

Ort::Env* gEnv = nullptr;
Ort::Session* gSession = nullptr;
Ort::AllocatorWithDefaultOptions gAllocator;

void initModel() {
    if (gSession) return;
    gEnv = new Ort::Env(ORT_LOGGING_LEVEL_WARNING, "EchoDetect");

    Ort::SessionOptions opts;
    opts.SetIntraOpNumThreads(1);
    opts.SetGraphOptimizationLevel(GraphOptimizationLevel::ORT_ENABLE_ALL);

    const char* paths[] = {
        "assets/echo_model/echo.onnx",
        "../assets/echo_model/echo.onnx",
        "../../assets/echo_model/echo.onnx",
        "/Users/somnusochi/Documents/coding/MaaWuWaX/assets/echo_model/echo.onnx",
    };
    for (const auto* p : paths) {
        std::ifstream f(p);
        if (f.good()) {
            try {
                gSession = new Ort::Session(*gEnv, p, opts);
                LogInfo << "EchoDetect: loaded echo.onnx from " << p;
                return;
            } catch (const Ort::Exception& e) {
                LogError << "EchoDetect: load failed: " << e.what();
            }
        }
    }
    LogError << "EchoDetect: echo.onnx not found";
}

struct Detection {
    float x, y, w, h;
    float conf;
};

std::vector<Detection> detect(const cv::Mat& img) {
    initModel();
    if (!gSession) return {};

    cv::Mat resized;
    cv::resize(img, resized, cv::Size(kModelSize, kModelSize));
    cv::Mat blob;
    resized.convertTo(blob, CV_32F, 1.0 / 255.0);

    std::vector<float> inputData(3 * kModelSize * kModelSize);
    for (int c = 0; c < 3; c++)
        for (int h = 0; h < kModelSize; h++)
            for (int w = 0; w < kModelSize; w++)
                inputData[c * kModelSize * kModelSize + h * kModelSize + w] =
                    blob.at<cv::Vec3f>(h, w)[c];

    std::array<int64_t, 4> shape = {1, 3, kModelSize, kModelSize};
    auto inputTensor = Ort::Value::CreateTensor<float>(
        gAllocator, shape.data(), shape.size());
    std::copy(inputData.begin(), inputData.end(),
              inputTensor.GetTensorMutableData<float>());

    auto inputName = gSession->GetInputNameAllocated(0, gAllocator);
    auto outputName = gSession->GetOutputNameAllocated(0, gAllocator);
    const char* inName = inputName.get();
    const char* outName = outputName.get();

    auto outputs = gSession->Run(Ort::RunOptions{nullptr}, &inName, &inputTensor, 1, &outName, 1);
    if (outputs.empty()) return {};

    float* data = outputs[0].GetTensorMutableData<float>();
    auto outShape = outputs[0].GetTensorTypeAndShapeInfo().GetShape();
    int numAnchors = outShape.size() >= 3 ? static_cast<int>(outShape[2]) : 8400;

    float xScale = static_cast<float>(img.cols) / kModelSize;
    float yScale = static_cast<float>(img.rows) / kModelSize;

    std::vector<Detection> detections;
    for (int i = 0; i < numAnchors; i++) {
        float conf = data[i * 5 + 4];
        if (conf < kConfThreshold) continue;
        Detection det;
        det.x = (data[i*5+0] - data[i*5+2]/2) * xScale;
        det.y = (data[i*5+1] - data[i*5+3]/2) * yScale;
        det.w = data[i*5+2] * xScale;
        det.h = data[i*5+3] * yScale;
        det.conf = conf;
        detections.push_back(det);
    }

    // NMS: sort by confidence, greedy suppression
    std::sort(detections.begin(), detections.end(),
              [](const Detection& a, const Detection& b) { return a.conf > b.conf; });

    std::vector<Detection> result;
    std::vector<bool> kept(detections.size(), true);
    for (size_t i = 0; i < detections.size(); i++) {
        if (!kept[i]) continue;
        result.push_back(detections[i]);
        for (size_t j = i+1; j < detections.size(); j++) {
            if (!kept[j]) continue;
            float ix1 = std::max(detections[i].x, detections[j].x);
            float iy1 = std::max(detections[i].y, detections[j].y);
            float ix2 = std::min(detections[i].x+detections[i].w, detections[j].x+detections[j].w);
            float iy2 = std::min(detections[i].y+detections[i].h, detections[j].y+detections[j].h);
            float iArea = std::max(0.0f, ix2-ix1) * std::max(0.0f, iy2-iy1);
            float uArea = detections[i].w*detections[i].h + detections[j].w*detections[j].h - iArea;
            if (uArea > 0 && iArea / uArea > 0.45f) kept[j] = false;
        }
    }
    return result;
}

} // namespace

MaaBool EchoDetectRecognitionRun(
    MaaContext* context, MaaTaskId task_id, const char* node_name,
    const char* custom_recognition_name, const char* custom_recognition_param,
    const MaaImageBuffer* image, const MaaRect* roi, void* trans_arg,
    MaaRect* out_box, MaaStringBuffer* out_detail)
{
    (void)context; (void)task_id; (void)node_name;
    (void)custom_recognition_name; (void)custom_recognition_param;
    (void)roi; (void)trans_arg; (void)out_detail;

    cv::Mat img = to_mat(image);
    if (img.empty()) return false;

    auto detections = detect(img);
    if (detections.empty()) return false;

    const auto& best = detections[0];
    if (out_box) {
        out_box->x = static_cast<int>(std::max(0.0f, best.x));
        out_box->y = static_cast<int>(std::max(0.0f, best.y));
        out_box->width = static_cast<int>(best.w);
        out_box->height = static_cast<int>(best.h);
    }
    return true;
}
