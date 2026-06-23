package discovery

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseATSURL_Workday(t *testing.T) {
	tests := []struct {
		name       string
		company    string
		rawURL     string
		wantTenant string
		wantSite   string
		wantURL    string
	}{
		{
			name:       "Standard Workday",
			company:    "Dexcom",
			rawURL:     "https://dexcom.wd1.myworkdayjobs.com/Dexcom",
			wantTenant: "dexcom",
			wantSite:   "Dexcom",
			wantURL:    "https://dexcom.wd1.myworkdayjobs.com/Dexcom/",
		},
		{
			name:       "Workday with trailing slash",
			company:    "Dexcom",
			rawURL:     "https://dexcom.wd1.myworkdayjobs.com/Dexcom/",
			wantTenant: "dexcom",
			wantSite:   "Dexcom",
			wantURL:    "https://dexcom.wd1.myworkdayjobs.com/Dexcom/",
		},
		{
			name:       "Workday with en-US locale",
			company:    "Illumina",
			rawURL:     "https://illumina.wd1.myworkdayjobs.com/en-US/illumina-careers/job/San-Diego/Software-Engineer_123",
			wantTenant: "illumina",
			wantSite:   "illumina-careers",
			wantURL:    "https://illumina.wd1.myworkdayjobs.com/en-US/illumina-careers/",
		},
		{
			name:       "Workday with other locale and subsegment",
			company:    "Illumina",
			rawURL:     "https://illumina.wd1.myworkdayjobs.com/de-DE/illumina-careers/",
			wantTenant: "illumina",
			wantSite:   "illumina-careers",
			wantURL:    "https://illumina.wd1.myworkdayjobs.com/de-DE/illumina-careers/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := ParseATSURL(tt.company, tt.rawURL)
			if tc == nil {
				t.Fatalf("ParseATSURL returned nil")
			}
			if tc.Platform != "workday" {
				t.Errorf("Expected platform workday, got %s", tc.Platform)
			}
			if tc.Tenant != tt.wantTenant {
				t.Errorf("Expected Tenant %s, got %s", tt.wantTenant, tc.Tenant)
			}
			if tc.Site != tt.wantSite {
				t.Errorf("Expected Site %s, got %s", tt.wantSite, tc.Site)
			}
			if tc.BaseURL != tt.wantURL {
				t.Errorf("Expected BaseURL %s, got %s", tt.wantURL, tc.BaseURL)
			}
		})
	}
}

func TestParseATSURL_Lever(t *testing.T) {
	tests := []struct {
		name       string
		company    string
		rawURL     string
		wantTenant string
	}{
		{
			name:       "Standard Lever",
			company:    "Datadog",
			rawURL:     "https://jobs.lever.co/datadog",
			wantTenant: "datadog",
		},
		{
			name:       "Lever with trailing slash",
			company:    "Datadog",
			rawURL:     "https://jobs.lever.co/datadog/",
			wantTenant: "datadog",
		},
		{
			name:       "Lever with job id",
			company:    "Datadog",
			rawURL:     "https://jobs.lever.co/datadog/123-456",
			wantTenant: "datadog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := ParseATSURL(tt.company, tt.rawURL)
			if tc == nil {
				t.Fatalf("ParseATSURL returned nil")
			}
			if tc.Platform != "lever" {
				t.Errorf("Expected platform lever, got %s", tc.Platform)
			}
			if tc.Tenant != tt.wantTenant {
				t.Errorf("Expected Tenant %s, got %s", tt.wantTenant, tc.Tenant)
			}
		})
	}
}

func TestParseATSURL_Ashby(t *testing.T) {
	tests := []struct {
		name       string
		company    string
		rawURL     string
		wantTenant string
	}{
		{
			name:       "Standard Ashby",
			company:    "Sentry",
			rawURL:     "https://jobs.ashbyhq.com/sentry",
			wantTenant: "sentry",
		},
		{
			name:       "Ashby with trailing slash",
			company:    "Sentry",
			rawURL:     "https://jobs.ashbyhq.com/sentry/",
			wantTenant: "sentry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := ParseATSURL(tt.company, tt.rawURL)
			if tc == nil {
				t.Fatalf("ParseATSURL returned nil")
			}
			if tc.Platform != "ashby" {
				t.Errorf("Expected platform ashby, got %s", tc.Platform)
			}
			if tc.Tenant != tt.wantTenant {
				t.Errorf("Expected Tenant %s, got %s", tt.wantTenant, tc.Tenant)
			}
		})
	}
}

func TestParseATSURL_Greenhouse(t *testing.T) {
	tests := []struct {
		name       string
		company    string
		rawURL     string
		wantTenant string
	}{
		{
			name:       "Standard Greenhouse",
			company:    "Stripe",
			rawURL:     "https://boards.greenhouse.io/stripe",
			wantTenant: "stripe",
		},
		{
			name:       "Greenhouse with trailing slash",
			company:    "Stripe",
			rawURL:     "https://boards.greenhouse.io/stripe/",
			wantTenant: "stripe",
		},
		{
			name:       "Greenhouse embed with query param",
			company:    "Stripe",
			rawURL:     "https://boards.greenhouse.io/embed/job_board?board=stripe",
			wantTenant: "stripe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := ParseATSURL(tt.company, tt.rawURL)
			if tc == nil {
				t.Fatalf("ParseATSURL returned nil")
			}
			if tc.Platform != "greenhouse" {
				t.Errorf("Expected platform greenhouse, got %s", tc.Platform)
			}
			if tc.Tenant != tt.wantTenant {
				t.Errorf("Expected Tenant %s, got %s", tt.wantTenant, tc.Tenant)
			}
		})
	}
}

// Mock RoundTripper for intercepting HTTP calls
type mockTransport struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestSearchCompanyCareers(t *testing.T) {
	// Keep track of the original transport
	origTransport := discoveryRoundTripper
	t.Cleanup(func() {
		discoveryRoundTripper = origTransport
	})

	t.Run("Direct DDG Match Workday", func(t *testing.T) {
		mockHTML := `<html><body>
			<a class="result__url" href="/l/?uddg=https%3A%2F%2Fdexcom.wd1.myworkdayjobs.com%2FDexcom%2F&amp;rut=123">Careers</a>
		</body></html>`

		discoveryRoundTripper = &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockHTML)),
					Request:    req,
				}, nil
			},
		}

		target, err := SearchCompanyCareers("Dexcom")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if target.Platform != "workday" || target.Tenant != "dexcom" || target.Site != "Dexcom" {
			t.Errorf("Unexpected target details: %+v", target)
		}
	})

	t.Run("Fallback Direct Workday Probing (Intuit)", func(t *testing.T) {
		// Mock DDG search returning no results, but direct Workday probe succeeding
		discoveryRoundTripper = &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				reqURL := req.URL.String()
				// DuckDuckGo search empty results
				if strings.Contains(reqURL, "duckduckgo.com") {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString("<html><body>No results</body></html>")),
						Request:    req,
					}, nil
				}
				// Mocking successful Intuit probe
				if strings.Contains(reqURL, "intuit.myworkdayjobs.com/Careers/") {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString("")),
						Request:    req,
					}, nil
				}
				// Return 404 for other probes
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewBufferString("")),
					Request:    req,
				}, nil
			},
		}

		target, err := SearchCompanyCareers("Intuit")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if target.Platform != "workday" || target.Tenant != "intuit" || target.Site != "Careers" {
			t.Errorf("Unexpected target details: %+v", target)
		}
	})

	t.Run("Direct DDG Match Lever", func(t *testing.T) {
		mockHTML := `<html><body>
			<a href="/l/?uddg=https%3A%2F%2Fjobs.lever.co%2Fdatadog&amp;rut=123">Careers</a>
		</body></html>`

		discoveryRoundTripper = &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockHTML)),
					Request:    req,
				}, nil
			},
		}

		target, err := SearchCompanyCareers("Datadog")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if target.Platform != "lever" || target.Tenant != "datadog" {
			t.Errorf("Unexpected target details: %+v", target)
		}
	})

	t.Run("Direct DDG Match Ashby", func(t *testing.T) {
		mockHTML := `<html><body>
			<a href="/l/?uddg=https%3A%2F%2Fjobs.ashbyhq.com%2Fsentry&amp;rut=123">Careers</a>
		</body></html>`

		discoveryRoundTripper = &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockHTML)),
					Request:    req,
				}, nil
			},
		}

		target, err := SearchCompanyCareers("Sentry")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if target.Platform != "ashby" || target.Tenant != "sentry" {
			t.Errorf("Unexpected target details: %+v", target)
		}
	})
}
