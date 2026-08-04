package tracer

import "github.com/IndependentImpact/standard2shape/internal/packagecontract"

type Manifest = packagecontract.Manifest

type SourceSummary struct {
	Path       string `json:"path"`
	Statements int    `json:"statements"`
}

type DocumentSummary struct {
	ID       string           `json:"id"`
	Label    string           `json:"label"`
	Sections []SectionSummary `json:"sections"`
}

type SectionSummary struct {
	ID             string             `json:"id"`
	Label          string             `json:"label"`
	Guidance       string             `json:"guidance"`
	GuidanceSource string             `json:"guidanceSource"`
	Placements     []PlacementSummary `json:"placements"`
}

type PlacementSummary struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Position int    `json:"position"`
	ShapeID  string `json:"shapeId"`
}

type ShapeSummary struct {
	ID          string            `json:"id"`
	TargetClass string            `json:"targetClass"`
	Source      string            `json:"source"`
	Properties  []PropertySummary `json:"properties"`
}

type PropertySummary struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Label    string `json:"label"`
	Guidance string `json:"guidance"`
	Datatype string `json:"datatype"`
	MinCount int    `json:"minCount"`
	Source   string `json:"source"`
}

type ValidationAssessment struct {
	PackageID        string           `json:"packageId"`
	PackageVersion   string           `json:"packageVersion"`
	Validator        string           `json:"validator"`
	ValidatorVersion string           `json:"validatorVersion"`
	Cases            []ValidationCase `json:"cases"`
}

type ValidationCase struct {
	Name             string   `json:"name"`
	Conforms         bool     `json:"conforms"`
	ExpectedConforms bool     `json:"expectedConforms"`
	Violations       []string `json:"violations"`
}

type PreviewForm struct {
	ID     string         `json:"id"`
	Title  string         `json:"title"`
	Fields []PreviewField `json:"fields"`
}

type PreviewField struct {
	Path     string `json:"path"`
	Label    string `json:"label"`
	Help     string `json:"help"`
	Required bool   `json:"required"`
}

type UnsupportedSummary struct {
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Digest string `json:"digest"`
	Detail string `json:"detail"`
}

type ChangeSummary struct {
	Source               string `json:"source"`
	Before               string `json:"before"`
	After                string `json:"after"`
	SemanticDiff         string `json:"semanticDiff"`
	Patch                string `json:"patch"`
	UnsupportedPreserved bool   `json:"unsupportedPreserved"`
}

type Snapshot struct {
	Manifest    Manifest             `json:"manifest"`
	Document    DocumentSummary      `json:"document"`
	Shapes      []ShapeSummary       `json:"shapes"`
	Sources     []SourceSummary      `json:"sources"`
	Assessment  ValidationAssessment `json:"assessment"`
	Preview     PreviewForm          `json:"preview"`
	Unsupported []UnsupportedSummary `json:"unsupported"`
	Change      *ChangeSummary       `json:"change,omitempty"`
}
