package assessmentcontract

import (
	"fmt"

	"github.com/IndependentImpact/standard2shape/internal/contract"
	"github.com/IndependentImpact/standard2shape/internal/packagecontract"
)

// CheckAgainstRequest verifies that an assessment answers exactly the given
// request: same package and reasoning profile, at least one result for every
// requested check, and no results for checks that were not requested.
func CheckAgainstRequest(request Request, assessment Assessment) error {
	var diagnostics []contract.Diagnostic
	if assessment.Package != request.Package {
		diagnostics = append(diagnostics, contract.Diag("assessment.request.package_mismatch", "package", "assessment answers package %s@%s, request named %s@%s", assessment.Package.ID, assessment.Package.Version, request.Package.ID, request.Package.Version))
	}
	if assessment.ReasoningProfile != request.ReasoningProfile {
		diagnostics = append(diagnostics, contract.Diag("assessment.request.profile_mismatch", "reasoningProfile", "assessment used reasoning profile %s@%s, request named %s@%s", assessment.ReasoningProfile.ID, assessment.ReasoningProfile.Version, request.ReasoningProfile.ID, request.ReasoningProfile.Version))
	}
	requested := map[string]bool{}
	for _, check := range request.Checks {
		requested[check] = true
	}
	answered := map[string]bool{}
	for index, result := range assessment.Results {
		if !requested[result.Check] {
			diagnostics = append(diagnostics, contract.Diag("assessment.request.check_unrequested", fmt.Sprintf("results[%d]", index), "check %s was not requested", result.Check))
		}
		answered[result.Check] = true
	}
	for _, check := range request.Checks {
		if !answered[check] {
			diagnostics = append(diagnostics, contract.Diag("assessment.request.check_missing", "results", "requested check %s has no result", check))
		}
	}
	return contract.ErrorFor(diagnostics)
}

// CheckSuiteAgainstPackage verifies that a conformance suite covers exactly
// the package's declared conformance vectors with matching expectations, and
// that it names the package by identity, version, and normalized-manifest
// digest.
func CheckSuiteAgainstPackage(suite Suite, pkg packagecontract.Package) error {
	var diagnostics []contract.Diagnostic
	manifestDigest := packagecontract.Digest(pkg.NormalizedManifest)
	if suite.Package.ID != pkg.Manifest.ID || suite.Package.Version != pkg.Manifest.Version || suite.Package.Digest != manifestDigest {
		diagnostics = append(diagnostics, contract.Diag("suite.package.mismatch", "package", "suite names %s@%s digest %s, package is %s@%s digest %s", suite.Package.ID, suite.Package.Version, suite.Package.Digest, pkg.Manifest.ID, pkg.Manifest.Version, manifestDigest))
	}
	declared := map[string]string{}
	for _, vector := range pkg.Manifest.ConformanceVectors {
		declared[vector.ID] = vector.Expected
	}
	covered := map[string]bool{}
	for index, vector := range suite.Vectors {
		location := fmt.Sprintf("vectors[%d]", index)
		expected, exists := declared[vector.ID]
		if !exists {
			diagnostics = append(diagnostics, contract.Diag("suite.vector.unknown", location+".id", "vector %s is not declared by the package manifest", vector.ID))
			continue
		}
		if vector.Expected != expected {
			diagnostics = append(diagnostics, contract.Diag("suite.vector.expected_mismatch", location+".expected", "vector %s expects %s in the manifest", vector.ID, expected))
		}
		covered[vector.ID] = true
	}
	for _, vector := range pkg.Manifest.ConformanceVectors {
		if !covered[vector.ID] {
			diagnostics = append(diagnostics, contract.Diag("suite.vector.missing", "vectors", "manifest vector %s is not categorized by the suite", vector.ID))
		}
	}
	return contract.ErrorFor(diagnostics)
}
