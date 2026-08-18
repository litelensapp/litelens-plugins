package helm

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	helmrest "github.com/litelensapp/litelens-plugins/plugins/helm/internal/api"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/dto"
	"github.com/litelensapp/litelens-plugins/plugins/helm/internal/kube"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

// HelmRepositoryLabel is the release label key used to store the chart's source repository.
const HelmRepositoryLabel = "meta.litelens.io/helm-repository-name"

func (s *Service) ListHelmReleases(namespace string) ([]dto.HelmRelease, error) {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return []dto.HelmRelease{}, nil
	}
	if rc == nil {
		return []dto.HelmRelease{}, nil
	}

	getter := &helmrest.Getter{
		RC:        rc,
		Rules:     kube.LoadingRules(kubeconfigPaths),
		Overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return []dto.HelmRelease{}, fmt.Errorf("helm: init configuration: %w", err)
	}

	list := action.NewList(cfg)
	list.AllNamespaces = namespace == ""
	list.StateMask = action.ListAll

	releases, err := list.Run()
	if err != nil {
		return []dto.HelmRelease{}, fmt.Errorf("helm: list releases: %w", err)
	}

	result := make([]dto.HelmRelease, 0, len(releases))
	for _, r := range releases {
		if r.Chart == nil || r.Chart.Metadata == nil {
			continue
		}
		repository := ""
		if r.Labels != nil {
			repository = r.Labels[HelmRepositoryLabel]
		}
		result = append(result, dto.HelmRelease{
			Name:              r.Name,
			Namespace:         r.Namespace,
			Chart:             r.Chart.Metadata.Name,
			ChartVersion:      r.Chart.Metadata.Version,
			AppVersion:        r.Chart.Metadata.AppVersion,
			Status:            r.Info.Status.String(),
			Revision:          r.Version,
			Updated:           helmAge(r.Info.LastDeployed.Time),
			UpdatedAt:         r.Info.LastDeployed.Time.Format(time.RFC3339),
			Repository:        repository,
			EncodedValuesYAML: compressValuesYAML(mergedValuesYAML(r)),
		})
	}
	return result, nil
}

// InstallHelmChart installs a helm chart into the specified namespace.
func (s *Service) InstallHelmChart(namespace, releaseName, repository, chartName, version, valuesYAML string) error {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return fmt.Errorf("helm: no REST config for active context")
	}

	// Load the chart index to get the download URL
	cacheDir := helmpath.CachePath("repository")
	indexPath := filepath.Join(cacheDir, repository+"-index.yaml")
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return fmt.Errorf("helm: load index: %w", err)
	}

	versions, ok := index.Entries[chartName]
	if !ok || len(versions) == 0 {
		return fmt.Errorf("helm: chart %q not found in repo %q", chartName, repository)
	}

	var chartVersion = versions[0]
	if version != "" {
		found := false
		for _, v := range versions {
			if v != nil && v.Version == version {
				chartVersion = v
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("helm: version %q of chart %q not found", version, chartName)
		}
	}

	if len(chartVersion.URLs) == 0 {
		return fmt.Errorf("helm: no download URLs found for chart %q version %q", chartName, chartVersion.Version)
	}
	chartURL, err := resolveChartURL(repository, chartVersion.URLs[0])
	if err != nil {
		return fmt.Errorf("helm: resolve chart URL: %w", err)
	}

	dlCtx, dlCancel := context.WithTimeout(s.provider.Ctx(), 60*time.Second)
	defer dlCancel()
	chart, err := downloadChart(dlCtx, chartURL)
	if err != nil {
		return err
	}

	// Set up helm configuration wired to the active cluster context.
	getter := &helmrest.Getter{
		RC:        rc,
		Rules:     kube.LoadingRules(kubeconfigPaths),
		Overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return fmt.Errorf("helm: init configuration: %w", err)
	}

	// Create install action (synchronous setup only)
	install := action.NewInstall(cfg)
	install.Namespace = namespace
	if releaseName == "" {
		install.ReleaseName = fmt.Sprintf("%s-%d", chartName, time.Now().Unix())
	} else {
		install.ReleaseName = releaseName
	}
	install.CreateNamespace = false
	install.Labels = map[string]string{
		HelmRepositoryLabel: repository,
	}

	// Parse custom values YAML if provided
	var vals map[string]interface{}
	if valuesYAML != "" {
		if err := yaml.Unmarshal([]byte(valuesYAML), &vals); err != nil {
			return fmt.Errorf("helm: parse custom values: %w", err)
		}
	}

	// Run the actual install asynchronously so the frontend receives control
	// immediately after setup and can display the pending-install release.
	relName := install.ReleaseName
	go func() {
		installCtx, installCancel := context.WithTimeout(s.provider.Ctx(), 5*time.Minute)
		defer installCancel()
		if _, err := install.RunWithContext(installCtx, chart, vals); err != nil {
			s.emit(s.provider.Ctx(), "helm:install:error", map[string]string{
				"releaseName": relName,
				"chartName":   chartName,
				"error":       err.Error(),
			})
			return
		}
		s.emit(s.provider.Ctx(), "helm:install:complete", map[string]string{
			"releaseName": relName,
			"chartName":   chartName,
		})
	}()

	return nil
}

// UpgradeHelmRelease upgrades an existing helm release to a different chart version.
func (s *Service) UpgradeHelmRelease(namespace, releaseName, repository, chartName, version, valuesYAML string) error {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return fmt.Errorf("helm: no REST config for active context")
	}

	// Load the chart index to get the download URL for the target version
	cacheDir := helmpath.CachePath("repository")
	indexPath := filepath.Join(cacheDir, repository+"-index.yaml")
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return fmt.Errorf("helm: load index: %w", err)
	}

	versions, ok := index.Entries[chartName]
	if !ok || len(versions) == 0 {
		return fmt.Errorf("helm: chart %q not found in repo %q", chartName, repository)
	}

	var chartVersion = versions[0]
	if version != "" {
		found := false
		for _, v := range versions {
			if v != nil && v.Version == version {
				chartVersion = v
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("helm: version %q of chart %q not found", version, chartName)
		}
	}

	if len(chartVersion.URLs) == 0 {
		return fmt.Errorf("helm: no download URLs found for chart %q version %q", chartName, chartVersion.Version)
	}
	chartURL, err := resolveChartURL(repository, chartVersion.URLs[0])
	if err != nil {
		return fmt.Errorf("helm: resolve chart URL: %w", err)
	}

	dlCtx, dlCancel := context.WithTimeout(s.provider.Ctx(), 60*time.Second)
	defer dlCancel()
	chart, err := downloadChart(dlCtx, chartURL)
	if err != nil {
		return err
	}

	// Set up helm configuration wired to the active cluster context.
	getter := &helmrest.Getter{
		RC:        rc,
		Rules:     kube.LoadingRules(kubeconfigPaths),
		Overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return fmt.Errorf("helm: init configuration: %w", err)
	}

	// Parse custom values YAML if provided, otherwise use empty map
	var vals map[string]any
	if valuesYAML != "" {
		if err := yaml.Unmarshal([]byte(valuesYAML), &vals); err != nil {
			return fmt.Errorf("helm: parse custom values: %w", err)
		}
	}

	// Run the upgrade asynchronously
	relName := releaseName
	go func() {
		upgradeCtx, upgradeCancel := context.WithTimeout(s.provider.Ctx(), 5*time.Minute)
		defer upgradeCancel()

		upgrade := action.NewUpgrade(cfg)
		upgrade.Namespace = namespace
		upgrade.Labels = map[string]string{
			HelmRepositoryLabel: repository,
		}

		if _, err := upgrade.RunWithContext(upgradeCtx, relName, chart, vals); err != nil {
			s.emit(s.provider.Ctx(), "helm:upgrade:error", map[string]string{
				"releaseName": relName,
				"chartName":   chartName,
				"error":       err.Error(),
			})
			return
		}
		s.emit(s.provider.Ctx(), "helm:upgrade:complete", map[string]string{
			"releaseName": relName,
			"chartName":   chartName,
		})
	}()

	return nil
}

// DeleteHelmRelease uninstalls a helm release from the specified namespace.
func (s *Service) DeleteHelmRelease(namespace, releaseName string) error {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return fmt.Errorf("helm: no REST config for active context")
	}

	// Set up helm configuration wired to the active cluster context.
	getter := &helmrest.Getter{
		RC:        rc,
		Rules:     kube.LoadingRules(kubeconfigPaths),
		Overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return fmt.Errorf("helm: init configuration: %w", err)
	}

	// Create and run uninstall action
	uninstall := action.NewUninstall(cfg)

	_, err := uninstall.Run(releaseName)
	if err != nil {
		return fmt.Errorf("helm: uninstall release: %w", err)
	}

	return nil
}

// DeleteHelmReleaseWithCleanup uninstalls a helm release synchronously, then
// asynchronously deletes any resources from the release manifest that the
// helm uninstall left behind (e.g. hook jobs with keep delete policy).
// Emits helm:cleanup:complete, helm:cleanup:partial, or helm:cleanup:error
// when the background sweep finishes.
func (s *Service) DeleteHelmReleaseWithCleanup(namespace, releaseName string) error {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return fmt.Errorf("helm: no REST config for active context")
	}

	getter := &helmrest.Getter{
		RC:        rc,
		Rules:     kube.LoadingRules(kubeconfigPaths),
		Overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return fmt.Errorf("helm: init configuration: %w", err)
	}

	// Capture the full manifest before uninstalling so we know exactly which
	// resources Helm deployed — regardless of what labels they carry.
	rel, err := action.NewGet(cfg).Run(releaseName)
	if err != nil {
		return fmt.Errorf("helm: get release manifest: %w", err)
	}
	manifestResources := parseManifestResources(rel.Manifest, namespace)

	// Hooks (pre-/post-install jobs etc.) live in rel.Hooks, not rel.Manifest.
	for _, hook := range rel.Hooks {
		if hook == nil {
			continue
		}
		manifestResources = append(manifestResources, parseManifestResources(hook.Manifest, namespace)...)
	}

	// Helm uninstall is synchronous — the release record is removed before we return.
	uninstall := action.NewUninstall(cfg)
	if _, err := uninstall.Run(releaseName); err != nil {
		return fmt.Errorf("helm: uninstall release: %w", err)
	}

	// Delete remaining manifest resources asynchronously. Helm already removes
	// most resources; we delete whatever is left (e.g. hook jobs with keep
	// delete policy). NotFound errors are silently ignored.
	go func() {
		dynClient, err := dynamic.NewForConfig(rc)
		if err != nil {
			s.emit(s.provider.Ctx(), "helm:cleanup:error", map[string]any{
				"releaseName": releaseName,
				"error":       fmt.Sprintf("build dynamic client: %s", err.Error()),
			})
			return
		}

		cleanCtx, cleanCancel := context.WithTimeout(s.provider.Ctx(), 60*time.Second)
		defer cleanCancel()

		var deleted int
		var cleanupErrs []string

		// Build Kind → GVR + namespaced flag from discovery so we can look up
		// resources by Kind (the only type info present in a manifest).
		// ServerPreferredResources can return partial results alongside an error;
		// record any error but still process whatever groups were returned.
		type gvrMeta struct {
			gvr        schema.GroupVersionResource
			namespaced bool
		}
		kindMap := map[string]gvrMeta{}
		serverResources, discoveryErr := cs.Discovery().ServerPreferredResources()
		if discoveryErr != nil {
			cleanupErrs = append(cleanupErrs, fmt.Sprintf("resource discovery: %s", discoveryErr.Error()))
		}
		for _, rl := range serverResources {
			gv, err := schema.ParseGroupVersion(rl.GroupVersion)
			if err != nil {
				continue
			}
			for _, r := range rl.APIResources {
				if strings.Contains(r.Name, "/") {
					continue
				}
				if helmHasVerb(r.Verbs, "delete") {
					kindMap[r.Kind] = gvrMeta{
						gvr:        schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: r.Name},
						namespaced: r.Namespaced,
					}
				}
			}
		}

		// Step 1: Build resource identity set.
		type resKey struct {
			kind      string
			namespace string
			name      string
		}
		inManifest := map[resKey]bool{}
		for _, res := range manifestResources {
			ns := res.Namespace
			if ns == "" {
				ns = namespace
			}
			inManifest[resKey{res.Kind, ns, res.Name}] = true
		}

		// Step 2: Fetch ownerReferences for each resource.
		ownerOf := map[resKey][]resKey{}
		for _, res := range manifestResources {
			gm, ok := kindMap[res.Kind]
			if !ok {
				continue
			}
			ns := res.Namespace
			if ns == "" {
				ns = namespace
			}
			var ri dynamic.ResourceInterface
			if gm.namespaced {
				ri = dynClient.Resource(gm.gvr).Namespace(ns)
			} else {
				ri = dynClient.Resource(gm.gvr)
			}
			obj, err := ri.Get(cleanCtx, res.Name, metav1.GetOptions{})
			if err != nil {
				if !k8serrors.IsNotFound(err) {
					cleanupErrs = append(cleanupErrs, fmt.Sprintf("%s/%s: get: %s", res.Kind, res.Name, err.Error()))
				}
				// IsNotFound = helm already deleted it; no owners to fetch
				continue
			}
			for _, ref := range obj.GetOwnerReferences() {
				ownerKey := resKey{ref.Kind, ns, ref.Name}
				if inManifest[ownerKey] {
					ownerOf[resKey{res.Kind, ns, res.Name}] = append(ownerOf[resKey{res.Kind, ns, res.Name}], ownerKey)
				}
			}
		}

		// Step 3: Topological deletion loop.
		deletedSet := map[resKey]bool{}

		for {
			if cleanCtx.Err() != nil {
				break
			}
			progress := false
			for _, res := range manifestResources {
				gm, ok := kindMap[res.Kind]
				if !ok {
					continue
				}
				ns := res.Namespace
				if ns == "" {
					ns = namespace
				}
				key := resKey{res.Kind, ns, res.Name}
				if deletedSet[key] {
					continue
				}

				// Only delete if all manifest-owners have already been deleted.
				ready := true
				for _, ownerKey := range ownerOf[key] {
					if !deletedSet[ownerKey] {
						ready = false
						break
					}
				}
				if !ready {
					continue
				}

				var ri dynamic.ResourceInterface
				if gm.namespaced {
					ri = dynClient.Resource(gm.gvr).Namespace(ns)
				} else {
					ri = dynClient.Resource(gm.gvr)
				}
				// Background propagation: object marked for deletion immediately; owned pods
				// drain concurrently. Label sweeps run after a grace period (see below).
				if err := ri.Delete(cleanCtx, res.Name, metav1.DeleteOptions{}); err != nil {
					if !k8serrors.IsNotFound(err) {
						cleanupErrs = append(cleanupErrs, fmt.Sprintf("%s/%s: %s", res.Kind, res.Name, err.Error()))
					}
				} else {
					deleted++
				}
				deletedSet[key] = true
				progress = true
			}
			if !progress {
				break // no more resources can be unblocked
			}
		}

		// Allow operator pods (deleted with Background propagation) to finish their
		// reconciliation cycle before label-based sweeps run. Background deletion
		// marks the parent for deletion but pods continue running briefly (~30s grace).
		// Use a context-aware select so the sleep exits immediately on cancellation.
		select {
		case <-time.After(5 * time.Second):
		case <-cleanCtx.Done():
		}
		if cleanCtx.Err() != nil {
			cleanupErrs = append(cleanupErrs, fmt.Sprintf("cleanup cancelled before label sweeps: %s", cleanCtx.Err()))
		} else {

			// ConfigMaps created by running operators are not in the manifest;
			// sweep them by the standard Helm instance label.
			cmList, cmListErr := cs.CoreV1().ConfigMaps(namespace).List(
				cleanCtx,
				metav1.ListOptions{LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName)},
			)
			if cmListErr != nil {
				cleanupErrs = append(cleanupErrs, fmt.Sprintf("list ConfigMaps: %s", cmListErr.Error()))
			} else {
				for _, cm := range cmList.Items {
					if err := cs.CoreV1().ConfigMaps(namespace).Delete(
						cleanCtx, cm.Name, metav1.DeleteOptions{},
					); err != nil {
						if !k8serrors.IsNotFound(err) {
							cleanupErrs = append(cleanupErrs, fmt.Sprintf("ConfigMap/%s: %s", cm.Name, err.Error()))
						}
					} else {
						deleted++
					}
				}
			}

			// PVCs created by StatefulSet volumeClaimTemplates are not in the manifest;
			// sweep them by the standard Helm instance label.
			pvcList, pvcListErr := cs.CoreV1().PersistentVolumeClaims(namespace).List(
				cleanCtx,
				metav1.ListOptions{LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName)},
			)
			if pvcListErr != nil {
				cleanupErrs = append(cleanupErrs, fmt.Sprintf("list PVCs: %s", pvcListErr.Error()))
			} else {
				for _, pvc := range pvcList.Items {
					if err := cs.CoreV1().PersistentVolumeClaims(namespace).Delete(
						cleanCtx, pvc.Name, metav1.DeleteOptions{},
					); err != nil {
						if !k8serrors.IsNotFound(err) {
							cleanupErrs = append(cleanupErrs, fmt.Sprintf("PersistentVolumeClaim/%s: %s", pvc.Name, err.Error()))
						}
					} else {
						deleted++
					}
				}
			}

			// Pods created by hook Jobs become orphaned once the Job is deleted;
			// delete terminal (Failed/Succeeded) pods that carry the release label.
			podList, podListErr := cs.CoreV1().Pods(namespace).List(
				cleanCtx,
				metav1.ListOptions{LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName)},
			)
			if podListErr != nil {
				cleanupErrs = append(cleanupErrs, fmt.Sprintf("list Pods: %s", podListErr.Error()))
			} else {
				for _, pod := range podList.Items {
					if pod.Status.Phase != corev1.PodFailed && pod.Status.Phase != corev1.PodSucceeded {
						continue
					}
					if err := cs.CoreV1().Pods(namespace).Delete(
						cleanCtx, pod.Name, metav1.DeleteOptions{},
					); err != nil {
						if !k8serrors.IsNotFound(err) {
							cleanupErrs = append(cleanupErrs, fmt.Sprintf("Pod/%s: %s", pod.Name, err.Error()))
						}
					} else {
						deleted++
					}
				}
			}
		} // end of context-check else block

		switch {
		case len(cleanupErrs) == 0:
			s.emit(s.provider.Ctx(), "helm:cleanup:complete", map[string]any{
				"releaseName": releaseName,
				"deleted":     deleted,
			})
		case deleted > 0:
			s.emit(s.provider.Ctx(), "helm:cleanup:partial", map[string]any{
				"releaseName": releaseName,
				"deleted":     deleted,
				"errors":      cleanupErrs,
			})
		default:
			s.emit(s.provider.Ctx(), "helm:cleanup:error", map[string]any{
				"releaseName": releaseName,
				"error":       strings.Join(cleanupErrs, "; "),
			})
		}
	}()

	return nil
}

// GetHelmReleaseByName returns detailed metadata for a single helm release.
func (s *Service) GetHelmReleaseByName(namespace, releaseName string) (*dto.HelmReleaseDetail, error) {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return nil, fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return nil, fmt.Errorf("helm: no REST config for active context")
	}

	// Set up helm configuration wired to the active cluster context.
	getter := &helmrest.Getter{
		RC:        rc,
		Rules:     kube.LoadingRules(kubeconfigPaths),
		Overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return nil, fmt.Errorf("helm: init configuration: %w", err)
	}

	// Create and run get action
	client := action.NewGet(cfg)
	rel, err := client.Run(releaseName)
	if err != nil {
		return nil, fmt.Errorf("helm: get release: %w", err)
	}

	var appVersion, chartVersion string
	if rel.Chart != nil && rel.Chart.Metadata != nil {
		appVersion = rel.Chart.Metadata.AppVersion
		chartVersion = rel.Chart.Metadata.Version
	}

	// Merge chart defaults with user-supplied overrides so the viewer shows all values.
	valuesYAML := mergedValuesYAML(rel)

	// Resolve the chart's source repository from the release label.
	// The label is set at install/upgrade time; pre-existing releases will have "" until re-upgraded.
	repository := ""
	if rel.Labels != nil {
		if repo, ok := rel.Labels[HelmRepositoryLabel]; ok && repo != "" {
			repository = repo
		}
	}

	return &dto.HelmReleaseDetail{
		Name:         rel.Name,
		Namespace:    rel.Namespace,
		Chart:        rel.Chart.Metadata.Name,
		ChartVersion: chartVersion,
		AppVersion:   appVersion,
		Status:       rel.Info.Status.String(),
		Revision:     rel.Version,
		Updated:      helmAge(rel.Info.LastDeployed.Time),
		UpdatedAt:    rel.Info.LastDeployed.Time.Format(time.RFC3339),
		Notes:        rel.Info.Notes,
		ValuesYAML:   valuesYAML,
		Resources:    parseManifestResources(rel.Manifest, rel.Namespace),
		Repository:   repository,
	}, nil
}

// GetHelmReleaseHistory returns revision history for a release, sorted newest-first.
func (s *Service) GetHelmReleaseHistory(namespace, releaseName string) ([]dto.HelmReleaseRevisionHistory, error) {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return []dto.HelmReleaseRevisionHistory{}, fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return []dto.HelmReleaseRevisionHistory{}, fmt.Errorf("helm: no REST config for active context")
	}

	// Set up helm configuration wired to the active cluster context.
	getter := &helmrest.Getter{
		RC:        rc,
		Rules:     kube.LoadingRules(kubeconfigPaths),
		Overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return []dto.HelmReleaseRevisionHistory{}, fmt.Errorf("helm: init configuration: %w", err)
	}

	// Get release history
	historyAction := action.NewHistory(cfg)
	releases, err := historyAction.Run(releaseName)
	if err != nil {
		return []dto.HelmReleaseRevisionHistory{}, fmt.Errorf("helm: get history: %w", err)
	}

	// Map releases to revision history DTOs
	result := make([]dto.HelmReleaseRevisionHistory, 0, len(releases))
	for _, rel := range releases {
		if rel == nil {
			continue
		}

		chartVersion := ""
		appVersion := ""
		if rel.Chart != nil && rel.Chart.Metadata != nil {
			chartVersion = rel.Chart.Metadata.Version
			appVersion = rel.Chart.Metadata.AppVersion
		}

		updatedAt := ""
		if rel.Info != nil && !rel.Info.LastDeployed.IsZero() {
			updatedAt = rel.Info.LastDeployed.Format(time.RFC3339)
		}

		status := ""
		if rel.Info != nil {
			status = rel.Info.Status.String()
		}

		result = append(result, dto.HelmReleaseRevisionHistory{
			Revision:     rel.Version,
			ChartVersion: chartVersion,
			AppVersion:   appVersion,
			Status:       status,
			UpdatedAt:    updatedAt,
		})
	}

	// Sort descending by Revision (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Revision > result[j].Revision
	})

	return result, nil
}

// RollbackHelmRelease rolls back a release to a previous revision, synchronously.
func (s *Service) RollbackHelmRelease(namespace, releaseName string, revision int) error {
	cs, rc, activeCtx, kubeconfigPaths := s.provider.ActiveClients()
	if cs == nil {
		return fmt.Errorf("helm: no active kubernetes context")
	}
	if rc == nil {
		return fmt.Errorf("helm: no REST config for active context")
	}

	// Set up helm configuration wired to the active cluster context.
	getter := &helmrest.Getter{
		RC:        rc,
		Rules:     kube.LoadingRules(kubeconfigPaths),
		Overrides: &clientcmd.ConfigOverrides{CurrentContext: activeCtx},
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secrets", func(string, ...any) {}); err != nil {
		return fmt.Errorf("helm: init configuration: %w", err)
	}

	// Create and run rollback action
	rollback := action.NewRollback(cfg)
	rollback.Version = revision

	err := rollback.Run(releaseName)
	if err != nil {
		return fmt.Errorf("helm: rollback: %w", err)
	}

	return nil
}
