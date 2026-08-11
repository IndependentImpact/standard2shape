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

Each result records the check, its outcome, the evidence checked (paths and digests), and its violations. Applicability results identify their requirement and version; a SHACL result may identify a shape as its requirement; syntax, package-structure, reasoning-profile, and test-vector results carry no requirement identity because the package or vector already identifies what was checked. Result identity is the triple (check, requirement, vector) and must be unique within an assessment.

The outcome enumeration distinguishes the five dispositions the contract must keep apart:

- `conforms` — the check ran and the evidence satisfies the requirements;
- `non-conforms` — the check ran and found at least one violation-severity finding (a validation failure of the evidence, not of the evaluator);
- `evaluator-failure` — the evaluator itself failed to complete the check;
- `unsupported` — the evaluator does not implement the requested capability;
- `indeterminate` — applicability could not be decided from the evidence.

The last three require an explanatory `message` and carry no violations. Violations record the violated requirement IRI, a severity (`violation`, `warning`, `info`), a message, and optionally the focus node, property path, and source member.

A test-vector result identifies its vector with the expected outcome and, when the check ran, the actual outcome. The result conforms exactly when actual matches expected; observed violations belong to the vector's evidence, so an expected-invalid vector that fails as expected is a `conforms` result carrying violations.

## Conformance suite

A suite categorizes a package's conformance vectors as `valid`, `invalid`, or `boundary` and must contain at least one of each. Valid vectors expect `conforms`, invalid vectors expect `non-conforms`, and boundary vectors pin the decided outcome at a requirement's edge. A suite covers its package exactly: every manifest vector appears once, with the manifest's expected outcome.

## Semantic equivalence

Normalization sorts results by (check, requirement, vector), violations and evidence by their stable keys, and serializes with two-space indentation and one trailing newline. `SemanticBytes` additionally omits the evaluator identity and timestamps, which are attestation metadata: two assessments are semantically equivalent exactly when their `SemanticBytes` are equal. Equivalent local and hosted adapters must therefore produce byte-identical semantic content while keeping distinct attestations.

## Stable diagnostics

Contract errors reuse the package-contract diagnostic form: stable code, location, message. Initial codes include:

- `request.field.required`, `assessment.field.required`, `suite.field.required` — a required member is absent (required arrays may be empty but never omitted);
- `request.field.unknown` / `.field.duplicate` (and the assessment/suite forms) — unknown, differently cased, or repeated JSON fields;
- `request.check.invalid` / `request.check.duplicate` — an unknown or repeated requested check;
- `assessment.result.duplicate` — repeated (check, requirement, vector) identity;
- `assessment.result.message_required` — a failure, unsupported, or indeterminate result without explanation;
- `assessment.result.violations_invalid` — violations inconsistent with the outcome or the observed vector result;
- `assessment.result.vector_required` / `.vector_forbidden` / `.vector_actual_required` / `.vector_actual_forbidden` / `.vector_outcome_mismatch` — vector identity and outcome coupling;
- `assessment.result.requirement_required` / `.requirement_forbidden` — requirement identity coupling;
- `assessment.timestamp.invalid` / `.timestamp.order` — timestamps that are not RFC 3339 UTC instants or run backwards;
- `suite.category.invalid` / `.category.missing` / `.category.expected_mismatch` — category coverage and expectation coupling;
- `suite.vector.duplicate` / `.vector.unknown` / `.vector.missing` / `.vector.expected_mismatch` — suite/package coverage;
- `assessment.request.package_mismatch` / `.profile_mismatch` / `.check_missing` / `.check_unrequested` — an assessment that does not answer its request.

## Schema and verifier alignment

The JSON Schemas express the structural rules and the outcome couplings; the verifier enforces the same rules plus the ones a schema cannot state: uniqueness over compound identities, suite/package coverage, request/assessment coverage, and calendar validity of timestamps. The schema timestamp pattern is a syntactic bound — a calendar-invalid instant such as February 30 passes the pattern but is rejected by the verifier. No other divergence is permitted: the path, version, and digest patterns accept exactly what the verifier accepts.

## Versioning

`requestVersion`, `assessmentVersion`, and `suiteVersion` are `0.1` and follow the package-manifest policy: `0.x` contracts are experimental, require exact consumer support, and reject unknown fields and unsupported versions. After `1.0`, a major version change signals an incompatible contract and additive optional fields may use a minor version.
