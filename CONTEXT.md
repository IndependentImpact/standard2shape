# standard2shape

`standard2shape` describes the domain of authoring canonical data requirements for complete standards-based documents as SHACL shape bundles.

## Language

**Standards Author**:
A domain expert responsible for defining a standard's canonical data requirements, who may not know RDF or Turtle deeply.
_Avoid_: Form designer, RDF user

**RDF Reviewer**:
An RDF or ontology specialist who reviews canonical-shape changes for semantic and graph-model correctness.
_Avoid_: Approver, technical user

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
