import { NavEntry, PackageIcon } from "@litelens/design-system";
import type { HelmViewType } from "./types";

export const HELM_NAV_ENTRY: NavEntry<HelmViewType> = {
  kind: "group",
  group: {
    id: "helm",
    label: "Helm",
    icon: PackageIcon,
    items: [
      { id: "helm-charts", label: "Charts", view: "helm-charts" },
      { id: "helm-releases", label: "Releases", view: "helm-releases" },
    ],
  },
};
