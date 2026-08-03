# standard2shape

Guided authoring and validation of canonical SHACL shape bundles for complex standards-based documents.

`standard2shape` is intended for standards authors who understand the document and its data requirements but may not know RDF or Turtle deeply. It helps them define an ordered bundle representing a complete document—such as a project design document (PDD) or monitoring report—while keeping the resulting SHACL reviewable by RDF specialists.

```text
standard requirements
        ↓
  standard2shape
        ↓
ordered canonical SHACL shape bundle
        ↓
    shape2form
        ↓
Flutter / HTML / React forms
```

The project is currently in requirements discovery. See [SPEC.md](SPEC.md) for the living product specification and [CONTEXT.md](CONTEXT.md) for the project vocabulary.

## Principles confirmed so far

- The unit of authorship is an ordered shape bundle representing one complete complex document or form.
- The primary user is a standards author; an RDF reviewer provides semantic and graph-model oversight.
- The output is canonical SHACL, not generated UI.
- Presentation annotations remain a separate concern handled by `shape2form` overlays and annotation tooling.
- Canonical guidance remains independently available; UI-specific guidance may supplement but never replace it.
- Semantic and quantitative applicability requirements are authoritative structured definitions; executable evaluators are versioned bindings with test vectors.
- Changes must be validated and reviewable before they alter a canonical standard.

