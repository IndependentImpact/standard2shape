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

The project is currently in requirements discovery with a phased delivery backlog. See [SPEC.md](SPEC.md) for the living product specification, [CONTEXT.md](CONTEXT.md) for the project vocabulary, and [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md) for the implementation sequence and GitHub issue map.

## Run the issue #7 tracer

The first executable slice uses only synthetic repository fixtures. It opens an isolated writable copy, never follows remote ontology imports, and binds its local server to loopback by default.

Prerequisites: Go 1.23 and a Vite-supported Node release (Node 20 or 22).

```sh
make tracer
```

Open <http://127.0.0.1:8090>. Edit the canonical section guidance and save it to see the semantic diff, minimal source patch, reload result, validation assessment, downstream preview, and unsupported-SHACL preservation guard.

Run all non-browser checks with:

```sh
npm ci
npm audit --audit-level=moderate
npm run typecheck
npm run build
go test ./...
go vet ./...
go build ./...
```

After installing Chromium once with `npx playwright install chromium`, run the desktop and mobile browser flow with `npm run test:e2e`. The tracer's architectural findings and limitations are recorded in [docs/architecture/tracer-7-evidence.md](docs/architecture/tracer-7-evidence.md).

## Principles confirmed so far

- The unit of authorship is an ordered shape bundle representing one complete complex document or form.
- The primary user is a standards author; an RDF reviewer provides semantic and graph-model oversight.
- The output is canonical SHACL, not generated UI.
- Presentation annotations remain a separate concern handled by `shape2form` overlays and annotation tooling.
- Canonical guidance remains independently available; UI-specific guidance may supplement but never replace it.
- Semantic and quantitative applicability requirements are authoritative structured definitions; executable evaluators are versioned bindings with test vectors.
- Changes must be validated and reviewable before they alter a canonical standard.
