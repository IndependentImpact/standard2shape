package assessmentcontract

import (
	"fmt"

	"github.com/IndependentImpact/standard2shape/internal/contract"
	"github.com/IndependentImpact/standard2shape/internal/packagecontract"
)

// CheckAgainstRequest verifies that an assessment answers exactly the given
// request against the given package: same package and reasoning profile, at
// least one result for every requested check and requirement, no results for
// checks or requirements that were not requested, one result per declared
// conformance vector when test-vectors was requested, and evidence
// attestations that neither drop, substitute, nor invent evidence — every
// attested path must be requested evidence or a declared package member,
// under its pinned digest.
func CheckAgainstRequest(request Request, assessment Assessment, pkg packagecontract.Package) error {
	var diagnostics []contract.Diagnostic
	if assessment.Package != request.Package {
		diagnostics = append(diagnostics, contract.Diag("assessment.request.package_mismatch", "package", "assessment answers package %s@%s, request named %s@%s", assessment.Package.ID, assessment.Package.Version, request.Package.ID, request.Package.Version))
	}
	if assessment.ReasoningProfile != request.ReasoningProfile {
		diagnostics = append(diagnostics, contract.Diag("assessment.request.profile_mismatch", "reasoningProfile", "assessment used reasoning profile %s@%s, request named %s@%s", assessment.ReasoningProfile.ID, assessment.ReasoningProfile.Version, request.ReasoningProfile.ID, request.ReasoningProfile.Version))
	}
	requestedChecks := map[string]bool{}
	for _, check := range request.Checks {
		requestedChecks[check] = true
	}
	requestedRequirements := map[string]bool{}
	for _, requirement := range request.Requirements {
		requestedRequirements[requirement] = true
	}
	requestedEvidence := map[string]string{}
	for _, evidence := range request.Evidence {
		requestedEvidence[evidence.Path] = evidence.Digest
	}
	memberDigests := map[string]string{}
	for _, artifact := range pkg.Manifest.Artifacts {
		memberDigests[artifact.Path] = artifact.Digest
	}
	declaredVectors := map[string]string{}
	for _, vector := range pkg.Manifest.ConformanceVectors {
		memberDigests[vector.Path] = vector.Digest
		declaredVectors[vector.ID] = vector.Expected
	}

	answeredChecks := map[string]bool{}
	answeredRequirements := map[string]bool{}
	answeredVectors := map[string]bool{}
	attestedEvidence := map[string]bool{}
	for index, result := range assessment.Results {
		location := fmt.Sprintf("results[%d]", index)
		if !requestedChecks[result.Check] {
			diagnostics = append(diagnostics, contract.Diag("assessment.request.check_unrequested", location, "check %s was not requested", result.Check))
		}
		answeredChecks[result.Check] = true
		if result.Requirement != nil {
			if !requestedRequirements[result.Requirement.ID] {
				diagnostics = append(diagnostics, contract.Diag("assessment.request.requirement_unrequested", location+".requirement", "requirement %s was not requested", result.Requirement.ID))
			}
			answeredRequirements[result.Requirement.ID] = true
		}
		if result.Vector != nil {
			expected, declared := declaredVectors[result.Vector.ID]
			if !declared {
				diagnostics = append(diagnostics, contract.Diag("assessment.request.vector_unknown", location+".vector.id", "vector %s is not declared by the package manifest", result.Vector.ID))
			} else if result.Vector.Expected != expected {
				diagnostics = append(diagnostics, contract.Diag("assessment.request.vector_expected_mismatch", location+".vector.expected", "vector %s expects %s in the manifest", result.Vector.ID, expected))
			}
			answeredVectors[result.Vector.ID] = true
		}
		for evidenceIndex, evidence := range result.EvidenceChecked {
			evidenceLocation := fmt.Sprintf("%s.evidenceChecked[%d]", location, evidenceIndex)
			expected, requested := requestedEvidence[evidence.Path]
			if !requested {
				memberDigest, member := memberDigests[evidence.Path]
				if !member {
					diagnostics = append(diagnostics, contract.Diag("assessment.request.evidence_unknown", evidenceLocation, "evidence %s is neither requested evidence nor a declared package member", evidence.Path))
				} else if evidence.Digest != memberDigest {
					diagnostics = append(diagnostics, contract.Diag("assessment.request.evidence_mismatch", evidenceLocation, "member %s was attested with digest %s, manifest pinned %s", evidence.Path, evidence.Digest, memberDigest))
				}
				continue
			}
			if evidence.Digest != expected {
				diagnostics = append(diagnostics, contract.Diag("assessment.request.evidence_mismatch", evidenceLocation, "evidence %s was attested with digest %s, request pinned %s", evidence.Path, evidence.Digest, expected))
				continue
			}
			attestedEvidence[evidence.Path] = true
		}
	}
	for _, check := range request.Checks {
		if !answeredChecks[check] {
			diagnostics = append(diagnostics, contract.Diag("assessment.request.check_missing", "results", "requested check %s has no result", check))
		}
	}
	for _, requirement := range request.Requirements {
		if !answeredRequirements[requirement] {
			diagnostics = append(diagnostics, contract.Diag("assessment.request.requirement_missing", "results", "requested requirement %s has no result", requirement))
		}
	}
	for _, evidence := range request.Evidence {
		if !attestedEvidence[evidence.Path] {
			diagnostics = append(diagnostics, contract.Diag("assessment.request.evidence_missing", "results", "requested evidence %s was not attested by any result", evidence.Path))
		}
	}
	if requestedChecks[CheckTestVectors] {
		for _, vector := range pkg.Manifest.ConformanceVectors {
			if !answeredVectors[vector.ID] {
				diagnostics = append(diagnostics, contract.Diag("assessment.request.vector_missing", "results", "declared vector %s has no result", vector.ID))
			}
		}
	}
	return contract.ErrorFor(diagnostics)
}

// CheckSuiteAgainstPackage verifies that a conformance suite covers exactly
// the package's declared conformance vectors with the manifest's expectations
// and requirement bindings, that every manifest-declared requirement keeps
// valid, invalid, and boundary coverage, and that the suite names the package
// by identity, version, and normalized-manifest digest.
func CheckSuiteAgainstPackage(suite Suite, pkg packagecontract.Package) error {
	var diagnostics []contract.Diagnostic
	manifestDigest := packagecontract.Digest(pkg.NormalizedManifest)
	if suite.Package.ID != pkg.Manifest.ID || suite.Package.Version != pkg.Manifest.Version || suite.Package.Digest != manifestDigest {
		diagnostics = append(diagnostics, contract.Diag("suite.package.mismatch", "package", "suite names %s@%s digest %s, package is %s@%s digest %s", suite.Package.ID, suite.Package.Version, suite.Package.Digest, pkg.Manifest.ID, pkg.Manifest.Version, manifestDigest))
	}
	declared := map[string]packagecontract.ConformanceVector{}
	for _, vector := range pkg.Manifest.ConformanceVectors {
		declared[vector.ID] = vector
	}
	covered := map[string]bool{}
	categoriesByRequirement := map[string]map[string]bool{}
	for index, vector := range suite.Vectors {
		location := fmt.Sprintf("vectors[%d]", index)
		declaredVector, exists := declared[vector.ID]
		if !exists {
			diagnostics = append(diagnostics, contract.Diag("suite.vector.unknown", location+".id", "vector %s is not declared by the package manifest", vector.ID))
			continue
		}
		if vector.Expected != declaredVector.Expected {
			diagnostics = append(diagnostics, contract.Diag("suite.vector.expected_mismatch", location+".expected", "vector %s expects %s in the manifest", vector.ID, declaredVector.Expected))
		}
		if vector.Requirement != declaredVector.Requirement {
			diagnostics = append(diagnostics, contract.Diag("suite.vector.requirement_mismatch", location+".requirement", "vector %s exercises requirement %s in the manifest", vector.ID, declaredVector.Requirement))
		}
		covered[vector.ID] = true
		if categoriesByRequirement[declaredVector.Requirement] == nil {
			categoriesByRequirement[declaredVector.Requirement] = map[string]bool{}
		}
		categoriesByRequirement[declaredVector.Requirement][vector.Category] = true
	}
	for _, vector := range pkg.Manifest.ConformanceVectors {
		if !covered[vector.ID] {
			diagnostics = append(diagnostics, contract.Diag("suite.vector.missing", "vectors", "manifest vector %s is not categorized by the suite", vector.ID))
		}
	}
	seenRequirements := map[string]bool{}
	for _, vector := range pkg.Manifest.ConformanceVectors {
		if seenRequirements[vector.Requirement] {
			continue
		}
		seenRequirements[vector.Requirement] = true
		categories := categoriesByRequirement[vector.Requirement]
		for _, category := range []string{"valid", "invalid", "boundary"} {
			if !categories[category] {
				diagnostics = append(diagnostics, contract.Diag("suite.category.missing", "vectors", "manifest requirement %s has no %s vector in the suite", vector.Requirement, category))
			}
		}
	}
	return contract.ErrorFor(diagnostics)
}
