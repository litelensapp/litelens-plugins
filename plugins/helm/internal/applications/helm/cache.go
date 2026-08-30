package helm

import (
	"sync"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type gvrMeta struct {
	gvr        schema.GroupVersionResource
	namespaced bool
}

type indexCacheEntry struct {
	index      *repo.IndexFile
	versionMap map[string]*repo.ChartVersion
	timestamp  time.Time
}

type cache struct {
	mu sync.RWMutex

	// configCache holds one action.Configuration per (context, namespace) pair —
	// action.Configuration.Init binds a fixed storage namespace, so entries must
	// not be shared across namespaces within the same cluster context.
	configCache map[configCacheKey]*action.Configuration

	// discoveryCache holds one server-resource discovery result per cluster
	// context. Discovery is cluster-wide (not namespace-scoped), so it is kept
	// separate from configCache rather than namespace-keyed.
	discoveryCache map[string]map[string]gvrMeta

	indexCache    map[string]*indexCacheEntry
	indexCacheTTL time.Duration
}

type configCacheKey struct {
	context   string
	namespace string
}

func newCache() *cache {
	return &cache{
		configCache:    make(map[configCacheKey]*action.Configuration),
		discoveryCache: make(map[string]map[string]gvrMeta),
		indexCache:     make(map[string]*indexCacheEntry),
		indexCacheTTL:  10 * time.Minute,
	}
}

func (c *cache) getConfig(contextName, namespace string) *action.Configuration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configCache[configCacheKey{context: contextName, namespace: namespace}]
}

func (c *cache) setConfig(contextName, namespace string, cfg *action.Configuration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configCache[configCacheKey{context: contextName, namespace: namespace}] = cfg
}

func (c *cache) getDiscovery(contextName string) map[string]gvrMeta {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.discoveryCache[contextName]
}

func (c *cache) setDiscovery(contextName string, gvr map[string]gvrMeta) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.discoveryCache[contextName] = gvr
}

func (c *cache) invalidateAllConfigs() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configCache = make(map[configCacheKey]*action.Configuration)
	c.discoveryCache = make(map[string]map[string]gvrMeta)
}

func (c *cache) getIndex(repository string) (*repo.IndexFile, map[string]*repo.ChartVersion) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if entry, ok := c.indexCache[repository]; ok {
		if time.Since(entry.timestamp) < c.indexCacheTTL {
			return entry.index, entry.versionMap
		}
	}
	return nil, nil
}

func (c *cache) setIndex(repository string, index *repo.IndexFile, versionMap map[string]*repo.ChartVersion) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.indexCache[repository] = &indexCacheEntry{
		index:      index,
		versionMap: versionMap,
		timestamp:  time.Now(),
	}
}

func (c *cache) invalidateIndex(repository string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.indexCache, repository)
}
