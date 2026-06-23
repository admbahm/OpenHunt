package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openhunt/openhunt/internal/scraper"
)

func TestVaultWriterIncludesApplyLink(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewVaultWriter(baseDir)

	job := scraper.JobListing{
		JobID:         "JR123",
		Title:         "Manager, Large Language Model Inference",
		LocationsText: "US, CA, Santa Clara",
		PostedOn:      "Posted Today",
		ExternalPath:  "https://nvidia.wd5.myworkdayjobs.com/wday/cxs/nvidia/NVIDIAExternalCareerSite/job/US-CA-Santa-Clara/Manager_JR123",
		Description:   "Build inference systems.",
	}
	analysis := &AnalysisResult{
		RoleType:      "Management",
		BaseSalaryMin: 184000,
		BaseSalaryMax: 356500,
		TechStack:     []string{"C++", "CUDA"},
	}

	if err := writer.WriteJob("NVIDIA", job, analysis); err != nil {
		t.Fatalf("WriteJob returned error: %v", err)
	}

	path := filepath.Join(baseDir, "@Active", "NVIDIA - Manager, Large Language Model Inference.md")
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read exported markdown: %v", err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, `url: "https://nvidia.wd5.myworkdayjobs.com/wday/cxs/nvidia/NVIDIAExternalCareerSite/job/US-CA-Santa-Clara/Manager_JR123"`) {
		t.Fatalf("expected url in frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, `**Apply:** [https://nvidia.wd5.myworkdayjobs.com/wday/cxs/nvidia/NVIDIAExternalCareerSite/job/US-CA-Santa-Clara/Manager_JR123](https://nvidia.wd5.myworkdayjobs.com/wday/cxs/nvidia/NVIDIAExternalCareerSite/job/US-CA-Santa-Clara/Manager_JR123)`) {
		t.Fatalf("expected apply link in markdown body, got:\n%s", content)
	}
}
