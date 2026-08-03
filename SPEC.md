# standard2shape product specification

Status: requirements discovery

## Summary

`standard2shape` is a guided authoring and validation environment for producing canonical SHACL shape bundles from the data requirements of complex standards-based documents. It sits upstream of `shape2form`: `standard2shape` defines what data the standard requires, while `shape2form` turns those canonical requirements and separate presentation annotations into usable form interfaces.

## Problem

Complex standard documents such as project design documents and monitoring reports contain ordered, nested data requirements. Today, expressing those requirements as canonical SHACL generally requires direct RDF/Turtle expertise. Standards authors need a safer way to create and evolve those requirements without losing semantic rigor, source provenance, or expert review.

## Confirmed users and roles

### Standards author

The primary user. A domain expert responsible for defining the standard's canonical data requirements; they may not know RDF or Turtle deeply.

### RDF reviewer

A specialist who reviews proposed canonical-shape changes for semantic and graph-model correctness.

## Confirmed unit of authorship

The author works on an **ordered shape bundle** representing one complete complex document or form, for example a PDD or monitoring report. The bundle may span multiple source files, but it is understood and validated as one coherent document definition.

Source-file ownership rules, import boundaries, and save behavior remain open requirements.

## Confirmed ordering boundary

The bundle's order is **document order**: the normative sequence of chapters, sections, requirement groups, and requirements in the standard's document definition. It belongs to `standard2shape` because changing it changes the canonical structure of the document.

**Presentation order** belongs to `shape2form`: it controls how a particular generated interface arranges fields for its users. Canonical document order must therefore not be encoded as `ui:order` or another presentation annotation.

## Confirmed structure and reuse model

The canonical document structure is an ordered **document tree**. Each node in that tree is a **document placement** that references a reusable canonical shape and supplies its parent and sequence within this document.

Canonical shapes remain graph entities rather than being copied into the tree. The same shape may therefore be referenced by multiple document bundles or by multiple placements in one bundle without duplicating its SHACL definition. Placement owns document context and order; the referenced shape owns semantic constraints.

## Product boundary

`standard2shape` owns canonical data requirements:

- node and property shapes;
- paths and target classes;
- datatypes and nested-node relationships;
- cardinality and value constraints;
- canonical document structure and document order;
- validation and review of proposed semantic changes.

`shape2form` remains downstream and owns form compilation, rendering, previews, and presentation-authoring workflows. UI presentation annotations are not canonical-shape edits and remain separate from this product.

## Required qualities

- Domain experts can author common SHACL structures without editing Turtle directly.
- The system preserves explicit provenance from graph entities back to owning source files.
- Invalid or contradictory semantic combinations are prevented or diagnosed before save.
- Every save produces a reviewable semantic change.
- Existing valid bundles can be loaded without flattening or silently rewriting their source layout.
- Advanced SHACL that the guided editor cannot safely represent remains visible and is never silently discarded.

## Candidate workflow

This workflow is provisional and must be tested during requirements grilling:

1. Open an existing bundle or start a bundle for a document type.
2. Explore the document hierarchy and its node/property shapes.
3. Add or edit requirements through structured controls.
4. See immediate structural, semantic, and downstream form-preview feedback.
5. Review a semantic diff and validation report.
6. Save a proposed change for RDF review.

## Non-goals under consideration

These have not yet been confirmed:

- general-purpose RDF or Turtle editing;
- presentation design or editing `ui:` overlays;
- silently normalizing arbitrary source files;
- replacing RDF/ontology review for advanced constructs;
- publishing standards directly without a review workflow.

## Open requirements

- Which document-tree nodes are structural containers, and which reference data-bearing canonical shapes?
- What owns a shape when the bundle spans multiple files?
- Which SHACL Core features belong in the first usable release?
- How are ontology terms discovered, selected, created, and reviewed?
- How should nested structures, reusable sections, and cross-file references appear in the UI?
- What validation levels are required before save, review, and publication?
- Is the output an in-place source edit, a generated patch, a branch, or a pull request?
- How should unsupported advanced SHACL and SHACL-SPARQL constructs be preserved?
- What relationship should the editor have to a live `shape2form` preview?

## Acceptance criteria

Acceptance criteria will be added as each open requirement is resolved. The specification is not implementation-ready while the status remains `requirements discovery`.
