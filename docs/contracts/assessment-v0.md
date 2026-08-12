# Validation assessment and conformance-suite contract v0.1

Status: experimental contract for issue #9

## Boundary

One technology-neutral request and result contract covers every check an evaluator can run against an immutable standard package: syntax, package structure, SHACL, reasoning profile, semantic applicability, quantitative applicability, and test-vector execution. Local tooling and hosted execution implement the same contract through adapters; neither deployment may change normative meaning. An adapter attests what it evaluated — it is never the source of truth for what the requirements mean.

The contract is plain JSON with closed schemas and no implementation-language-specific types: strings, arrays, objects, and closed enumerations only.

## Contract files

- [`contracts/v0/validation-request.schema.json`](../../contracts/v0/validation-request.schema.json) defines a validation request.
- [`contracts/v0/assessment.schema.json`](../../contracts/v0/assessment.schema.json) defines a validation assessment.
- [`contracts/v0/conformance-suite.schema.json`](../../contracts/v0/conformance-suite.schema.json) defines a categorized conformance suite.
- [`fixtures/assessment-v0`](../../fixtures/assessment-v0) holds valid request, assessment, and suite examples plus rejected examples with stable diagnostic codes.

Field names in all three documents are matched exactly, duplicate fields are rejected, and the required array fields are always present — empty collections serialize as `[]`, never `null` — so a normalized document always satisfies its published schema.

## Request

A request names the package by identity, version, and normalized-manifest digest; the reasoning profile under which semantic requirements must be read; the set of checks to run (at least one, no duplicates); the evidence graphs to validate by package-relative path and digest; and the specific requirement IRIs to evaluate (both arrays may be empty). Because the package is pinned by digest, equal requests against equal packages are answerable by any conforming adapter.

## Assessment

An assessment answers one request. It repeats the package and reasoning-profile identity, attests the evaluator identity and version, records `startedAt` and `completedAt` as RFC 3339 UTC instants, and carries one result per executed check unit.

Each result records the check, its outcome, the evidence checked (paths and digests), and its violations. Applicability results identify their requirement and version — a methodology-owned applicability requirement the package manifest inventories from its canonical graph, not a document shape. All other results carry no requirement identity: the package, the shapes in its graph, or the vector already identify what was checked, and violated shapes are named per violation. Result identity is the triple (check, requirement, vector) and must be unique within an assessment.

The outcome enumeration distinguishes the five dispositions the contract must keep apart:

- `conforms` — the check ran and the evidence satisfies the requirements;
- `non-conforms` — the check ran and found at least one violation-severity finding (a validation failure of the evidence, not of the evaluator);
- `evaluator-failure` — the evaluator itself failed to complete the check;
- `unsupported` — the evaluator does not implement the requested capability;
- `indeterminate` — applicability could not be decided from the evidence; it is defined only for the two applicability checks.

The last three require an explanatory `message` and carry no violations. Violations record the violated requirement IRI, a severity (`violation`, `warning`, `info`), a message, and optionally the focus node, property path, and source member.

A test-vector result identifies its vector with the expected outcome and, when the check ran, the actual outcome. The result conforms exactly when actual matches expected; observed violations belong to the vector's evidence, so an expected-invalid vector that fails as expected is a `conforms` result carrying violations.

An assessment answers exactly its request against its package, and the request itself must be bound to the supplied package: its package identity, version, and normalized-manifest digest and its reasoning profile must match the package, and it may request only requirements the manifest declares — so no other package can supply the accepted vector and evidence inventory. Every requested check and every requested requirement has at least one result and no result covers an unrequested check or requirement. Requirement coverage is kind-paired: a requested requirement is answered only by the applicability check matching its declared kind and version, and a result pairing a requirement with the other kind's check is rejected — adapters cannot swap evaluator semantics between requirements undetected. When test-vector execution is requested, every conformance vector the manifest declares has a result carrying the manifest's expected outcome, so no vector can be silently omitted. Every requested evidence graph is attested by at least one result with the pinned digest, and every attested path must be either requested evidence or a declared package member — in both cases under its pinned digest. A request whose checks include applicability must request at least one requirement of the matching kind and must include the kind-matched check for every requested requirement, so every accepted request is answerable.

## Conformance suite

A suite categorizes a package's conformance vectors as `valid`, `invalid`, or `boundary`, each naming the executable requirement it exercises. The requirement labels are not self-declared: the package manifest canonically binds each vector to a declared canonical shape or reference, and a suite whose labels differ from the manifest is rejected. Coverage is per requirement and measured against the manifest: every requirement the manifest's vectors declare must have at least one suite vector of each category, so no executable requirement ships without valid, invalid, and boundary cases. Valid vectors expect `conforms`, invalid vectors expect `non-conforms`, and boundary vectors pin the decided outcome at the requirement's edge. A suite covers its package exactly: every manifest vector appears once, with the manifest's expected outcome.

## Semantic equivalence

Normalization sorts results by (check, requirement, vector), violations and evidence by their stable keys, and serializes with two-space indentation and one trailing newline. `SemanticBytes` additionally omits the evaluator identity and timestamps, which are attestation metadata: two assessments are semantically equivalent exactly when their `SemanticBytes` are equal. Equivalent local and hosted adapters must therefore produce byte-identical semantic content while keeping distinct attestations.

## Stable diagnostics

Contract errors reuse the package-contract diagnostic form: stable code, location, message. Initial codes include:

- `request.field.required`, `assessment.field.required`, `suite.field.required` — a required member is absent, whether scalar, object, or array (required arrays may be empty but never omitted);
- `request.field.unknown` / `.field.duplicate` / `.field.null` / `.field.empty` (and the assessment/suite forms) — unknown, differently cased, repeated, explicitly null, or present-but-empty JSON fields;
- `assessment.field.blank` — a present optional string containing only whitespace;
- `request.check.invalid` / `request.check.duplicate` — an unknown or repeated requested check;
- `request.requirements.empty` — applicability checks requested without any requirement;
- `assessment.result.duplicate` — repeated (check, requirement, vector) identity;
- `assessment.result.message_required` — a failure, unsupported, or indeterminate result without explanation;
- `assessment.result.outcome_forbidden` — indeterminate used outside the applicability checks;
- `assessment.result.violations_invalid` — violations inconsistent with the outcome or the observed vector result;
- `assessment.result.vector_required` / `.vector_forbidden` / `.vector_actual_required` / `.vector_actual_forbidden` / `.vector_outcome_mismatch` — vector identity and outcome coupling;
- `assessment.result.requirement_required` / `.requirement_forbidden` — requirement identity coupling;
- `assessment.timestamp.invalid` / `.timestamp.order` — timestamps that are not RFC 3339 UTC instants or run backwards;
- `suite.category.invalid` / `.category.missing` / `.category.expected_mismatch` — per-requirement category coverage and expectation coupling;
- `suite.vector.duplicate` / `.vector.unknown` / `.vector.missing` / `.vector.expected_mismatch` / `.vector.requirement_mismatch` — suite/package coverage and manifest binding;
- `assessment.request.package_unbound` / `.profile_unbound` / `.requirement_unknown` / `.requirement_uncheckable` / `.check_unanswerable` — a request that is not bound to the supplied package or cannot be answered against it;
- `assessment.request.package_mismatch` / `.profile_mismatch` / `.check_missing` / `.check_unrequested` / `.requirement_missing` / `.requirement_unrequested` / `.requirement_kind_mismatch` / `.requirement_version_mismatch` / `.evidence_missing` / `.evidence_mismatch` / `.evidence_unknown` / `.vector_missing` / `.vector_unknown` / `.vector_expected_mismatch` — an assessment that does not answer its request against its package.

## Schema and verifier alignment

The JSON Schemas express the structural rules and the outcome couplings; the verifier enforces the same rules plus the ones a schema cannot state: uniqueness over compound identities, per-requirement suite coverage, suite/package coverage, request/assessment coverage, and calendar validity of timestamps. The schema timestamp pattern is a syntactic bound — a calendar-invalid instant such as February 30 passes the pattern but is rejected by the verifier. No other divergence is permitted: the path, version, and digest patterns accept exactly what the verifier accepts; explicit `null` values and present-but-empty or whitespace-only strings are rejected by both, with blankness following the ECMA-262 `\s` classes JSON Schema regexes are defined against (including NBSP, excluding NEL); and the test suite runs every fixture and every normalized document through a Draft 2020-12 validator against the published schemas with format assertion enabled and `iri` bound to the verifier's own IRI predicate. JSON Schema treats `format` as an annotation unless a consumer asserts it; conforming consumers must assert formats or apply an equivalent IRI check, and validators whose regex engine is not ECMA-262 (for example RE2) differ from the published semantics on exotic Unicode whitespace.

## Versioning

`requestVersion`, `assessmentVersion`, and `suiteVersion` are `0.1` and follow the package-manifest policy: `0.x` contracts are experimental, require exact consumer support, and reject unknown fields and unsupported versions. After `1.0`, a major version change signals an incompatible contract and additive optional fields may use a minor version.
