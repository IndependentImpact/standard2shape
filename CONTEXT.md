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
A position in the ordered document tree that references a reusable canonical shape and supplies its document-specific parent and sequence.
_Avoid_: Shape copy, field instance

**Document Tree**:
The ordered hierarchy of document sections and document placements that defines the normative structure of one complete standards-based document.
_Avoid_: Shape graph, file tree, UI layout

**Document Section**:
A first-class structural container in the document tree that groups child sections or placements without capturing or constraining data itself.
_Avoid_: Empty shape, heading field, UI group
