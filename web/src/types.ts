export interface Snapshot {
  manifest: Manifest;
  document: DocumentSummary;
  shapes: ShapeSummary[];
  sources: SourceSummary[];
  assessment: ValidationAssessment;
  preview: PreviewForm;
  unsupported: UnsupportedSummary[];
  change?: ChangeSummary;
}

export interface Manifest {
  id: string;
  version: string;
  document: string;
  sources: string[];
  dataTests: Array<{ name: string; path: string; expectConforms: boolean }>;
}

export interface DocumentSummary {
  id: string;
  label: string;
  sections: SectionSummary[];
}

export interface SectionSummary {
  id: string;
  label: string;
  guidance: string;
  guidanceSource: string;
  placements: PlacementSummary[];
}

export interface PlacementSummary {
  id: string;
  label: string;
  position: number;
  shapeId: string;
}

export interface ShapeSummary {
  id: string;
  targetClass: string;
  source: string;
  properties: PropertySummary[];
}

export interface PropertySummary {
  id: string;
  path: string;
  label: string;
  guidance: string;
  datatype: string;
  minCount: number;
  source: string;
}

export interface SourceSummary {
  path: string;
  statements: number;
}

export interface ValidationAssessment {
  packageId: string;
  packageVersion: string;
  validator: string;
  validatorVersion: string;
  cases: ValidationCase[];
}

export interface ValidationCase {
  name: string;
  conforms: boolean;
  expectedConforms: boolean;
  violations: string[];
}

export interface PreviewForm {
  id: string;
  title: string;
  fields: PreviewField[];
}

export interface PreviewField {
  path: string;
  label: string;
  help: string;
  required: boolean;
}

export interface UnsupportedSummary {
  kind: string;
  source: string;
  digest: string;
  detail: string;
}

export interface ChangeSummary {
  source: string;
  before: string;
  after: string;
  semanticDiff: string;
  patch: string;
  unsupportedPreserved: boolean;
}

