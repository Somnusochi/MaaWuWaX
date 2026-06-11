#include "FarmMapWalkStep.h"

#include <algorithm>
#include <array>
#include <chrono>
#include <cmath>
#include <cstdint>
#include <cstring>
#include <filesystem>
#include <fstream>
#include <optional>
#include <numbers>
#include <string>
#include <thread>
#include <vector>

#include <meojson/json.hpp>

#include <MaaUtils/Logger.h>
#include <MaaUtils/NoWarningCV.hpp>

#include "../MapLocator/MapAlgorithm.h"
#include "../MapNavigator/action_wrapper.h"
#include "../utils.h"

namespace farmmap
{

namespace
{

constexpr MaaBool kMaaTrue = 1;
constexpr MaaBool kMaaFalse = 0;

struct PointBox
{
    int x = 0;
    int y = 0;
    int w = 0;
    int h = 0;

    MEO_JSONIZATION(MEO_KEY("x") x, MEO_KEY("y") y, MEO_KEY("w") w, MEO_KEY("h") h)
};

struct PersistedState
{
    std::string big_map_path;
    std::vector<PointBox> stars;
    PointBox my_box;
    PointBox mini_map_box;
    double last_distance = 0.0;
    int stuck_index = 0;
    int too_far_count = 0;
    bool done = false;

    MEO_JSONIZATION(
        MEO_KEY("big_map_path") big_map_path,
        MEO_KEY("stars") stars,
        MEO_KEY("my_box") my_box,
        MEO_KEY("mini_map_box") mini_map_box,
        MEO_KEY("last_distance") last_distance,
        MEO_KEY("stuck_index") stuck_index,
        MEO_OPT MEO_KEY("too_far_count") too_far_count,
        MEO_KEY("done") done)
};

struct WalkStepParam
{
    int reach_distance = 24;
    int search_radius = 280;
    int walk_ms = 850;

    MEO_JSONIZATION(
        MEO_OPT MEO_KEY("reach_distance") reach_distance,
        MEO_OPT MEO_KEY("search_radius") search_radius,
        MEO_OPT MEO_KEY("walk_ms") walk_ms)
};

std::optional<std::filesystem::path> find_existing_from_parents(const std::filesystem::path& relative_path)
{
    std::error_code ec;
    std::filesystem::path current = std::filesystem::current_path(ec);
    if (ec) {
        return std::nullopt;
    }

    while (true) {
        const std::filesystem::path candidate = current / relative_path;
        if (std::filesystem::exists(candidate, ec) && !ec) {
            return candidate;
        }
        const std::filesystem::path parent = current.parent_path();
        if (parent.empty() || parent == current) {
            break;
        }
        current = parent;
    }

    return std::nullopt;
}

std::optional<std::filesystem::path> resolve_path_from_state(const std::string& path_text)
{
    if (path_text.empty()) {
        return std::nullopt;
    }

    const std::filesystem::path input(path_text);
    if (input.is_absolute()) {
        return std::filesystem::exists(input) ? std::optional<std::filesystem::path>(input) : std::nullopt;
    }
    return find_existing_from_parents(input);
}

std::optional<PersistedState> read_state()
{
    const auto state_path = find_existing_from_parents(std::filesystem::path("debug") / "farmmap" / "state.json");
    if (!state_path) {
        LogWarn << "FarmMapWalkStepCpp: state.json not found.";
        return std::nullopt;
    }

    const auto parsed = json::open(state_path->string());
    if (!parsed) {
        LogWarn << "FarmMapWalkStepCpp: failed to parse state json." << VAR(state_path->string());
        return std::nullopt;
    }

    PersistedState state;
    if (!state.from_json(*parsed)) {
        LogWarn << "FarmMapWalkStepCpp: invalid state json." << VAR(state_path->string());
        return std::nullopt;
    }
    return state;
}

bool write_state(const PersistedState& state)
{
    const auto state_path = find_existing_from_parents(std::filesystem::path("debug") / "farmmap" / "state.json");
    if (!state_path) {
        LogWarn << "FarmMapWalkStepCpp: cannot resolve state path for writing.";
        return false;
    }

    std::ofstream out(state_path->string(), std::ios::binary | std::ios::trunc);
    if (!out.is_open()) {
        LogWarn << "FarmMapWalkStepCpp: failed to open state for writing." << VAR(state_path->string());
        return false;
    }

    out << json::value(state).dumps(2);
    return static_cast<bool>(out);
}

cv::Mat ensure_bgr(const cv::Mat& input)
{
    if (input.empty()) {
        return {};
    }
    cv::Mat output;
    switch (input.channels()) {
    case 4:
        cv::cvtColor(input, output, cv::COLOR_BGRA2BGR);
        return output;
    case 3:
        return input.clone();
    case 1:
        cv::cvtColor(input, output, cv::COLOR_GRAY2BGR);
        return output;
    default:
        return {};
    }
}

cv::Rect to_rect(const PointBox& box)
{
    return cv::Rect(box.x, box.y, std::max(0, box.w), std::max(0, box.h));
}

PointBox scale_box(const PointBox& box, double scale)
{
    const double cx = static_cast<double>(box.x) + static_cast<double>(box.w) / 2.0;
    const double cy = static_cast<double>(box.y) + static_cast<double>(box.h) / 2.0;
    int w = static_cast<int>(std::round(static_cast<double>(box.w) * scale));
    int h = static_cast<int>(std::round(static_cast<double>(box.h) * scale));
    w = std::max(w, 40);
    h = std::max(h, 40);
    return PointBox {
        .x = static_cast<int>(std::round(cx - static_cast<double>(w) / 2.0)),
        .y = static_cast<int>(std::round(cy - static_cast<double>(h) / 2.0)),
        .w = w,
        .h = h,
    };
}

PointBox clamp_box(const PointBox& box, const cv::Rect& bounds)
{
    const cv::Rect clipped = to_rect(box) & bounds;
    return PointBox { .x = clipped.x, .y = clipped.y, .w = clipped.width, .h = clipped.height };
}

double center_x(const PointBox& box)
{
    return static_cast<double>(box.x) + static_cast<double>(box.w) / 2.0;
}

double center_y(const PointBox& box)
{
    return static_cast<double>(box.y) + static_cast<double>(box.h) / 2.0;
}

double distance_between(const PointBox& lhs, const PointBox& rhs)
{
    return std::hypot(center_x(rhs) - center_x(lhs), center_y(rhs) - center_y(lhs));
}

double angle_clockwise(const PointBox& from, const PointBox& to)
{
    double degree = std::atan2(center_y(to) - center_y(from), center_x(to) - center_x(from)) * 180.0 / std::numbers::pi;
    if (degree < 0.0) {
        degree += 360.0;
    }
    return degree;
}

double angle_between(double current, double target)
{
    double turn = target - current;
    while (turn > 180.0) {
        turn -= 360.0;
    }
    while (turn < -180.0) {
        turn += 360.0;
    }
    return turn;
}

cv::Mat create_circle_mask_with_hole(const cv::Size& size)
{
    cv::Mat mask(size, CV_8UC1, cv::Scalar(0));
    const int center_x_value = size.width / 2;
    const int center_y_value = size.height / 2;
    const int radius = std::min(size.width, size.height) / 2;
    cv::circle(mask, cv::Point(center_x_value, center_y_value), radius, cv::Scalar(255), -1);

    const int rect_w = static_cast<int>(std::round(static_cast<double>(size.width) / 4.4));
    const int rect_h = static_cast<int>(std::round(static_cast<double>(size.height) / 4.4));
    const int rect_x1 = center_x_value - rect_w / 2;
    const int rect_y1 = center_y_value - rect_h / 2;
    cv::rectangle(mask, cv::Rect(rect_x1, rect_y1, rect_w, rect_h), cv::Scalar(0), -1);
    return mask;
}

std::optional<PointBox> match_minimap_in_bigmap(const cv::Mat& big_map, const cv::Mat& minimap, const PointBox& search_box)
{
    if (big_map.empty() || minimap.empty()) {
        return std::nullopt;
    }

    const cv::Rect bounds(0, 0, big_map.cols, big_map.rows);
    const PointBox clamped = clamp_box(search_box, bounds);
    const cv::Rect roi = to_rect(clamped);
    if (roi.width < minimap.cols || roi.height < minimap.rows) {
        return std::nullopt;
    }

    const cv::Mat search = big_map(roi);
    cv::Mat result;
    cv::matchTemplate(search, minimap, result, cv::TM_CCORR_NORMED, create_circle_mask_with_hole(minimap.size()));

    double max_val = 0.0;
    cv::Point max_loc;
    cv::minMaxLoc(result, nullptr, &max_val, nullptr, &max_loc);
    if (max_val < 0.20) {
        return std::nullopt;
    }

    return PointBox {
        .x = roi.x + max_loc.x,
        .y = roi.y + max_loc.y,
        .w = minimap.cols,
        .h = minimap.rows,
    };
}

std::optional<PointBox> locate_current_position(
    const PersistedState& state,
    const cv::Mat& big_map,
    const cv::Mat& screenshot,
    int search_radius)
{
    const cv::Rect screenshot_bounds(0, 0, screenshot.cols, screenshot.rows);
    const PointBox mini_box = clamp_box(state.mini_map_box, screenshot_bounds);
    const cv::Rect mini_roi = to_rect(mini_box);
    if (mini_roi.width <= 0 || mini_roi.height <= 0) {
        return std::nullopt;
    }

    const cv::Mat minimap = ensure_bgr(screenshot(mini_roi));
    if (minimap.empty()) {
        return std::nullopt;
    }

    const cv::Rect big_bounds(0, 0, big_map.cols, big_map.rows);
    PointBox predicted = clamp_box(
        PointBox {
            .x = state.my_box.x - search_radius,
            .y = state.my_box.y - search_radius,
            .w = state.my_box.w + search_radius * 2,
            .h = state.my_box.h + search_radius * 2,
        },
        big_bounds);

    if (auto located = match_minimap_in_bigmap(big_map, minimap, predicted)) {
        return located;
    }
    return match_minimap_in_bigmap(big_map, minimap, PointBox { .x = 0, .y = 0, .w = big_map.cols, .h = big_map.rows });
}

PointBox nearest_star(const std::vector<PointBox>& stars, const PointBox& current, size_t* out_index)
{
    size_t best_index = 0;
    double best_distance = std::numeric_limits<double>::max();
    for (size_t index = 0; index < stars.size(); ++index) {
        const double distance = distance_between(current, stars[index]);
        if (distance < best_distance) {
            best_distance = distance;
            best_index = index;
        }
    }
    if (out_index != nullptr) {
        *out_index = best_index;
    }
    return stars[best_index];
}

void perform_stuck_step(mapnavigator::ActionWrapper& wrapper, int stuck_index)
{
    switch (stuck_index % 4) {
    case 0:
        wrapper.TriggerJumpSync(80);
        break;
    case 1:
        wrapper.SetMovementStateSync(false, true, false, false, 700);
        wrapper.SetMovementStateSync(false, false, false, false, 0);
        break;
    case 2:
        wrapper.SetMovementStateSync(false, false, false, true, 700);
        wrapper.SetMovementStateSync(false, false, false, false, 0);
        break;
    default:
        wrapper.TriggerSprintSync();
        std::this_thread::sleep_for(std::chrono::milliseconds(250));
        break;
    }
}

void walk_forward_for(mapnavigator::ActionWrapper& wrapper, int walk_ms)
{
    wrapper.SetMovementStateSync(true, false, false, false, 0);
    std::this_thread::sleep_for(std::chrono::milliseconds(std::max(0, walk_ms)));
    wrapper.SetMovementStateSync(false, false, false, false, 0);
}

void steer_toward_target(mapnavigator::ActionWrapper& wrapper, double turn, int walk_ms)
{
    if (turn > 135.0 || turn < -135.0) {
        wrapper.SendViewDeltaSync(turn > 0 ? 420 : -420, 0);
        walk_forward_for(wrapper, walk_ms);
        return;
    }
    if (turn > 45.0) {
        wrapper.SendViewDeltaSync(260, 0);
        walk_forward_for(wrapper, walk_ms);
        return;
    }
    if (turn < -45.0) {
        wrapper.SendViewDeltaSync(-260, 0);
        walk_forward_for(wrapper, walk_ms);
        return;
    }
    if (turn > 10.0) {
        wrapper.SetMovementStateSync(false, false, false, true, 120);
        wrapper.SetMovementStateSync(false, false, false, false, 0);
        walk_forward_for(wrapper, walk_ms);
        return;
    }
    if (turn < -10.0) {
        wrapper.SetMovementStateSync(false, true, false, false, 120);
        wrapper.SetMovementStateSync(false, false, false, false, 0);
        walk_forward_for(wrapper, walk_ms);
        return;
    }
    walk_forward_for(wrapper, walk_ms);
}

double too_far_threshold(const PersistedState& state, const WalkStepParam& param)
{
    const double reach_threshold = static_cast<double>(std::max(param.reach_distance, 24)) * 8.0;
    const double locate_threshold = static_cast<double>(std::max(state.my_box.w, state.my_box.h)) * 4.0;
    const double minimap_threshold = static_cast<double>(std::max(state.mini_map_box.w, state.mini_map_box.h)) * 0.9;
    return std::max({reach_threshold, locate_threshold, minimap_threshold});
}

std::vector<PointBox> to_runtime_boxes(const std::vector<PointBox>& input)
{
    return input;
}

} // namespace

MaaBool MAA_CALL FarmMapWalkStepCppRun(
    MaaContext* context,
    [[maybe_unused]] MaaTaskId task_id,
    [[maybe_unused]] const char* node_name,
    [[maybe_unused]] const char* custom_action_name,
    const char* custom_action_param,
    [[maybe_unused]] MaaRecoId reco_id,
    [[maybe_unused]] const MaaRect* box,
    [[maybe_unused]] void* trans_arg)
{
    if (context == nullptr) {
        LogError << "FarmMapWalkStepCpp: null context";
        return kMaaFalse;
    }

    WalkStepParam param;
    if (custom_action_param != nullptr && std::strlen(custom_action_param) > 0) {
        const auto parsed = json::parse(custom_action_param);
        if (parsed && !param.from_json(*parsed)) {
            LogWarn << "FarmMapWalkStepCpp: failed to parse custom_action_param" << VAR(custom_action_param);
        }
    }

    const auto state_opt = read_state();
    if (!state_opt.has_value()) {
        return kMaaFalse;
    }
    PersistedState state = *state_opt;
    if (state.done || state.stars.empty()) {
        return kMaaTrue;
    }

    const auto big_map_path = resolve_path_from_state(state.big_map_path);
    if (!big_map_path) {
        LogWarn << "FarmMapWalkStepCpp: failed to resolve big map path." << VAR(state.big_map_path);
        return kMaaFalse;
    }

    const cv::Mat big_map = ensure_bgr(cv::imread(big_map_path->string(), cv::IMREAD_UNCHANGED));
    if (big_map.empty()) {
        LogWarn << "FarmMapWalkStepCpp: failed to load big map image." << VAR(big_map_path->string());
        return kMaaFalse;
    }

    MaaTasker* tasker = MaaContextGetTasker(context);
    if (tasker == nullptr) {
        LogError << "FarmMapWalkStepCpp: no tasker";
        return kMaaFalse;
    }
    MaaController* controller = MaaTaskerGetController(tasker);
    if (controller == nullptr) {
        LogError << "FarmMapWalkStepCpp: no controller";
        return kMaaFalse;
    }

    const MaaCtrlId screencap_id = MaaControllerPostScreencap(controller);
    MaaControllerWait(controller, screencap_id);

    MaaImageBuffer* image_buffer = MaaImageBufferCreate();
    if (image_buffer == nullptr || !MaaControllerCachedImage(controller, image_buffer) || MaaImageBufferIsEmpty(image_buffer)) {
        if (image_buffer != nullptr) {
            MaaImageBufferDestroy(image_buffer);
        }
        LogWarn << "FarmMapWalkStepCpp: failed to capture image";
        return kMaaFalse;
    }

    const cv::Mat screenshot = ensure_bgr(to_mat(image_buffer));
    MaaImageBufferDestroy(image_buffer);
    if (screenshot.empty()) {
        LogWarn << "FarmMapWalkStepCpp: screenshot is empty";
        return kMaaFalse;
    }

    auto current_loc = locate_current_position(state, big_map, screenshot, param.search_radius);
    const bool locate_fallback = !current_loc.has_value();
    if (!current_loc.has_value()) {
        current_loc = state.my_box;
        LogWarn << "FarmMapWalkStepCpp: failed to locate current position, using last known position.";
    }

    std::vector<PointBox> stars = to_runtime_boxes(state.stars);
    size_t target_index = 0;
    const PointBox target = nearest_star(stars, *current_loc, &target_index);
    const double distance = distance_between(*current_loc, target);

    if (distance <= static_cast<double>(param.reach_distance)) {
        stars.erase(stars.begin() + static_cast<std::ptrdiff_t>(target_index));
        state.stars = stars;
        state.last_distance = 0.0;
        state.stuck_index = 0;
        state.too_far_count = 0;
        state.my_box = scale_box(target, 1.3);
        state.done = state.stars.empty();
        write_state(state);
        LogInfo << "FarmMapWalkStepCpp reached star." << VAR(distance) << VAR(state.stars.size()) << VAR(state.done);
        return kMaaTrue;
    }

    mapnavigator::ActionWrapper wrapper(context);
    if (!wrapper.is_supported()) {
        LogWarn << "FarmMapWalkStepCpp: unsupported input backend." << VAR(wrapper.controller_type()) << VAR(wrapper.unsupported_reason());
        return kMaaFalse;
    }

    const double far_threshold = too_far_threshold(state, param);
    if (distance > far_threshold) {
        state.too_far_count += 1;
        LogWarn << "FarmMapWalkStepCpp: target distance looks suspiciously far."
                << VAR(distance) << VAR(far_threshold) << VAR(state.too_far_count) << VAR(locate_fallback);
        if (state.too_far_count >= 3) {
            perform_stuck_step(wrapper, state.stuck_index);
            state.stuck_index += 1;
            state.last_distance = distance;
            write_state(state);
            return kMaaTrue;
        }
    }
    else {
        state.too_far_count = 0;
    }

    if (std::abs(distance - state.last_distance) < 1.0) {
        perform_stuck_step(wrapper, state.stuck_index);
        state.stuck_index += 1;
    }
    else {
        const cv::Rect screenshot_bounds(0, 0, screenshot.cols, screenshot.rows);
        const PointBox mini_box = clamp_box(state.mini_map_box, screenshot_bounds);
        const cv::Rect mini_roi = to_rect(mini_box);
        const cv::Mat minimap = ensure_bgr(screenshot(mini_roi));
        const double facing = maplocator::InferYellowArrowRotation(minimap);
        const double target_angle = angle_clockwise(*current_loc, target);
        const double turn = angle_between(facing < 0.0 ? 0.0 : facing, target_angle);
        steer_toward_target(wrapper, turn, param.walk_ms);
        state.stuck_index = 0;
        LogDebug << "FarmMapWalkStepCpp moved."
                 << VAR(distance) << VAR(facing) << VAR(target_angle) << VAR(turn) << VAR(locate_fallback);
    }

    state.last_distance = distance;
    state.my_box = scale_box(*current_loc, 1.3);
    write_state(state);
    return kMaaTrue;
}

} // namespace farmmap
