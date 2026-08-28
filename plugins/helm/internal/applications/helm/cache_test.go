package helm

import (
	"testing"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/repo"
)

// TestCache_ConfigKeyedByNamespace is a regression test for the bug where
// getOrCreateConfig cached action.Configuration per cluster context only,
// ignoring namespace. action.Configuration.Init binds a fixed storage
// namespace, so whichever namespace first populated a context's cache entry
// silently "stuck" for every later call into that context, breaking installs
// into any other namespace. The fix keys configCache by (context, namespace).
func TestCache_ConfigKeyedByNamespace(t *testing.T) {
	c := newCache()

	cfgDefault := new(action.Configuration)
	cfgKubeSystem := new(action.Configuration)

	c.setConfig("docker-desktop", "default", cfgDefault)
	c.setConfig("docker-desktop", "kube-system", cfgKubeSystem)

	if got := c.getConfig("docker-desktop", "default"); got != cfgDefault {
		t.Errorf("getConfig(ctx, default) = %p; want %p", got, cfgDefault)
	}
	if got := c.getConfig("docker-desktop", "kube-system"); got != cfgKubeSystem {
		t.Errorf("getConfig(ctx, kube-system) = %p; want %p", got, cfgKubeSystem)
	}

	// A namespace never explicitly cached must miss, not silently return
	// another namespace's configuration.
	if got := c.getConfig("docker-desktop", "other-ns"); got != nil {
		t.Errorf("getConfig(ctx, other-ns) = %p; want nil (cache miss)", got)
	}
}

// TestCache_ConfigIsolatedAcrossContexts ensures the same namespace name in
// two different cluster contexts doesn't collide.
func TestCache_ConfigIsolatedAcrossContexts(t *testing.T) {
	c := newCache()

	cfgA := new(action.Configuration)
	cfgB := new(action.Configuration)

	c.setConfig("context-a", "default", cfgA)
	c.setConfig("context-b", "default", cfgB)

	if got := c.getConfig("context-a", "default"); got != cfgA {
		t.Errorf("getConfig(context-a, default) = %p; want %p", got, cfgA)
	}
	if got := c.getConfig("context-b", "default"); got != cfgB {
		t.Errorf("getConfig(context-b, default) = %p; want %p", got, cfgB)
	}
}

// TestCache_DiscoveryKeyedByContextOnly verifies discovery results (cluster-wide
// API resource discovery) are shared across namespaces within a context, unlike
// the per-namespace config cache.
func TestCache_DiscoveryKeyedByContextOnly(t *testing.T) {
	c := newCache()

	gvrMap := map[string]gvrMeta{"Deployment": {}}
	c.setDiscovery("docker-desktop", gvrMap)

	got := c.getDiscovery("docker-desktop")
	if len(got) != 1 {
		t.Fatalf("getDiscovery returned %d entries; want 1", len(got))
	}
	if _, ok := got["Deployment"]; !ok {
		t.Errorf("getDiscovery missing expected key %q", "Deployment")
	}

	if got := c.getDiscovery("other-context"); got != nil {
		t.Errorf("getDiscovery(other-context) = %v; want nil (cache miss)", got)
	}
}

// TestCache_InvalidateAllConfigs verifies both the config and discovery caches
// are cleared, since a context switch invalidates both.
func TestCache_InvalidateAllConfigs(t *testing.T) {
	c := newCache()

	c.setConfig("ctx", "default", new(action.Configuration))
	c.setDiscovery("ctx", map[string]gvrMeta{"Pod": {}})

	c.invalidateAllConfigs()

	if got := c.getConfig("ctx", "default"); got != nil {
		t.Errorf("getConfig after invalidate = %p; want nil", got)
	}
	if got := c.getDiscovery("ctx"); got != nil {
		t.Errorf("getDiscovery after invalidate = %v; want nil", got)
	}
}

func TestCache_Index(t *testing.T) {
	c := newCache()

	if idx, vm := c.getIndex("bitnami"); idx != nil || vm != nil {
		t.Fatalf("getIndex on empty cache = (%v, %v); want (nil, nil)", idx, vm)
	}

	index := &repo.IndexFile{}
	versionMap := map[string]*repo.ChartVersion{"1.0.0": {}}
	c.setIndex("bitnami", index, versionMap)

	gotIndex, gotVersionMap := c.getIndex("bitnami")
	if gotIndex != index {
		t.Errorf("getIndex index = %p; want %p", gotIndex, index)
	}
	if len(gotVersionMap) != 1 {
		t.Errorf("getIndex versionMap len = %d; want 1", len(gotVersionMap))
	}
}
