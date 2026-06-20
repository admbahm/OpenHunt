# Implementation Plan - Debug Workday CXS 422 Errors

We are diagnosing and repairing HTTP 422 Unprocessable Entity errors from Workday CXS endpoints.

## 1. Data Structure Refinement (`internal/scraper/models.go`)
- Ensure `WorkdayRequest` struct has `AppliedFacets` field correctly tagged.
- JSON tag: `json:"appliedFacets"` (explicitly remove `omitempty` if present).
- Type: `map[string][]string`.

## 2. HTTP Client Enhancements (`internal/scraper/client.go`)
- **Verbose Error Logging**: 
    - In `FetchJobs`, if `resp.StatusCode != http.StatusOK`:
    - Use `io.ReadAll(resp.Body)` to capture the raw response.
    - Log the raw body string to console: `fmt.Printf("Workday Error Body: %s\n", string(body))`.
- **Complete Header Synthesis**:
    - Add/Update headers in `FetchJobs`:
        - `"Content-Type": "application/json"`
        - `"Accept": "application/json, text/plain, */*"`
        - `"Accept-Language": "en-US"`
        - `"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"`
- **Strict Initialization**:
    - Ensure `AppliedFacets` is initialized with `make(map[string][]string)` before marshaling.

## 3. Unit Testing (`internal/scraper/client_test.go`)
- Use `httptest.NewServer`.
- **Test Case 1: Headers**: Verify all 4 required headers are present and correct.
- **Test Case 2: Payload Serialization**: 
    - Inspect the JSON body received by the test server.
    - Assert that `appliedFacets` is serialized as `{}` and not `null`.

## 4. Execution Workflow
1. Create branch `feature/debug-cxs-422`.
2. Apply changes to `internal/scraper/models.go`.
3. Apply changes to `internal/scraper/client.go`.
4. Create and run `internal/scraper/client_test.go`.
5. Commit and push.
