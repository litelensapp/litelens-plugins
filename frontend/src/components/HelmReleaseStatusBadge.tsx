import { Badge } from "@litelens/design-system";
import { FC } from "react";

function statusVariant(status: string) {
  switch (status.toLowerCase()) {
    case "deployed":
      return "success";
    case "pending-install":
    case "pending-upgrade":
    case "pending-rollback":
      return "warning";
    case "uninstalling":
      return "danger";
    case "failed":
      return "destructive";
    case "superseded":
      return "info";
    default:
      return "ghost";
  }
}

export const HelmReleaseStatusBadge: FC<{ status: string }> = ({ status }) => {
  return <Badge variant={statusVariant(status)}>{status}</Badge>;
};
