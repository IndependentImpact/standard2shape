# standard2shape implementation plan

Status: planned

## Outcome

Deliver a local-first authoring environment in which a standards author can open or create an ordered canonical SHACL bundle, edit supported requirements without writing Turtle, inspect every semantic change, run the bundle's conformance suite, and produce a deterministic release for RDF review and downstream use by `shape2form` and `ii-backend`.

## Delivery principles

- Canonical RDF, SHACL, guidance, document order, and standard authorizations remain the source of truth.
- Indicator and methodology packages are referenced and inspected without transferring their ownership to the standard.
- Existing multi-file bundles retain source provenance; unsupported graph content is visible and never silently discarded.
- Every phase delivers a thin, demonstrable path across graph handling, validation, user interaction, persistence, and tests.
- Local and hosted execution use the same technology-neutral validation interface, immutable packages, conformance vectors, and assessment contract.
- `standard2shape` remains local-first. `ii-backend` owns hosted access, governance, storage, and production orchestration.
- SHACL, ontology, and RDF fixtures remain local to the development and test environment unless deliberately published as repository fixtures.

## Target component map

The implementation should preserve these seams even if their technology changes during the first tracer:

1. **Authoring interface** — the standards-author and RDF-reviewer workspace.
2. **Bundle workspace** — loads multi-file RDF, tracks statement and entity provenance, and applies controlled semantic changes.
3. **Canonical model** — document tree, placements, shapes, guidance, authorizations, and referenced-artifact identities.
4. **Validation interface** — syntax, structural, SHACL, reasoning-profile, conformance-suite, and assessment behavior behind one interface.
5. **Release builder** — deterministic manifests, digests, version metadata, source patches, and review artifacts.
6. **Adapters** — local files and CLI first; `shape2form`, `ii-backend`, and trusted-compute contracts downstream.

The first tracer will evaluate a Go core/local process with a React and TypeScript authoring interface because that aligns with `ii-backend` and the reusable `shape2form` packages. The kept architecture must be selected from evidence produced by the tracer and recorded before broad implementation begins.

## GitHub delivery map

| Milestone | Epic | Sub-issues |
| --- | --- | --- |
| M0 | [#1 — Runnable foundation and package contracts](https://github.com/IndependentImpact/standard2shape/issues/1) | [#7](https://github.com/IndependentImpact/standard2shape/issues/7), [#8](https://github.com/IndependentImpact/standard2shape/issues/8), [#9](https://github.com/IndependentImpact/standard2shape/issues/9), [#10](https://github.com/IndependentImpact/standard2shape/issues/10) |
| M1 | [#2 — Lossless multi-file bundle workspace](https://github.com/IndependentImpact/standard2shape/issues/2) | [#11](https://github.com/IndependentImpact/standard2shape/issues/11), [#12](https://github.com/IndependentImpact/standard2shape/issues/12), [#13](https://github.com/IndependentImpact/standard2shape/issues/13), [#14](https://github.com/IndependentImpact/standard2shape/issues/14) |
| M2 | [#3 — Canonical document and SHACL authoring](https://github.com/IndependentImpact/standard2shape/issues/3) | [#15](https://github.com/IndependentImpact/standard2shape/issues/15), [#16](https://github.com/IndependentImpact/standard2shape/issues/16), [#17](https://github.com/IndependentImpact/standard2shape/issues/17), [#18](https://github.com/IndependentImpact/standard2shape/issues/18) |
| M3 | [#4 — Authorized indicators and methodologies](https://github.com/IndependentImpact/standard2shape/issues/4) | [#19](https://github.com/IndependentImpact/standard2shape/issues/19), [#20](https://github.com/IndependentImpact/standard2shape/issues/20), [#21](https://github.com/IndependentImpact/standard2shape/issues/21), [#22](https://github.com/IndependentImpact/standard2shape/issues/22) |
| M4 | [#5 — Conformance, review, and deterministic releases](https://github.com/IndependentImpact/standard2shape/issues/5) | [#23](https://github.com/IndependentImpact/standard2shape/issues/23), [#24](https://github.com/IndependentImpact/standard2shape/issues/24), [#25](https://github.com/IndependentImpact/standard2shape/issues/25), [#26](https://github.com/IndependentImpact/standard2shape/issues/26) |
| M5 | [#6 — Downstream integration and release hardening](https://github.com/IndependentImpact/standard2shape/issues/6) | [#27](https://github.com/IndependentImpact/standard2shape/issues/27), [#28](https://github.com/IndependentImpact/standard2shape/issues/28), [#29](https://github.com/IndependentImpact/standard2shape/issues/29), [#30](https://github.com/IndependentImpact/standard2shape/issues/30) |

## Delivery sequence

### Epic 1 — Runnable foundation and package contracts

Prove one complete fixture-driven loop before committing to the implementation architecture. Define the minimum package, validation, and assessment contracts that every later slice uses.

Demonstrable outcome: a local application opens a representative bundle, displays its document and validation summary, invokes a `shape2form` preview adapter, makes one controlled change, and emits a reviewable result under CI.

Sub-issues:

1. Build an executable open–validate–preview–change tracer.
2. Define the versioned canonical package manifest and document vocabulary.
3. Define the validation assessment and conformance-suite contracts.
4. Establish the kept application skeleton and quality gates.

### Epic 2 — Lossless multi-file bundle workspace

Make existing standards safe to load and change. Preserve file ownership, graph identity, unsupported constructs, and untouched source content.

Demonstrable outcome: an RDF reviewer can import a multi-file fixture, trace every managed entity to its source, apply a supported edit, reload it, and prove that unrelated or unsupported graph content survived unchanged.

Sub-issues:

1. Import a multi-file bundle with statement and entity provenance.
2. Explore the document tree and reusable shape graph.
3. Save one controlled edit as a reviewable source patch and reload it.
4. Preserve and expose unsupported advanced SHACL without data loss.

### Epic 3 — Canonical document and SHACL authoring

Give standards authors guided controls for the first useful SHACL Core subset and the canonical document tree.

Demonstrable outcome: a standards author creates a nested document structure, authors scalar and constrained requirements, reuses a shape in multiple placements, adds canonical guidance, validates the result, and saves a semantic patch.

Sub-issues:

1. Create, nest, and reorder document sections and placements.
2. Author SHACL property constraints through guided controls.
3. Author nested and reusable shapes without placement overrides.
4. Author canonical guidance with preserved provenance.

### Epic 4 — Authorized indicators and methodologies

Represent the standard's governance role without absorbing indicator or methodology definitions. External packages remain read-only references in this product.

Demonstrable outcome: a standards author loads pinned indicator and methodology packages, understands their relationship and applicability evidence, and records traceable standard authorizations without changing the referenced artifacts.

Sub-issues:

1. Load and inspect pinned indicator and methodology packages read-only.
2. Preserve the IndicatorFormula and EquationStep identity boundary.
3. Author versioned standard authorizations and compatibility declarations.
4. Inspect methodology applicability and add non-overriding placement context.

### Epic 5 — Conformance, review, and deterministic releases

Turn authored changes into trustworthy release candidates. Make validation levels explicit and require semantic review before release production.

Demonstrable outcome: a candidate bundle passes syntax, structural, SHACL, reasoning, and test-vector checks; an RDF reviewer approves its semantic diff; and both UI and CLI produce the same deterministic release and assessment.

Sub-issues:

1. Run staged local validation and package conformance suites.
2. Review and approve semantic diffs before applying or exporting changes.
3. Build deterministic, versioned release packages with digests.
4. Validate the same package through a local CLI and shared validation interface.

### Epic 6 — Downstream integration and release hardening

Prove that the package is useful outside the editor and prepare the first supported release without creating a second hosted platform.

Demonstrable outcome: one released fixture previews correctly in `shape2form`, passes the `ii-backend` ingestion contract, exercises a versioned evaluator binding, and survives browser, accessibility, offline, security, and performance checks.

Sub-issues:

1. Integrate a live `shape2form` preview with canonical guidance provenance.
2. Publish and verify the `ii-backend` ingestion contract fixture.
3. Verify evaluator-binding and trusted-compute conformance contracts.
4. Complete accessibility, resilience, performance, security, and release documentation.

## Detailed first-epic specification

The first tracer uses a deliberately small but representative fixture:

- one standard release;
- one ordered document bundle with a section and placement;
- one reusable node shape with one required property shape;
- canonical guidance;
- one authorized indicator and one methodology reference;
- one valid and one invalid data graph;
- one unsupported SHACL construct that must survive unchanged.

The tracer is complete when it can:

1. open the fixture without network ontology resolution;
2. show the document tree, shape summary, source files, and validation result;
3. invoke an adapter that demonstrates the downstream `shape2form` representation;
4. change one supported cardinality or guidance value;
5. present the semantic difference;
6. emit and reapply a source patch without changing the unsupported construct;
7. return a structured assessment identifying package, test vector, validator, and result;
8. run through one command in CI and one browser-level test.

The tracer must record decisions for runtime composition, RDF parsing and patching, blank-node identity, validation-engine isolation, front-end packaging, and the local-file security model. Throwaway code may be discarded, but the fixture, contracts, tests, and resulting decisions are kept.

## Verification matrix

| Area | Required verification |
| --- | --- |
| RDF integrity | Local syntax parsing, namespace checks, deterministic graph comparison, no remote import fetching by default |
| SHACL | Valid, invalid, missing, duplicate, boundary, nested, enum, and unsupported-construct fixtures |
| Round trip | Reload every saved patch; assert managed semantic change and preservation of unrelated source content |
| Domain contracts | Standard, indicator, methodology, guidance, formula, authorization, and applicability ownership tests |
| Validation | Contract tests for assessment serialization, reasoning profiles, evaluator identity, and conformance vectors |
| UI | Component tests plus Playwright coverage of every changed control and complete authoring paths |
| Accessibility | Keyboard operation, focus management, labels, errors, guidance disclosure, and automated accessibility checks |
| Integration | `shape2form` preview fixture, `ii-backend` ingestion fixture, and trusted-compute evaluator-binding fixture |
| Release | Formatting, lint, type checks, unit/integration tests, browser tests, deterministic build, checksums, and clean-install smoke test |

## Milestones

- **M0 — Architecture proven:** Epic 1 complete.
- **M1 — Existing bundles safe:** Epic 2 complete.
- **M2 — Useful authoring:** Epic 3 complete.
- **M3 — Governance complete:** Epic 4 complete.
- **M4 — Release candidates trustworthy:** Epic 5 complete.
- **M5 — First supported release:** Epic 6 complete.

Epics are sequenced, but independent sub-issues within an epic may proceed in parallel where their recorded blockers allow it. No later milestone should bypass an unresolved graph-contract or provenance failure from an earlier milestone.
