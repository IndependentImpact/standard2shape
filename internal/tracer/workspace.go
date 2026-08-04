package tracer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/knakk/rdf"

	"github.com/IndependentImpact/standard2shape/internal/packagecontract"
)

const (
	rdfType           = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	rdfsLabel         = "http://www.w3.org/2000/01/rdf-schema#label"
	shNodeShape       = "http://www.w3.org/ns/shacl#NodeShape"
	shPropertyShape   = "http://www.w3.org/ns/shacl#PropertyShape"
	shTargetClass     = "http://www.w3.org/ns/shacl#targetClass"
	shProperty        = "http://www.w3.org/ns/shacl#property"
	shPath            = "http://www.w3.org/ns/shacl#path"
	shMinCount        = "http://www.w3.org/ns/shacl#minCount"
	shDatatype        = "http://www.w3.org/ns/shacl#datatype"
	shName            = "http://www.w3.org/ns/shacl#name"
	shDescription     = "http://www.w3.org/ns/shacl#description"
	shSPARQL          = "http://www.w3.org/ns/shacl#sparql"
	s2sBundle         = "https://standard2shape.dev/vocab#OrderedShapeBundle"
	s2sSection        = "https://standard2shape.dev/vocab#DocumentSection"
	s2sPlacement      = "https://standard2shape.dev/vocab#DocumentPlacement"
	s2sRootSection    = "https://standard2shape.dev/vocab#hasRootSection"
	s2sCanonicalGuide = "https://standard2shape.dev/vocab#canonicalGuidance"
	s2sMember         = "https://standard2shape.dev/vocab#member"
	s2sPosition       = "https://standard2shape.dev/vocab#position"
	s2sShape          = "https://standard2shape.dev/vocab#shape"
	validatorName     = "standard2shape tracer validator"
	validatorVersion  = "0.1.0-tracer"
)

type sourcedTriple struct {
	Triple rdf.Triple
	Source string
}

type loadedBundle struct {
	Snapshot Snapshot
	Triples  []sourcedTriple
}

// Session owns an isolated writable copy of the synthetic fixture. The
// repository fixture is never changed by browser interactions or tests.
type Session struct {
	mu       sync.RWMutex
	root     string
	snapshot Snapshot
}

func NewSession(fixtureRoot string) (*Session, error) {
	absFixture, err := filepath.Abs(fixtureRoot)
	if err != nil {
		return nil, err
	}
	pkg, err := packagecontract.Open(absFixture)
	if err != nil {
		return nil, err
	}
	manifest := pkg.Manifest
	tempRoot, err := os.MkdirTemp("", "standard2shape-tracer-")
	if err != nil {
		return nil, err
	}
	cleanup := func(loadErr error) (*Session, error) {
		_ = os.RemoveAll(tempRoot)
		return nil, loadErr
	}
	paths := append([]string{"manifest.json"}, manifest.LocalPaths()...)
	for _, rel := range paths {
		source, err := safeJoin(absFixture, rel)
		if err != nil {
			return cleanup(err)
		}
		target, err := safeJoin(tempRoot, rel)
		if err != nil {
			return cleanup(err)
		}
		if err := copyFile(source, target); err != nil {
			return cleanup(err)
		}
	}
	loaded, err := loadBundle(tempRoot)
	if err != nil {
		return cleanup(err)
	}
	return &Session{root: tempRoot, snapshot: loaded.Snapshot}, nil
}

func (s *Session) Close() {
	if s == nil {
		return
	}
	_ = os.RemoveAll(s.root)
}

func (s *Session) Root() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.root
}

func (s *Session) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

func (s *Session) ChangeGuidance(next string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next = strings.TrimSpace(next)
	if next == "" {
		return Snapshot{}, errors.New("canonical guidance must not be empty")
	}
	if strings.ContainsRune(next, '\x00') {
		return Snapshot{}, errors.New("canonical guidance contains an invalid null character")
	}
	if len(s.snapshot.Document.Sections) == 0 {
		return Snapshot{}, errors.New("bundle has no editable document section")
	}
	section := s.snapshot.Document.Sections[0]
	if section.GuidanceSource == "" {
		return Snapshot{}, errors.New("canonical guidance has no unambiguous source")
	}
	if section.Guidance == next {
		return s.snapshot, nil
	}
	path, err := safeJoin(s.root, section.GuidanceSource)
	if err != nil {
		return Snapshot{}, err
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	oldToken := fmt.Sprintf("\"%s\"@en", escapeTurtleString(section.Guidance))
	newToken := fmt.Sprintf("\"%s\"@en", escapeTurtleString(next))
	if strings.Count(string(original), oldToken) != 1 {
		return Snapshot{}, fmt.Errorf("expected one managed guidance token in %s", section.GuidanceSource)
	}
	updatedSource := strings.Replace(string(original), oldToken, newToken, 1)
	if err := os.WriteFile(path, []byte(updatedSource), 0o644); err != nil {
		return Snapshot{}, err
	}
	loaded, err := loadBundle(s.root)
	if err != nil {
		_ = os.WriteFile(path, original, 0o644)
		return Snapshot{}, fmt.Errorf("reload changed bundle: %w", err)
	}
	beforeDigest := firstUnsupportedDigest(s.snapshot.Unsupported)
	afterDigest := firstUnsupportedDigest(loaded.Snapshot.Unsupported)
	if beforeDigest == "" || beforeDigest != afterDigest {
		_ = os.WriteFile(path, original, 0o644)
		return Snapshot{}, errors.New("unsupported SHACL changed during managed edit; original source restored")
	}
	loaded.Snapshot.Change = &ChangeSummary{
		Source:               section.GuidanceSource,
		Before:               section.Guidance,
		After:                next,
		SemanticDiff:         fmt.Sprintf("Canonical guidance changed on %s\n− %s\n+ %s", compactIRI(section.ID), section.Guidance, next),
		Patch:                guidancePatch(section.GuidanceSource, oldToken, newToken),
		UnsupportedPreserved: true,
	}
	s.snapshot = loaded.Snapshot
	return s.snapshot, nil
}

// Load parses an existing workspace without following owl:imports or any other
// remote reference. It is exported for round-trip and integration tests.
func Load(root string) (Snapshot, error) {
	loaded, err := loadBundle(root)
	if err != nil {
		return Snapshot{}, err
	}
	return loaded.Snapshot, nil
}

func loadBundle(root string) (loadedBundle, error) {
	manifest, err := readManifest(root)
	if err != nil {
		return loadedBundle{}, err
	}
	var triples []sourcedTriple
	var sources []SourceSummary
	for _, artifact := range manifest.Artifacts {
		rel := artifact.Path
		path, err := safeJoin(root, rel)
		if err != nil {
			return loadedBundle{}, err
		}
		parsed, err := parseTurtle(path)
		if err != nil {
			return loadedBundle{}, fmt.Errorf("parse %s: %w", rel, err)
		}
		for _, triple := range parsed {
			triples = append(triples, sourcedTriple{Triple: triple, Source: rel})
		}
		sources = append(sources, SourceSummary{Path: rel, Statements: len(parsed)})
	}
	document, shapes, err := summarizeGraph(manifest, triples)
	if err != nil {
		return loadedBundle{}, err
	}
	unsupported := summarizeUnsupported(triples)
	assessment, err := assessData(root, manifest, shapes)
	if err != nil {
		return loadedBundle{}, err
	}
	preview := buildPreview(document, shapes)
	return loadedBundle{
		Snapshot: Snapshot{
			Manifest:    manifest,
			Document:    document,
			Shapes:      shapes,
			Sources:     sources,
			Assessment:  assessment,
			Preview:     preview,
			Unsupported: unsupported,
		},
		Triples: triples,
	}, nil
}

func readManifest(root string) (Manifest, error) {
	return packagecontract.ReadManifest(root)
}

func summarizeGraph(manifest Manifest, triples []sourcedTriple) (DocumentSummary, []ShapeSummary, error) {
	if len(manifest.DocumentRoots) == 0 {
		return DocumentSummary{}, nil, errors.New("manifest contains no document root")
	}
	documentID := manifest.DocumentRoots[0].ID
	if !hasType(triples, documentID, s2sBundle) {
		return DocumentSummary{}, nil, fmt.Errorf("manifest document %s is not an OrderedShapeBundle", documentID)
	}
	document := DocumentSummary{ID: documentID, Label: literalObject(triples, documentID, rdfsLabel)}
	for _, sectionID := range objectValues(triples, documentID, s2sRootSection) {
		if !hasType(triples, sectionID, s2sSection) {
			return DocumentSummary{}, nil, fmt.Errorf("root section %s is not a DocumentSection", sectionID)
		}
		section := SectionSummary{
			ID:             sectionID,
			Label:          literalObject(triples, sectionID, rdfsLabel),
			Guidance:       literalObject(triples, sectionID, s2sCanonicalGuide),
			GuidanceSource: sourceFor(triples, sectionID, s2sCanonicalGuide),
		}
		for _, placementID := range objectValues(triples, sectionID, s2sMember) {
			if !hasType(triples, placementID, s2sPlacement) {
				continue
			}
			position, _ := strconv.Atoi(literalObject(triples, placementID, s2sPosition))
			section.Placements = append(section.Placements, PlacementSummary{
				ID:       placementID,
				Label:    literalObject(triples, placementID, rdfsLabel),
				Position: position,
				ShapeID:  objectValue(triples, placementID, s2sShape),
			})
		}
		sort.Slice(section.Placements, func(i, j int) bool { return section.Placements[i].Position < section.Placements[j].Position })
		document.Sections = append(document.Sections, section)
	}
	if len(document.Sections) == 0 {
		return DocumentSummary{}, nil, errors.New("ordered bundle has no root section")
	}

	var shapes []ShapeSummary
	for _, statement := range triples {
		if statement.Triple.Pred.String() != rdfType || statement.Triple.Obj.String() != shNodeShape {
			continue
		}
		shapeID := statement.Triple.Subj.String()
		shape := ShapeSummary{
			ID:          shapeID,
			TargetClass: objectValue(triples, shapeID, shTargetClass),
			Source:      statement.Source,
		}
		for _, propertyID := range objectValues(triples, shapeID, shProperty) {
			if !hasType(triples, propertyID, shPropertyShape) {
				return DocumentSummary{}, nil, fmt.Errorf("shape %s references %s, which is not a PropertyShape", shapeID, propertyID)
			}
			minCount, _ := strconv.Atoi(literalObject(triples, propertyID, shMinCount))
			shape.Properties = append(shape.Properties, PropertySummary{
				ID:       propertyID,
				Path:     objectValue(triples, propertyID, shPath),
				Label:    literalObject(triples, propertyID, shName),
				Guidance: literalObject(triples, propertyID, shDescription),
				Datatype: objectValue(triples, propertyID, shDatatype),
				MinCount: minCount,
				Source:   sourceFor(triples, propertyID, rdfType),
			})
		}
		shapes = append(shapes, shape)
	}
	if len(shapes) == 0 {
		return DocumentSummary{}, nil, errors.New("bundle contains no supported NodeShape")
	}
	sort.Slice(shapes, func(i, j int) bool { return shapes[i].ID < shapes[j].ID })
	return document, shapes, nil
}

func assessData(root string, manifest Manifest, shapes []ShapeSummary) (ValidationAssessment, error) {
	assessment := ValidationAssessment{
		PackageID:        manifest.ID,
		PackageVersion:   manifest.Version,
		Validator:        validatorName,
		ValidatorVersion: validatorVersion,
	}
	for _, test := range manifest.ConformanceVectors {
		path, err := safeJoin(root, test.Path)
		if err != nil {
			return ValidationAssessment{}, err
		}
		data, err := parseTurtle(path)
		if err != nil {
			return ValidationAssessment{}, fmt.Errorf("parse test vector %s: %w", test.Name, err)
		}
		var violations []string
		for _, shape := range shapes {
			for _, focus := range subjectsWith(data, rdfType, shape.TargetClass) {
				for _, property := range shape.Properties {
					count := countPredicate(data, focus, property.Path)
					if count < property.MinCount {
						violations = append(violations, fmt.Sprintf("%s requires at least %d value for %s; found %d", compactIRI(focus), property.MinCount, property.Label, count))
					}
				}
			}
		}
		assessment.Cases = append(assessment.Cases, ValidationCase{
			Name:             test.Name,
			Conforms:         len(violations) == 0,
			ExpectedConforms: test.Expected == "conforms",
			Violations:       violations,
		})
	}
	return assessment, nil
}

func buildPreview(document DocumentSummary, shapes []ShapeSummary) PreviewForm {
	preview := PreviewForm{ID: document.ID, Title: document.Label}
	for _, shape := range shapes {
		for _, property := range shape.Properties {
			preview.Fields = append(preview.Fields, PreviewField{
				Path:     property.Path,
				Label:    property.Label,
				Help:     property.Guidance,
				Required: property.MinCount > 0,
			})
		}
	}
	return preview
}

func summarizeUnsupported(triples []sourcedTriple) []UnsupportedSummary {
	var summaries []UnsupportedSummary
	for _, statement := range triples {
		if statement.Triple.Pred.String() != shSPARQL {
			continue
		}
		collected := []sourcedTriple{statement}
		queue := []string{statement.Triple.Obj.String()}
		seen := map[string]bool{}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			if seen[current] {
				continue
			}
			seen[current] = true
			for _, candidate := range triples {
				if candidate.Source != statement.Source || candidate.Triple.Subj.String() != current {
					continue
				}
				collected = append(collected, candidate)
				if _, blank := candidate.Triple.Obj.(rdf.Blank); blank {
					queue = append(queue, candidate.Triple.Obj.String())
				}
			}
		}
		serialized := make([]string, 0, len(collected))
		for _, item := range collected {
			serialized = append(serialized, item.Triple.Serialize(rdf.NTriples))
		}
		sort.Strings(serialized)
		digest := sha256.Sum256([]byte(strings.Join(serialized, "\n")))
		summaries = append(summaries, UnsupportedSummary{
			Kind:   "SHACL-SPARQL",
			Source: statement.Source,
			Digest: "sha256:" + hex.EncodeToString(digest[:]),
			Detail: "Visible and preserved, but not editable in the guided tracer.",
		})
	}
	return summaries
}

func parseTurtle(path string) ([]rdf.Triple, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := rdf.NewTripleDecoder(file, rdf.Turtle)
	var triples []rdf.Triple
	for {
		triple, err := decoder.Decode()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		triples = append(triples, triple)
	}
	return triples, nil
}

func safeJoin(root, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute package path is not allowed: %s", rel)
	}
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	joined := filepath.Join(cleanRoot, filepath.Clean(rel))
	relative, err := filepath.Rel(cleanRoot, joined)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("package path escapes bundle root: %s", rel)
	}
	return joined, nil
}

func copyFile(source, target string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func hasType(triples []sourcedTriple, subject, object string) bool {
	for _, candidate := range objectValues(triples, subject, rdfType) {
		if candidate == object {
			return true
		}
	}
	return false
}

func objectValue(triples []sourcedTriple, subject, predicate string) string {
	values := objectValues(triples, subject, predicate)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func objectValues(triples []sourcedTriple, subject, predicate string) []string {
	var values []string
	for _, statement := range triples {
		if statement.Triple.Subj.String() == subject && statement.Triple.Pred.String() == predicate {
			values = append(values, statement.Triple.Obj.String())
		}
	}
	return values
}

func literalObject(triples []sourcedTriple, subject, predicate string) string {
	for _, statement := range triples {
		if statement.Triple.Subj.String() != subject || statement.Triple.Pred.String() != predicate {
			continue
		}
		if literal, ok := statement.Triple.Obj.(rdf.Literal); ok {
			return literal.String()
		}
	}
	return ""
}

func sourceFor(triples []sourcedTriple, subject, predicate string) string {
	for _, statement := range triples {
		if statement.Triple.Subj.String() == subject && statement.Triple.Pred.String() == predicate {
			return statement.Source
		}
	}
	return ""
}

func subjectsWith(triples []rdf.Triple, predicate, object string) []string {
	var subjects []string
	for _, triple := range triples {
		if triple.Pred.String() == predicate && triple.Obj.String() == object {
			subjects = append(subjects, triple.Subj.String())
		}
	}
	return subjects
}

func countPredicate(triples []rdf.Triple, subject, predicate string) int {
	count := 0
	for _, triple := range triples {
		if triple.Subj.String() == subject && triple.Pred.String() == predicate {
			count++
		}
	}
	return count
}

func firstUnsupportedDigest(items []UnsupportedSummary) string {
	if len(items) == 0 {
		return ""
	}
	return items[0].Digest
}

func escapeTurtleString(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	)
	return replacer.Replace(value)
}

func guidancePatch(source, before, after string) string {
	return fmt.Sprintf("--- a/%s\n+++ b/%s\n@@ canonical guidance @@\n-  s2s:canonicalGuidance %s ;\n+  s2s:canonicalGuidance %s ;", source, source, before, after)
}

func compactIRI(iri string) string {
	trimmed := strings.TrimRight(iri, "/#")
	if index := strings.LastIndexAny(trimmed, "/#"); index >= 0 {
		return trimmed[index+1:]
	}
	return iri
}
