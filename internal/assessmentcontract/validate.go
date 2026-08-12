package assessmentcontract

import (
	"fmt"
	"time"

	"github.com/IndependentImpact/standard2shape/internal/contract"
)

func validateRequest(request Request) []contract.Diagnostic {
	var diagnostics []contract.Diagnostic
	prefix := "request"
	if request.RequestVersion == "" {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", "requestVersion", "field is required"))
	} else if request.RequestVersion != RequestVersionV01 {
		diagnostics = append(diagnostics, contract.Diag(prefix+".version.unsupported", "requestVersion", "supported request version is %s, got %q", RequestVersionV01, request.RequestVersion))
	}
	diagnostics = append(diagnostics, checkPackageRef(prefix, "package", request.Package)...)
	diagnostics = append(diagnostics, checkEntityRef(prefix, "reasoningProfile", request.ReasoningProfile)...)
	if len(request.Checks) == 0 {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", "checks", "at least one check is required"))
	}
	seenChecks := map[string]bool{}
	for index, check := range request.Checks {
		location := fmt.Sprintf("checks[%d]", index)
		if !isCheck(check) {
			diagnostics = append(diagnostics, contract.Diag(prefix+".check.invalid", location, "unknown check %q", check))
		}
		if seenChecks[check] {
			diagnostics = append(diagnostics, contract.Diag(prefix+".check.duplicate", location, "check %q is requested more than once", check))
		}
		seenChecks[check] = true
	}
	if request.Evidence == nil {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", "evidence", "evidence must be an array, possibly empty"))
	}
	seenEvidence := map[string]bool{}
	for index, evidence := range request.Evidence {
		location := fmt.Sprintf("evidence[%d]", index)
		diagnostics = append(diagnostics, checkEvidenceRef(prefix, location, evidence)...)
		if seenEvidence[evidence.Path] {
			diagnostics = append(diagnostics, contract.Diag(prefix+".evidence.duplicate", location+".path", "evidence path is declared more than once"))
		}
		seenEvidence[evidence.Path] = true
	}
	if request.Requirements == nil {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", "requirements", "requirements must be an array, possibly empty"))
	}
	if (seenChecks[CheckSemanticApplicability] || seenChecks[CheckQuantitativeApplicability]) && len(request.Requirements) == 0 {
		diagnostics = append(diagnostics, contract.Diag(prefix+".requirements.empty", "requirements", "applicability checks require at least one requested requirement"))
	}
	seenRequirements := map[string]bool{}
	for index, requirement := range request.Requirements {
		location := fmt.Sprintf("requirements[%d]", index)
		if !contract.IsIRI(requirement) {
			diagnostics = append(diagnostics, contract.Diag(prefix+".iri.invalid", location, "expected an absolute IRI, got %q", requirement))
		}
		if seenRequirements[requirement] {
			diagnostics = append(diagnostics, contract.Diag(prefix+".requirement.duplicate", location, "requirement is requested more than once"))
		}
		seenRequirements[requirement] = true
	}
	return diagnostics
}

func validateAssessment(assessment Assessment) []contract.Diagnostic {
	var diagnostics []contract.Diagnostic
	prefix := "assessment"
	if assessment.AssessmentVersion == "" {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", "assessmentVersion", "field is required"))
	} else if assessment.AssessmentVersion != AssessmentVersionV01 {
		diagnostics = append(diagnostics, contract.Diag(prefix+".version.unsupported", "assessmentVersion", "supported assessment version is %s, got %q", AssessmentVersionV01, assessment.AssessmentVersion))
	}
	diagnostics = append(diagnostics, checkPackageRef(prefix, "package", assessment.Package)...)
	diagnostics = append(diagnostics, checkEntityRef(prefix, "reasoningProfile", assessment.ReasoningProfile)...)
	diagnostics = append(diagnostics, checkEntityRef(prefix, "evaluator", assessment.Evaluator)...)
	diagnostics = append(diagnostics, checkTimestampField(prefix, "startedAt", assessment.StartedAt)...)
	diagnostics = append(diagnostics, checkTimestampField(prefix, "completedAt", assessment.CompletedAt)...)
	if contract.IsTimestamp(assessment.StartedAt) && contract.IsTimestamp(assessment.CompletedAt) {
		started, _ := time.Parse(time.RFC3339Nano, assessment.StartedAt)
		completed, _ := time.Parse(time.RFC3339Nano, assessment.CompletedAt)
		if completed.Before(started) {
			diagnostics = append(diagnostics, contract.Diag(prefix+".timestamp.order", "completedAt", "completedAt precedes startedAt"))
		}
	}
	if len(assessment.Results) == 0 {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", "results", "at least one check result is required"))
	}
	seenResults := map[string]bool{}
	for index, result := range assessment.Results {
		location := fmt.Sprintf("results[%d]", index)
		diagnostics = append(diagnostics, validateResult(location, result)...)
		key := resultKey(result)
		if seenResults[key] {
			diagnostics = append(diagnostics, contract.Diag(prefix+".result.duplicate", location, "result identity (check, requirement, vector) is declared more than once"))
		}
		seenResults[key] = true
	}
	return diagnostics
}

func validateResult(location string, result CheckResult) []contract.Diagnostic {
	var diagnostics []contract.Diagnostic
	prefix := "assessment"
	if result.Check == "" {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", location+".check", "field is required"))
		return diagnostics
	}
	if !isCheck(result.Check) {
		diagnostics = append(diagnostics, contract.Diag(prefix+".check.invalid", location+".check", "unknown check %q", result.Check))
		return diagnostics
	}
	failureOutcome := false
	switch result.Outcome {
	case OutcomeConforms, OutcomeNonConforms:
	case OutcomeEvaluatorFailure, OutcomeUnsupported:
		failureOutcome = true
	case OutcomeIndeterminate:
		failureOutcome = true
		if result.Check != CheckSemanticApplicability && result.Check != CheckQuantitativeApplicability {
			diagnostics = append(diagnostics, contract.Diag(prefix+".result.outcome_forbidden", location+".outcome", "indeterminate is only defined for applicability checks"))
		}
	case "":
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", location+".outcome", "field is required"))
		return diagnostics
	default:
		diagnostics = append(diagnostics, contract.Diag(prefix+".outcome.invalid", location+".outcome", "unknown outcome %q", result.Outcome))
		return diagnostics
	}

	isVectorCheck := result.Check == CheckTestVectors
	if isVectorCheck && result.Vector == nil {
		diagnostics = append(diagnostics, contract.Diag(prefix+".result.vector_required", location, "a test-vectors result must identify its vector"))
	}
	if !isVectorCheck && result.Vector != nil {
		diagnostics = append(diagnostics, contract.Diag(prefix+".result.vector_forbidden", location, "only test-vectors results may carry a vector"))
	}
	requiresRequirement := result.Check == CheckSemanticApplicability || result.Check == CheckQuantitativeApplicability
	if requiresRequirement && result.Requirement == nil {
		diagnostics = append(diagnostics, contract.Diag(prefix+".result.requirement_required", location, "an applicability result must identify its requirement"))
	}
	if !requiresRequirement && result.Requirement != nil {
		diagnostics = append(diagnostics, contract.Diag(prefix+".result.requirement_forbidden", location, "a %s result must not carry a requirement identity", result.Check))
	}
	if result.Requirement != nil {
		diagnostics = append(diagnostics, checkEntityRef(prefix, location+".requirement", *result.Requirement)...)
	}
	if result.Message != "" && contract.IsBlank(result.Message) {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.blank", location+".message", "message must not be blank"))
	}

	if result.Violations == nil {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", location+".violations", "violations must be an array, possibly empty"))
	}
	for index, violation := range result.Violations {
		diagnostics = append(diagnostics, validateViolation(fmt.Sprintf("%s.violations[%d]", location, index), violation)...)
	}
	if result.EvidenceChecked == nil {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", location+".evidenceChecked", "evidenceChecked must be an array, possibly empty"))
	}
	seenEvidence := map[string]bool{}
	for index, evidence := range result.EvidenceChecked {
		evidenceLocation := fmt.Sprintf("%s.evidenceChecked[%d]", location, index)
		diagnostics = append(diagnostics, checkEvidenceRef(prefix, evidenceLocation, evidence)...)
		if seenEvidence[evidence.Path] {
			diagnostics = append(diagnostics, contract.Diag(prefix+".evidence.duplicate", evidenceLocation+".path", "evidence path is attested more than once"))
		}
		seenEvidence[evidence.Path] = true
	}

	if failureOutcome {
		if contract.IsBlank(result.Message) {
			diagnostics = append(diagnostics, contract.Diag(prefix+".result.message_required", location, "a %s result must explain itself in message", result.Outcome))
		}
		if len(result.Violations) != 0 {
			diagnostics = append(diagnostics, contract.Diag(prefix+".result.violations_invalid", location+".violations", "a %s result must not report violations", result.Outcome))
		}
		if result.Vector != nil && result.Vector.Actual != "" {
			diagnostics = append(diagnostics, contract.Diag(prefix+".result.vector_actual_forbidden", location+".vector.actual", "a %s result must not report an actual vector outcome", result.Outcome))
		}
	}

	if result.Vector != nil {
		diagnostics = append(diagnostics, validateVectorResult(location+".vector", result, failureOutcome)...)
	} else if !failureOutcome {
		switch result.Outcome {
		case OutcomeConforms:
			if len(result.Violations) != 0 {
				diagnostics = append(diagnostics, contract.Diag(prefix+".result.violations_invalid", location+".violations", "a conforms result must not report violations"))
			}
		case OutcomeNonConforms:
			if !hasBlockingViolation(result.Violations) {
				diagnostics = append(diagnostics, contract.Diag(prefix+".result.violations_invalid", location+".violations", "a non-conforms result must report at least one violation-severity entry"))
			}
		}
	}
	return diagnostics
}

func validateVectorResult(location string, result CheckResult, failureOutcome bool) []contract.Diagnostic {
	var diagnostics []contract.Diagnostic
	prefix := "assessment"
	vector := result.Vector
	diagnostics = append(diagnostics, checkIRIField(prefix, location+".id", vector.ID)...)
	if vector.Expected == "" {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", location+".expected", "field is required"))
		return diagnostics
	}
	if vector.Expected != OutcomeConforms && vector.Expected != OutcomeNonConforms {
		diagnostics = append(diagnostics, contract.Diag(prefix+".result.vector_expected_invalid", location+".expected", "expected must be conforms or non-conforms"))
		return diagnostics
	}
	if failureOutcome {
		return diagnostics
	}
	if vector.Actual != OutcomeConforms && vector.Actual != OutcomeNonConforms {
		diagnostics = append(diagnostics, contract.Diag(prefix+".result.vector_actual_required", location+".actual", "a decided vector result must record the actual outcome"))
		return diagnostics
	}
	if vector.Actual == OutcomeConforms && len(result.Violations) != 0 {
		diagnostics = append(diagnostics, contract.Diag(prefix+".result.violations_invalid", location, "a vector observed as conforms must not report violations"))
	}
	if vector.Actual == OutcomeNonConforms && !hasBlockingViolation(result.Violations) {
		diagnostics = append(diagnostics, contract.Diag(prefix+".result.violations_invalid", location, "a vector observed as non-conforms must report at least one violation-severity entry"))
	}
	matched := vector.Expected == vector.Actual
	if matched && result.Outcome != OutcomeConforms {
		diagnostics = append(diagnostics, contract.Diag(prefix+".result.vector_outcome_mismatch", location, "a vector matching its expectation must conform"))
	}
	if !matched && result.Outcome != OutcomeNonConforms {
		diagnostics = append(diagnostics, contract.Diag(prefix+".result.vector_outcome_mismatch", location, "a vector missing its expectation must not conform"))
	}
	return diagnostics
}

func validateViolation(location string, violation Violation) []contract.Diagnostic {
	prefix := "assessment"
	diagnostics := checkIRIField(prefix, location+".requirement", violation.Requirement)
	if violation.Severity == "" {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", location+".severity", "field is required"))
	} else if violation.Severity != "violation" && violation.Severity != "warning" && violation.Severity != "info" {
		diagnostics = append(diagnostics, contract.Diag(prefix+".severity.invalid", location+".severity", "severity must be violation, warning, or info"))
	}
	if contract.IsBlank(violation.Message) {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", location+".message", "violation message is required"))
	}
	if violation.Focus != "" && contract.IsBlank(violation.Focus) {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.blank", location+".focus", "focus must not be blank"))
	}
	if violation.Path != "" && contract.IsBlank(violation.Path) {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.blank", location+".path", "path must not be blank"))
	}
	if violation.Source != "" && !contract.IsNormalizedPath(violation.Source) {
		diagnostics = append(diagnostics, contract.Diag(prefix+".path.invalid", location+".source", "expected a normalized POSIX package-relative path, got %q", violation.Source))
	}
	return diagnostics
}

func validateSuite(suite Suite) []contract.Diagnostic {
	var diagnostics []contract.Diagnostic
	prefix := "suite"
	if suite.SuiteVersion == "" {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", "suiteVersion", "field is required"))
	} else if suite.SuiteVersion != SuiteVersionV01 {
		diagnostics = append(diagnostics, contract.Diag(prefix+".version.unsupported", "suiteVersion", "supported suite version is %s, got %q", SuiteVersionV01, suite.SuiteVersion))
	}
	diagnostics = append(diagnostics, checkPackageRef(prefix, "package", suite.Package)...)
	if len(suite.Vectors) == 0 {
		diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", "vectors", "at least one conformance vector is required"))
	}
	seen := map[string]bool{}
	categoriesByRequirement := map[string]map[string]bool{}
	for index, vector := range suite.Vectors {
		location := fmt.Sprintf("vectors[%d]", index)
		diagnostics = append(diagnostics, checkIRIField(prefix, location+".id", vector.ID)...)
		diagnostics = append(diagnostics, checkIRIField(prefix, location+".requirement", vector.Requirement)...)
		if seen[vector.ID] {
			diagnostics = append(diagnostics, contract.Diag(prefix+".vector.duplicate", location+".id", "vector is declared more than once"))
		}
		seen[vector.ID] = true
		if vector.Category == "" {
			diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", location+".category", "field is required"))
			continue
		}
		if vector.Category != "valid" && vector.Category != "invalid" && vector.Category != "boundary" {
			diagnostics = append(diagnostics, contract.Diag(prefix+".category.invalid", location+".category", "category must be valid, invalid, or boundary"))
			continue
		}
		if categoriesByRequirement[vector.Requirement] == nil {
			categoriesByRequirement[vector.Requirement] = map[string]bool{}
		}
		categoriesByRequirement[vector.Requirement][vector.Category] = true
		if vector.Expected == "" {
			diagnostics = append(diagnostics, contract.Diag(prefix+".field.required", location+".expected", "field is required"))
			continue
		}
		if vector.Expected != OutcomeConforms && vector.Expected != OutcomeNonConforms {
			diagnostics = append(diagnostics, contract.Diag(prefix+".expected.invalid", location+".expected", "expected must be conforms or non-conforms"))
			continue
		}
		if vector.Category == "valid" && vector.Expected != OutcomeConforms {
			diagnostics = append(diagnostics, contract.Diag(prefix+".category.expected_mismatch", location, "a valid vector must expect conforms"))
		}
		if vector.Category == "invalid" && vector.Expected != OutcomeNonConforms {
			diagnostics = append(diagnostics, contract.Diag(prefix+".category.expected_mismatch", location, "an invalid vector must expect non-conforms"))
		}
	}
	for requirement, categories := range categoriesByRequirement {
		for _, category := range []string{"valid", "invalid", "boundary"} {
			if !categories[category] {
				diagnostics = append(diagnostics, contract.Diag(prefix+".category.missing", "vectors", "requirement %s has no %s vector", requirement, category))
			}
		}
	}
	return diagnostics
}

// Absent required members decode to Go zero values; the documented contract
// reports them as <prefix>.field.required, never as a value-format error.
func checkField(prefix, location, value string, valid func(string) bool, code, format string) []contract.Diagnostic {
	if value == "" {
		return []contract.Diagnostic{contract.Diag(prefix+".field.required", location, "field is required")}
	}
	if !valid(value) {
		return []contract.Diagnostic{contract.Diag(prefix+"."+code, location, format, value)}
	}
	return nil
}

func checkIRIField(prefix, location, value string) []contract.Diagnostic {
	return checkField(prefix, location, value, contract.IsIRI, "iri.invalid", "expected an absolute IRI, got %q")
}

func checkVersionField(prefix, location, value string) []contract.Diagnostic {
	return checkField(prefix, location, value, contract.IsVersion, "version.invalid", "expected semantic version, got %q")
}

func checkDigestField(prefix, location, value string) []contract.Diagnostic {
	return checkField(prefix, location, value, contract.IsDigest, "digest.invalid", "expected lowercase sha256 digest, got %q")
}

func checkPathField(prefix, location, value string) []contract.Diagnostic {
	return checkField(prefix, location, value, contract.IsNormalizedPath, "path.invalid", "expected a normalized POSIX package-relative path, got %q")
}

func checkTimestampField(prefix, location, value string) []contract.Diagnostic {
	return checkField(prefix, location, value, contract.IsTimestamp, "timestamp.invalid", "expected an RFC 3339 UTC instant, got %q")
}

func checkPackageRef(prefix, location string, ref PackageRef) []contract.Diagnostic {
	diagnostics := checkIRIField(prefix, location+".id", ref.ID)
	diagnostics = append(diagnostics, checkVersionField(prefix, location+".version", ref.Version)...)
	diagnostics = append(diagnostics, checkDigestField(prefix, location+".digest", ref.Digest)...)
	return diagnostics
}

func checkEntityRef(prefix, location string, ref EntityRef) []contract.Diagnostic {
	diagnostics := checkIRIField(prefix, location+".id", ref.ID)
	diagnostics = append(diagnostics, checkVersionField(prefix, location+".version", ref.Version)...)
	return diagnostics
}

func checkEvidenceRef(prefix, location string, ref EvidenceRef) []contract.Diagnostic {
	diagnostics := checkPathField(prefix, location+".path", ref.Path)
	diagnostics = append(diagnostics, checkDigestField(prefix, location+".digest", ref.Digest)...)
	return diagnostics
}

func hasBlockingViolation(violations []Violation) bool {
	for _, violation := range violations {
		if violation.Severity == "violation" {
			return true
		}
	}
	return false
}

func isCheck(check string) bool {
	for _, known := range Checks {
		if check == known {
			return true
		}
	}
	return false
}
