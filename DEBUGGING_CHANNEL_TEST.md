# Debugging Channel Test "openai_error" Issue

## Problem
You're seeing "Request error occurred: openai_error" when testing image generation channels, even though the image appears to generate successfully.

## Root Causes (Most Likely)

### 1. **Response Body Validation Failure**
The backend is receiving a successful response from the upstream provider, but the response format doesn't match expectations.

**Location**: `controller/channel-test.go` lines 736-742
```go
if bodyErr := validateTestResponseBody(respBody, isStream); bodyErr != nil {
    return testResult{
        context:     c,
        localErr:    bodyErr,
        newAPIError: types.NewOpenAIError(bodyErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
    }
}
```

### 2. **Usage Parsing Failure**
The adaptor is failing to parse usage/billing information from the response.

**Location**: `controller/channel-test.go` lines 719-726
```go
usage, usageErr := coerceTestUsage(usageA, isStream, info.GetEstimatePromptTokens())
if usageErr != nil {
    return testResult{
        context:     c,
        localErr:    usageErr,
        newAPIError: types.NewOpenAIError(usageErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError),
    }
}
```

### 3. **Model Price Configuration Missing**
The model "sd2.0" doesn't have pricing configured, causing billing calculation to fail.

**Location**: `controller/channel-test.go` lines 519-531

## Diagnostic Steps

### Step 1: Check Server Logs
Look for detailed error messages in your backend logs:

```bash
# If using systemd
sudo journalctl -u new-api -f

# If running directly
# Check your console output for lines like:
# "channel test bad response: channel_id=X name=Y type=Z model=sd2.0"
```

### Step 2: Enable Debug Logging
Add this to check what the actual response looks like:

1. Find line 763 in `controller/channel-test.go`:
```go
common.SysLog(fmt.Sprintf("testing channel #%d, response: \n%s", channel.Id, string(respBody)))
```

2. This should already be logging the response. Check your logs for the actual upstream response.

### Step 3: Check Model Price Configuration
The model "sd2.0" might not have pricing configured:

1. Go to Settings → Model Pricing
2. Search for "sd2.0"
3. If not found, add pricing for this model

### Step 4: Test with a Standard Model
Try testing with a known working model first:
- For image generation: try "dall-e-3" or "dall-e-2"
- This will help determine if it's model-specific or a general issue

## Quick Fixes to Try

### Fix 1: Add Model Pricing (Most Likely Solution)
If "sd2.0" doesn't have pricing configured:

1. Go to `/console/setting?tab=ratio`
2. Add a new model price entry for "sd2.0"
3. Set a price (e.g., $0.02 per image)
4. Save and test again

### Fix 2: Check Channel Configuration
1. Verify the channel base URL is correct
2. Verify the API key is valid
3. Check if the channel type matches the provider

### Fix 3: Simplify Test Request
In the test interface, try:
- Uncheck "Stream Mode" (already unchecked in your screenshot ✓)
- Change endpoint type from "auto" to "image-generation"
- Try with a simpler prompt: "a cat"

## Getting More Details

To see the EXACT error, you need to check the backend response. The frontend is only showing the error code.

### Method 1: Browser DevTools
1. Open browser DevTools (F12)
2. Go to Network tab
3. Run the test again
4. Click on the request to `/api/channel/test/[id]`
5. Check the Response tab for the full error message

### Method 2: Add Logging to Frontend
Modify `web/default/src/features/channels/lib/channel-actions.ts` line 228:

```typescript
const response = await testChannel(id, payload)
console.log('Test response:', response)  // ADD THIS LINE
if (response.success) {
```

Then check browser console for the full response.

## Common Error Patterns

| Error Pattern | Cause | Solution |
|--------------|-------|----------|
| "model_price_error" | Model pricing not configured | Add model price in settings |
| "upstream error: invalid_request" | Wrong model name or parameters | Check model name spelling |
| "DoRequest failed" | Network/connectivity issue | Check base URL and API key |
| "BadResponseBody" | Response parsing failed | Check provider response format |
| "ConvertRequest failed" | Request conversion issue | Check endpoint type setting |

## Next Steps

1. **Check your backend logs right now** - The actual error is already being logged there
2. **Check browser DevTools Network tab** - See the full error response
3. **Verify model pricing** - Go to settings and check if "sd2.0" has pricing configured
4. **Report back with the actual error** from either logs or DevTools, and I can provide a specific fix

## If You Need Code Changes

If the issue is a bug in the code (not configuration), please share:
1. The actual error message from backend logs or DevTools
2. Your channel type (OpenAI, Azure, Custom, etc.)
3. The provider you're connecting to

Then I can provide a targeted code fix.
