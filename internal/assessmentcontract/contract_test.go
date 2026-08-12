package assessmentcontract

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/IndependentImpact/standard2shape/internal/contract"
	"github.com/IndependentImpact/standard2shape/internal/packagecontract"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "assessment-v0", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestValidFixturesDecodeAndCrossCheck(t *testing.T) {
	request, err := DecodeRequest(fixture(t, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := DecodeAssessment(fixture(t, "assessment-valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeAssessment(fixture(t, "assessment-non-conforming.json")); err != nil {
		t.Fatal(err)
	}
	suite, err := DecodeSuite(fixture(t, "suite.json"))
	if err != nil {
		t.Fatal(err)
	}

	pkg, err := packagecontract.Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckAgainstRequest(request, assessment, pkg); err != nil {
		t.Fatalf("valid assessment must answer the fixture request: %v", err)
	}
	if err := CheckSuiteAgainstPackage(suite, pkg); err != nil {
		t.Fatalf("suite must cover the tracer package: %v", err)
	}
	if request.Package.Digest != packagecontract.Digest(pkg.NormalizedManifest) {
		t.Fatalf("fixture package digest is stale: expected %s", packagecontract.Digest(pkg.NormalizedManifest))
	}
}

func TestInvalidFixturesHaveStableDiagnostics(t *testing.T) {
	tests := []struct {
		fixture string
		code    string
	}{
		{fixture: "invalid-conforms-with-violations.json", code: "assessment.result.violations_invalid"},
		{fixture: "invalid-missing-message.json", code: "assessment.result.message_required"},
		{fixture: "invalid-vector-outcome-mismatch.json", code: "assessment.result.vector_outcome_mismatch"},
		{fixture: "invalid-suite-missing-boundary.json", code: "suite.category.missing"},
		{fixture: "invalid-suite-category-mismatch.json", code: "suite.category.expected_mismatch"},
	}
	for _, test := range tests {
		t.Run(test.fixture, func(t *testing.T) {
			data := fixture(t, test.fixture)
			decode := func() error {
				if bytes.Contains(data, []byte("suiteVersion")) {
					_, err := DecodeSuite(data)
					return err
				}
				_, err := DecodeAssessment(data)
				return err
			}
			err := decode()
			var contractErr *contract.Error
			if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, test.code) {
				t.Fatalf("missing diagnostic %q, got %v", test.code, err)
			}
			repeated := decode()
			if repeated == nil || repeated.Error() != err.Error() {
				t.Fatalf("diagnostic is not stable:\nfirst:  %v\nsecond: %v", err, repeated)
			}
		})
	}
}

func TestAssessmentNormalizationIsDeterministic(t *testing.T) {
	assessment, err := DecodeAssessment(fixture(t, "assessment-valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := NormalizeAssessment(assessment)
	if err != nil {
		t.Fatal(err)
	}

	reordered := assessment
	reordered.Results = append([]CheckResult{}, assessment.Results...)
	slices.Reverse(reordered.Results)
	for index, result := range reordered.Results {
		result.Violations = append([]Violation{}, result.Violations...)
		result.EvidenceChecked = append([]EvidenceRef{}, result.EvidenceChecked...)
		slices.Reverse(result.Violations)
		slices.Reverse(result.EvidenceChecked)
		reordered.Results[index] = result
	}
	renormalized, err := NormalizeAssessment(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(normalized, renormalized) {
		t.Fatalf("normalization depends on input order:\nfirst:\n%s\nsecond:\n%s", normalized, renormalized)
	}

	roundTripped, err := DecodeAssessment(normalized)
	if err != nil {
		t.Fatalf("normalized assessment violates its own contract: %v", err)
	}
	if _, err := SemanticBytes(roundTripped); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizationOrdersBySeverity(t *testing.T) {
	assessment, err := DecodeAssessment(fixture(t, "assessment-non-conforming.json"))
	if err != nil {
		t.Fatal(err)
	}
	base := assessment.Results[0].Violations[0]
	warning := base
	warning.Severity = "warning"
	assessment.Results[0].Violations = []Violation{base, warning}
	first, err := NormalizeAssessment(assessment)
	if err != nil {
		t.Fatal(err)
	}
	assessment.Results[0].Violations = []Violation{warning, base}
	second, err := NormalizeAssessment(assessment)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("violation order depends on input severity order:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestExplicitNullFieldsAreRejected(t *testing.T) {
	data := fixture(t, "assessment-non-conforming.json")
	nullified := bytes.Replace(data, []byte(`"message": "the evidence does not describe the project scope this requirement constrains"`), []byte(`"message": null`), 1)
	_, err := DecodeAssessment(nullified)
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.field.null") {
		t.Fatalf("expected explicit-null diagnostic, got %v", err)
	}
}

func TestMissingRequiredFieldsReportFieldRequired(t *testing.T) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(fixture(t, "request.json"), &fields); err != nil {
		t.Fatal(err)
	}
	delete(fields, "requestVersion")
	delete(fields, "package")
	stripped, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	_, err = DecodeRequest(stripped)
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) {
		t.Fatalf("error type = %T: %v", err, err)
	}
	for _, location := range []string{"requestVersion", "package.id", "package.version", "package.digest"} {
		found := slices.ContainsFunc(contractErr.Diagnostics, func(d contract.Diagnostic) bool {
			return d.Code == "request.field.required" && d.Location == location
		})
		if !found {
			t.Fatalf("missing request.field.required at %s in %#v", location, contractErr.Diagnostics)
		}
	}
}

func TestSuiteCoverageIsPerRequirement(t *testing.T) {
	suite, err := DecodeSuite(fixture(t, "suite.json"))
	if err != nil {
		t.Fatal(err)
	}
	suite.Vectors = append(suite.Vectors, SuiteVector{
		ID:          "https://example.org/standards/demo/tests/second-requirement-valid",
		Requirement: "https://example.org/standard/AnotherRequirement",
		Category:    "valid",
		Expected:    "conforms",
	})
	err = contract.ErrorFor(validateSuite(suite))
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "suite.category.missing") {
		t.Fatalf("a requirement with partial category coverage must be rejected, got %v", err)
	}
}

func TestIndeterminateIsOnlyForApplicabilityChecks(t *testing.T) {
	assessment, err := DecodeAssessment(fixture(t, "assessment-valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	assessment.Results[0] = CheckResult{
		Check:           CheckSyntax,
		Outcome:         OutcomeIndeterminate,
		Message:         "cannot decide",
		Violations:      []Violation{},
		EvidenceChecked: []EvidenceRef{},
	}
	err = contract.ErrorFor(validateAssessment(assessment))
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.result.outcome_forbidden") {
		t.Fatalf("indeterminate outside applicability checks must be rejected, got %v", err)
	}
}

func TestAssessmentMustAnswerRequestedRequirementsAndEvidence(t *testing.T) {
	request, err := DecodeRequest(fixture(t, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := DecodeAssessment(fixture(t, "assessment-valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := packagecontract.Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}

	dropped := assessment
	dropped.Results = slices.DeleteFunc(append([]CheckResult{}, assessment.Results...), func(result CheckResult) bool {
		return result.Check == CheckSemanticApplicability
	})
	err = CheckAgainstRequest(request, dropped, pkg)
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.request.requirement_missing") {
		t.Fatalf("dropping a requested requirement must be rejected, got %v", err)
	}

	substituted := assessment
	substituted.Results = append([]CheckResult{}, assessment.Results...)
	for index, result := range substituted.Results {
		if result.Check == CheckSemanticApplicability {
			swapped := *result.Requirement
			swapped.ID = "https://example.org/standard/SomethingElse"
			result.Requirement = &swapped
			substituted.Results[index] = result
		}
	}
	err = CheckAgainstRequest(request, substituted, pkg)
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.request.requirement_unrequested") {
		t.Fatalf("substituting a requirement must be rejected, got %v", err)
	}

	tampered := assessment
	tampered.Results = append([]CheckResult{}, assessment.Results...)
	for index, result := range tampered.Results {
		result.EvidenceChecked = append([]EvidenceRef{}, result.EvidenceChecked...)
		for evidenceIndex, evidence := range result.EvidenceChecked {
			if evidence.Path == "data-valid.ttl" {
				evidence.Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				result.EvidenceChecked[evidenceIndex] = evidence
			}
		}
		tampered.Results[index] = result
	}
	err = CheckAgainstRequest(request, tampered, pkg)
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.request.evidence_mismatch") {
		t.Fatalf("substituting requested evidence must be rejected, got %v", err)
	}
	if !hasDiagnostic(contractErr.Diagnostics, "assessment.request.evidence_missing") {
		t.Fatalf("dropping requested evidence must be rejected, got %v", err)
	}
}

func TestSuiteRequirementLabelsAreBoundToTheManifest(t *testing.T) {
	suite, err := DecodeSuite(fixture(t, "suite.json"))
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := packagecontract.Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}
	suite.Vectors[0].Requirement = "https://example.org/standard/RelabelledRequirement"
	suite.Vectors[0].Category = "valid"
	err = CheckSuiteAgainstPackage(suite, pkg)
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "suite.vector.requirement_mismatch") {
		t.Fatalf("relabelling a vector's requirement must be rejected, got %v", err)
	}
}

func TestAssessmentEvidenceMustBeRequestedOrPackageMembers(t *testing.T) {
	request, err := DecodeRequest(fixture(t, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := DecodeAssessment(fixture(t, "assessment-valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := packagecontract.Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}

	foreign := assessment
	foreign.Results = append([]CheckResult{}, assessment.Results...)
	result := foreign.Results[0]
	result.EvidenceChecked = append(append([]EvidenceRef{}, result.EvidenceChecked...), EvidenceRef{
		Path:   "somewhere-else.ttl",
		Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	})
	foreign.Results[0] = result
	err = CheckAgainstRequest(request, foreign, pkg)
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.request.evidence_unknown") {
		t.Fatalf("attesting a non-member path must be rejected, got %v", err)
	}

	tamperedMember := assessment
	tamperedMember.Results = append([]CheckResult{}, assessment.Results...)
	memberResult := tamperedMember.Results[0]
	memberResult.EvidenceChecked = append([]EvidenceRef{}, memberResult.EvidenceChecked...)
	for index, evidence := range memberResult.EvidenceChecked {
		if evidence.Path == "shapes.ttl" {
			evidence.Digest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
			memberResult.EvidenceChecked[index] = evidence
		}
	}
	tamperedMember.Results[0] = memberResult
	err = CheckAgainstRequest(request, tamperedMember, pkg)
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.request.evidence_mismatch") {
		t.Fatalf("attesting a package member under a foreign digest must be rejected, got %v", err)
	}
}

func TestRequestedTestVectorsRequireCompleteResults(t *testing.T) {
	request, err := DecodeRequest(fixture(t, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := DecodeAssessment(fixture(t, "assessment-valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := packagecontract.Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}
	assessment.Results = slices.DeleteFunc(append([]CheckResult{}, assessment.Results...), func(result CheckResult) bool {
		return result.Vector != nil && result.Vector.ID == "https://example.org/standards/demo/tests/missing-title"
	})
	err = CheckAgainstRequest(request, assessment, pkg)
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.request.vector_missing") {
		t.Fatalf("omitting a declared vector must be rejected, got %v", err)
	}
}

func TestApplicabilityRequestsRequireRequirements(t *testing.T) {
	requestData := bytes.Replace(fixture(t, "request.json"), []byte(`"requirements": [
    "https://example.org/standard/ProjectTitleRequirement"
  ]`), []byte(`"requirements": []`), 1)
	_, err := DecodeRequest(requestData)
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "request.requirements.empty") {
		t.Fatalf("applicability checks without requirements must be rejected, got %v", err)
	}
	requestSchema := compileSchema(t, "validation-request.schema.json")
	if err := validateAgainstSchema(t, requestSchema, requestData); err == nil {
		t.Fatal("the schema must also reject applicability checks without requirements")
	}
}

func TestEmptyAndBlankStringsAgreeWithSchemas(t *testing.T) {
	assessmentSchema := compileSchema(t, "assessment.schema.json")
	base := fixture(t, "assessment-non-conforming.json")
	target := []byte(`"message": "the evidence does not describe the project scope this requirement constrains"`)

	empty := bytes.Replace(base, target, []byte(`"message": ""`), 1)
	_, err := DecodeAssessment(empty)
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.field.empty") {
		t.Fatalf("explicit empty message must be rejected, got %v", err)
	}
	if err := validateAgainstSchema(t, assessmentSchema, empty); err == nil {
		t.Fatal("the schema must also reject an explicit empty message")
	}

	blank := bytes.Replace(base, target, []byte(`"message": " \t "`), 1)
	if _, err := DecodeAssessment(blank); err == nil {
		t.Fatal("whitespace-only message must be rejected by the verifier")
	}
	if err := validateAgainstSchema(t, assessmentSchema, blank); err == nil {
		t.Fatal("the schema must also reject a whitespace-only message")
	}

	blankFocus := bytes.Replace(base, []byte(`"focus": "https://example.org/standard/ProjectWithoutTitle"`), []byte(`"focus": " \t "`), 1)
	_, err = DecodeAssessment(blankFocus)
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.field.blank") {
		t.Fatalf("whitespace-only focus must be rejected, got %v", err)
	}
	if err := validateAgainstSchema(t, assessmentSchema, blankFocus); err == nil {
		t.Fatal("the schema must also reject a whitespace-only focus")
	}

	badIRI := bytes.Replace(base, []byte(`"id": "https://standard2shape.dev/evaluators/local-tracer"`), []byte(`"id": "not-an-iri"`), 1)
	if _, err := DecodeAssessment(badIRI); err == nil {
		t.Fatal("a non-IRI evaluator id must be rejected by the verifier")
	}
	if err := validateAgainstSchema(t, assessmentSchema, badIRI); err == nil {
		t.Fatal("the schema must also reject a non-IRI evaluator id with format assertion")
	}
}

func TestRequestMustBindToSuppliedPackage(t *testing.T) {
	request, err := DecodeRequest(fixture(t, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := DecodeAssessment(fixture(t, "assessment-valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := packagecontract.Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}

	repinned := request
	repinned.Package.Digest = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	tampered := assessment
	tampered.Package = repinned.Package
	err = CheckAgainstRequest(repinned, tampered, pkg)
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.request.package_unbound") {
		t.Fatalf("a request pinned to a different package digest must be rejected, got %v", err)
	}

	reprofiled := request
	reprofiled.ReasoningProfile = EntityRef{ID: "https://example.org/standard/OtherProfile", Version: "2.0.0"}
	reprofiledAssessment := assessment
	reprofiledAssessment.ReasoningProfile = reprofiled.ReasoningProfile
	err = CheckAgainstRequest(reprofiled, reprofiledAssessment, pkg)
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.request.profile_unbound") {
		t.Fatalf("a request naming a foreign reasoning profile must be rejected, got %v", err)
	}
}

func TestUnicodeBlankStringsAreRejected(t *testing.T) {
	base := fixture(t, "assessment-non-conforming.json")
	target := []byte(`"message": "the evidence does not describe the project scope this requirement constrains"`)
	nbspOnly := bytes.Replace(base, target, []byte("\"message\": \"\u00a0\u00a0\""), 1)
	_, err := DecodeAssessment(nbspOnly)
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.result.message_required") {
		t.Fatalf("a message of only Unicode whitespace must be blank under ECMA semantics, got %v", err)
	}
}

func TestRequirementKindPairingIsEnforced(t *testing.T) {
	request, err := DecodeRequest(fixture(t, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := DecodeAssessment(fixture(t, "assessment-valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := packagecontract.Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}
	assessment.Results = append([]CheckResult{}, assessment.Results...)
	for index, result := range assessment.Results {
		if result.Check == CheckSemanticApplicability {
			result.Check = CheckQuantitativeApplicability
			result.Outcome = OutcomeUnsupported
			result.Message = "swapped evaluator semantics"
			result.EvidenceChecked = []EvidenceRef{}
			assessment.Results[index] = result
		}
	}
	err = CheckAgainstRequest(request, assessment, pkg)
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.request.requirement_kind_mismatch") {
		t.Fatalf("answering a semantic requirement with a quantitative check must be rejected, got %v", err)
	}
	if !hasDiagnostic(contractErr.Diagnostics, "assessment.request.requirement_missing") {
		t.Fatalf("the swapped requirement must not count as answered, got %v", err)
	}
}

func TestApplicabilityChecksNeedKindMatchedRequirements(t *testing.T) {
	request, err := DecodeRequest(fixture(t, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := DecodeAssessment(fixture(t, "assessment-valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := packagecontract.Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}
	request.Checks = append(append([]string{}, request.Checks...), CheckQuantitativeApplicability)
	err = CheckAgainstRequest(request, assessment, pkg)
	var contractErr *contract.Error
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "assessment.request.check_unanswerable") {
		t.Fatalf("a quantitative check without a quantitative requirement must be rejected, got %v", err)
	}
}

// localAdapter independently derives an assessment from the package on disk,
// faking every evaluation as its deterministic canned outcome.
type localAdapter struct {
	root string
}

func (adapter localAdapter) Evaluate(request Request) (Assessment, error) {
	pkg, err := packagecontract.Open(adapter.root)
	if err != nil {
		return Assessment{}, err
	}
	artifactDigests := map[string]string{}
	for _, artifact := range pkg.Manifest.Artifacts {
		artifactDigests[artifact.Path] = artifact.Digest
	}
	assessment := Assessment{
		AssessmentVersion: AssessmentVersionV01,
		Package:           request.Package,
		ReasoningProfile:  request.ReasoningProfile,
		Evaluator:         EntityRef{ID: "https://standard2shape.dev/evaluators/local-tracer", Version: "0.1.0"},
		StartedAt:         "2026-08-11T08:00:00Z",
		CompletedAt:       "2026-08-11T08:00:02Z",
	}

	syntaxEvidence := []EvidenceRef{}
	for _, artifact := range pkg.Manifest.Artifacts {
		syntaxEvidence = append(syntaxEvidence, EvidenceRef{Path: artifact.Path, Digest: artifact.Digest})
	}
	assessment.Results = append(assessment.Results,
		CheckResult{Check: CheckSyntax, Outcome: OutcomeConforms, Violations: []Violation{}, EvidenceChecked: syntaxEvidence},
		CheckResult{Check: CheckPackageStructure, Outcome: OutcomeConforms, Violations: []Violation{}, EvidenceChecked: []EvidenceRef{}},
		CheckResult{Check: CheckSHACL, Outcome: OutcomeConforms, Violations: []Violation{}, EvidenceChecked: append([]EvidenceRef{}, request.Evidence...)},
		CheckResult{Check: CheckReasoningProfile, Outcome: OutcomeConforms, Violations: []Violation{}, EvidenceChecked: []EvidenceRef{{Path: pkg.Manifest.ReasoningProfile.Source, Digest: artifactDigests[pkg.Manifest.ReasoningProfile.Source]}}},
	)
	declarations := map[string]packagecontract.RequirementDeclaration{}
	for _, declaration := range pkg.Manifest.Requirements {
		declarations[declaration.ID] = declaration
	}
	for _, requirement := range request.Requirements {
		declaration := declarations[requirement]
		if declaration.Kind == "semantic" {
			assessment.Results = append(assessment.Results, CheckResult{
				Check:           CheckSemanticApplicability,
				Outcome:         OutcomeConforms,
				Requirement:     &EntityRef{ID: declaration.ID, Version: declaration.Version},
				Violations:      []Violation{},
				EvidenceChecked: append([]EvidenceRef{}, request.Evidence...),
			})
			continue
		}
		assessment.Results = append(assessment.Results, CheckResult{
			Check:           CheckQuantitativeApplicability,
			Outcome:         OutcomeUnsupported,
			Requirement:     &EntityRef{ID: declaration.ID, Version: declaration.Version},
			Message:         "no quantitative evaluation binding is available to this evaluator",
			Violations:      []Violation{},
			EvidenceChecked: []EvidenceRef{},
		})
	}
	for _, vector := range pkg.Manifest.ConformanceVectors {
		violations := []Violation{}
		if vector.Expected == OutcomeNonConforms {
			violations = append(violations, Violation{
				Requirement: "https://example.org/standard/ProjectTitleShape",
				Severity:    "violation",
				Message:     "a project requires exactly one title",
				Focus:       "https://example.org/standard/ProjectWithoutTitle",
				Path:        "https://example.org/standard/title",
				Source:      vector.Path,
			})
		}
		assessment.Results = append(assessment.Results, CheckResult{
			Check:           CheckTestVectors,
			Outcome:         OutcomeConforms,
			Violations:      violations,
			EvidenceChecked: []EvidenceRef{{Path: vector.Path, Digest: vector.Digest}},
			Vector:          &VectorResult{ID: vector.ID, Expected: vector.Expected, Actual: vector.Expected},
		})
	}
	return assessment, nil
}

// hostedAdapter replays the same normative content from its own store with a
// different evaluator identity, run timestamps, and internal ordering.
type hostedAdapter struct {
	stored []byte
}

func (adapter hostedAdapter) Evaluate(request Request) (Assessment, error) {
	assessment, err := DecodeAssessment(adapter.stored)
	if err != nil {
		return Assessment{}, err
	}
	assessment.Evaluator = EntityRef{ID: "https://ii.example.org/evaluators/hosted-runner", Version: "3.2.1"}
	assessment.StartedAt = "2026-08-11T09:30:00.250Z"
	assessment.CompletedAt = "2026-08-11T09:30:04Z"
	slices.Reverse(assessment.Results)
	return assessment, nil
}

func TestEquivalentAdaptersProduceEquivalentAssessments(t *testing.T) {
	request, err := DecodeRequest(fixture(t, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := packagecontract.Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}
	adapters := []Adapter{
		localAdapter{root: filepath.Join("..", "..", "fixtures", "tracer")},
		hostedAdapter{stored: fixture(t, "assessment-valid.json")},
	}

	var semantic [][]byte
	var full [][]byte
	for _, adapter := range adapters {
		assessment, err := adapter.Evaluate(request)
		if err != nil {
			t.Fatal(err)
		}
		if err := CheckAgainstRequest(request, assessment, pkg); err != nil {
			t.Fatalf("adapter %T does not answer the request: %v", adapter, err)
		}
		semanticBytes, err := SemanticBytes(assessment)
		if err != nil {
			t.Fatal(err)
		}
		semantic = append(semantic, semanticBytes)
		normalized, err := NormalizeAssessment(assessment)
		if err != nil {
			t.Fatal(err)
		}
		full = append(full, normalized)
	}

	if !bytes.Equal(semantic[0], semantic[1]) {
		t.Fatalf("adapters disagree on normative content:\nlocal:\n%s\nhosted:\n%s", semantic[0], semantic[1])
	}
	if bytes.Equal(full[0], full[1]) {
		t.Fatal("full normalization must preserve distinct evaluator attestations")
	}
}

func TestSchemaValuePatternsMatchVerifier(t *testing.T) {
	schemaData, err := os.ReadFile(filepath.Join("..", "..", "contracts", "v0", "assessment.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs struct {
			Path struct {
				Pattern string `json:"pattern"`
			} `json:"path"`
			Timestamp struct {
				Pattern string `json:"pattern"`
			} `json:"timestamp"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		t.Fatal(err)
	}
	pathPattern := regexp.MustCompile(schema.Defs.Path.Pattern)
	timestampPattern := regexp.MustCompile(schema.Defs.Timestamp.Pattern)

	paths := []struct {
		value string
		valid bool
	}{
		{value: "data-valid.ttl", valid: true},
		{value: "evidence/data.ttl", valid: true},
		{value: "./data.ttl", valid: false},
		{value: "evidence//data.ttl", valid: false},
		{value: "/data.ttl", valid: false},
		{value: "../data.ttl", valid: false},
	}
	for _, test := range paths {
		if pathPattern.MatchString(test.value) != test.valid || contract.IsNormalizedPath(test.value) != test.valid {
			t.Errorf("path %q: expected valid=%t, schema=%t, verifier=%t", test.value, test.valid, pathPattern.MatchString(test.value), contract.IsNormalizedPath(test.value))
		}
	}

	timestamps := []struct {
		value string
		valid bool
	}{
		{value: "2026-08-11T08:00:00Z", valid: true},
		{value: "2026-08-11T23:59:59.999999999Z", valid: true},
		{value: "2026-08-11T08:00:00+02:00", valid: false},
		{value: "2026-08-11T08:00Z", valid: false},
		{value: "2026-08-11 08:00:00Z", valid: false},
		{value: "2026-13-11T08:00:00Z", valid: false},
		{value: "", valid: false},
	}
	for _, test := range timestamps {
		if timestampPattern.MatchString(test.value) != test.valid || contract.IsTimestamp(test.value) != test.valid {
			t.Errorf("timestamp %q: expected valid=%t, schema=%t, verifier=%t", test.value, test.valid, timestampPattern.MatchString(test.value), contract.IsTimestamp(test.value))
		}
	}

	// The schema pattern is a syntactic bound: calendar-invalid instants pass
	// it but the verifier rejects them, as documented in the contract.
	calendarInvalid := "2026-02-30T08:00:00Z"
	if !timestampPattern.MatchString(calendarInvalid) || contract.IsTimestamp(calendarInvalid) {
		t.Errorf("calendar-invalid instant: schema=%t (want true), verifier=%t (want false)", timestampPattern.MatchString(calendarInvalid), contract.IsTimestamp(calendarInvalid))
	}
}

func compileSchema(t *testing.T, name string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	compiler.RegisterFormat(&jsonschema.Format{
		Name: "iri",
		Validate: func(v any) error {
			value, ok := v.(string)
			if !ok {
				return nil
			}
			if !contract.IsIRI(value) {
				return errors.New("not an absolute IRI")
			}
			return nil
		},
	})
	schema, err := compiler.Compile(filepath.Join("..", "..", "contracts", "v0", name))
	if err != nil {
		t.Fatalf("compile %s: %v", name, err)
	}
	return schema
}

func validateAgainstSchema(t *testing.T, schema *jsonschema.Schema, data []byte) error {
	t.Helper()
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return schema.Validate(document)
}

func TestFixturesValidateAgainstPublishedSchemas(t *testing.T) {
	requestSchema := compileSchema(t, "validation-request.schema.json")
	assessmentSchema := compileSchema(t, "assessment.schema.json")
	suiteSchema := compileSchema(t, "conformance-suite.schema.json")
	manifestSchema := compileSchema(t, "package-manifest.schema.json")

	positives := []struct {
		schema *jsonschema.Schema
		data   []byte
	}{
		{schema: requestSchema, data: fixture(t, "request.json")},
		{schema: assessmentSchema, data: fixture(t, "assessment-valid.json")},
		{schema: assessmentSchema, data: fixture(t, "assessment-non-conforming.json")},
		{schema: suiteSchema, data: fixture(t, "suite.json")},
	}
	manifestData, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "tracer", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	positives = append(positives, struct {
		schema *jsonschema.Schema
		data   []byte
	}{schema: manifestSchema, data: manifestData})
	for index, positive := range positives {
		if err := validateAgainstSchema(t, positive.schema, positive.data); err != nil {
			t.Errorf("positive fixture %d fails its published schema: %v", index, err)
		}
	}

	negatives := []struct {
		schema  *jsonschema.Schema
		fixture string
	}{
		{schema: assessmentSchema, fixture: "invalid-conforms-with-violations.json"},
		{schema: assessmentSchema, fixture: "invalid-missing-message.json"},
		{schema: assessmentSchema, fixture: "invalid-vector-outcome-mismatch.json"},
		{schema: suiteSchema, fixture: "invalid-suite-missing-boundary.json"},
		{schema: suiteSchema, fixture: "invalid-suite-category-mismatch.json"},
	}
	for _, negative := range negatives {
		if err := validateAgainstSchema(t, negative.schema, fixture(t, negative.fixture)); err == nil {
			t.Errorf("negative fixture %s passes the published schema", negative.fixture)
		}
	}
}

func TestNormalizedDocumentsSatisfyPublishedSchemas(t *testing.T) {
	requestSchema := compileSchema(t, "validation-request.schema.json")
	assessmentSchema := compileSchema(t, "assessment.schema.json")
	suiteSchema := compileSchema(t, "conformance-suite.schema.json")

	request, err := DecodeRequest(fixture(t, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	request.Checks = []string{CheckSyntax, CheckPackageStructure}
	request.Evidence = []EvidenceRef{}
	request.Requirements = []string{}
	normalizedRequest, err := NormalizeRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateAgainstSchema(t, requestSchema, normalizedRequest); err != nil {
		t.Errorf("normalized request violates its schema: %v", err)
	}

	assessment, err := DecodeAssessment(fixture(t, "assessment-valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	normalizedAssessment, err := NormalizeAssessment(assessment)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateAgainstSchema(t, assessmentSchema, normalizedAssessment); err != nil {
		t.Errorf("normalized assessment violates its schema: %v", err)
	}

	suite, err := DecodeSuite(fixture(t, "suite.json"))
	if err != nil {
		t.Fatal(err)
	}
	normalizedSuite, err := NormalizeSuite(suite)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateAgainstSchema(t, suiteSchema, normalizedSuite); err != nil {
		t.Errorf("normalized suite violates its schema: %v", err)
	}
}

func TestVersionedContractArtifactsParseLocally(t *testing.T) {
	for _, name := range []string{"validation-request.schema.json", "assessment.schema.json", "conformance-suite.schema.json"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "contracts", "v0", name))
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if schema["$id"] != "https://standard2shape.dev/contracts/v0/"+name {
			t.Fatalf("%s schema id = %v", name, schema["$id"])
		}
	}
}

func hasDiagnostic(diagnostics []contract.Diagnostic, code string) bool {
	return slices.ContainsFunc(diagnostics, func(diagnostic contract.Diagnostic) bool { return diagnostic.Code == code })
}
