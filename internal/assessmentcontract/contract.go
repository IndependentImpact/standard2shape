package assessmentcontract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/IndependentImpact/standard2shape/internal/contract"
)

var (
	packageRefSpec = contract.ObjectSpec{"id": nil, "version": nil, "digest": nil}
	entityRefSpec  = contract.ObjectSpec{"id": nil, "version": nil}
	evidenceSpec   = contract.ObjectSpec{"path": nil, "digest": nil}
	violationSpec  = contract.ObjectSpec{"requirement": nil, "severity": nil, "message": nil, "focus": contract.NonEmpty{}, "path": contract.NonEmpty{}, "source": contract.NonEmpty{}}
	vectorSpec     = contract.ObjectSpec{"id": nil, "expected": nil, "actual": contract.NonEmpty{}}
	resultSpec     = contract.ObjectSpec{
		"check":           nil,
		"outcome":         nil,
		"requirement":     entityRefSpec,
		"message":         contract.NonEmpty{},
		"violations":      contract.ArraySpec{Element: violationSpec},
		"evidenceChecked": contract.ArraySpec{Element: evidenceSpec},
		"vector":          vectorSpec,
	}

	requestSpec = contract.ObjectSpec{
		"requestVersion":   nil,
		"package":          packageRefSpec,
		"reasoningProfile": entityRefSpec,
		"checks":           contract.ArraySpec{Element: nil},
		"evidence":         contract.ArraySpec{Element: evidenceSpec},
		"requirements":     contract.ArraySpec{Element: nil},
	}
	assessmentSpec = contract.ObjectSpec{
		"assessmentVersion": nil,
		"package":           packageRefSpec,
		"reasoningProfile":  entityRefSpec,
		"evaluator":         entityRefSpec,
		"startedAt":         nil,
		"completedAt":       nil,
		"results":           contract.ArraySpec{Element: resultSpec},
	}
	suiteSpec = contract.ObjectSpec{
		"suiteVersion": nil,
		"package":      packageRefSpec,
		"vectors":      contract.ArraySpec{Element: contract.ObjectSpec{"id": nil, "requirement": nil, "category": nil, "expected": nil}},
	}
)

func DecodeRequest(data []byte) (Request, error) {
	var request Request
	if err := decodeClosed(data, "request", "validation-request", requestSpec, &request); err != nil {
		return Request{}, err
	}
	if err := contract.ErrorFor(validateRequest(request)); err != nil {
		return Request{}, err
	}
	return request, nil
}

func DecodeAssessment(data []byte) (Assessment, error) {
	var assessment Assessment
	if err := decodeClosed(data, "assessment", "assessment", assessmentSpec, &assessment); err != nil {
		return Assessment{}, err
	}
	if err := contract.ErrorFor(validateAssessment(assessment)); err != nil {
		return Assessment{}, err
	}
	return assessment, nil
}

func DecodeSuite(data []byte) (Suite, error) {
	var suite Suite
	if err := decodeClosed(data, "suite", "conformance-suite", suiteSpec, &suite); err != nil {
		return Suite{}, err
	}
	if err := contract.ErrorFor(validateSuite(suite)); err != nil {
		return Suite{}, err
	}
	return suite, nil
}

func decodeClosed(data []byte, codePrefix, document string, spec contract.ObjectSpec, target any) error {
	if json.Valid(data) {
		if err := contract.ErrorFor(contract.CheckExactFields(data, codePrefix, document, spec)); err != nil {
			return err
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return contract.ErrorFor([]contract.Diagnostic{contract.Diag(codePrefix+".invalid", document, "cannot decode closed v0.1 contract: %v", err)})
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return contract.ErrorFor([]contract.Diagnostic{contract.Diag(codePrefix+".invalid", document, "invalid trailing content: %v", err)})
	}
	return nil
}

func NormalizeRequest(request Request) ([]byte, error) {
	if err := contract.ErrorFor(validateRequest(request)); err != nil {
		return nil, err
	}
	normalized := request
	normalized.Checks = append([]string{}, request.Checks...)
	normalized.Evidence = append([]EvidenceRef{}, request.Evidence...)
	normalized.Requirements = append([]string{}, request.Requirements...)
	sort.Strings(normalized.Checks)
	sort.Strings(normalized.Requirements)
	sort.Slice(normalized.Evidence, func(i, j int) bool { return normalized.Evidence[i].Path < normalized.Evidence[j].Path })
	return marshalContract(normalized)
}

func NormalizeAssessment(assessment Assessment) ([]byte, error) {
	if err := contract.ErrorFor(validateAssessment(assessment)); err != nil {
		return nil, err
	}
	return marshalContract(sortedAssessment(assessment))
}

// SemanticBytes serializes an assessment's normative content: evaluator
// identity and timestamps are attestation metadata, so two assessments are
// semantically equivalent exactly when their SemanticBytes are equal.
func SemanticBytes(assessment Assessment) ([]byte, error) {
	if err := contract.ErrorFor(validateAssessment(assessment)); err != nil {
		return nil, err
	}
	sorted := sortedAssessment(assessment)
	return marshalContract(struct {
		AssessmentVersion string        `json:"assessmentVersion"`
		Package           PackageRef    `json:"package"`
		ReasoningProfile  EntityRef     `json:"reasoningProfile"`
		Results           []CheckResult `json:"results"`
	}{
		AssessmentVersion: sorted.AssessmentVersion,
		Package:           sorted.Package,
		ReasoningProfile:  sorted.ReasoningProfile,
		Results:           sorted.Results,
	})
}

func NormalizeSuite(suite Suite) ([]byte, error) {
	if err := contract.ErrorFor(validateSuite(suite)); err != nil {
		return nil, err
	}
	normalized := suite
	normalized.Vectors = append([]SuiteVector{}, suite.Vectors...)
	sort.Slice(normalized.Vectors, func(i, j int) bool { return normalized.Vectors[i].ID < normalized.Vectors[j].ID })
	return marshalContract(normalized)
}

func sortedAssessment(assessment Assessment) Assessment {
	sorted := assessment
	sorted.Results = append([]CheckResult{}, assessment.Results...)
	for index, result := range sorted.Results {
		result.Violations = append([]Violation{}, result.Violations...)
		result.EvidenceChecked = append([]EvidenceRef{}, result.EvidenceChecked...)
		sort.Slice(result.Violations, func(i, j int) bool { return violationKey(result.Violations[i]) < violationKey(result.Violations[j]) })
		sort.Slice(result.EvidenceChecked, func(i, j int) bool { return result.EvidenceChecked[i].Path < result.EvidenceChecked[j].Path })
		sorted.Results[index] = result
	}
	sort.Slice(sorted.Results, func(i, j int) bool { return resultKey(sorted.Results[i]) < resultKey(sorted.Results[j]) })
	return sorted
}

func resultKey(result CheckResult) string {
	requirement := ""
	if result.Requirement != nil {
		requirement = result.Requirement.ID
	}
	vector := ""
	if result.Vector != nil {
		vector = result.Vector.ID
	}
	return result.Check + "\x00" + requirement + "\x00" + vector
}

func violationKey(violation Violation) string {
	return violation.Requirement + "\x00" + violation.Severity + "\x00" + violation.Focus + "\x00" + violation.Path + "\x00" + violation.Source + "\x00" + violation.Message
}

func marshalContract(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("normalize contract value: %w", err)
	}
	return append(data, '\n'), nil
}
