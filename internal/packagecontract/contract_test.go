package packagecontract

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/knakk/rdf"
)

func TestOpenValidTracerPackage(t *testing.T) {
	pkg, err := Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}

	if pkg.Manifest.ManifestVersion != ManifestVersionV01 {
		t.Fatalf("manifest version = %q", pkg.Manifest.ManifestVersion)
	}
	if pkg.Manifest.StandardRelease.ID != "https://example.org/standard/DemoStandardRelease" {
		t.Fatalf("standard release = %#v", pkg.Manifest.StandardRelease)
	}
	if len(pkg.Manifest.DocumentRoots) != 1 || len(pkg.Manifest.CanonicalShapes) != 2 {
		t.Fatalf("roots=%#v shapes=%#v", pkg.Manifest.DocumentRoots, pkg.Manifest.CanonicalShapes)
	}
	if len(pkg.Manifest.Artifacts) != 3 || len(pkg.Manifest.References) != 2 || len(pkg.Manifest.ConformanceVectors) != 2 {
		t.Fatalf("artifacts=%d references=%d vectors=%d", len(pkg.Manifest.Artifacts), len(pkg.Manifest.References), len(pkg.Manifest.ConformanceVectors))
	}
	if pkg.StatementCount == 0 {
		t.Fatal("expected locally parsed RDF statements")
	}
	if !bytes.HasSuffix(pkg.NormalizedManifest, []byte("\n")) {
		t.Fatal("normalized manifest must end with one newline")
	}
}

func TestManifestNormalizationIsDeterministic(t *testing.T) {
	pkg, err := Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}

	reordered := pkg.Manifest
	slices.Reverse(reordered.Artifacts)
	slices.Reverse(reordered.Imports)
	slices.Reverse(reordered.References)
	slices.Reverse(reordered.DocumentRoots)
	slices.Reverse(reordered.CanonicalShapes)
	slices.Reverse(reordered.ConformanceVectors)
	normalized, err := Normalize(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pkg.NormalizedManifest, normalized) {
		t.Fatalf("normalization depends on input order:\nfirst:\n%s\nsecond:\n%s", pkg.NormalizedManifest, normalized)
	}
}

func TestNormalizedManifestKeepsEmptyCollections(t *testing.T) {
	pkg, err := Open(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}

	manifest := pkg.Manifest
	manifest.Imports = []ImportReference{}
	manifest.References = []ArtifactReference{}
	normalized, err := Normalize(manifest)
	if err != nil {
		t.Fatal(err)
	}
	roundTripped, err := decodeManifest(normalized)
	if err != nil {
		t.Fatal(err)
	}
	if roundTripped.Imports == nil || roundTripped.References == nil {
		t.Fatalf("normalization drops required empty arrays:\n%s", normalized)
	}
	if diagnostics := validateManifest(roundTripped); len(diagnostics) != 0 {
		t.Fatalf("normalized manifest violates its own contract: %#v", diagnostics)
	}
}

func TestManifestRequiresImportAndReferenceArrays(t *testing.T) {
	manifestData, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "tracer", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(manifestData, &fields); err != nil {
		t.Fatal(err)
	}
	delete(fields, "imports")
	fields["references"] = json.RawMessage("null")
	stripped, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}

	manifest, err := decodeManifest(stripped)
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := validateManifest(manifest)
	for _, location := range []string{"imports", "references"} {
		found := slices.ContainsFunc(diagnostics, func(d Diagnostic) bool {
			return d.Code == "manifest.field.required" && d.Location == location
		})
		if !found {
			t.Fatalf("missing %s diagnostic %q in %#v", location, "manifest.field.required", diagnostics)
		}
	}
}

func TestManifestFieldNamesAreExact(t *testing.T) {
	manifestData, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "tracer", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}

	cased := bytes.Replace(manifestData, []byte(`"imports"`), []byte(`"Imports"`), 1)
	_, err = decodeManifest(cased)
	var contractErr *ContractError
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "manifest.field.unknown") {
		t.Fatalf("expected exact-case field diagnostic, got %v", err)
	}

	unknown := bytes.Replace(manifestData, []byte(`"manifestVersion":`), []byte("\"foo\": true,\n  \"manifestVersion\":"), 1)
	_, err = decodeManifest(unknown)
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "manifest.field.unknown") {
		t.Fatalf("expected unknown field diagnostic, got %v", err)
	}

	duplicated := bytes.Replace(manifestData, []byte(`"version": "0.1.0",`), []byte("\"version\": \"0.1.0\",\n  \"version\": \"0.1.0\","), 1)
	_, err = decodeManifest(duplicated)
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "manifest.field.duplicate") {
		t.Fatalf("expected duplicate field diagnostic, got %v", err)
	}
}

func TestSymlinkedMemberCannotEscapePackageRoot(t *testing.T) {
	root := copyFixture(t, filepath.Join("..", "..", "fixtures", "tracer"))
	shapesPath := filepath.Join(root, "shapes.ttl")
	shapes, err := os.ReadFile(shapesPath)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "shapes.ttl")
	if err := os.WriteFile(outside, shapes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(shapesPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, shapesPath); err != nil {
		t.Skipf("cannot create symlinks: %v", err)
	}

	_, err = Open(root)
	var contractErr *ContractError
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "package.member.escape") {
		t.Fatalf("expected escaping-member diagnostic, got %v", err)
	}
}

func TestDualTypedDocumentMembersAreRejected(t *testing.T) {
	root := copyFixture(t, filepath.Join("..", "..", "fixtures", "tracer"))
	documentPath := filepath.Join(root, "document.ttl")
	document, err := os.ReadFile(documentPath)
	if err != nil {
		t.Fatal(err)
	}
	document = append(document, []byte("\n<https://example.org/standard/ProjectDetailsPlacement> a <https://standard2shape.dev/vocab#DocumentSection> .\n")...)
	if err := os.WriteFile(documentPath, document, 0o644); err != nil {
		t.Fatal(err)
	}
	updateArtifactDigest(t, root, "document.ttl", document)

	_, err = Open(root)
	var contractErr *ContractError
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "graph.document_member.invalid") {
		t.Fatalf("expected dual-typed member diagnostic, got %v", err)
	}
}

func TestOrphanDocumentNodesAreRejected(t *testing.T) {
	root := copyFixture(t, filepath.Join("..", "..", "fixtures", "tracer"))
	documentPath := filepath.Join(root, "document.ttl")
	document, err := os.ReadFile(documentPath)
	if err != nil {
		t.Fatal(err)
	}
	document = append(document, []byte("\n<https://example.org/standard/OrphanSection> a <https://standard2shape.dev/vocab#DocumentSection> .\n")...)
	if err := os.WriteFile(documentPath, document, 0o644); err != nil {
		t.Fatal(err)
	}
	updateArtifactDigest(t, root, "document.ttl", document)

	_, err = Open(root)
	var contractErr *ContractError
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "graph.document_node.orphan") {
		t.Fatalf("expected orphan document-node diagnostic, got %v", err)
	}
}

func TestSchemaPathPatternMatchesVerifier(t *testing.T) {
	schemaData, err := os.ReadFile(filepath.Join("..", "..", "contracts", "v0", "package-manifest.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs struct {
			Path struct {
				Pattern string `json:"pattern"`
			} `json:"path"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		t.Fatal(err)
	}
	pattern, err := regexp.Compile(schema.Defs.Path.Pattern)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path  string
		valid bool
	}{
		{path: "shapes.ttl", valid: true},
		{path: "graphs/shapes.ttl", valid: true},
		{path: ".hidden.ttl", valid: true},
		{path: "..archive.ttl", valid: true},
		{path: "", valid: false},
		{path: ".", valid: false},
		{path: "..", valid: false},
		{path: "./shapes.ttl", valid: false},
		{path: "../shapes.ttl", valid: false},
		{path: "graphs/./shapes.ttl", valid: false},
		{path: "graphs/../shapes.ttl", valid: false},
		{path: "graphs//shapes.ttl", valid: false},
		{path: "graphs/shapes.ttl/", valid: false},
		{path: "/shapes.ttl", valid: false},
		{path: "graphs\\shapes.ttl", valid: false},
	}
	for _, test := range tests {
		schemaValid := pattern.MatchString(test.path)
		verifierValid := len(validatePath("path", test.path)) == 0
		if schemaValid != test.valid || verifierValid != test.valid {
			t.Errorf("path %q: expected valid=%t, schema=%t, verifier=%t", test.path, test.valid, schemaValid, verifierValid)
		}
	}
}

func TestInvalidPackageFixturesHaveStableDiagnostics(t *testing.T) {
	tests := []struct {
		fixture string
		code    string
	}{
		{fixture: "missing-member", code: "package.member.missing"},
		{fixture: "duplicate-member", code: "package.member.duplicate"},
		{fixture: "conflicting-member", code: "package.member.conflict"},
	}
	for _, test := range tests {
		t.Run(test.fixture, func(t *testing.T) {
			_, err := Open(filepath.Join("..", "..", "fixtures", "package-v0", "invalid-"+test.fixture))
			if err == nil {
				t.Fatal("expected package to be rejected")
			}
			var contractErr *ContractError
			if !errors.As(err, &contractErr) {
				t.Fatalf("error type = %T: %v", err, err)
			}
			if !hasDiagnostic(contractErr.Diagnostics, test.code) {
				t.Fatalf("missing diagnostic %q in %#v", test.code, contractErr.Diagnostics)
			}
			_, repeated := Open(filepath.Join("..", "..", "fixtures", "package-v0", "invalid-"+test.fixture))
			if repeated == nil || repeated.Error() != err.Error() {
				t.Fatalf("diagnostic is not stable:\nfirst:  %v\nsecond: %v", err, repeated)
			}
		})
	}
}

func TestCanonicalSourcesRejectPresentationOrder(t *testing.T) {
	root := copyFixture(t, filepath.Join("..", "..", "fixtures", "tracer"))
	shapesPath := filepath.Join(root, "shapes.ttl")
	shapes, err := os.ReadFile(shapesPath)
	if err != nil {
		t.Fatal(err)
	}
	shapes = append(shapes, []byte("\n<https://example.org/standard/ProjectTitleShape> <https://shape2form.dev/vocab/ui#order> 1 .\n")...)
	if err := os.WriteFile(shapesPath, shapes, 0o644); err != nil {
		t.Fatal(err)
	}
	updateArtifactDigest(t, root, "shapes.ttl", shapes)

	_, err = Open(root)
	var contractErr *ContractError
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "graph.presentation_order.forbidden") {
		t.Fatalf("expected forbidden presentation-order diagnostic, got %v", err)
	}
}

func TestCanonicalNodeShapesMustBeDeclared(t *testing.T) {
	root := copyFixture(t, filepath.Join("..", "..", "fixtures", "tracer"))
	shapesPath := filepath.Join(root, "shapes.ttl")
	shapes, err := os.ReadFile(shapesPath)
	if err != nil {
		t.Fatal(err)
	}
	shapes = append(shapes, []byte("\n<https://example.org/standard/HiddenShape> a <http://www.w3.org/ns/shacl#NodeShape> .\n")...)
	if err := os.WriteFile(shapesPath, shapes, 0o644); err != nil {
		t.Fatal(err)
	}
	updateArtifactDigest(t, root, "shapes.ttl", shapes)

	_, err = Open(root)
	var contractErr *ContractError
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "graph.canonical_shape.undeclared") {
		t.Fatalf("expected undeclared canonical-shape diagnostic, got %v", err)
	}
}

func TestCanonicalPropertyShapesMustBeDeclared(t *testing.T) {
	root := copyFixture(t, filepath.Join("..", "..", "fixtures", "tracer"))
	shapesPath := filepath.Join(root, "shapes.ttl")
	shapes, err := os.ReadFile(shapesPath)
	if err != nil {
		t.Fatal(err)
	}
	shapes = append(shapes, []byte("\n<https://example.org/standard/HiddenTitleShape> a <http://www.w3.org/ns/shacl#PropertyShape> .\n")...)
	if err := os.WriteFile(shapesPath, shapes, 0o644); err != nil {
		t.Fatal(err)
	}
	updateArtifactDigest(t, root, "shapes.ttl", shapes)

	_, err = Open(root)
	var contractErr *ContractError
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "graph.canonical_shape.undeclared") {
		t.Fatalf("expected undeclared canonical-shape diagnostic, got %v", err)
	}
}

func TestPinnedReferenceDoesNotImplyAuthorization(t *testing.T) {
	root := copyFixture(t, filepath.Join("..", "..", "fixtures", "tracer"))
	referencesPath := filepath.Join(root, "references.ttl")
	references, err := os.ReadFile(referencesPath)
	if err != nil {
		t.Fatal(err)
	}
	references = append(references, []byte(`
<https://example.org/standard/ContextIndicator> a <https://standard2shape.dev/vocab#IndicatorReference> ;
  <https://standard2shape.dev/vocab#artifactVersion> "1.0.0" ;
  <https://standard2shape.dev/vocab#artifactDigest> "sha256:4444444444444444444444444444444444444444444444444444444444444444" .
`)...)
	if err := os.WriteFile(referencesPath, references, 0o644); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(root, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.References = append(manifest.References, ArtifactReference{
		Kind:    "indicator",
		ID:      "https://example.org/standard/ContextIndicator",
		Version: "1.0.0",
		Digest:  "sha256:4444444444444444444444444444444444444444444444444444444444444444",
		Source:  "references.ttl",
	})
	for index := range manifest.Artifacts {
		if manifest.Artifacts[index].Path == "references.ttl" {
			manifest.Artifacts[index].Digest = digest(references)
		}
	}
	updated, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(updated, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Open(root); err != nil {
		t.Fatalf("a pinned contextual reference must not require authorization: %v", err)
	}
}

func TestVersionedContractArtifactsParseLocally(t *testing.T) {
	schemaData, err := os.ReadFile(filepath.Join("..", "..", "contracts", "v0", "package-manifest.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		t.Fatalf("parse manifest schema: %v", err)
	}
	if schema["$id"] != "https://standard2shape.dev/contracts/v0/package-manifest.schema.json" {
		t.Fatalf("schema id = %v", schema["$id"])
	}

	vocabulary, err := os.Open(filepath.Join("..", "..", "contracts", "v0", "standard2shape.ttl"))
	if err != nil {
		t.Fatal(err)
	}
	defer vocabulary.Close()
	decoder := rdf.NewTripleDecoder(vocabulary, rdf.Turtle)
	statements := 0
	for {
		_, err := decoder.Decode()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("parse vocabulary: %v", err)
		}
		statements++
	}
	if statements == 0 {
		t.Fatal("vocabulary contains no statements")
	}
}

func hasDiagnostic(diagnostics []Diagnostic, code string) bool {
	return slices.ContainsFunc(diagnostics, func(diagnostic Diagnostic) bool { return diagnostic.Code == code })
}

func copyFixture(t *testing.T, source string) string {
	t.Helper()
	target := t.TempDir()
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(source, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, entry.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return target
}

func updateArtifactDigest(t *testing.T, root, relative string, data []byte) {
	t.Helper()
	manifestPath := filepath.Join(root, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	for index := range manifest.Artifacts {
		if manifest.Artifacts[index].Path == relative {
			manifest.Artifacts[index].Digest = digest(data)
		}
	}
	updated, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(updated, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
