package dto

type HelmChart struct {
	Name        string
	Description string
	Version     string
	AppVersion  string
	Repository  string
	Icon        string
}

type HelmChartDetail struct {
	Name        string
	Description string
	Version     string
	AppVersion  string
	Repository  string
	Icon        string
	Home        string
	Keywords    []string
	Sources     []string
	Maintainers []string // "Name <email>" or just "Name"
}

type HelmRepository struct {
	Name string
	URL  string
}

type HelmRelease struct {
	Name              string
	Namespace         string
	Chart             string
	ChartVersion      string
	AppVersion        string
	Status            string
	Revision          int
	Updated           string
	UpdatedAt         string
	Repository        string
	EncodedValuesYAML string
}

type HelmReleaseResource struct {
	Kind      string
	Name      string
	Namespace string
}

type HelmReleaseDetail struct {
	Name         string
	Namespace    string
	Chart        string
	ChartVersion string
	AppVersion   string
	Status       string
	Revision     int
	Updated      string
	UpdatedAt    string
	Notes        string
	ValuesYAML   string
	Resources    []HelmReleaseResource
	Repository   string
}

// HelmReleaseRevisionHistory represents a single revision in a release's history.
type HelmReleaseRevisionHistory struct {
	Revision     int
	ChartVersion string
	AppVersion   string
	Status       string
	UpdatedAt    string
}
