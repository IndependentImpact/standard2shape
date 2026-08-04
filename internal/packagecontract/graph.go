package packagecontract

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/knakk/rdf"
)

const (
	rdfType            = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	owlImports         = "http://www.w3.org/2002/07/owl#imports"
	shNodeShape        = "http://www.w3.org/ns/shacl#NodeShape"
	uiOrder            = "https://shape2form.dev/vocab/ui#order"
	s2sStandardRelease = "https://standard2shape.dev/vocab#StandardRelease"
	s2sBundle          = "https://standard2shape.dev/vocab#OrderedShapeBundle"
	s2sSection         = "https://standard2shape.dev/vocab#DocumentSection"
	s2sPlacement       = "https://standard2shape.dev/vocab#DocumentPlacement"
	s2sIndicatorRef    = "https://standard2shape.dev/vocab#IndicatorReference"
	s2sMethodologyRef  = "https://standard2shape.dev/vocab#MethodologyReference"
	s2sReasoning       = "https://standard2shape.dev/vocab#ReasoningProfile"
	s2sDefinesDocument = "https://standard2shape.dev/vocab#definesDocument"
	s2sRootSection     = "https://standard2shape.dev/vocab#hasRootSection"
	s2sMember          = "https://standard2shape.dev/vocab#member"
	s2sPosition        = "https://standard2shape.dev/vocab#position"
	s2sShape           = "https://standard2shape.dev/vocab#shape"
	s2sCanonicalGuide  = "https://standard2shape.dev/vocab#canonicalGuidance"
	s2sReleaseVersion  = "https://standard2shape.dev/vocab#releaseVersion"
	s2sArtifactVersion = "https://standard2shape.dev/vocab#artifactVersion"
	s2sArtifactDigest  = "https://standard2shape.dev/vocab#artifactDigest"
	s2sUsesReasoning   = "https://standard2shape.dev/vocab#usesReasoningProfile"
	s2sProfileVersion  = "https://standard2shape.dev/vocab#profileVersion"
	s2sEntailment      = "https://standard2shape.dev/vocab#entailmentRegime"
	s2sAuthIndicator   = "https://standard2shape.dev/vocab#authorizesIndicator"
	s2sAuthMethodology = "https://standard2shape.dev/vocab#authorizesMethodology"
	s2sForIndicator    = "https://standard2shape.dev/vocab#forIndicator"
	xsdPositiveInteger = "http://www.w3.org/2001/XMLSchema#positiveInteger"
)

type sourcedTriple struct {
	Triple rdf.Triple
	Source string
}

func parseTurtle(reader io.Reader) ([]rdf.Triple, error) {
	decoder := rdf.NewTripleDecoder(reader, rdf.Turtle)
	var triples []rdf.Triple
	for {
		triple, err := decoder.Decode()
		if errors.Is(err, io.EOF) {
			return triples, nil
		}
		if err != nil {
			return nil, err
		}
		triples = append(triples, triple)
	}
}

func validateGraph(manifest Manifest, triples []sourcedTriple) []Diagnostic {
	var diagnostics []Diagnostic
	for _, statement := range triples {
		if statement.Triple.Pred.String() == uiOrder {
			diagnostics = append(diagnostics, diagnostic("graph.presentation_order.forbidden", statement.Source, "canonical sources must use s2s:position, not shape2form ui:order"))
		}
		if statement.Triple.Pred.String() == s2sCanonicalGuide {
			literal, ok := statement.Triple.Obj.(rdf.Literal)
			if !ok || literal.Lang() == "" {
				diagnostics = append(diagnostics, diagnostic("graph.guidance.invalid", statement.Source, "s2s:canonicalGuidance must be a language-tagged string"))
			}
		}
	}
	diagnostics = append(diagnostics, validateImports(manifest, triples)...)
	diagnostics = append(diagnostics, validateStandardRelease(manifest, triples)...)
	diagnostics = append(diagnostics, validateReasoningProfile(manifest, triples)...)
	diagnostics = append(diagnostics, validateReferences(manifest, triples)...)
	diagnostics = append(diagnostics, validateDocuments(manifest, triples)...)
	diagnostics = append(diagnostics, validateShapes(manifest, triples)...)
	return diagnostics
}

func validateImports(manifest Manifest, triples []sourcedTriple) []Diagnostic {
	declared := map[string]bool{}
	for _, item := range manifest.Imports {
		declared[item.Source+"\x00"+item.IRI] = true
	}
	observed := map[string]bool{}
	for _, statement := range triples {
		if statement.Triple.Pred.String() == owlImports {
			observed[statement.Source+"\x00"+statement.Triple.Obj.String()] = true
		}
	}
	var diagnostics []Diagnostic
	for key := range observed {
		if !declared[key] {
			parts := strings.SplitN(key, "\x00", 2)
			diagnostics = append(diagnostics, diagnostic("graph.import.mismatch", parts[0], "owl:imports %s is not pinned in the manifest", parts[1]))
		}
	}
	for key := range declared {
		if !observed[key] {
			parts := strings.SplitN(key, "\x00", 2)
			diagnostics = append(diagnostics, diagnostic("graph.import.mismatch", parts[0], "manifest import %s is absent from the source graph", parts[1]))
		}
	}
	return diagnostics
}

func validateStandardRelease(manifest Manifest, triples []sourcedTriple) []Diagnostic {
	entity := manifest.StandardRelease
	var diagnostics []Diagnostic
	diagnostics = append(diagnostics, requireTypedEntity(triples, entity.ID, s2sStandardRelease, entity.Source, "graph.standard_release.invalid")...)
	for _, subject := range subjectsWith(triples, rdfType, s2sStandardRelease) {
		if subject != entity.ID {
			diagnostics = append(diagnostics, diagnostic("graph.standard_release.invalid", sourceForType(triples, subject, s2sStandardRelease), "standard release %s is not declared by the manifest", subject))
		}
	}
	if value := singleLiteral(triples, entity.ID, s2sReleaseVersion); value != entity.Version {
		diagnostics = append(diagnostics, diagnostic("graph.standard_release.invalid", entity.Source, "release version for %s is %q, expected %q", entity.ID, value, entity.Version))
	}
	documentIDs := make([]string, 0, len(manifest.DocumentRoots))
	for _, root := range manifest.DocumentRoots {
		documentIDs = append(documentIDs, root.ID)
	}
	if !sameValues(objectValues(triples, entity.ID, s2sDefinesDocument), documentIDs) {
		diagnostics = append(diagnostics, diagnostic("graph.standard_release.invalid", entity.Source, "standard release document declarations must exactly match manifest document roots"))
	}
	if !sameValues(objectValues(triples, entity.ID, s2sUsesReasoning), []string{manifest.ReasoningProfile.ID}) {
		diagnostics = append(diagnostics, diagnostic("graph.standard_release.invalid", entity.Source, "standard release must select exactly the manifest reasoning profile %s", manifest.ReasoningProfile.ID))
	}
	indicatorIDs, methodologyIDs := referenceIDs(manifest.References)
	if !sameValues(objectValues(triples, entity.ID, s2sAuthIndicator), indicatorIDs) {
		diagnostics = append(diagnostics, diagnostic("graph.reference.invalid", entity.Source, "authorized indicators must exactly match manifest references"))
	}
	if !sameValues(objectValues(triples, entity.ID, s2sAuthMethodology), methodologyIDs) {
		diagnostics = append(diagnostics, diagnostic("graph.reference.invalid", entity.Source, "authorized methodologies must exactly match manifest references"))
	}
	return diagnostics
}

func validateReasoningProfile(manifest Manifest, triples []sourcedTriple) []Diagnostic {
	profile := manifest.ReasoningProfile
	diagnostics := requireTypedEntity(triples, profile.ID, s2sReasoning, profile.Source, "graph.reasoning_profile.invalid")
	for _, subject := range subjectsWith(triples, rdfType, s2sReasoning) {
		if subject != profile.ID {
			diagnostics = append(diagnostics, diagnostic("graph.reasoning_profile.invalid", sourceForType(triples, subject, s2sReasoning), "reasoning profile %s is not declared by the manifest", subject))
		}
	}
	if value := singleLiteral(triples, profile.ID, s2sProfileVersion); value != profile.Version {
		diagnostics = append(diagnostics, diagnostic("graph.reasoning_profile.invalid", profile.Source, "profile version for %s is %q, expected %q", profile.ID, value, profile.Version))
	}
	if strings.TrimSpace(singleLiteral(triples, profile.ID, s2sEntailment)) == "" {
		diagnostics = append(diagnostics, diagnostic("graph.reasoning_profile.invalid", profile.Source, "reasoning profile %s must declare an entailment regime", profile.ID))
	}
	return diagnostics
}

func validateReferences(manifest Manifest, triples []sourcedTriple) []Diagnostic {
	var diagnostics []Diagnostic
	indicatorIDs := map[string]bool{}
	declared := map[string]bool{}
	for _, reference := range manifest.References {
		declared[reference.Kind+"\x00"+reference.ID] = true
		if reference.Kind == "indicator" {
			indicatorIDs[reference.ID] = true
		}
	}
	for _, subject := range subjectsWith(triples, rdfType, s2sIndicatorRef) {
		if !declared["indicator\x00"+subject] {
			diagnostics = append(diagnostics, diagnostic("graph.reference.invalid", sourceForType(triples, subject, s2sIndicatorRef), "indicator reference %s is not declared by the manifest", subject))
		}
	}
	for _, subject := range subjectsWith(triples, rdfType, s2sMethodologyRef) {
		if !declared["methodology\x00"+subject] {
			diagnostics = append(diagnostics, diagnostic("graph.reference.invalid", sourceForType(triples, subject, s2sMethodologyRef), "methodology reference %s is not declared by the manifest", subject))
		}
	}
	for _, reference := range manifest.References {
		typeIRI := s2sIndicatorRef
		authorization := s2sAuthIndicator
		if reference.Kind == "methodology" {
			typeIRI = s2sMethodologyRef
			authorization = s2sAuthMethodology
		}
		diagnostics = append(diagnostics, requireTypedEntity(triples, reference.ID, typeIRI, reference.Source, "graph.reference.invalid")...)
		if singleLiteral(triples, reference.ID, s2sArtifactVersion) != reference.Version {
			diagnostics = append(diagnostics, diagnostic("graph.reference.invalid", reference.Source, "%s reference %s has a different version", reference.Kind, reference.ID))
		}
		if singleLiteral(triples, reference.ID, s2sArtifactDigest) != reference.Digest {
			diagnostics = append(diagnostics, diagnostic("graph.reference.invalid", reference.Source, "%s reference %s has a different digest", reference.Kind, reference.ID))
		}
		if !contains(objectValues(triples, manifest.StandardRelease.ID, authorization), reference.ID) {
			diagnostics = append(diagnostics, diagnostic("graph.reference.invalid", manifest.StandardRelease.Source, "standard release does not authorize declared %s %s", reference.Kind, reference.ID))
		}
		if reference.Kind == "methodology" {
			targets := objectValues(triples, reference.ID, s2sForIndicator)
			if len(targets) != 1 || !indicatorIDs[targets[0]] {
				diagnostics = append(diagnostics, diagnostic("graph.reference.invalid", reference.Source, "methodology %s must target exactly one declared indicator reference", reference.ID))
			}
		}
	}
	return diagnostics
}

func validateDocuments(manifest Manifest, triples []sourcedTriple) []Diagnostic {
	var diagnostics []Diagnostic
	documentIDs := map[string]bool{}
	for _, root := range manifest.DocumentRoots {
		documentIDs[root.ID] = true
	}
	for _, subject := range subjectsWith(triples, rdfType, s2sBundle) {
		if !documentIDs[subject] {
			diagnostics = append(diagnostics, diagnostic("graph.document_root.invalid", sourceForType(triples, subject, s2sBundle), "ordered shape bundle %s is not declared by the manifest", subject))
		}
	}
	shapeIDs := map[string]bool{}
	for _, shape := range manifest.CanonicalShapes {
		shapeIDs[shape.ID] = true
	}
	parents := map[string]string{}
	for _, root := range manifest.DocumentRoots {
		diagnostics = append(diagnostics, requireTypedEntity(triples, root.ID, s2sBundle, root.Source, "graph.document_root.invalid")...)
		sections := objectValues(triples, root.ID, s2sRootSection)
		if len(sections) == 0 {
			diagnostics = append(diagnostics, diagnostic("graph.document_root.invalid", root.Source, "document root %s has no root section", root.ID))
			continue
		}
		if len(sections) > 1 {
			diagnostics = append(diagnostics, validateSiblingPositions(triples, root.Source, sections)...)
		}
		visited := map[string]bool{}
		active := map[string]bool{}
		for _, section := range sections {
			diagnostics = append(diagnostics, validateSection(triples, section, root.ID, root.Source, shapeIDs, parents, visited, active)...)
		}
	}
	return diagnostics
}

func validateSection(triples []sourcedTriple, section, parent, fallbackSource string, shapeIDs map[string]bool, parents map[string]string, visited, active map[string]bool) []Diagnostic {
	source := sourceForType(triples, section, s2sSection)
	if source == "" {
		source = fallbackSource
	}
	if active[section] {
		return []Diagnostic{diagnostic("graph.document_tree.cycle", source, "document section cycle reaches %s", section)}
	}
	diagnostics := claimParent(parents, section, parent, source)
	if visited[section] {
		return diagnostics
	}
	visited[section] = true
	active[section] = true
	defer delete(active, section)
	diagnostics = append(diagnostics, requireTypedEntityAnySource(triples, section, s2sSection, "graph.document_section.invalid")...)
	children := objectValues(triples, section, s2sMember)
	diagnostics = append(diagnostics, validateSiblingPositions(triples, source, children)...)
	for _, child := range children {
		switch {
		case hasType(triples, child, s2sSection):
			diagnostics = append(diagnostics, validateSection(triples, child, section, source, shapeIDs, parents, visited, active)...)
		case hasType(triples, child, s2sPlacement):
			diagnostics = append(diagnostics, claimParent(parents, child, section, source)...)
			diagnostics = append(diagnostics, validatePlacement(triples, child, source, shapeIDs)...)
		default:
			diagnostics = append(diagnostics, diagnostic("graph.document_member.invalid", source, "member %s is neither a DocumentSection nor DocumentPlacement", child))
		}
	}
	return diagnostics
}

func claimParent(parents map[string]string, child, parent, source string) []Diagnostic {
	if previous, exists := parents[child]; exists && previous != parent {
		return []Diagnostic{diagnostic("graph.document_tree.invalid", source, "document node %s has multiple parents: %s and %s", child, previous, parent)}
	}
	parents[child] = parent
	return nil
}

func validateSiblingPositions(triples []sourcedTriple, source string, children []string) []Diagnostic {
	var diagnostics []Diagnostic
	positions := map[int]string{}
	seenChildren := map[string]bool{}
	for _, child := range children {
		if seenChildren[child] {
			diagnostics = append(diagnostics, diagnostic("graph.document_order.invalid", source, "member %s is linked more than once", child))
		}
		seenChildren[child] = true
		values := literalTerms(triples, child, s2sPosition)
		if len(values) != 1 {
			diagnostics = append(diagnostics, diagnostic("graph.document_order.invalid", source, "member %s must have exactly one position", child))
			continue
		}
		if values[0].DataType.String() != xsdPositiveInteger {
			diagnostics = append(diagnostics, diagnostic("graph.document_order.invalid", source, "member %s position must use xsd:positiveInteger", child))
			continue
		}
		position, err := strconv.Atoi(values[0].String())
		if err != nil || position < 1 {
			diagnostics = append(diagnostics, diagnostic("graph.document_order.invalid", source, "member %s has invalid positive position %q", child, values[0].String()))
			continue
		}
		if previous, exists := positions[position]; exists {
			diagnostics = append(diagnostics, diagnostic("graph.document_order.invalid", source, "members %s and %s both use position %d", previous, child, position))
		} else {
			positions[position] = child
		}
	}
	return diagnostics
}

func validatePlacement(triples []sourcedTriple, placement, fallbackSource string, shapeIDs map[string]bool) []Diagnostic {
	source := sourceForType(triples, placement, s2sPlacement)
	if source == "" {
		source = fallbackSource
	}
	diagnostics := requireTypedEntityAnySource(triples, placement, s2sPlacement, "graph.document_placement.invalid")
	shapes := objectValues(triples, placement, s2sShape)
	if len(shapes) != 1 || !shapeIDs[first(shapes)] {
		diagnostics = append(diagnostics, diagnostic("graph.placement.shape_invalid", source, "placement %s must reference exactly one declared canonical shape", placement))
	}
	return diagnostics
}

func validateShapes(manifest Manifest, triples []sourcedTriple) []Diagnostic {
	var diagnostics []Diagnostic
	declared := map[string]bool{}
	for _, shape := range manifest.CanonicalShapes {
		declared[shape.ID] = true
		diagnostics = append(diagnostics, requireTypedEntity(triples, shape.ID, shNodeShape, shape.Source, "graph.canonical_shape.invalid")...)
	}
	for _, subject := range subjectsWith(triples, rdfType, shNodeShape) {
		if !declared[subject] {
			diagnostics = append(diagnostics, diagnostic("graph.canonical_shape.undeclared", sourceForType(triples, subject, shNodeShape), "canonical node shape %s is not declared by the manifest", subject))
		}
	}
	return diagnostics
}

func requireTypedEntity(triples []sourcedTriple, subject, typeIRI, source, code string) []Diagnostic {
	matches := 0
	wrongSource := false
	for _, statement := range triples {
		if statement.Triple.Subj.String() == subject && statement.Triple.Pred.String() == rdfType && statement.Triple.Obj.String() == typeIRI {
			matches++
			wrongSource = wrongSource || statement.Source != source
		}
	}
	if matches != 1 || wrongSource {
		return []Diagnostic{diagnostic(code, source, "%s must have exactly one %s type statement owned by %s", subject, typeIRI, source)}
	}
	return nil
}

func requireTypedEntityAnySource(triples []sourcedTriple, subject, typeIRI, code string) []Diagnostic {
	matches := 0
	source := "graph"
	for _, statement := range triples {
		if statement.Triple.Subj.String() == subject && statement.Triple.Pred.String() == rdfType && statement.Triple.Obj.String() == typeIRI {
			matches++
			source = statement.Source
		}
	}
	if matches != 1 {
		return []Diagnostic{diagnostic(code, source, "%s must have exactly one %s type statement", subject, typeIRI)}
	}
	return nil
}

func hasType(triples []sourcedTriple, subject, typeIRI string) bool {
	return contains(objectValues(triples, subject, rdfType), typeIRI)
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

func literalValues(triples []sourcedTriple, subject, predicate string) []string {
	terms := literalTerms(triples, subject, predicate)
	values := make([]string, 0, len(terms))
	for _, literal := range terms {
		values = append(values, literal.String())
	}
	return values
}

func literalTerms(triples []sourcedTriple, subject, predicate string) []rdf.Literal {
	var values []rdf.Literal
	for _, statement := range triples {
		if statement.Triple.Subj.String() != subject || statement.Triple.Pred.String() != predicate {
			continue
		}
		if literal, ok := statement.Triple.Obj.(rdf.Literal); ok {
			values = append(values, literal)
		}
	}
	return values
}

func singleLiteral(triples []sourcedTriple, subject, predicate string) string {
	values := literalValues(triples, subject, predicate)
	if len(values) != 1 {
		return ""
	}
	return values[0]
}

func sourceForType(triples []sourcedTriple, subject, typeIRI string) string {
	for _, statement := range triples {
		if statement.Triple.Subj.String() == subject && statement.Triple.Pred.String() == rdfType && statement.Triple.Obj.String() == typeIRI {
			return statement.Source
		}
	}
	return ""
}

func subjectsWith(triples []sourcedTriple, predicate, object string) []string {
	var subjects []string
	seen := map[string]bool{}
	for _, statement := range triples {
		if statement.Triple.Pred.String() == predicate && statement.Triple.Obj.String() == object && !seen[statement.Triple.Subj.String()] {
			subjects = append(subjects, statement.Triple.Subj.String())
			seen[statement.Triple.Subj.String()] = true
		}
	}
	return subjects
}

func referenceIDs(references []ArtifactReference) (indicators, methodologies []string) {
	for _, reference := range references {
		if reference.Kind == "indicator" {
			indicators = append(indicators, reference.ID)
		} else if reference.Kind == "methodology" {
			methodologies = append(methodologies, reference.ID)
		}
	}
	return indicators, methodologies
}

func sameValues(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := map[string]int{}
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
