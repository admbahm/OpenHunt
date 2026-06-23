package discovery

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/openhunt/openhunt/internal/scraper"
)

var (
	// RoundTripper used for outgoing HTTP requests (overridden in tests)
	discoveryRoundTripper http.RoundTripper = http.DefaultTransport

	// Locales to strip when finding the Workday Site segment
	locales = map[string]bool{
		"en-us": true, "en-gb": true, "zh-cn": true, "fr-fr": true,
		"de-de": true, "ja-jp": true, "es-es": true, "pt-br": true,
	}

	// Regexes to extract links from DDG HTML search
	ddgHrefRegex = regexp.MustCompile(`href="([^"]+)"`)
)

// SearchCompanyCareers queries DuckDuckGo for the company's career page
// and parses the search results to find a matching ATS URL.
func SearchCompanyCareers(companyName string) (*scraper.TargetCompany, error) {
	query := fmt.Sprintf("%s careers", companyName)
	u := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: discoveryRoundTripper,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned bad status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search response: %w", err)
	}
	body := string(bodyBytes)

	matches := ddgHrefRegex.FindAllStringSubmatch(body, -1)
	var candidateURLs []string

	for _, match := range matches {
		link := match[1]
		if strings.Contains(link, "uddg=") {
			uDecoded, err := url.QueryUnescape(link)
			if err == nil {
				idx := strings.Index(uDecoded, "uddg=")
				if idx != -1 {
					actualURL := uDecoded[idx+5:]
					if ampersand := strings.Index(actualURL, "&"); ampersand != -1 {
						actualURL = actualURL[:ampersand]
					}
					// Collect potential matches
					candidateURLs = append(candidateURLs, actualURL)
				}
			}
		}
	}

	// 1. First pass: look for direct ATS URLs in the candidates
	for _, rawURL := range candidateURLs {
		if tc := ParseATSURL(companyName, rawURL); tc != nil {
			return tc, nil
		}
	}

	// 2. Second pass: inspect top custom career domains (up to 3) to see if they embed/redirect to an ATS
	checked := 0
	for _, rawURL := range candidateURLs {
		// Only check clean HTTP/HTTPS links, skip search engine junk or social networks
		if !strings.HasPrefix(rawURL, "http") ||
			strings.Contains(rawURL, "duckduckgo.com") ||
			strings.Contains(rawURL, "linkedin.com") ||
			strings.Contains(rawURL, "indeed.com") ||
			strings.Contains(rawURL, "glassdoor.com") {
			continue
		}

		checked++
		if checked > 3 {
			break
		}

		if tc, err := inspectCustomPage(companyName, rawURL); err == nil && tc != nil {
			return tc, nil
		}
	}

	// 3. Fallback: try direct platform URL probing using name guesses
	if tc := probeDirectFallbacks(companyName); tc != nil {
		return tc, nil
	}

	return nil, fmt.Errorf("could not find a supported job board for %s", companyName)
}

// inspectCustomPage fetches the custom page HTML to look for embedded ATS patterns
func inspectCustomPage(companyName, pageURL string) (*scraper.TargetCompany, error) {
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: discoveryRoundTripper,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check final redirected URL
	finalURL := resp.Request.URL.String()
	if tc := ParseATSURL(companyName, finalURL); tc != nil {
		return tc, nil
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // limit to 1MB
	if err != nil {
		return nil, err
	}
	body := string(bodyBytes)

	// Search body for greenhouse boards URL
	// E.g., boards.greenhouse.io/company
	ghRegex := regexp.MustCompile(`boards\.greenhouse\.io/([^"'/ ]+)`)
	if ghMatch := ghRegex.FindStringSubmatch(body); len(ghMatch) > 1 {
		return &scraper.TargetCompany{
			Name:     companyName,
			Tenant:   ghMatch[1],
			Platform: "greenhouse",
		}, nil
	}

	// Search body for workday jobs URL
	wdRegex := regexp.MustCompile(`https://([^"'/ ]+\.myworkdayjobs\.com/[^"'/ ]+)`)
	if wdMatch := wdRegex.FindStringSubmatch(body); len(wdMatch) > 1 {
		if tc := ParseATSURL(companyName, wdMatch[1]); tc != nil {
			return tc, nil
		}
	}

	// Search body for lever boards URL
	leverRegex := regexp.MustCompile(`jobs\.lever\.co/([^"'/ ]+)`)
	if leverMatch := leverRegex.FindStringSubmatch(body); len(leverMatch) > 1 {
		return &scraper.TargetCompany{
			Name:     companyName,
			Tenant:   leverMatch[1],
			Platform: "lever",
		}, nil
	}

	// Search body for ashby boards URL
	ashbyRegex := regexp.MustCompile(`jobs\.ashbyhq\.com/([^"'/ ]+)`)
	if ashbyMatch := ashbyRegex.FindStringSubmatch(body); len(ashbyMatch) > 1 {
		return &scraper.TargetCompany{
			Name:     companyName,
			Tenant:   ashbyMatch[1],
			Platform: "ashby",
		}, nil
	}

	return nil, nil
}

// ParseATSURL attempts to parse a URL into a TargetCompany if it's a known ATS URL format
func ParseATSURL(companyName, rawURL string) *scraper.TargetCompany {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	host := strings.ToLower(parsed.Host)

	// Workday checks
	if strings.Contains(host, "myworkdayjobs.com") {
		// Tenant is the first subdomain segment (e.g. dexcom.wd1.myworkdayjobs.com -> dexcom)
		parts := strings.Split(host, ".")
		if len(parts) < 3 {
			return nil
		}
		tenant := parts[0]

		// Site and locale parsing
		// E.g. /en-US/Dexcom/ -> segments: "", "en-US", "Dexcom"
		pathParts := strings.Split(parsed.Path, "/")
		site := ""
		for _, part := range pathParts {
			partLower := strings.ToLower(part)
			if part == "" || locales[partLower] || partLower == "job" {
				continue
			}
			site = part
			break
		}

		if site == "" {
			site = tenant
		}

		// Rebuild clean BaseURL
		baseURL := fmt.Sprintf("%s://%s/%s/", parsed.Scheme, parsed.Host, site)
		for _, part := range pathParts {
			partLower := strings.ToLower(part)
			if locales[partLower] {
				baseURL = fmt.Sprintf("%s://%s/%s/%s/", parsed.Scheme, parsed.Host, part, site)
				break
			}
		}

		return &scraper.TargetCompany{
			Name:     companyName,
			Tenant:   tenant,
			Site:     site,
			BaseURL:  baseURL,
			Platform: "workday",
		}
	}

	// Greenhouse checks
	if strings.Contains(host, "greenhouse.io") {
		pathParts := strings.Split(parsed.Path, "/")
		tenant := ""

		if strings.Contains(parsed.Path, "embed/job_board") {
			tenant = parsed.Query().Get("board")
		} else {
			for _, part := range pathParts {
				if part != "" && part != "embed" {
					tenant = part
					break
				}
			}
		}

		if tenant != "" {
			return &scraper.TargetCompany{
				Name:     companyName,
				Tenant:   tenant,
				Platform: "greenhouse",
			}
		}
	}

	// Lever checks
	if strings.Contains(host, "lever.co") {
		pathParts := strings.Split(parsed.Path, "/")
		tenant := ""
		for _, part := range pathParts {
			if part != "" {
				tenant = part
				break
			}
		}
		if tenant != "" {
			return &scraper.TargetCompany{
				Name:     companyName,
				Tenant:   tenant,
				Platform: "lever",
			}
		}
	}

	// Ashby checks
	if strings.Contains(host, "ashbyhq.com") {
		pathParts := strings.Split(parsed.Path, "/")
		tenant := ""
		for _, part := range pathParts {
			if part != "" {
				tenant = part
				break
			}
		}
		if tenant != "" {
			return &scraper.TargetCompany{
				Name:     companyName,
				Tenant:   tenant,
				Platform: "ashby",
			}
		}
	}

	return nil
}

// probeDirectFallbacks attempts to probe standard subdomains/endpoints directly if search fails to yield links.
func probeDirectFallbacks(companyName string) *scraper.TargetCompany {
	// Clean the name to create a valid tenant name
	// Strip spaces, convert to lowercase, keep only alphanumeric
	reg := regexp.MustCompile("[^a-zA-Z0-9]")
	cleanName := strings.ToLower(reg.ReplaceAllString(companyName, ""))

	client := &http.Client{
		Timeout:   3 * time.Second,
		Transport: discoveryRoundTripper,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	// 1. Try Greenhouse
	ghURL := fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs", cleanName)
	if resp, err := client.Get(ghURL); err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return &scraper.TargetCompany{
				Name:     companyName,
				Tenant:   cleanName,
				Platform: "greenhouse",
			}
		}
	}

	// 2. Try Workday (e.g. companyname.myworkdayjobs.com or companyname.wd1.myworkdayjobs.com)
	wdHosts := []string{
		fmt.Sprintf("%s.myworkdayjobs.com", cleanName),
		fmt.Sprintf("%s.wd1.myworkdayjobs.com", cleanName),
	}
	siteGuesses := []string{
		cleanName,
		cleanName + "-careers",
		strings.Title(cleanName),
		strings.Title(cleanName) + "-Careers",
		"Careers",
		"careers",
	}

	for _, wdHost := range wdHosts {
		for _, site := range siteGuesses {
			wdURL := fmt.Sprintf("https://%s/%s/", wdHost, site)
			req, _ := http.NewRequest("GET", wdURL, nil)
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
			if resp, err := client.Do(req); err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound {
					finalURL := resp.Request.URL.String()
					if finalURL == "" {
						finalURL = wdURL
					}
					if tc := ParseATSURL(companyName, finalURL); tc != nil {
						return tc
					}
				}
			}
		}
	}

	// 3. Try Lever
	leverURL := fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", cleanName)
	if resp, err := client.Get(leverURL); err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return &scraper.TargetCompany{
				Name:     companyName,
				Tenant:   cleanName,
				Platform: "lever",
			}
		}
	}

	// 4. Try Ashby public hosted boards.
	ashbyURL := fmt.Sprintf("https://jobs.ashbyhq.com/%s", cleanName)
	if resp, err := client.Get(ashbyURL); err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return &scraper.TargetCompany{
				Name:     companyName,
				Tenant:   cleanName,
				Platform: "ashby",
			}
		}
	}

	return nil
}
