package packagecontract

const ManifestVersionV01 = "0.1"

type Manifest struct {
	ManifestVersion    string               `json:"manifestVersion"`
	ID                 string               `json:"id"`
	Version            string               `json:"version"`
	StandardRelease    VersionedGraphEntity `json:"standardRelease"`
	DocumentRoots      []GraphEntity        `json:"documentRoots"`
	CanonicalShapes    []GraphEntity        `json:"canonicalShapes"`
	Artifacts          []SourceArtifact     `json:"artifacts"`
	Imports            []ImportReference    `json:"imports"`
	References         []ArtifactReference  `json:"references"`
	ReasoningProfile   VersionedGraphEntity `json:"reasoningProfile"`
	ConformanceVectors []ConformanceVector  `json:"conformanceVectors"`
}

type GraphEntity struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

type VersionedGraphEntity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Source  string `json:"source"`
}

type SourceArtifact struct {
	Path      string `json:"path"`
	Role      string `json:"role"`
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
}

type ImportReference struct {
	Source  string `json:"source"`
	IRI     string `json:"iri"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
	Policy  string `json:"policy"`
}

type ArtifactReference struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
	Source  string `json:"source"`
}

type ConformanceVector struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Digest   string `json:"digest"`
	Expected string `json:"expected"`
}

type Package struct {
	Root               string
	Manifest           Manifest
	NormalizedManifest []byte
	StatementCount     int
}

func (manifest Manifest) LocalPaths() []string {
	paths := make([]string, 0, len(manifest.Artifacts)+len(manifest.ConformanceVectors))
	for _, artifact := range manifest.Artifacts {
		paths = append(paths, artifact.Path)
	}
	for _, vector := range manifest.ConformanceVectors {
		paths = append(paths, vector.Path)
	}
	return paths
}
