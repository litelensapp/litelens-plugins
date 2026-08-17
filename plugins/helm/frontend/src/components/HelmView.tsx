import type { SharedNamespaceContext, SharedUnifiedTrayContext } from "@litelens/design-system";
import { FC } from "react";
import { HelmProvider } from "../HelmContext";
import { HelmChartsView } from "./chart/HelmChartsView";
import { HelmReleasesView } from "./release/HelmReleasesView";

interface HelmViewProps {
  activeResource: string;
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

export const HelmView: FC<HelmViewProps> = ({
  activeResource,
  activeContext,
  namespace,
  onNavigateToView,
  onToggleNamespaceDetail,
  namespaces,
  unifiedTray,
  getResourceLinks,
}) => {
  return (
    <HelmProvider
      activeContext={activeContext}
      namespace={namespace}
      onNavigateToView={onNavigateToView}
      onToggleNamespaceDetail={onToggleNamespaceDetail}
      namespaces={namespaces}
      unifiedTray={unifiedTray}
      getResourceLinks={getResourceLinks}
    >
      {activeResource === "helm-charts" && <HelmChartsView />}
      {activeResource === "helm-releases" && <HelmReleasesView />}
    </HelmProvider>
  );
};
