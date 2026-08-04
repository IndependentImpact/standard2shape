import { useEffect, useMemo, useState } from "react";
import { loadWorkspace, saveGuidance } from "./api";
import type { Snapshot } from "./types";

function compact(value: string): string {
  const parts = value.split(/[/#]/).filter(Boolean);
  return parts.at(-1) ?? value;
}

function StatusMark({ good, children }: { good: boolean; children: React.ReactNode }) {
  return <span className={good ? "status status--good" : "status status--warn"}>{children}</span>;
}

export function App() {
  const [snapshot, setSnapshot] = useState<Snapshot>();
  const [guidance, setGuidance] = useState("");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("Loading the local fixture…");
  const [error, setError] = useState<string>();

  useEffect(() => {
    loadWorkspace()
      .then((next) => {
        setSnapshot(next);
        setGuidance(next.document.sections[0]?.guidance ?? "");
        setMessage("Local fixture loaded. No remote imports were fetched.");
      })
      .catch((reason: unknown) => {
        setError(reason instanceof Error ? reason.message : String(reason));
      });
  }, []);

  const expectedCases = useMemo(
    () => snapshot?.assessment.cases.filter((testCase) => testCase.conforms === testCase.expectedConforms).length ?? 0,
    [snapshot],
  );

  if (error) {
    return (
      <main className="state-page">
        <p className="eyebrow">standard2shape · tracer #7</p>
        <h1>The local bundle could not be opened</h1>
        <p role="alert">{error}</p>
      </main>
    );
  }
  if (!snapshot) {
    return (
      <main className="state-page" aria-busy="true">
        <p className="eyebrow">standard2shape · tracer #7</p>
        <h1>Opening the canonical bundle</h1>
        <p>{message}</p>
      </main>
    );
  }

  const section = snapshot.document.sections[0];
  const shape = snapshot.shapes[0];
  const dirty = guidance.trim() !== section.guidance;
  const save = async () => {
    setSaving(true);
    setError(undefined);
    setMessage("Applying the managed guidance patch…");
    try {
      const next = await saveGuidance(guidance);
      setSnapshot(next);
      setGuidance(next.document.sections[0]?.guidance ?? "");
      setMessage("Saved as a source patch and reloaded successfully.");
    } catch (reason: unknown) {
      setError(reason instanceof Error ? reason.message : String(reason));
      setMessage("The change was not applied.");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">standard2shape · executable tracer #7</p>
          <h1>{snapshot.document.label}</h1>
        </div>
        <div className="release-meta" aria-label="Package release">
          <span>release {snapshot.manifest.version}</span>
          <span className="offline-dot">local only</span>
        </div>
      </header>

      <div className="notice" role="status" aria-live="polite">
        <span aria-hidden="true">◆</span>
        <span>{message}</span>
      </div>
      {error ? <p className="inline-error" role="alert">{error}</p> : null}

      <main className="workspace">
        <aside className="rail" aria-label="Bundle navigation">
          <section className="panel tree-panel">
            <div className="panel-heading">
              <div>
                <p className="kicker">Bundle structure</p>
                <h2>Document tree</h2>
              </div>
              <span className="count">1 section</span>
            </div>
            <ol className="tree">
              <li>
                <div className="tree-node tree-node--section">
                  <span className="tree-icon" aria-hidden="true">§</span>
                  <span><strong>{section.label}</strong><small>Document section</small></span>
                </div>
                <ol>
                  {section.placements.map((placement) => (
                    <li key={placement.id} className="tree-node tree-node--placement">
                      <span className="tree-index">{placement.position}</span>
                      <span><strong>{placement.label}</strong><small>uses {compact(placement.shapeId)}</small></span>
                    </li>
                  ))}
                </ol>
              </li>
            </ol>
          </section>

          <section className="panel source-panel">
            <div className="panel-heading">
              <div>
                <p className="kicker">Provenance</p>
                <h2>Source files</h2>
              </div>
              <span className="count">{snapshot.sources.length}</span>
            </div>
            <ul className="source-list">
              {snapshot.sources.map((source) => (
                <li key={source.path}>
                  <code>{source.path}</code>
                  <span>{source.statements} statements</span>
                </li>
              ))}
            </ul>
          </section>
        </aside>

        <section className="editor-column" aria-label="Canonical authoring">
          <article className="panel editor-panel">
            <div className="panel-heading panel-heading--rule">
              <div>
                <p className="kicker">Selected section</p>
                <h2>{section.label}</h2>
              </div>
              <span className="canonical-badge">Canonical</span>
            </div>

            <div className="provenance-line">
              <span>Entity</span><code>{compact(section.id)}</code>
              <span>Source</span><code>{section.guidanceSource}</code>
            </div>

            <label className="field-label" htmlFor="canonical-guidance">
              Canonical guidance
              <span>Authoritative help text supplied by the standard</span>
            </label>
            <textarea
              id="canonical-guidance"
              value={guidance}
              rows={6}
              onChange={(event) => setGuidance(event.target.value)}
              aria-describedby="guidance-boundary"
            />
            <p id="guidance-boundary" className="field-note">
              Presentation UIs may add examples, but they cannot replace or clear this text.
            </p>
            <div className="editor-actions">
              <span className={dirty ? "dirty-indicator dirty-indicator--active" : "dirty-indicator"}>
                {dirty ? "Unsaved semantic change" : "Source and workspace agree"}
              </span>
              <button type="button" onClick={save} disabled={!dirty || saving || guidance.trim() === ""}>
                {saving ? "Applying patch…" : "Save canonical guidance"}
              </button>
            </div>
          </article>

          <article className="panel shape-panel">
            <div className="panel-heading">
              <div>
                <p className="kicker">Reusable graph entity</p>
                <h2>{compact(shape.id)}</h2>
              </div>
              <span className="count">{shape.properties.length} requirement</span>
            </div>
            <div className="shape-target"><span>Targets</span><code>{compact(shape.targetClass)}</code><span>from</span><code>{shape.source}</code></div>
            {shape.properties.map((property) => (
              <div className="requirement" key={property.id}>
                <div>
                  <strong>{property.label}</strong>
                  <code>{compact(property.path)}</code>
                </div>
                <div className="constraint-chips">
                  <span>{compact(property.datatype)}</span>
                  <span>min {property.minCount}</span>
                </div>
                <p>{property.guidance}</p>
                <small>{property.source} · {compact(property.id)}</small>
              </div>
            ))}
          </article>
        </section>

        <aside className="inspector" aria-label="Validation and downstream preview">
          <section className="panel preview-panel">
            <div className="panel-heading panel-heading--rule">
              <div>
                <p className="kicker">Downstream adapter</p>
                <h2>shape2form preview</h2>
              </div>
              <StatusMark good>Ready</StatusMark>
            </div>
            <div className="form-preview">
              <h3>{snapshot.preview.title}</h3>
              {snapshot.preview.fields.map((field) => (
                <div key={field.path} className="preview-field">
                  <label htmlFor="preview-title">{field.label}{field.required ? <span aria-label="required"> *</span> : null}</label>
                  <input id="preview-title" disabled placeholder="Project title" />
                  <p><span aria-hidden="true">ⓘ</span> {field.help}</p>
                </div>
              ))}
            </div>
          </section>

          <section className="panel validation-panel">
            <div className="panel-heading">
              <div>
                <p className="kicker">Local assessment</p>
                <h2>Fixture checks</h2>
              </div>
              <StatusMark good={expectedCases === snapshot.assessment.cases.length}>
                {expectedCases}/{snapshot.assessment.cases.length} expected
              </StatusMark>
            </div>
            <ul className="validation-list">
              {snapshot.assessment.cases.map((testCase) => (
                <li key={testCase.name}>
                  <span className={testCase.conforms === testCase.expectedConforms ? "case-dot case-dot--pass" : "case-dot"} aria-hidden="true" />
                  <span><strong>{testCase.name}</strong><small>{testCase.conforms ? "Conforms" : `${testCase.violations.length} expected violation`}</small></span>
                </li>
              ))}
            </ul>
            <p className="validator-id">{snapshot.assessment.validator} · {snapshot.assessment.validatorVersion}</p>
          </section>

          {snapshot.unsupported.map((item) => (
            <section className="panel unsupported-panel" key={item.digest}>
              <div className="unsupported-title"><span aria-hidden="true">!</span><div><p className="kicker">Preservation guard</p><h2>{item.kind}</h2></div></div>
              <p>{item.detail}</p>
              <dl>
                <div><dt>Source</dt><dd><code>{item.source}</code></dd></div>
                <div><dt>Digest</dt><dd><code>{item.digest.slice(0, 24)}…</code></dd></div>
              </dl>
            </section>
          ))}
        </aside>

        {snapshot.change ? (
          <section className="review-grid" aria-label="Change review">
            <article className="panel review-panel">
              <div className="panel-heading panel-heading--rule">
                <div><p className="kicker">Semantic review</p><h2>Meaning changed</h2></div>
                <StatusMark good={snapshot.change.unsupportedPreserved}>Unsupported graph preserved</StatusMark>
              </div>
              <pre data-testid="semantic-diff">{snapshot.change.semanticDiff}</pre>
            </article>
            <article className="panel review-panel">
              <div className="panel-heading panel-heading--rule">
                <div><p className="kicker">Source review</p><h2>Minimal patch</h2></div>
                <code>{snapshot.change.source}</code>
              </div>
              <pre data-testid="source-patch">{snapshot.change.patch}</pre>
            </article>
          </section>
        ) : null}
      </main>
    </div>
  );
}
