package helm

import (
	"sync"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type configCacheEntry struct {
	cfg *action.Configuration
	gvr map[string]gvrMeta
}

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
	mu               sync.RWMutex
	configCache      map[string]*configCacheEntry
	indexCache       map[string]*indexCacheEntry
	indexCacheTTL    time.Duration
	currentContext   string
}

func newCache() *cache {
	return &cache{
		configCache:   make(map[string]*configCacheEntry),
		indexCache:    make(map[string]*indexCacheEntry),
		indexCacheTTL: 10 * time.Minute,
	}
}

func (c *cache) getConfig(contextName string) (*action.Configuration, map[string]gvrMeta) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if entry, ok := c.configCache[contextName]; ok {
		return entry.cfg, entry.gvr
	}
	return nil, nil
}

func (c *cache) setConfig(contextName string, cfg *action.Configuration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configCache[contextName] = &configCacheEntry{
		cfg: cfg,
		gvr: make(map[string]gvrMeta),
	}
	c.currentContext = contextName
}

func (c *cache) setDiscovery(contextName string, gvr map[string]gvrMeta) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.configCache[contextName]; ok {
		entry.gvr = gvr
	}
}

func (c *cache) invalidateAllConfigs() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configCache = make(map[string]*configCacheEntry)
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
