package telemetry

// AnalysisResult captures the structured intelligence extracted from a job description.
type AnalysisResult struct {
	BaseSalaryMin   int      `json:"base_salary_min"`
	BaseSalaryMax   int      `json:"base_salary_max"`
	TechStack       []string `json:"tech_stack"`
	RegulatoryGates []string `json:"regulatory_gates"`
	RoleType        string   `json:"role_type"`
}
