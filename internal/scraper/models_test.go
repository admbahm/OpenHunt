package scraper

import "testing"

func TestMatchLocation(t *testing.T) {
	tests := []struct {
		name           string
		jobLoc         string
		targetLocation string
		targetCountry  string
		expected       bool
	}{
		{
			name:           "Empty filters",
			jobLoc:         "San Diego, CA",
			targetLocation: "",
			targetCountry:  "",
			expected:       true,
		},
		{
			name:           "All filters",
			jobLoc:         "San Diego, CA",
			targetLocation: "All",
			targetCountry:  "All",
			expected:       true,
		},
		{
			name:           "Remote filter matches remote job",
			jobLoc:         "Remote, US",
			targetLocation: "Remote",
			targetCountry:  "All",
			expected:       true,
		},
		{
			name:           "Remote filter mismatches onsite job",
			jobLoc:         "San Diego, CA",
			targetLocation: "Remote",
			targetCountry:  "All",
			expected:       false,
		},
		{
			name:           "Exact city match",
			jobLoc:         "San Diego, CA",
			targetLocation: "San Diego, California",
			targetCountry:  "United States of America",
			expected:       true,
		},
		{
			name:           "Case-insensitive city match",
			jobLoc:         "san diego, ca",
			targetLocation: "San Diego, California",
			targetCountry:  "United States of America",
			expected:       true,
		},
		{
			name:           "City match on first token",
			jobLoc:         "San Diego Office",
			targetLocation: "San Diego, California",
			targetCountry:  "United States of America",
			expected:       true,
		},
		{
			name:           "City mismatch",
			jobLoc:         "Los Angeles, CA",
			targetLocation: "San Diego, California",
			targetCountry:  "United States of America",
			expected:       false,
		},
		{
			name:           "Country match (US)",
			jobLoc:         "San Diego, CA",
			targetLocation: "All",
			targetCountry:  "United States of America",
			expected:       true,
		},
		{
			name:           "Country mismatch",
			jobLoc:         "Dublin, Ireland",
			targetLocation: "All",
			targetCountry:  "United States of America",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchLocation(tt.jobLoc, tt.targetLocation, tt.targetCountry)
			if got != tt.expected {
				t.Errorf("MatchLocation(%q, %q, %q) = %v; want %v", tt.jobLoc, tt.targetLocation, tt.targetCountry, got, tt.expected)
			}
		})
	}
}

func TestMatchCategory(t *testing.T) {
	tests := []struct {
		name           string
		jobCategory    string
		targetCategory string
		expected       bool
	}{
		{
			name:           "Empty filter",
			jobCategory:    "Software Engineering",
			targetCategory: "",
			expected:       true,
		},
		{
			name:           "All filter",
			jobCategory:    "Software Engineering",
			targetCategory: "All",
			expected:       true,
		},
		{
			name:           "Category match",
			jobCategory:    "Software Engineering",
			targetCategory: "Engineering",
			expected:       true,
		},
		{
			name:           "Case-insensitive match",
			jobCategory:    "software engineering",
			targetCategory: "Engineering",
			expected:       true,
		},
		{
			name:           "Category mismatch",
			jobCategory:    "Marketing",
			targetCategory: "Engineering",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchCategory(tt.jobCategory, tt.targetCategory)
			if got != tt.expected {
				t.Errorf("MatchCategory(%q, %q) = %v; want %v", tt.jobCategory, tt.targetCategory, got, tt.expected)
			}
		})
	}
}
