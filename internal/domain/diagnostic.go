package domain

// Diagnostic reports a problem with the analysis itself, such as a parse failure.
// A required error diagnostic makes the whole analysis incomplete.
type Diagnostic struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	File     string   `json:"file,omitempty"`
	Required bool     `json:"required"`
}

// IsRequiredError reports whether the diagnostic invalidates the analysis.
func (d Diagnostic) IsRequiredError() bool {
	return d.Required && d.Severity == SeverityError
}

// HasRequiredError reports whether any diagnostic invalidates the analysis.
func HasRequiredError(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.IsRequiredError() {
			return true
		}
	}
	return false
}

// RequiredError builds a diagnostic that makes the analysis incomplete.
func RequiredError(code, file, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: SeverityError, Message: message, File: file, Required: true}
}

// Advisory builds a non-required diagnostic with the given severity.
func Advisory(code, file, message string, severity Severity) Diagnostic {
	return Diagnostic{Code: code, Severity: severity, Message: message, File: file, Required: false}
}
