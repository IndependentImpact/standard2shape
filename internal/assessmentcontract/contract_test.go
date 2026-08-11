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

	if err := CheckAgainstRequest(request, assessment); err != nil {
		t.Fatalf("valid assessment must answer the fixture request: %v", err)
	}
	pkg, err := packagecontract.Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
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
		CheckResult{
			Check:           CheckSemanticApplicability,
			Outcome:         OutcomeConforms,
			Requirement:     &EntityRef{ID: "https://example.org/standard/DemoMethodologyV1/requirements/semantic-scope", Version: "1.0.0"},
			Violations:      []Violation{},
			EvidenceChecked: append([]EvidenceRef{}, request.Evidence...),
		},
		CheckResult{
			Check:           CheckQuantitativeApplicability,
			Outcome:         OutcomeUnsupported,
			Requirement:     &EntityRef{ID: "https://example.org/standard/DemoMethodologyV1/requirements/minimum-annual-yield", Version: "1.0.0"},
			Message:         "no quantitative evaluation binding is available to this evaluator",
			Violations:      []Violation{},
			EvidenceChecked: []EvidenceRef{},
		},
	)
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
		if err := CheckAgainstRequest(request, assessment); err != nil {
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
