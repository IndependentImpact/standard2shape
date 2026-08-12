package assessmentcontract

const (
	RequestVersionV01    = "0.1"
	AssessmentVersionV01 = "0.1"
	SuiteVersionV01      = "0.1"
)

const (
	CheckSyntax                    = "syntax"
	CheckPackageStructure          = "package-structure"
	CheckSHACL                     = "shacl"
	CheckReasoningProfile          = "reasoning-profile"
	CheckSemanticApplicability     = "semantic-applicability"
	CheckQuantitativeApplicability = "quantitative-applicability"
	CheckTestVectors               = "test-vectors"
)

const (
	OutcomeConforms         = "conforms"
	OutcomeNonConforms      = "non-conforms"
	OutcomeEvaluatorFailure = "evaluator-failure"
	OutcomeUnsupported      = "unsupported"
	OutcomeIndeterminate    = "indeterminate"
)

var Checks = []string{
	CheckSyntax,
	CheckPackageStructure,
	CheckSHACL,
	CheckReasoningProfile,
	CheckSemanticApplicability,
	CheckQuantitativeApplicability,
	CheckTestVectors,
}

type PackageRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

type EntityRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type EvidenceRef struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type Request struct {
	RequestVersion   string        `json:"requestVersion"`
	Package          PackageRef    `json:"package"`
	ReasoningProfile EntityRef     `json:"reasoningProfile"`
	Checks           []string      `json:"checks"`
	Evidence         []EvidenceRef `json:"evidence"`
	Requirements     []string      `json:"requirements"`
}

type Violation struct {
	Requirement string `json:"requirement"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Focus       string `json:"focus,omitempty"`
	Path        string `json:"path,omitempty"`
	Source      string `json:"source,omitempty"`
}

type VectorResult struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Actual   string `json:"actual,omitempty"`
}

type CheckResult struct {
	Check           string        `json:"check"`
	Outcome         string        `json:"outcome"`
	Requirement     *EntityRef    `json:"requirement,omitempty"`
	Message         string        `json:"message,omitempty"`
	Violations      []Violation   `json:"violations"`
	EvidenceChecked []EvidenceRef `json:"evidenceChecked"`
	Vector          *VectorResult `json:"vector,omitempty"`
}

type Assessment struct {
	AssessmentVersion string        `json:"assessmentVersion"`
	Package           PackageRef    `json:"package"`
	ReasoningProfile  EntityRef     `json:"reasoningProfile"`
	Evaluator         EntityRef     `json:"evaluator"`
	StartedAt         string        `json:"startedAt"`
	CompletedAt       string        `json:"completedAt"`
	Results           []CheckResult `json:"results"`
}

type SuiteVector struct {
	ID       string `json:"id"`
	Target   string `json:"target"`
	Category string `json:"category"`
	Expected string `json:"expected"`
}

type Suite struct {
	SuiteVersion string        `json:"suiteVersion"`
	Package      PackageRef    `json:"package"`
	Vectors      []SuiteVector `json:"vectors"`
}

// Adapter is the shared validation interface: local tooling and hosted
// execution implement it against the same request and assessment contracts,
// so neither deployment can change normative meaning.
type Adapter interface {
	Evaluate(request Request) (Assessment, error)
}
