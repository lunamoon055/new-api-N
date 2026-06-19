# Video Fixed Fee Billing Design

Status: approved for design, pending implementation plan
Date: 2026-06-19

## Summary

Add an explicit per-model video billing mode so administrators can choose whether a video model keeps the current duration/specification billing or uses a fixed one-time fee per task.

The default remains the current behavior:

```text
quota = base model price * group ratio * seconds ratio * size ratio * other task ratios
```

When a video model is set to fixed one-time billing:

```text
quota = base model price * group ratio
```

The change must not remove duration-based billing. It adds an administrator-controlled override for video models that should charge once per generation.

## Current Behavior

Video task adapters return `OtherRatios` from request parameters such as `seconds`, `size`, `durationSeconds`, or provider-specific media options. `RelayTaskSubmit` applies those ratios to `PriceData.Quota` before pre-consumption.

For example, with `ModelPrice = 0.1` and `seconds = 4`, the current quota is effectively:

```text
0.1 * 4 = 0.4
```

This is correct for providers and models that should bill by duration or specification. The problem is that there is no clear admin-facing per-video-model setting for fixed one-time billing.

## Goals

- Keep existing video duration/specification billing as the default.
- Add a model-level setting for all video models:
  - dynamic media billing
  - fixed one-time billing
- Make the setting available from the existing model pricing management area.
- Ensure fixed one-time billing affects both pre-consumption and later task settlement.
- Preserve request parameters sent upstream; `seconds`, `size`, and related fields still belong in the provider request.
- Keep logs and task metadata understandable for support and audits.
- Avoid broad refactors of provider adapters or the billing expression system.

## Non-Goals

- Do not remove `OtherRatios`.
- Do not make all video models fixed-price by default.
- Do not change non-video chat, image, token, or tiered expression pricing.
- Do not require administrators to rewrite billing expressions for simple fixed-price video models.
- Do not change provider request payloads or model mappings.

## Recommended Approach

Introduce a dedicated per-model video billing mode map in settings:

```json
{
  "sora2": "fixed",
  "video-2.0": "dynamic"
}
```

The values are:

- `dynamic`: apply `OtherRatios` normally. This is the default when a model is absent from the map.
- `fixed`: use the base per-call model price once and skip task `OtherRatios` when calculating quota.

This should be separate from general model pricing mode. Existing `ModelPrice`, `ModelRatio`, and tiered billing settings keep their current meaning. The new setting only decides whether video task ratios are billable.

Fixed mode requires a fixed model price. If a model is set to fixed video billing but has no `ModelPrice` entry or default fixed price, the backend should return a clear configuration error instead of silently falling back to model-ratio pre-consumption.

## Alternatives Considered

### Reuse `TaskPricePatches`

This is the smallest backend change because `RelayTaskSubmit` already skips `OtherRatios` for models in `TaskPricePatches`.

Rejected as the primary design because it is an internal compatibility list, not a clear administrator-facing pricing control. It would also make the UI wording and long-term maintenance less obvious.

### Use billing expressions only

A billing expression can represent fixed or dynamic prices, but it is too heavy for this requirement. The user needs a simple per-video-model entry, not a full expression authoring workflow.

Rejected because it adds unnecessary complexity and makes simple fixed video pricing harder to operate.

### Recommended: explicit video billing mode

This keeps the existing dynamic behavior intact and adds a small, named configuration for the new behavior. It is the clearest option for admins and the lowest-risk option for the codebase.

## Data Model

Add a new settings-backed JSON map for video billing modes. Suggested names:

- backend option key: `VideoBillingMode`
- frontend form key: `VideoBillingMode`
- helper package location: near existing ratio or billing settings helpers, following local settings patterns

The setting stores model names as keys and mode strings as values:

```json
{
  "sora2": "fixed",
  "sora-2": "fixed",
  "video-2.0": "dynamic",
  "video-2.0-fast": "dynamic",
  "kling-v3": "fixed"
}
```

Unknown or empty values are treated as `dynamic`. This provides backward compatibility for all existing deployments.

## Backend Flow

### Submit-time billing

1. Build the base `PriceData` through `ModelPriceHelperPerCall`.
2. Let the task adapter estimate media ratios as it does today.
3. Decide whether those ratios are billable:
   - non-video task: unchanged
   - video task with missing or `dynamic` mode: apply `OtherRatios`
   - video task with `fixed` mode and a fixed model price: do not multiply `Quota` by `OtherRatios`
   - video task with `fixed` mode but no fixed model price: fail with a configuration error
4. Pre-consume the resulting quota.

The implementation should use a small helper so the decision is not scattered:

```text
ShouldApplyTaskOtherRatios(modelName, platform, action, priceData) bool
```

The exact helper signature can adjust to existing code, but the rule should remain centralized.

### Submit-time adjustment

Some adapters can adjust billing after the upstream submit response. Fixed one-time billing must skip these ratio-based adjustments as well; otherwise the task could be pre-consumed as fixed and then changed back to dynamic.

### Task metadata

`TaskBillingContext` should record enough information for later settlement and logs. The design should distinguish:

- media ratios observed from the request
- whether those ratios were applied to quota
- the selected video billing mode

If the implementation keeps only one `OtherRatios` field, fixed mode must set `PerCallBilling` or an equivalent explicit flag so polling settlement does not reapply multipliers.

### Completion settlement

`settleTaskBillingOnComplete` already skips adjustment when `BillingContext.PerCallBilling` is true. Fixed one-time video tasks must land in that path.

Dynamic video tasks should keep current behavior.

## Frontend Flow

The model pricing management UI should expose a video-specific billing control for each model:

- label: `视频计费方式`
- options:
  - `按时长/规格计费`
  - `固定每次调用`
- default display: `按时长/规格计费`

This belongs near the existing fixed price/per-request controls, because administrators will usually configure the fixed price and the video billing mode together.

The control can be visible for models known or classified as video models. If model classification is uncertain, showing the field for all models is acceptable as long as the backend only uses it for task/video relay paths.

## Logging and Auditing

For dynamic mode, logs continue showing applied calculation parameters such as `seconds` and `size`.

For fixed mode, logs should say that the task used fixed one-time billing. The observed media ratios may still be recorded in structured metadata for debugging, but the visible consume text should not imply those ratios affected the charge.

Suggested log data:

```json
{
  "video_billing_mode": "fixed",
  "media_ratios": {
    "seconds": 4,
    "size": 1
  },
  "applied_other_ratios": false
}
```

## Compatibility

- Existing settings with no video billing mode map behave exactly as today.
- Existing per-second/per-size billing remains available.
- Existing task provider adapters continue producing `OtherRatios`.
- Existing model price settings remain the source of the base fixed fee.
- Database changes should use the existing settings/options mechanism and remain compatible with SQLite, MySQL, and PostgreSQL.

## Test Plan

Backend tests:

- Dynamic video model with `ModelPrice = 0.1` and `seconds = 4` charges `0.4` equivalent quota.
- Fixed video model with `ModelPrice = 0.1` and `seconds = 4` charges `0.1` equivalent quota.
- Fixed mode skips submit-time adapter ratio adjustment.
- Dynamic mode still applies submit-time adapter ratio adjustment.
- Polling settlement skips fixed one-time video tasks.
- Non-video task billing is unchanged.
- Unknown mode values fall back to dynamic.

Frontend tests:

- Model pricing form can load, edit, and save video billing mode.
- Existing model price, model ratio, and tiered billing fields are not cleared when video billing mode changes.
- The default UI state for models without the new setting is dynamic billing.
- Fixed video billing shows a validation hint when the selected model has no fixed price.

Manual verification:

- Configure a video model as dynamic and confirm a 4-second request charges four units of the model price.
- Configure the same model as fixed and confirm the same 4-second request charges one unit of the model price.
- Confirm request payload still sends `seconds` and `size` to upstream.
- Confirm task logs show fixed one-time billing when selected.

## Implementation Boundaries

Keep the code change small and local:

- add settings parse/save support
- add a centralized backend helper for applying task ratios
- update task submit and task metadata handling
- add the pricing UI control
- add targeted tests

Do not rewrite adapters, model mapping, the pricing catalog, or billing expression internals for this feature.
