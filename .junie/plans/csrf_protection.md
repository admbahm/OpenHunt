# Implementation Plan - CSRF Protection & Stateful Scraper

We are updating the `scraper` module to handle Workday's CSRF protection, which requires maintaining session cookies and providing specific CSRF headers.

## 1. Stateful Client Initialization (`internal/scraper/client.go`)
- Update `Client` struct:
    - The `httpClient` field will now hold a client initialized with a `cookiejar`.
- Update `NewClient()`:
    - Initialize a new `cookiejar.Jar`.
    - Create the `http.Client` using this jar.
    - Maintain existing timeout and user-agent settings.

## 2. CSRF Header Management (`internal/scraper/client.go`)
- Workday uses `X-Calypso-Csrf-Token`.
- To obtain the token and session cookies, we may need an initial "handshake" request (GET) to the job board base URL before making POST requests to the CXS API.
- Update `fetchJobsAt` or add a helper to:
    - Check if we have cookies/token.
    - Extract the CSRF token from cookies (e.g., `CALYPSO_CSRF_TOKEN`).
    - Set the `X-Calypso-Csrf-Token` header in the POST request.

## 3. Unit Testing Updates (`internal/scraper/client_test.go`)
- Enhance the mock server to:
    - Simulate a GET request that sets cookies.
    - Verify that subsequent POST requests include the correct cookies and the `X-Calypso-Csrf-Token` header.
- Test client persistence (ensure cookies are carried over between requests).

## 4. Execution Workflow
1. Ensure we are on `feature/debug-cxs-422`.
2. Modify `internal/scraper/client.go` with stateful initialization.
3. Implement header/cookie extraction logic.
4. Update and run tests.
5. Commit and push.
