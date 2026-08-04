package tracer

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSessionOpensRepresentativeBundleOffline(t *testing.T) {
	session, err := NewSession(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(session.Close)

	snapshot := session.Snapshot()
	if snapshot.Document.Label != "Project design document" {
		t.Fatalf("document label = %q", snapshot.Document.Label)
	}
	if len(snapshot.Document.Sections) != 1 || len(snapshot.Document.Sections[0].Placements) != 1 {
		t.Fatalf("unexpected document tree: %#v", snapshot.Document)
	}
	if len(snapshot.Shapes) != 1 || len(snapshot.Shapes[0].Properties) != 1 {
		t.Fatalf("unexpected shape summary: %#v", snapshot.Shapes)
	}
	if len(snapshot.Sources) != 3 {
		t.Fatalf("source count = %d", len(snapshot.Sources))
	}
	if len(snapshot.Unsupported) != 1 || snapshot.Unsupported[0].Kind != "SHACL-SPARQL" {
		t.Fatalf("unsupported summary = %#v", snapshot.Unsupported)
	}
	if len(snapshot.Assessment.Cases) != 2 {
		t.Fatalf("validation cases = %#v", snapshot.Assessment.Cases)
	}
	for _, testCase := range snapshot.Assessment.Cases {
		if testCase.Conforms != testCase.ExpectedConforms {
			t.Fatalf("case %q conforms=%v expected=%v", testCase.Name, testCase.Conforms, testCase.ExpectedConforms)
		}
	}
	if len(snapshot.Preview.Fields) != 1 || !snapshot.Preview.Fields[0].Required {
		t.Fatalf("preview = %#v", snapshot.Preview)
	}
}

func TestGuidanceChangeProducesPatchAndPreservesUnsupportedGraph(t *testing.T) {
	session, err := NewSession(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(session.Close)

	before := session.Snapshot()
	originalDigest := before.Unsupported[0].Digest
	updated := "Explain the project purpose, location, and intended outcomes for an independent reviewer."

	after, err := session.ChangeGuidance(updated)
	if err != nil {
		t.Fatal(err)
	}
	if after.Document.Sections[0].Guidance != updated {
		t.Fatalf("guidance = %q", after.Document.Sections[0].Guidance)
	}
	if after.Change == nil || !after.Change.UnsupportedPreserved {
		t.Fatalf("change = %#v", after.Change)
	}
	if !strings.Contains(after.Change.Patch, "-  s2s:canonicalGuidance") || !strings.Contains(after.Change.Patch, "+  s2s:canonicalGuidance") {
		t.Fatalf("patch does not describe guidance replacement:\n%s", after.Change.Patch)
	}
	if after.Unsupported[0].Digest != originalDigest {
		t.Fatalf("unsupported digest changed: %s -> %s", originalDigest, after.Unsupported[0].Digest)
	}

	reloaded, err := Load(session.Root())
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Document.Sections[0].Guidance != updated {
		t.Fatalf("reloaded guidance = %q", reloaded.Document.Sections[0].Guidance)
	}
	after.Change = nil
	if !reflect.DeepEqual(after, reloaded) {
		t.Fatalf("reloaded graph-derived workspace differs:\nafter: %#v\nreload: %#v", after, reloaded)
	}
}

func TestGuidanceChangeRejectsEmptyText(t *testing.T) {
	session, err := NewSession(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(session.Close)

	if _, err := session.ChangeGuidance("   "); err == nil {
		t.Fatal("expected empty guidance to be rejected")
	}
}

func TestSafeJoinRejectsBundleEscape(t *testing.T) {
	root := t.TempDir()
	if _, err := safeJoin(root, filepath.Join("..", "outside.ttl")); err == nil {
		t.Fatal("expected parent path to be rejected")
	}
	if _, err := safeJoin(root, filepath.Join(root, "absolute.ttl")); err == nil {
		t.Fatal("expected absolute path to be rejected")
	}
}
