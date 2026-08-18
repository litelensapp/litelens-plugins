// Package api provides HTTP handlers plus a genericclioptions.RESTClientGetter
// implementation backed by an existing rest.Config, so Helm actions can be wired to
// the active cluster context without re-reading kubeconfig from disk.
package api

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	memorycache "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// Getter implements genericclioptions.RESTClientGetter using an existing rest.Config.
type Getter struct {
	RC        *rest.Config
	Rules     *clientcmd.ClientConfigLoadingRules
	Overrides *clientcmd.ConfigOverrides
}

func (g *Getter) ToRESTConfig() (*rest.Config, error) {
	return rest.CopyConfig(g.RC), nil
}

func (g *Getter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(g.RC)
	if err != nil {
		return nil, err
	}
	return memorycache.NewMemCacheClient(dc), nil
}

func (g *Getter) ToRESTMapper() (meta.RESTMapper, error) {
	dc, err := g.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(dc), nil
}

func (g *Getter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(g.Rules, g.Overrides)
}
