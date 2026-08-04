# standard2shape

`standard2shape` describes the domain of authoring canonical data requirements for complete standards-based documents as SHACL shape bundles.

## Language

**Standards Author**:
A domain expert responsible for defining a standard's canonical data requirements, who may not know RDF or Turtle deeply.
_Avoid_: Form designer, RDF user

**RDF Reviewer**:
An RDF or ontology specialist who reviews canonical-shape changes for semantic and graph-model correctness.
_Avoid_: Approver, technical user

**Standard**:
A normative governance instrument that authorizes indicators and methodologies and defines the document structures, requirements, and guidance under which they may be used.
_Avoid_: Methodology, indicator, validation bundle

**Indicator**:
The domain unit targeted by a methodology: the canonical definition of what is measured, quantified, or reported, including its meaning and applicable measurement unit or dimension.
_Avoid_: Methodology result value, formula, applicability condition

**Indicator Formula**:
An indicator's identity-defining mathematical expression: the canonical relationship among quantities that distinguishes what the indicator means, independently of any methodology used to determine it.
_Avoid_: Methodology equation, calculation step, evaluator code

**Methodology**:
A canonical procedure for determining an indicator, including its semantic and quantitative applicability conditions, required inputs, calculations, and evidence requirements.
_Avoid_: Standard, indicator, evaluator implementation

**Equation Step**:
A methodology-owned transformation that consumes variables or constants and produces an intermediate or final variable as one step in determining an indicator. It may operationalize part of an indicator formula but is not itself an indicator formula.
_Avoid_: Indicator formula, indicator definition

**Standard Authorization**:
A traceable normative statement that a standard permits an identified indicator or methodology to be used. Authorization does not transfer ownership of the indicator or methodology definition into the standard.
_Avoid_: Definition, copy, import

**Ordered Shape Bundle**:
An ordered collection of canonical SHACL shapes that together define the data requirements of one complete complex document or form.
_Avoid_: Form, Turtle file, schema file

**Canonical Shape**:
A SHACL node or property shape that expresses the standard's normative data structure or constraints, independent of UI presentation.
_Avoid_: Form field, UI schema

**Document Order**:
The normative sequence of chapters, sections, requirement groups, and requirements in a standards-based document.
_Avoid_: UI order, field order, layout order

**Presentation Order**:
The sequence in which a particular generated interface presents fields to its users; it is not part of the canonical document definition.
_Avoid_: Document order, canonical order

**Document Placement**:
A position in the ordered document tree that references a reusable canonical shape and supplies document-specific context, parent, and sequence without overriding the shape's constraints.
_Avoid_: Shape copy, field instance

**Document Tree**:
The ordered hierarchy of document sections and document placements that defines the normative structure of one complete standards-based document.
_Avoid_: Shape graph, file tree, UI layout

**Document Section**:
A first-class structural container in the document tree that groups child sections or placements without capturing or constraining data itself.
_Avoid_: Empty shape, heading field, UI group

**Canonical Guidance**:
Authoritative instructional or explanatory text that is part of the standard, remains independently available in every conforming UI, and may be attached to a document section or canonical shape.
_Avoid_: UI help, tooltip copy, placeholder

**Supplemental Guidance**:
Non-canonical text added by a UI developer for a particular interface, such as examples or task-specific clarification; it can accompany but never replace canonical guidance.
_Avoid_: Canonical guidance, normative text

**Placement Context**:
Canonical context owned by a document placement, such as applicability guidance, that explains the referenced shape's role in that document without changing its constraints.
_Avoid_: Constraint override, UI context

**Applicability Requirement**:
A canonical semantic or quantitative rule owned by a methodology that must be satisfied before that methodology may be used for its target indicator.
_Avoid_: Visibility rule, UI condition

**Semantic Requirement**:
An applicability requirement evaluated against RDF meaning and relationships, normally by SHACL under an explicit reasoning profile.
_Avoid_: Text rule, category label

**Quantitative Requirement**:
An applicability requirement expressed as a structured rule over typed quantities, units, thresholds, or formulas.
_Avoid_: Numerical requirement, R check

**Evaluation Binding**:
A versioned link from an authoritative requirement definition to a software implementation that evaluates it.
_Avoid_: Requirement definition, source code ownership

**Applicability Assessment**:
The result of evaluating applicability requirements, containing conformance, violations, and an attestation of what was checked.
_Avoid_: Boolean result, form validation
