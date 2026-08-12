package packagecontract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IndependentImpact/standard2shape/internal/contract"
)

func Open(root string) (Package, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Package{}, err
	}
	manifest, err := ReadManifest(absRoot)
	if err != nil {
		return Package{}, err
	}

	var diagnostics []Diagnostic
	var statements []sourcedTriple
	for _, artifact := range manifest.Artifacts {
		memberData, memberDiagnostics := readMember(absRoot, artifact.Path, artifact.Digest)
		diagnostics = append(diagnostics, memberDiagnostics...)
		if len(memberDiagnostics) > 0 {
			continue
		}
		parsed, parseErr := parseTurtle(bytes.NewReader(memberData))
		if parseErr != nil {
			diagnostics = append(diagnostics, diagnostic("package.member.invalid_rdf", artifact.Path, "cannot parse Turtle: %v", parseErr))
			continue
		}
		for _, triple := range parsed {
			statements = append(statements, sourcedTriple{Triple: triple, Source: artifact.Path})
		}
	}
	for _, vector := range manifest.ConformanceVectors {
		memberData, memberDiagnostics := readMember(absRoot, vector.Path, vector.Digest)
		diagnostics = append(diagnostics, memberDiagnostics...)
		if len(memberDiagnostics) > 0 {
			continue
		}
		if _, parseErr := parseTurtle(bytes.NewReader(memberData)); parseErr != nil {
			diagnostics = append(diagnostics, diagnostic("package.member.invalid_rdf", vector.Path, "cannot parse Turtle test vector: %v", parseErr))
		}
	}
	if len(diagnostics) > 0 {
		return Package{}, contractError(diagnostics)
	}
	diagnostics = append(diagnostics, validateGraph(manifest, statements)...)
	if len(diagnostics) > 0 {
		return Package{}, contractError(diagnostics)
	}
	normalized, err := Normalize(manifest)
	if err != nil {
		return Package{}, err
	}
	return Package{
		Root:               absRoot,
		Manifest:           manifest,
		NormalizedManifest: normalized,
		StatementCount:     len(statements),
	}, nil
}

// ReadManifest reads and structurally validates the closed v0.1 manifest
// without reading package members. Open is the package-verification entry
// point; this narrower function supports an already-open mutable workspace.
func ReadManifest(root string) (Manifest, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Manifest{}, err
	}
	data, err := os.ReadFile(filepath.Join(absRoot, "manifest.json"))
	if err != nil {
		code := "manifest.invalid"
		if errors.Is(err, os.ErrNotExist) {
			code = "package.member.missing"
		}
		return Manifest{}, contractError([]Diagnostic{diagnostic(code, "manifest.json", "cannot read manifest: %v", err)})
	}
	manifest, err := decodeManifest(data)
	if err != nil {
		return Manifest{}, err
	}
	if err := contractError(validateManifest(manifest)); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func Normalize(manifest Manifest) ([]byte, error) {
	if err := contractError(validateManifest(manifest)); err != nil {
		return nil, err
	}
	normalized := manifest
	normalized.Artifacts = append([]SourceArtifact(nil), manifest.Artifacts...)
	normalized.Imports = append([]ImportReference{}, manifest.Imports...)
	normalized.References = append([]ArtifactReference{}, manifest.References...)
	normalized.Requirements = append([]RequirementDeclaration(nil), manifest.Requirements...)
	normalized.DocumentRoots = append([]GraphEntity(nil), manifest.DocumentRoots...)
	normalized.CanonicalShapes = append([]GraphEntity(nil), manifest.CanonicalShapes...)
	normalized.ConformanceVectors = append([]ConformanceVector(nil), manifest.ConformanceVectors...)
	sort.Slice(normalized.Artifacts, func(i, j int) bool { return normalized.Artifacts[i].Path < normalized.Artifacts[j].Path })
	sort.Slice(normalized.Imports, func(i, j int) bool {
		return normalized.Imports[i].Source+"\x00"+normalized.Imports[i].IRI < normalized.Imports[j].Source+"\x00"+normalized.Imports[j].IRI
	})
	sort.Slice(normalized.References, func(i, j int) bool {
		return normalized.References[i].Kind+"\x00"+normalized.References[i].ID < normalized.References[j].Kind+"\x00"+normalized.References[j].ID
	})
	sort.Slice(normalized.Requirements, func(i, j int) bool { return normalized.Requirements[i].ID < normalized.Requirements[j].ID })
	sort.Slice(normalized.DocumentRoots, func(i, j int) bool { return normalized.DocumentRoots[i].ID < normalized.DocumentRoots[j].ID })
	sort.Slice(normalized.CanonicalShapes, func(i, j int) bool { return normalized.CanonicalShapes[i].ID < normalized.CanonicalShapes[j].ID })
	sort.Slice(normalized.ConformanceVectors, func(i, j int) bool { return normalized.ConformanceVectors[i].ID < normalized.ConformanceVectors[j].ID })
	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("normalize manifest: %w", err)
	}
	return append(data, '\n'), nil
}

func decodeManifest(data []byte) (Manifest, error) {
	if json.Valid(data) {
		if err := contractError(checkExactFields(data)); err != nil {
			return Manifest{}, err
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, contractError([]Diagnostic{diagnostic("manifest.invalid", "manifest.json", "cannot decode closed v0.1 contract: %v", err)})
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return Manifest{}, contractError([]Diagnostic{diagnostic("manifest.invalid", "manifest.json", "invalid trailing content: %v", err)})
	}
	return manifest, nil
}

var (
	graphEntitySpec     = contract.ObjectSpec{"id": nil, "source": nil}
	versionedEntitySpec = contract.ObjectSpec{"id": nil, "version": nil, "source": nil}
	manifestSpec        = contract.ObjectSpec{
		"manifestVersion":    nil,
		"id":                 nil,
		"version":            nil,
		"standardRelease":    versionedEntitySpec,
		"documentRoots":      contract.ArraySpec{Element: graphEntitySpec},
		"canonicalShapes":    contract.ArraySpec{Element: graphEntitySpec},
		"artifacts":          contract.ArraySpec{Element: contract.ObjectSpec{"path": nil, "role": nil, "mediaType": nil, "digest": nil}},
		"imports":            contract.ArraySpec{Element: contract.ObjectSpec{"source": nil, "iri": nil, "version": nil, "digest": nil, "policy": nil}},
		"references":         contract.ArraySpec{Element: contract.ObjectSpec{"kind": nil, "id": nil, "version": nil, "digest": nil, "source": nil}},
		"requirements":       contract.ArraySpec{Element: contract.ObjectSpec{"id": nil, "version": nil, "kind": nil, "digest": nil, "source": nil}},
		"reasoningProfile":   versionedEntitySpec,
		"conformanceVectors": contract.ArraySpec{Element: contract.ObjectSpec{"id": nil, "name": nil, "requirement": nil, "path": nil, "digest": nil, "expected": nil}},
	}
)

func checkExactFields(data []byte) []Diagnostic {
	return contract.CheckExactFields(data, "manifest", "manifest.json", manifestSpec)
}

func validateManifest(manifest Manifest) []Diagnostic {
	var diagnostics []Diagnostic
	if manifest.ManifestVersion != ManifestVersionV01 {
		diagnostics = append(diagnostics, diagnostic("manifest.version.unsupported", "manifestVersion", "supported manifest version is %s, got %q", ManifestVersionV01, manifest.ManifestVersion))
	}
	diagnostics = append(diagnostics, validateIRI("id", manifest.ID)...)
	diagnostics = append(diagnostics, validateVersion("version", manifest.Version)...)
	diagnostics = append(diagnostics, validateVersionedEntity("standardRelease", manifest.StandardRelease)...)
	diagnostics = append(diagnostics, validateVersionedEntity("reasoningProfile", manifest.ReasoningProfile)...)
	if len(manifest.DocumentRoots) == 0 {
		diagnostics = append(diagnostics, diagnostic("manifest.field.required", "documentRoots", "at least one document root is required"))
	}
	if len(manifest.CanonicalShapes) == 0 {
		diagnostics = append(diagnostics, diagnostic("manifest.field.required", "canonicalShapes", "at least one canonical shape is required"))
	}
	if len(manifest.Artifacts) == 0 {
		diagnostics = append(diagnostics, diagnostic("manifest.field.required", "artifacts", "at least one canonical artifact is required"))
	}
	if len(manifest.ConformanceVectors) == 0 {
		diagnostics = append(diagnostics, diagnostic("manifest.field.required", "conformanceVectors", "at least one conformance vector is required"))
	}
	if manifest.Imports == nil {
		diagnostics = append(diagnostics, diagnostic("manifest.field.required", "imports", "imports must be an array, possibly empty"))
	}
	if manifest.References == nil {
		diagnostics = append(diagnostics, diagnostic("manifest.field.required", "references", "references must be an array, possibly empty"))
	}

	artifactByPath := map[string]SourceArtifact{}
	for index, artifact := range manifest.Artifacts {
		location := fmt.Sprintf("artifacts[%d]", index)
		diagnostics = append(diagnostics, validatePath(location+".path", artifact.Path)...)
		if artifact.Role != "document" && artifact.Role != "shapes" && artifact.Role != "references" {
			diagnostics = append(diagnostics, diagnostic("manifest.artifact.role_invalid", location+".role", "unsupported canonical artifact role %q", artifact.Role))
		}
		if artifact.MediaType != "text/turtle" {
			diagnostics = append(diagnostics, diagnostic("manifest.artifact.media_type_unsupported", location+".mediaType", "v0.1 supports only text/turtle"))
		}
		diagnostics = append(diagnostics, validateDigest(location+".digest", artifact.Digest)...)
		if previous, exists := artifactByPath[artifact.Path]; exists {
			code := "package.member.conflict"
			message := "path is assigned incompatible artifact descriptors"
			if previous == artifact {
				code = "package.member.duplicate"
				message = "artifact descriptor is declared more than once"
			}
			diagnostics = append(diagnostics, diagnostic(code, artifact.Path, message))
		} else {
			artifactByPath[artifact.Path] = artifact
		}
	}

	diagnostics = append(diagnostics, validateGraphEntities("documentRoots", manifest.DocumentRoots, artifactByPath)...)
	diagnostics = append(diagnostics, validateGraphEntities("canonicalShapes", manifest.CanonicalShapes, artifactByPath)...)
	diagnostics = append(diagnostics, requireArtifactSource("standardRelease.source", manifest.StandardRelease.Source, artifactByPath)...)
	diagnostics = append(diagnostics, requireArtifactSource("reasoningProfile.source", manifest.ReasoningProfile.Source, artifactByPath)...)

	importKeys := map[string]bool{}
	for index, item := range manifest.Imports {
		location := fmt.Sprintf("imports[%d]", index)
		diagnostics = append(diagnostics, requireArtifactSource(location+".source", item.Source, artifactByPath)...)
		diagnostics = append(diagnostics, validateIRI(location+".iri", item.IRI)...)
		diagnostics = append(diagnostics, validateVersion(location+".version", item.Version)...)
		diagnostics = append(diagnostics, validateDigest(location+".digest", item.Digest)...)
		if item.Policy != "reference-only" {
			diagnostics = append(diagnostics, diagnostic("manifest.import.policy_invalid", location+".policy", "v0.1 imports must be reference-only"))
		}
		key := item.Source + "\x00" + item.IRI
		if importKeys[key] {
			diagnostics = append(diagnostics, diagnostic("manifest.import.duplicate", location, "import is declared more than once"))
		}
		importKeys[key] = true
	}

	referenceKeys := map[string]bool{}
	for index, item := range manifest.References {
		location := fmt.Sprintf("references[%d]", index)
		if item.Kind != "indicator" && item.Kind != "methodology" {
			diagnostics = append(diagnostics, diagnostic("manifest.reference.kind_invalid", location+".kind", "reference kind must be indicator or methodology"))
		}
		diagnostics = append(diagnostics, validateIRI(location+".id", item.ID)...)
		diagnostics = append(diagnostics, validateVersion(location+".version", item.Version)...)
		diagnostics = append(diagnostics, validateDigest(location+".digest", item.Digest)...)
		diagnostics = append(diagnostics, requireArtifactSource(location+".source", item.Source, artifactByPath)...)
		key := item.Kind + "\x00" + item.ID
		if referenceKeys[key] {
			diagnostics = append(diagnostics, diagnostic("manifest.reference.duplicate", location, "reference is declared more than once"))
		}
		referenceKeys[key] = true
	}

	if len(manifest.Requirements) == 0 {
		diagnostics = append(diagnostics, diagnostic("manifest.field.required", "requirements", "at least one executable requirement is required"))
	}
	declaredRequirements := map[string]bool{}
	for index, requirement := range manifest.Requirements {
		location := fmt.Sprintf("requirements[%d]", index)
		diagnostics = append(diagnostics, validateIRI(location+".id", requirement.ID)...)
		diagnostics = append(diagnostics, validateVersion(location+".version", requirement.Version)...)
		if requirement.Kind != "semantic" && requirement.Kind != "quantitative" {
			diagnostics = append(diagnostics, diagnostic("manifest.requirement.kind_invalid", location+".kind", "requirement kind must be semantic or quantitative"))
		}
		diagnostics = append(diagnostics, validateDigest(location+".digest", requirement.Digest)...)
		diagnostics = append(diagnostics, requireArtifactSource(location+".source", requirement.Source, artifactByPath)...)
		if declaredRequirements[requirement.ID] {
			diagnostics = append(diagnostics, diagnostic("manifest.requirement.duplicate", location+".id", "requirement is declared more than once"))
		}
		declaredRequirements[requirement.ID] = true
	}

	vectorPaths := map[string]ConformanceVector{}
	vectorIDs := map[string]bool{}
	vectorsByRequirement := map[string]map[string]int{}
	for index, vector := range manifest.ConformanceVectors {
		location := fmt.Sprintf("conformanceVectors[%d]", index)
		diagnostics = append(diagnostics, validateIRI(location+".id", vector.ID)...)
		if contract.IsBlank(vector.Name) {
			diagnostics = append(diagnostics, diagnostic("manifest.field.required", location+".name", "vector name is required"))
		}
		diagnostics = append(diagnostics, validateIRI(location+".requirement", vector.Requirement)...)
		if vector.Requirement != "" && !declaredRequirements[vector.Requirement] {
			diagnostics = append(diagnostics, diagnostic("manifest.vector.requirement_unknown", location+".requirement", "requirement %s is not a declared executable requirement", vector.Requirement))
		}
		if vectorsByRequirement[vector.Requirement] == nil {
			vectorsByRequirement[vector.Requirement] = map[string]int{}
		}
		vectorsByRequirement[vector.Requirement][vector.Expected]++
		diagnostics = append(diagnostics, validatePath(location+".path", vector.Path)...)
		diagnostics = append(diagnostics, validateDigest(location+".digest", vector.Digest)...)
		if vector.Expected != "conforms" && vector.Expected != "non-conforms" {
			diagnostics = append(diagnostics, diagnostic("manifest.vector.expected_invalid", location+".expected", "expected must be conforms or non-conforms"))
		}
		if _, exists := artifactByPath[vector.Path]; exists {
			diagnostics = append(diagnostics, diagnostic("package.member.conflict", vector.Path, "path cannot be both canonical source and conformance evidence"))
		}
		if previous, exists := vectorPaths[vector.Path]; exists {
			code := "package.member.conflict"
			if previous == vector {
				code = "package.member.duplicate"
			}
			diagnostics = append(diagnostics, diagnostic(code, vector.Path, "conformance-vector path is declared more than once"))
		} else {
			vectorPaths[vector.Path] = vector
		}
		if vectorIDs[vector.ID] {
			diagnostics = append(diagnostics, diagnostic("manifest.vector.duplicate", location+".id", "vector identity is declared more than once"))
		}
		vectorIDs[vector.ID] = true
	}

	// SPEC: every executable requirement has mandatory valid, invalid, and
	// boundary test vectors, which at manifest level means at least three
	// vectors including both expected outcomes; the suite contract binds the
	// categories.
	for index, requirement := range manifest.Requirements {
		counts := vectorsByRequirement[requirement.ID]
		if counts["conforms"]+counts["non-conforms"] < 3 || counts["conforms"] == 0 || counts["non-conforms"] == 0 {
			diagnostics = append(diagnostics, diagnostic("manifest.requirement.vectors_missing", fmt.Sprintf("requirements[%d]", index), "requirement %s needs at least three vectors covering both expected outcomes", requirement.ID))
		}
	}
	return diagnostics
}

func validateGraphEntities(field string, entities []GraphEntity, artifacts map[string]SourceArtifact) []Diagnostic {
	var diagnostics []Diagnostic
	identities := map[string]bool{}
	for index, entity := range entities {
		location := fmt.Sprintf("%s[%d]", field, index)
		diagnostics = append(diagnostics, validateIRI(location+".id", entity.ID)...)
		diagnostics = append(diagnostics, requireArtifactSource(location+".source", entity.Source, artifacts)...)
		if identities[entity.ID] {
			diagnostics = append(diagnostics, diagnostic("manifest.graph_entity.duplicate", location+".id", "graph identity is declared more than once"))
		}
		identities[entity.ID] = true
	}
	return diagnostics
}

func validateVersionedEntity(location string, entity VersionedGraphEntity) []Diagnostic {
	diagnostics := validateIRI(location+".id", entity.ID)
	diagnostics = append(diagnostics, validateVersion(location+".version", entity.Version)...)
	diagnostics = append(diagnostics, validatePath(location+".source", entity.Source)...)
	return diagnostics
}

func validateIRI(location, value string) []Diagnostic {
	if !contract.IsIRI(value) {
		return []Diagnostic{diagnostic("manifest.iri.invalid", location, "expected an absolute IRI, got %q", value)}
	}
	return nil
}

func validateVersion(location, value string) []Diagnostic {
	if !contract.IsVersion(value) {
		return []Diagnostic{diagnostic("manifest.version.invalid", location, "expected semantic version, got %q", value)}
	}
	return nil
}

func validateDigest(location, value string) []Diagnostic {
	if !contract.IsDigest(value) {
		return []Diagnostic{diagnostic("manifest.digest.invalid", location, "expected lowercase sha256 digest")}
	}
	return nil
}

func validatePath(location, value string) []Diagnostic {
	if !contract.IsNormalizedPath(value) {
		return []Diagnostic{diagnostic("manifest.path.invalid", location, "expected a normalized POSIX package-relative path, got %q", value)}
	}
	return nil
}

func requireArtifactSource(location, source string, artifacts map[string]SourceArtifact) []Diagnostic {
	diagnostics := validatePath(location, source)
	if _, exists := artifacts[source]; !exists && source != "" {
		diagnostics = append(diagnostics, diagnostic("package.member.missing", location, "source %q is not declared as a canonical artifact", source))
	}
	return diagnostics
}

func readMember(root, relative, expectedDigest string) ([]byte, []Diagnostic) {
	path, err := safeJoin(root, relative)
	if err != nil {
		return nil, []Diagnostic{diagnostic("manifest.path.invalid", relative, "%v", err)}
	}
	if diagnostics := checkResolvedContainment(root, path, relative); len(diagnostics) > 0 {
		return nil, diagnostics
	}
	data, err := os.ReadFile(path)
	if err != nil {
		code := "package.member.unreadable"
		if errors.Is(err, os.ErrNotExist) {
			code = "package.member.missing"
		}
		return nil, []Diagnostic{diagnostic(code, relative, "cannot read declared member: %v", err)}
	}
	actual := Digest(data)
	if actual != expectedDigest {
		return nil, []Diagnostic{diagnostic("package.member.digest_mismatch", relative, "expected %s, got %s", expectedDigest, actual)}
	}
	return data, nil
}

// safeJoin is only lexical; reading a member follows symlinks, so the
// resolved location must also stay inside the resolved package root.
func checkResolvedContainment(root, path, relative string) []Diagnostic {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return []Diagnostic{diagnostic("package.member.unreadable", relative, "cannot resolve declared member: %v", err)}
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return []Diagnostic{diagnostic("package.member.unreadable", relative, "cannot resolve package root: %v", err)}
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return []Diagnostic{diagnostic("package.member.escape", relative, "member resolves outside the package root")}
	}
	return nil
}

func safeJoin(root, relative string) (string, error) {
	if diagnostics := validatePath("path", relative); len(diagnostics) > 0 {
		return "", errors.New(diagnostics[0].Message)
	}
	joined := filepath.Join(root, filepath.FromSlash(relative))
	rel, err := filepath.Rel(root, joined)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes package root: %s", relative)
	}
	return joined, nil
}

// Digest returns the canonical sha256:<hex> form used across the contracts.
func Digest(data []byte) string {
	return contract.SHA256(data)
}
