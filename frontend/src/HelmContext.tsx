import { createContext, FC, ReactNode, use, useMemo, useReducer } from "react";
import type { SharedNamespaceContext, SharedUnifiedTrayContext } from "@litelens/design-system";

interface HelmContextValue {
  activeContext: string;
  namespace: string;

  selectedHelmChartName: string | null;
  selectedHelmChartRepo: string | null;
  onToggleHelmChartDetail: (repo?: string, name?: string) => void;

  selectedHelmReleaseName: string | null;
  selectedHelmReleaseNamespace: string | null;
  onToggleHelmReleaseDetail: (namespace?: string, name?: string) => void;

  onNavigateToView: (view: string) => void;
  onToggleNamespaceDetail: (name?: string) => void;

  // Injected from the parent app context
  namespaces: SharedNamespaceContext[];
  unifiedTray: SharedUnifiedTrayContext | null;
  getResourceLinks: (resource: {
    kind: string;
    name: string;
    namespace?: string;
  }) => Array<{ label: string; href: string }>;
}

interface HelmState {
  selectedHelmChartName: string | null;
  selectedHelmChartRepo: string | null;
  selectedHelmReleaseName: string | null;
  selectedHelmReleaseNamespace: string | null;
}

type HelmAction =
  | { type: "toggleHelmChart"; repo?: string; name?: string }
  | { type: "toggleHelmRelease"; namespace?: string; name?: string };

const initialState: HelmState = {
  selectedHelmChartName: null,
  selectedHelmChartRepo: null,
  selectedHelmReleaseName: null,
  selectedHelmReleaseNamespace: null,
};

function helmReducer(state: HelmState, action: HelmAction): HelmState {
  switch (action.type) {
    case "toggleHelmChart":
      return {
        ...state,
        selectedHelmChartRepo: action.repo ?? null,
        selectedHelmChartName: action.name ?? null,
      };
    case "toggleHelmRelease":
      return {
        ...state,
        selectedHelmReleaseNamespace: action.namespace ?? null,
        selectedHelmReleaseName: action.name ?? null,
      };
  }
}

const HelmCtx = createContext<HelmContextValue | null>(null);

export const useHelmContext = (): HelmContextValue => {
  const ctx = use(HelmCtx);
  if (!ctx) throw new Error("useHelmContext must be used inside HelmProvider");
  return ctx;
};

interface HelmProviderProps {
  children: ReactNode;
  activeContext: string;
  namespace: string;
  onNavigateToView: (view: string) => void;
  onToggleNamespaceDetail: (name?: string) => void;
  namespaces: SharedNamespaceContext[];
  unifiedTray: SharedUnifiedTrayContext | null;
  getResourceLinks: (resource: {
    kind: string;
    name: string;
    namespace?: string;
  }) => Array<{ label: string; href: string }>;
}

export const HelmProvider: FC<HelmProviderProps> = ({
  children,
  activeContext,
  namespace,
  onNavigateToView,
  onToggleNamespaceDetail,
  namespaces,
  unifiedTray,
  getResourceLinks,
}) => {
  const [state, dispatch] = useReducer(helmReducer, initialState);

  const ctxValue = useMemo<HelmContextValue>(
    () => ({
      activeContext,
      namespace,

      selectedHelmChartName: state.selectedHelmChartName,
      selectedHelmChartRepo: state.selectedHelmChartRepo,
      onToggleHelmChartDetail: (repo, name) => dispatch({ type: "toggleHelmChart", repo, name }),

      selectedHelmReleaseName: state.selectedHelmReleaseName,
      selectedHelmReleaseNamespace: state.selectedHelmReleaseNamespace,
      onToggleHelmReleaseDetail: (namespace, name) =>
        dispatch({ type: "toggleHelmRelease", namespace, name }),

      onNavigateToView,
      onToggleNamespaceDetail,

      namespaces,
      unifiedTray,
      getResourceLinks,
    }),
    [
      state,
      activeContext,
      namespace,
      onNavigateToView,
      onToggleNamespaceDetail,
      namespaces,
      unifiedTray,
      getResourceLinks,
    ]
  );

  return <HelmCtx.Provider value={ctxValue}>{children}</HelmCtx.Provider>;
};
