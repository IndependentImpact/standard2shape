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

