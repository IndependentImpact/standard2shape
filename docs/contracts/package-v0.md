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

`manifestVersion` selects this contract and is currently exactly `0.1`. `id` and `version` identify the package; `standardRelease` identifies the standard-owned release represented by it. Document roots and canonical shape roots are named RDF graph entities and declare the source artifact that owns their defining type statement.

Every local canonical artifact records a package-relative path, one role, the `text/turtle` media type, and a SHA-256 digest of its exact bytes. Conformance vectors are local members with the same path and digest guarantees but are separated from canonical artifacts because they are evidence, not normative source graphs.

All paths are lexical package-relative POSIX paths. On an indicator, methodology, or import entry, `source` identifies the standard-owned file containing the reference declaration—not the source or ownership of the external artifact. A pinned reference does not itself constitute a standard authorization; every authorization must target a declared reference, but references may also exist for context or dependency resolution. Remote import resolution is disabled: each `owl:imports` statement must have a matching `reference-only` manifest declaration. Consumers verify pinned external references through an approved artifact source; they never infer ownership from inclusion in a standard manifest.

## Document vocabulary

An `s2s:OrderedShapeBundle` has one or more `s2s:hasRootSection` links to `s2s:DocumentSection` nodes. A section contains ordered child sections or `s2s:DocumentPlacement` nodes through `s2s:member`. Every child has one positive `s2s:position`, unique among its siblings. A placement has one `s2s:shape` link to a reusable `sh:NodeShape`; the shape is not copied into the tree.

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

- `package.member.missing` — a declared local member is absent;
- `package.member.duplicate` — the same member descriptor is declared more than once;
- `package.member.conflict` — one path is assigned incompatible descriptors or roles;
- `package.member.digest_mismatch` — local bytes do not match the manifest;
- `graph.document_root.invalid` — a declared root is absent, duplicated, mistyped, or defined in another source;
- `graph.placement.shape_invalid` — a placement does not reference exactly one declared reusable shape;
- `graph.document_order.invalid` — sibling positions are absent, invalid, or duplicated;
- `graph.presentation_order.forbidden` — canonical RDF contains shape2form presentation order;
- `graph.import.mismatch` — graph and manifest import declarations differ;
- `graph.reference.invalid` — a pinned external reference does not match its RDF reference record.

Messages may become clearer without breaking compatibility; consumers must key automation on codes, not English text.

## Normalization and versioning

Normalization sorts all set-like manifest arrays by stable identity and serializes the typed manifest with two-space JSON indentation and one trailing newline. It never rewrites RDF. Equivalent manifests therefore normalize to identical bytes regardless of input array order.

Manifest `0.x` versions are experimental and require exact consumer support; unknown fields and unsupported manifest versions are rejected. After `1.0`, a major manifest-version change will indicate an incompatible contract, while additive optional fields may use a minor contract version. Package `version` follows semantic versioning independently: a breaking normative model change increments major, an additive normative change increments minor, and guidance, metadata, or non-breaking corrections increment patch. Any byte change updates the affected member digest and any semantic change requires a new immutable standard release.

External artifact compatibility is exact in v0: identity, version, and digest must all match. A later compatibility-range policy must be introduced as an explicit contract revision; consumers must not infer compatibility from semantic-version ranges.
