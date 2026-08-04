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
	if len(pkg.Manifest.DocumentRoots) != 1 || len(pkg.Manifest.CanonicalShapes) != 1 {
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
		if manifest.Artifacts[index].Path == "shapes.ttl" {
			manifest.Artifacts[index].Digest = digest(shapes)
		}
	}
	updated, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(updated, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = Open(root)
	var contractErr *ContractError
	if !errors.As(err, &contractErr) || !hasDiagnostic(contractErr.Diagnostics, "graph.presentation_order.forbidden") {
		t.Fatalf("expected forbidden presentation-order diagnostic, got %v", err)
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

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
