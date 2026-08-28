package async

import (
	"encoding/json"
	"fmt"

	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/applications/dto"
)

func deserializeClusterContext(data []byte) (*dto.ClusterContextEvent, error) {
	var e dto.ClusterContextEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("unmarshal cluster context event: %w", err)
	}
	return &e, nil
}

func deserializeActiveNamespaces(data []byte) (*dto.ActiveNamespacesEvent, error) {
	var e dto.ActiveNamespacesEvent
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("unmarshal active namespaces event: %w", err)
	}
	return &e, nil
}
