package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/openhunt/openhunt/internal/discovery"
	"github.com/openhunt/openhunt/internal/scraper"
)

func TestNormalizeCompanyFlagCombinesTrailingWords(t *testing.T) {
	got := normalizeCompanyFlag("General", []string{"Atomics"})
	if got != "General Atomics" {
		t.Fatalf("company = %q, want %q", got, "General Atomics")
	}
}

func TestNormalizeCompanyFlagTrimsEmptyParts(t *testing.T) {
	got := normalizeCompanyFlag(" Apple ", []string{" ", "Careers"})
	if got != "Apple Careers" {
		t.Fatalf("company = %q, want %q", got, "Apple Careers")
	}
}

func TestFilterTargetsByCompanyAndPlatform(t *testing.T) {
	targets := []scraper.TargetCompany{
		{Name: "Apple", Platform: "workday"},
		{Name: "Apple Bank", Platform: "greenhouse"},
		{Name: "Stripe", Platform: "greenhouse"},
	}

	got := filterTargets(targets, "apple", "workday")
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Name != "Apple" {
		t.Fatalf("got target %q, want Apple", got[0].Name)
	}
}

func TestDiscoveryFailureResult(t *testing.T) {
	result := discoveryFailureResult("Apple", "workday", errString("not found"))
	if result.OK {
		t.Fatal("expected failure result")
	}
	if result.Company != "Apple" || result.Platform != "workday" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Errors) != 1 || !strings.Contains(result.Errors[0], "not found") {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
}

func TestPrintDiscoveryFailureUnsupportedATS(t *testing.T) {
	output := captureStdout(t, func() {
		printDiscoveryFailure("General Atomics", &discovery.UnsupportedATSError{
			ATS: "brassring",
			URL: "https://sjobs.brassring.com/TGNewUI/Search/Home/Home?partnerid=25539&siteid=5313",
		}, nil)
	})

	if !strings.Contains(output, "Detected unsupported ATS: brassring") {
		t.Fatalf("output did not include unsupported ATS: %s", output)
	}
	if strings.Contains(output, "Configured targets:") {
		t.Fatalf("unsupported ATS output should not list configured targets: %s", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writePipe

	fn()

	if err := writePipe.Close(); err != nil {
		t.Fatalf("close write pipe: %v", err)
	}
	os.Stdout = orig

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, readPipe); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}

type errString string

func (e errString) Error() string {
	return string(e)
}
