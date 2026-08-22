export const HelmView = {
  HelmCharts: "helm-charts",
  HelmReleases: "helm-releases",
} as const;

export type HelmViewType = (typeof HelmView)[keyof typeof HelmView];
