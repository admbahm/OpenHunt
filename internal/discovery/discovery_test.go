package discovery

import (
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
