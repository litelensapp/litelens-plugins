package dto

type ClusterContextEvent struct {
	ContextName    string `json:"contextName"`
	KubeconfigPath string `json:"kubeconfigPath"`
}

type ActiveNamespacesEvent struct {
	Namespaces []string `json:"namespaces"`
}
