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

## Confirmed standard, indicator, and methodology boundary

The model keeps three normative concepts separate:

- an **indicator** is the domain unit that is measured, quantified, or reported, including its canonical meaning and applicable measurement unit or dimension;
- a **methodology** targets an indicator and defines how it is determined, including the methodology's semantic and quantitative applicability conditions, required inputs, calculations, and evidence; and
- a **standard** authorizes indicators and methodologies and defines the normative document structure, requirements, and guidance governing their use.

A standard authorization references an indicator or methodology; it does not absorb or redefine it. Likewise, applicability conditions belong to the methodology, not to its target indicator or to a generic document-validation layer. Executable packages and assessments must preserve these authority and provenance boundaries.

The required granularity of authorization—including whether a standard authorizes exact indicator–methodology version pairings or independently authorizes compatible versions—remains an open governance decision.

## Confirmed formula boundary

An **indicator formula** and a methodology **equation step** are both mathematical expressions, but they have different identity and ownership:

- the indicator formula is the indicator's identity-defining mathematical expression and remains stable across methodologies that determine the same indicator; and
- an equation step belongs to one methodology and is one ordered transformation in that methodology's operational procedure.

They are sibling specializations of a neutral mathematical-expression concept, not a subtype relationship in either direction. A methodology may use multiple equation steps to operationalize one indicator formula, and two methodologies may use different equation steps while targeting the same indicator. The model may link a methodology explicitly to the indicator formula it implements, but must not infer that each equation step is itself an indicator formula.

## Confirmed unit of authorship

The author works on an **ordered shape bundle** representing one complete complex document or form, for example a PDD or monitoring report. The bundle may span multiple source files, but it is understood and validated as one coherent document definition.

Source-file ownership rules, import boundaries, and save behavior remain open requirements.

## Confirmed ordering boundary

The bundle's order is **document order**: the normative sequence of chapters, sections, requirement groups, and requirements in the standard's document definition. It belongs to `standard2shape` because changing it changes the canonical structure of the document.

**Presentation order** belongs to `shape2form`: it controls how a particular generated interface arranges fields for its users. Canonical document order must therefore not be encoded as `ui:order` or another presentation annotation.

## Confirmed structure and reuse model

The canonical document structure is an ordered **document tree** containing two kinds of node:

- a **document section**, which is a structural container and may contain child sections or placements without capturing or constraining data; and
- a **document placement**, which references a reusable canonical shape and supplies its parent and sequence within this document.

Canonical shapes remain graph entities rather than being copied into the tree. The same shape may therefore be referenced by multiple document bundles or by multiple placements in one bundle without duplicating its SHACL definition. Placement owns document context and order; the referenced shape owns semantic constraints.

## Confirmed placement boundary

A document placement may own canonical context specific to that use of a reusable shape, including applicability guidance. It may not override the referenced shape's datatype, cardinality, value, or other semantic constraints.

If two uses require different constraints, they must reference distinct canonical shapes or an explicit specialization relationship. Contextual constraint overrides are not permitted because they would make the same shape mean different things depending on where it appears.

## Confirmed applicability model

Each methodology's applicability conditions are represented by first-class **applicability requirements**, each carrying canonical guidance and one of two evaluation models:

- a **semantic requirement** is defined by SHACL and an explicit reasoning profile, including required preprocessing such as SKOS hierarchy materialisation; and
- a **quantitative requirement** is an authoritative structured definition over typed inputs, units, thresholds, and formulas.

A quantitative requirement's software implementation—for example an R applicability function—is a versioned **evaluation binding**, not the source of truth. Simple numeric bounds may compile to SHACL constraints; multi-variable calculations may require an external evaluator. The canonical requirement remains independent of implementation language.

Every executable requirement has mandatory valid, invalid, and boundary test vectors. Semantic and quantitative evaluators return one **applicability assessment** contract containing conformance, violations, and an attestation identifying the requirement, version, evaluator, and evidence checked.

## Confirmed execution boundary

`standard2shape` authors and locally validates canonical standard packages; it does not operate a separate hosted validation service. One reusable validation interface applies immutable packages and returns the common assessment contract, so local tooling and hosted execution cannot silently implement different semantics.

`ii-backend` owns the II Platform's hosted API, authentication, authorization, artifact registry and storage, package selection, execution orchestration, and attestations. Approved methodology calculations may be delegated to trusted-compute implementations through versioned evaluation bindings. Neither deployment mode becomes the source of truth: the normative packages and their conformance suites remain authoritative.

## Confirmed guidance boundary

**Canonical guidance** is part of the standard. Document sections and canonical node or property shapes may carry authoritative instructions, explanations, applicability guidance, and definitions. Downstream UIs use that content as the default source for help text and extended information, reducing the amount of presentation copy each UI developer must create.

How guidance is revealed—always visible, hover, click, disclosure, or another interaction—is a presentation concern owned by `shape2form`. A UI developer may also add **supplemental guidance**, such as examples or task-specific clarification, for a particular interface.

Canonical guidance and supplemental guidance are separate content channels. A conforming UI must keep canonical guidance available and traceable to the standard; supplemental guidance may accompany it but may not replace, clear, or silently hide it. This requires downstream representations to preserve the provenance of both channels rather than collapsing them into one description field.

## Product boundary

`standard2shape` owns canonical data requirements:

- node and property shapes;
- paths and target classes;
- datatypes and nested-node relationships;
- cardinality and value constraints;
- canonical document structure and document order;
- validation and review of proposed semantic changes.

It must preserve the separate authority of standards, indicators, and methodologies. Authoring or executing a standard package must not silently turn referenced indicator definitions or methodology applicability conditions into standard-owned rules.

`shape2form` remains downstream and owns form compilation, rendering, previews, and presentation-authoring workflows. UI presentation annotations are not canonical-shape edits and remain separate from this product.

`standard2shape` does not own hosted user accounts, operational artifact storage, a public validation API, standard authorization workflows, or production execution orchestration. Those responsibilities belong to `ii-backend`; duplicating them here is a non-goal.

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

- How should each renderer visually compose canonical and supplemental guidance while preserving their distinct provenance?
- What owns a shape when the bundle spans multiple files?
- Which SHACL Core features belong in the first usable release?
- How are ontology terms discovered, selected, created, and reviewed?
- How should nested structures, reusable sections, and cross-file references appear in the UI?
- What RDF vocabulary and serialization represent quantitative definitions, evaluation bindings, reasoning profiles, test vectors, and applicability assessments?
- What validation levels are required before save, review, and publication?
- Does a standard authorize exact compatible indicator–methodology version pairings, or authorize indicator and methodology versions independently?
- Which repository or release process publishes each normative package, and which component assembles them for execution without changing their ownership?
- What technology-neutral validation interface and assessment serialization allow local tooling, `ii-backend`, and trusted-compute bindings to run the same conformance suites?
- Is the output an in-place source edit, a generated patch, a branch, or a pull request?
- How should unsupported advanced SHACL and SHACL-SPARQL constructs be preserved?
- What relationship should the editor have to a live `shape2form` preview?

## Acceptance criteria

Acceptance criteria will be added as each open requirement is resolved. The specification is not implementation-ready while the status remains `requirements discovery`.
