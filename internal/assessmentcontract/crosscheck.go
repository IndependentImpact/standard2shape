package assessmentcontract

import (
	"fmt"

	"github.com/IndependentImpact/standard2shape/internal/contract"
	"github.com/IndependentImpact/standard2shape/internal/packagecontract"
)

// CheckAgainstRequest verifies that an assessment answers exactly the given
// request against the given package. The request must itself be bound to the
// supplied package by identity, version, normalized-manifest digest, and
// reasoning profile, and may only request requirements the manifest declares
// — otherwise a different package could supply the accepted vector and
// evidence inventory. The assessment must carry at least one result for every
// requested check, answer every requested requirement through the
// applicability check matching its declared kind, cover every declared
// conformance vector when test-vectors was requested, and attest only
// requested evidence or declared package members, always under the pinned
// digest.
func CheckAgainstRequest(request Request, assessment Assessment, pkg packagecontract.Package) error {
	var diagnostics []contract.Diagnostic
	manifestDigest := packagecontract.Digest(pkg.NormalizedManifest)
	if request.Package.ID != pkg.Manifest.ID || request.Package.Version != pkg.Manifest.Version || request.Package.Digest != manifestDigest {
		diagnostics = append(diagnostics, contract.Diag("assessment.request.package_unbound", "package", "request pins %s@%s digest %s, supplied package is %s@%s digest %s", request.Package.ID, request.Package.Version, request.Package.Digest, pkg.Manifest.ID, pkg.Manifest.Version, manifestDigest))
	}
	if request.ReasoningProfile.ID != pkg.Manifest.ReasoningProfile.ID || request.ReasoningProfile.Version != pkg.Manifest.ReasoningProfile.Version {
		diagnostics = append(diagnostics, contract.Diag("assessment.request.profile_unbound", "reasoningProfile", "request names reasoning profile %s@%s, supplied package declares %s@%s", request.ReasoningProfile.ID, request.ReasoningProfile.Version, pkg.Manifest.ReasoningProfile.ID, pkg.Manifest.ReasoningProfile.Version))
	}
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
	declaredKinds := map[string]packagecontract.RequirementDeclaration{}
	for _, requirement := range pkg.Manifest.Requirements {
		declaredKinds[requirement.ID] = requirement
	}
	checkForKind := map[string]string{"semantic": CheckSemanticApplicability, "quantitative": CheckQuantitativeApplicability}
	requestedRequirements := map[string]bool{}
	for index, requirement := range request.Requirements {
		requestedRequirements[requirement] = true
		declaration, declared := declaredKinds[requirement]
		if !declared {
			diagnostics = append(diagnostics, contract.Diag("assessment.request.requirement_unknown", fmt.Sprintf("requirements[%d]", index), "requirement %s is not declared by the package manifest", requirement))
			continue
		}
		if !requestedChecks[checkForKind[declaration.Kind]] {
			diagnostics = append(diagnostics, contract.Diag("assessment.request.requirement_uncheckable", fmt.Sprintf("requirements[%d]", index), "requirement %s is %s but the request omits the %s check", requirement, declaration.Kind, checkForKind[declaration.Kind]))
		}
	}
	for _, kind := range []string{"semantic", "quantitative"} {
		if !requestedChecks[checkForKind[kind]] {
			continue
		}
		answerable := false
		for _, requirement := range request.Requirements {
			if declaration, declared := declaredKinds[requirement]; declared && declaration.Kind == kind {
				answerable = true
			}
		}
		if !answerable {
			diagnostics = append(diagnostics, contract.Diag("assessment.request.check_unanswerable", "checks", "the %s check is requested but no requested requirement is %s", checkForKind[kind], kind))
		}
	}
	requestedEvidence := map[string]string{}
	for _, evidence := range request.Evidence {
		requestedEvidence[evidence.Path] = evidence.Digest
	}
	memberDigests := map[string]string{}
	for _, artifact := range pkg.Manifest.Artifacts {
		memberDigests[artifact.Path] = artifact.Digest
	}
	canonicalShapes := map[string]bool{}
	for _, shape := range pkg.Manifest.CanonicalShapes {
		canonicalShapes[shape.ID] = true
	}
	violationTargets := map[string]bool{}
	for shape := range canonicalShapes {
		violationTargets[shape] = true
	}
	for _, requirement := range pkg.Manifest.Requirements {
		violationTargets[requirement.ID] = true
	}
	declaredVectors := map[string]packagecontract.ConformanceVector{}
	for _, vector := range pkg.Manifest.ConformanceVectors {
		memberDigests[vector.Path] = vector.Digest
		declaredVectors[vector.ID] = vector
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
			if declaration, declared := declaredKinds[result.Requirement.ID]; declared {
				if result.Requirement.Version != declaration.Version {
					diagnostics = append(diagnostics, contract.Diag("assessment.request.requirement_version_mismatch", location+".requirement.version", "requirement %s is declared at version %s", result.Requirement.ID, declaration.Version))
				}
				if checkForKind[declaration.Kind] != result.Check {
					diagnostics = append(diagnostics, contract.Diag("assessment.request.requirement_kind_mismatch", location+".requirement", "requirement %s is %s and cannot be answered by %s", result.Requirement.ID, declaration.Kind, result.Check))
				} else {
					answeredRequirements[result.Requirement.ID] = true
				}
			}
		}
		if result.Vector != nil {
			declaration, declared := declaredVectors[result.Vector.ID]
			if !declared {
				diagnostics = append(diagnostics, contract.Diag("assessment.request.vector_unknown", location+".vector.id", "vector %s is not declared by the package manifest", result.Vector.ID))
			} else {
				if result.Vector.Expected != declaration.Expected {
					diagnostics = append(diagnostics, contract.Diag("assessment.request.vector_expected_mismatch", location+".vector.expected", "vector %s expects %s in the manifest", result.Vector.ID, declaration.Expected))
				}
				attested := false
				for _, evidence := range result.EvidenceChecked {
					if evidence.Path == declaration.Path && evidence.Digest == declaration.Digest {
						attested = true
					}
				}
				if !attested {
					diagnostics = append(diagnostics, contract.Diag("assessment.request.vector_evidence_missing", location+".evidenceChecked", "vector %s must attest its own evidence %s under the manifest digest", result.Vector.ID, declaration.Path))
				}
			}
			answeredVectors[result.Vector.ID] = true
		}
		for violationIndex, violation := range result.Violations {
			violationLocation := fmt.Sprintf("%s.violations[%d].requirement", location, violationIndex)
			if !violationTargets[violation.Requirement] {
				diagnostics = append(diagnostics, contract.Diag("assessment.request.violation_requirement_unknown", violationLocation, "violation names %s, which is neither a declared canonical shape nor a declared executable requirement", violation.Requirement))
				continue
			}
			// Attribution is bound to the result's own evaluation context,
			// not the package-wide identity union.
			switch {
			case result.Requirement != nil:
				if violation.Requirement != result.Requirement.ID {
					diagnostics = append(diagnostics, contract.Diag("assessment.request.violation_requirement_mismatch", violationLocation, "an applicability violation must attribute to the result's requirement %s", result.Requirement.ID))
				}
			case result.Vector != nil:
				vectorTarget := declaredVectors[result.Vector.ID].Target
				if !canonicalShapes[violation.Requirement] && violation.Requirement != vectorTarget {
					diagnostics = append(diagnostics, contract.Diag("assessment.request.violation_requirement_mismatch", violationLocation, "a vector violation must attribute to a declared canonical shape or the vector's target %s", vectorTarget))
				}
			default:
				if !canonicalShapes[violation.Requirement] {
					diagnostics = append(diagnostics, contract.Diag("assessment.request.violation_requirement_mismatch", violationLocation, "a %s violation must attribute to a declared canonical shape", result.Check))
				}
			}
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
		if _, declared := declaredKinds[requirement]; declared && !answeredRequirements[requirement] {
			diagnostics = append(diagnostics, contract.Diag("assessment.request.requirement_missing", "results", "requested requirement %s has no result from its kind-matched applicability check", requirement))
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
	categoriesByTarget := map[string]map[string]bool{}
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
		if vector.Target != declaredVector.Target {
			diagnostics = append(diagnostics, contract.Diag("suite.vector.target_mismatch", location+".target", "vector %s exercises target %s in the manifest", vector.ID, declaredVector.Target))
		}
		if vector.Category != declaredVector.Category {
			diagnostics = append(diagnostics, contract.Diag("suite.vector.category_mismatch", location+".category", "vector %s is categorized %s in the manifest", vector.ID, declaredVector.Category))
		}
		covered[vector.ID] = true
		if categoriesByTarget[declaredVector.Target] == nil {
			categoriesByTarget[declaredVector.Target] = map[string]bool{}
		}
		categoriesByTarget[declaredVector.Target][vector.Category] = true
	}
	for _, vector := range pkg.Manifest.ConformanceVectors {
		if !covered[vector.ID] {
			diagnostics = append(diagnostics, contract.Diag("suite.vector.missing", "vectors", "manifest vector %s is not categorized by the suite", vector.ID))
		}
	}
	seenTargets := map[string]bool{}
	for _, vector := range pkg.Manifest.ConformanceVectors {
		if seenTargets[vector.Target] {
			continue
		}
		seenTargets[vector.Target] = true
		categories := categoriesByTarget[vector.Target]
		for _, category := range []string{"valid", "invalid", "boundary"} {
			if !categories[category] {
				diagnostics = append(diagnostics, contract.Diag("suite.category.missing", "vectors", "manifest target %s has no %s vector in the suite", vector.Target, category))
			}
		}
	}
	return contract.ErrorFor(diagnostics)
}
