# Canonical standard package contract v0.1

Status: experimental contract for issue #8

## Boundary

A standard package is a local, versioned inventory around one immutable standard release. It contains only standard-owned canonical source artifacts and conformance vectors. Indicator, methodology, and imported ontology definitions remain separately owned: the package records their identity, exact version, and SHA-256 digest but does not vendor or redefine them.

The JSON manifest inventories package members and roots. RDF remains authoritative for the document tree, SHACL constraints, canonical guidance, standard authorizations, and reasoning-profile declaration.

## Contract files

- [`contracts/v0/package-manifest.schema.json`](../../contracts/v0/package-manifest.schema.json) defines the closed JSON shape of `manifest.json`.
- [`contracts/v0/standard2shape.ttl`](../../contracts/v0/standard2shape.ttl) defines the v0 RDF vocabulary.
- [`fixtures/tracer`](../../fixtures/tracer) is the executable valid example.
- [`fixtures/package-v0`](../../fixtures/package-v0) contains rejected package examples with stable diagnostic codes.

## Identity and membership

`manifestVersion` selects this contract and is currently exactly `0.1`. `id` and `version` identify the package; `standardRelease` identifies the standard-owned release represented by it. Document roots and canonical shape roots are named RDF graph entities and declare the source artifact that owns their defining type statement. `canonicalShapes` inventories every canonical shape, `sh:NodeShape` and `sh:PropertyShape` alike; a shape typed in a canonical source but absent from the inventory is rejected. Document placements may reference only inventoried node shapes.

Manifest field names are matched exactly: differently cased or duplicate fields are rejected, matching the closed JSON Schema. Members are read from within the package root; a member that resolves outside it through a symlink is rejected.

Every local canonical artifact records a package-relative path, one role, the `text/turtle` media type, and a SHA-256 digest of its exact bytes. Conformance vectors are local members with the same path and digest guarantees but are separated from canonical artifacts because they are evidence, not normative source graphs. Each vector names the target it exercises — a declared executable requirement or a declared canonical shape — and its category (`valid`, `invalid`, or `boundary`), so both the binding and the categorization are covered by the normalized-manifest digest and immutable with the package. Valid vectors expect `conforms`, invalid vectors expect `non-conforms`, and every tested target must have all three categories.

`requirements` inventories every executable applicability requirement of the package: its IRI, version, kind (`semantic` or `quantitative`), the SHA-256 digest pinning its externally owned normative definition, and the source artifact declaring it. The inventory may be empty — a standard package without authorized methodology requirements ships shape-targeted conformance vectors only. Requirement records follow the same authority boundary and resolution model as every other reference: applicability requirements are methodology-owned, so the package records a pinned reference — identity, version, kind, and definition digest — and never recreates the rule itself. As with methodology and indicator digests, the referenced definition is not a package member and carries no locator: consumers verify pinned external definitions through an approved artifact source, and an evaluator that cannot resolve one must answer `evaluator-failure` or `unsupported`, never substitute its own rule. Locally, the requirement's meaning is behaviorally pinned by its mandatory conformance vectors: the evidence bytes and expected outcomes are digest-pinned package members, so local validation proves evaluator behavior on the requirement's full vector set even though the definition itself is external. A requirement identity may not collide with a canonical shape or reference identity. The inventory is derived from, not asserted over, the canonical graph: each entry must resolve to an RDF requirement record typed `s2s:SemanticRequirement` or `s2s:QuantitativeRequirement` (the classes are disjoint; a record typed as both is rejected) matching the declared kind, carrying the declared `s2s:requirementVersion` and `s2s:requirementDigest`, owned by its declared source, and attached to exactly one declared methodology reference through `s2s:hasRequirement`. A requirement record present in the graph but absent from the inventory is rejected — including records typed only as the parent `s2s:ApplicabilityRequirement` class and untyped objects of `s2s:hasRequirement`, whose range already makes them requirements — so omitting a requirement and its vectors cannot pass. Every declared requirement must be exercised by valid, invalid, and boundary vectors — the rule that every executable requirement has mandatory test vectors of all three categories.

All paths are normalized package-relative POSIX paths: no backslashes and no leading, trailing, empty, `.`, or `..` segments. The JSON Schema path pattern and the verifier accept exactly the same paths. On an indicator, methodology, or import entry, `source` identifies the standard-owned file containing the reference declaration—not the source or ownership of the external artifact. A pinned reference does not itself constitute a standard authorization; every authorization must target a declared reference, but references may also exist for context or dependency resolution. Remote import resolution is disabled: each `owl:imports` statement must have a matching `reference-only` manifest declaration. Consumers verify pinned external references through an approved artifact source; they never infer ownership from inclusion in a standard manifest.

## Document vocabulary

An `s2s:OrderedShapeBundle` has one or more `s2s:hasRootSection` links to `s2s:DocumentSection` nodes. A section contains ordered child sections or `s2s:DocumentPlacement` nodes through `s2s:member`. Every child has one positive `s2s:position`, unique among its siblings. A placement has one `s2s:shape` link to a reusable `sh:NodeShape`; the shape is not copied into the tree. Every typed `s2s:DocumentSection` and `s2s:DocumentPlacement` must be reachable from a declared document root; orphan document nodes are rejected because they would escape tree validation. The two classes are disjoint: a node typed as both is rejected.

Document-node guidance uses `s2s:canonicalGuidance`. Canonical guidance on SHACL node and property shapes uses `sh:description`, matching the downstream shape2form canonical channel. Canonical document order is expressed only by `s2s:position`; `https://shape2form.dev/vocab/ui#order` is a presentation annotation and is rejected from canonical source artifacts.

For example, the manifest names the graph roots without embedding their definitions:

```json
{
  "documentRoots": [{ "id": "https://example.org/standard/ProjectDocument", "source": "document.ttl" }],
  "canonicalShapes": [{ "id": "https://example.org/standard/ProjectShape", "source": "shapes.ttl" }]
}
```

The canonical source then owns the ordered relationship and guidance:

```turtle
ex:ProjectOverview a s2s:DocumentSection ;
  s2s:canonicalGuidance "Explain the project to an independent reviewer."@en ;
  s2s:member ex:ProjectDetailsPlacement .

ex:ProjectDetailsPlacement a s2s:DocumentPlacement ;
  s2s:position "1"^^xsd:positiveInteger ;
  s2s:shape ex:ProjectShape .
```

## Stable diagnostics

Contract errors expose a stable code, package-relative location, and explanatory message. Initial codes include:

- `manifest.field.unknown` — a manifest field is unknown or differently cased;
- `manifest.field.duplicate` — a manifest field is declared more than once;
- `manifest.field.null` — a manifest field is explicitly null;
- `package.member.missing` — a declared local member is absent;
- `package.member.duplicate` — the same member descriptor is declared more than once;
- `package.member.conflict` — one path is assigned incompatible descriptors or roles;
- `package.member.digest_mismatch` — local bytes do not match the manifest;
- `package.member.escape` — a member resolves outside the package root;
- `manifest.vector.requirement_unknown` — a vector's target is not a declared executable requirement or canonical shape;
- `manifest.vector.category_invalid` / `.vector.category_mismatch` — a vector category is unknown or inconsistent with its expected outcome;
- `manifest.requirement.kind_invalid` / `.requirement.duplicate` / `.requirement.conflict` — a requirement entry does not use a known kind, a unique identity, or an identity distinct from every canonical shape and reference;
- `manifest.requirement.vectors_missing` — a declared requirement or tested target lacks a valid, invalid, or boundary vector;
- `graph.requirement.invalid` — an inventoried requirement is missing from the graph or its type, version, source, or methodology ownership differs;
- `graph.requirement.undeclared` — a requirement definition in the graph is not inventoried by the manifest;
- `graph.document_root.invalid` — a declared root is absent, duplicated, mistyped, or defined in another source;
- `graph.placement.shape_invalid` — a placement does not reference exactly one declared reusable shape;
- `graph.document_order.invalid` — sibling positions are absent, invalid, or duplicated;
- `graph.document_node.orphan` — a typed document node is unreachable from every declared root;
- `graph.document_member.invalid` — a member is neither, or both, a DocumentSection and a DocumentPlacement;
- `graph.presentation_order.forbidden` — canonical RDF contains shape2form presentation order;
- `graph.import.mismatch` — graph and manifest import declarations differ;
- `graph.reference.invalid` — a pinned external reference does not match its RDF reference record.

Messages may become clearer without breaking compatibility; consumers must key automation on codes, not English text.

## Normalization and versioning

Normalization sorts all set-like manifest arrays by stable identity and serializes the typed manifest with two-space JSON indentation and one trailing newline. Every collection is always present: `imports` and `references` may be empty but never omitted or `null`, and normalization serializes them as `[]`, so a normalized manifest always satisfies the published JSON Schema. It never rewrites RDF. Equivalent manifests therefore normalize to identical bytes regardless of input array order.

Manifest `0.x` versions are experimental and require exact consumer support; unknown fields and unsupported manifest versions are rejected. After `1.0`, a major manifest-version change will indicate an incompatible contract, while additive optional fields may use a minor contract version. Package `version` follows semantic versioning independently: a breaking normative model change increments major, an additive normative change increments minor, and guidance, metadata, or non-breaking corrections increment patch. Any byte change updates the affected member digest and any semantic change requires a new immutable standard release.

External artifact compatibility is exact in v0: identity, version, and digest must all match. A later compatibility-range policy must be introduced as an explicit contract revision; consumers must not infer compatibility from semantic-version ranges.
