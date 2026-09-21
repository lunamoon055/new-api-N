---
name: ghlink-testing
description: Automated testing and validation for GHLINK video generation API models. Use when testing model endpoints, validating parameters, or verifying API responses.
---

# GHLINK API Testing Skill

## Purpose
Automate testing and validation of GHLINK's 21 video/image generation models using Chrome DevTools MCP.

## When to Use
- Testing model API endpoints
- Validating request parameters
- Verifying response formats
- Checking authentication flows
- Monitoring task status polling
- Debugging integration issues

## Available Models

### Original Models (10)
**Video Models (6)**: sora2, veo31, veo31-fast, veo31-ref, kling-v3, grok-imagine-video
**Image Models (4)**: nano-banana, nano-banana2, nano-banana-pro, gpt-image2

### New Models (11)
**MiniMax H3 (1)**: minimax-h3
**SD2 Series (3)**: sd2-1080P(933按秒), sd2-4k(933按秒), sd2-720P(933)
**Video 2.0 Series (6)**: video-2.0, video-2.0-fast, video-2.0-mini, video-2.0-480p, video-2.0-fast-480p, video-2.0-mini-480p
**Videos 4 (1)**: videos-4（4图3视频1音频）

## Testing Workflow

### Step 1: Open GHLINK Website
```javascript
// Open the main page
await chrome.newPage("https://ghlink.top/");
```

### Step 2: Navigate to Model Marketplace
```javascript
// Click on "模型广场" to view available models
await chrome.click("模型广场");
// Note: May require login
```

### Step 3: Test API Endpoint
```javascript
// Example: Test minimax-h3 model
const testRequest = {
  model: "minimax-h3",
  prompt: "一位宇航员走在红色沙漠上，远处是巨大的蓝色行星",
  duration: 8,
  aspect_ratio: "16:9"
};

// Submit via network request monitoring
await chrome.networkMonitor.capture({
  url: "https://ghlink.top/v1/video/async-generations",
  method: "POST",
  body: JSON.stringify(testRequest)
});
```

### Step 4: Validate Response
```javascript
// Check response format
// Expected: { task_id: "task_xxx", status: "processing" }
```

### Step 5: Monitor Task Status
```javascript
// Poll task status every 3-5 seconds
const taskId = response.task_id;
const pollUrl = `https://ghlink.top/v1/video/async-generations/${taskId}`;

// Check until status === "completed"
```

## Common Test Cases

### Test Case 1: Model Parameter Validation

**minimax-h3**:
- ✅ prompt: 1-2000 characters
- ✅ duration: 5-15 seconds
- ✅ aspect_ratio: 16:9, 9:16, 1:1, 4:3, 3:4, 21:9
- ❌ resolution: Should NOT be sent (fixed 2K)

**SD2 Series**:
- ✅ prompt: 1-5000 characters
- ✅ duration: 4-15 seconds
- ✅ ratio: 16:9, 9:16, 1:1
- ⚠️ resolution: Only for sd2-720P (720p or 480p)

**Video 2.0 Series**:
- ✅ prompt: 1-5000 characters
- ✅ duration: 4-15 seconds
- ✅ aspect_ratio: 9:16, 16:9, 1:1
- ✅ resolution: Must match model (720p or 480p)
- ✅ async: true (recommended)

### Test Case 2: Authentication
```javascript
// Test with valid token
headers: {
  "Authorization": "Bearer sk-valid-token",
  "Content-Type": "application/json"
}
// Expected: 200 OK

// Test with invalid token
headers: {
  "Authorization": "Bearer invalid-token"
}
// Expected: 401 Unauthorized
```

### Test Case 3: Error Handling
- 400: Invalid parameters
- 401: Invalid/missing token
- 403: Insufficient permissions
- 404: Task not found
- 429: Rate limit/quota exceeded
- 500/503: Upstream service error

## Browser Automation Commands

### Login Flow
```javascript
// Navigate to login page
await chrome.navigatePage({ type: "url", url: "https://ghlink.top/sign-in" });

// Fill credentials
await chrome.fill({ uid: "username_field", value: "your_username" });
await chrome.fill({ uid: "password_field", value: "your_password" });

// Submit
await chrome.click("登录");
```

### Access Model Marketplace
```javascript
// After login, navigate to pricing page
await chrome.navigatePage({ type: "url", url: "https://ghlink.top/pricing" });

// Take snapshot to verify models loaded
await chrome.takeSnapshot();
```

### Monitor Network Requests
```javascript
// List network requests
const requests = await chrome.listNetworkRequests({
  resourceTypes: ["fetch", "xhr"]
});

// Filter API calls
const apiCalls = requests.filter(req => 
  req.url.includes("/v1/video/async-generations")
);
```

### Check Console Errors
```javascript
// List console messages
const messages = await chrome.listConsoleMessages({
  types: ["error", "warn"]
});
```

## Parameter Validation Rules

### Model-Specific Rules

| Model | prompt_max | duration | ratio/aspect | resolution | ref_images | ref_videos | ref_audios |
|-------|-----------|----------|--------------|-----------|-----------|-----------|-----------|
| minimax-h3 | 2000 | 5-15s | aspect_ratio | ❌ fixed 2K | 0-5 | ❌ | 0-1 |
| sd2-1080P | 5000 | 4-15s | ratio | ❌ fixed 1080P | 0-9 | 0-3 | 0-3 |
| sd2-4k | 5000 | 4-15s | ratio | ❌ fixed 4K | 0-9 | 0-3 | 0-3 |
| sd2-720P | 5000 | 4-15s | ratio | ✅ 720p/480p | 0-9 | 0-3 | 0-3 |
| video-2.0* | 5000 | 4-15s | aspect_ratio | ✅ 720p/480p | 0-4 | 0-3 | 0-1 |
| videos-4 | 5000 | 4-15s | ratio | ✅ 720p/480p | 0-4 | 0-3 | 0-1 |

### Common Validation Checks
1. Model name matches exactly (case-sensitive, includes parentheses)
2. Authorization header format: `Bearer <space> token`
3. URL does not duplicate `/v1` prefix
4. Reference media URLs are publicly accessible
5. task_id is preserved from submit response
6. Polling interval is 3-5 seconds
7. Maximum wait time is 5-10 minutes

## Debugging Checklist

When a model test fails, check:

1. ✅ Model name is copied exactly from model marketplace
2. ✅ Authorization header has space after "Bearer"
3. ✅ Base URL is `https://ghlink.top` (no duplicate /v1)
4. ✅ Fixed-resolution models (1080P, 4K) do NOT send resolution
5. ✅ Variable-resolution models (720P, Video 2.0) DO send resolution
6. ✅ SD2/Videos 4 use `ratio`, MiniMax/Video 2.0 use `aspect_ratio`
7. ✅ Reference media URLs return 200 OK when accessed directly
8. ✅ Reference media counts don't exceed limits
9. ✅ prompt is not empty and within character limit
10. ✅ duration and aspect values are in supported range

## Integration with new-api Project

This skill complements the new-api relay system. When adding GHLINK models to the relay:

1. Add channel adapter in `relay/channel/ghlink/`
2. Implement model-specific parameter mapping
3. Handle async task submission and polling
4. Map GHLINK errors to relay error types
5. Use this skill to validate the integration

### Example Adapter Structure
```go
// relay/channel/ghlink/adaptor.go
package ghlink

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo) (*model.ErrorWithStatusCode, *relaycommon.Usage) {
    // Map relay request to GHLINK format
    request := ConvertRequest(info)
    
    // Submit async task
    taskID, err := a.submitTask(request)
    
    // Poll until complete
    result, err := a.pollTask(taskID)
    
    return result, usage
}
```

## Quick Test Script

Use this script to quickly test all model endpoints:

```javascript
const models = {
  minimax: {
    model: "minimax-h3",
    prompt: "测试视频生成",
    duration: 8,
    aspect_ratio: "16:9"
  },
  sd2_1080p: {
    model: "sd2-1080P(933按秒)",
    prompt: "测试视频生成",
    duration: 8,
    ratio: "16:9"
  },
  video20: {
    model: "video-2.0-fast",
    prompt: "测试视频生成",
    duration: 8,
    aspect_ratio: "16:9",
    resolution: "720p",
    async: true
  }
};

// Test each model
for (const [name, config] of Object.entries(models)) {
  console.log(`Testing ${name}...`);
  // Submit request and validate response
}
```

## Notes

- All 11 new models use the same async workflow
- Query endpoint format is identical across all models
- Response parsing logic is unified (Chapter 6 of original docs)
- Reference media must be publicly accessible URLs
- Task polling should respect rate limits (3-5s interval)

## Related Files

- `/Users/mymac/Desktop/new-api-main/GHLINK_模型对接汇总.md` - Full model documentation
- `relay/channel/ghlink/` - Channel adapter implementation (to be created)
- `constant/channel.go` - Add GHLINK channel type constant

---

**Created**: 2026-09-21  
**For**: GHLINK API (https://ghlink.top)
