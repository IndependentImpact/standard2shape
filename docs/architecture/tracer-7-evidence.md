# Issue 7 tracer evidence and architecture recommendation

Status: recommendation from the executable tracer

## Recommendation

Keep a local Go process as the composition root and graph/validation host, with a React and TypeScript authoring interface served over a loopback-only HTTP connection. Preserve narrow Go interfaces around bundle loading, controlled source changes, validation, release building, and downstream adapters so the same graph and validation behavior can later be composed into `ii-backend` without moving browser or HTTP concerns into the canonical model.

This is evidence for the architecture decision required by issue #10; it does not prematurely freeze the package manifest, complete SHACL feature set, or production patch format that later issues own.

## What the tracer proved

- A Go process can load a synthetic multi-file Turtle bundle, retain source-file provenance, derive the ordered document tree and reusable shape summary, and avoid resolving a deliberately remote `owl:imports` reference.
- A small validation interface can evaluate valid and invalid local data vectors and return one structured assessment to both HTTP and browser callers.
- A managed canonical-guidance edit can be assigned to its owning file, represented as a semantic diff and minimal source patch, reloaded into an equivalent graph, and guarded by a digest over an adjacent unsupported SHACL-SPARQL subgraph.
- A framework-neutral downstream adapter can project the canonical property shape into the minimum information a `shape2form` preview needs without writing presentation annotations into canonical RDF.
- The React interface can expose authoring, provenance, validation, unsupported-content warnings, semantic review, and source review in one browser workflow at desktop and mobile widths.
- The repository can verify Go, TypeScript, production web assets, local RDF fixtures, HTTP behavior, and real browser interaction in one CI job.

## Recommended module seams

1. **Bundle workspace** — local package access, parsing, source ownership, graph identity, and controlled changes.
2. **Canonical model** — document tree, placements, shapes, guidance, references, and authorizations without HTTP or browser types.
3. **Validation interface** — immutable package and evidence in; structured assessment out. The tracer's cardinality check is only a test adapter, not the production SHACL engine.
4. **Source-change adapter** — converts reviewed semantic changes into recoverable source operations while preserving unsupported content.
5. **Preview adapter** — maps supported canonical content to a downstream `shape2form` contract; it does not own UI annotations.
6. **Local delivery adapter** — loopback HTTP and static React assets. Hosted storage, governance, and production orchestration remain in `ii-backend`.

## Risks exposed by the tracer

- The RDF parser provides triples, not concrete-syntax source spans. The tracer can safely replace one uniquely identified literal, but production editing must not generalize this string replacement. Issue #13 must select a source-aware patch strategy or an explicit RDF patch workflow and test comments, prefixes, multiline literals, blank nodes, and conflicts.
- Blank-node labels are document-local and parser-dependent. The named synthetic blank node makes the preservation proof deterministic, but issue #11 must define identity and provenance across real multi-file bundles.
- The tracer validator intentionally implements only the `sh:minCount` fixture check. A conformant SHACL engine remains isolated behind the validation interface and is selected by issues #9 and #23.
- The preview is a contract-shaped adapter, not a direct dependency on the current `shape2form` implementation. The real integration belongs to issue #27 so the tracer does not couple two evolving repositories prematurely.
- Lexical file paths are restricted to manifest members beneath one package root, and browser requests cannot choose paths. Production import, symlink, size, archive, and hostile-input policy still requires hardening.

## Rejected alternatives

- **Browser-only RDF editing:** rejected because safe multi-file access, recoverable patches, local validators, and future Go backend reuse would be pushed into browser-specific code or permission APIs.
- **Go-rendered HTML with no React interface:** rejected because it would not exercise the authoring interaction architecture or align with the reusable React path already present in `shape2form`.
- **Electron or another desktop shell now:** rejected because the tracer needs local file authority, not a second packaging and security platform. A loopback local process proves the seam with less lock-in.
- **Re-serialize the entire RDF graph after every edit:** rejected because it would rewrite prefixes, comments, blank-node labels, and unrelated source layout, violating the preservation requirement.
- **A separate hosted standard2shape service:** rejected by ADR 0003 because `ii-backend` already owns hosted access, storage, governance, and orchestration.
