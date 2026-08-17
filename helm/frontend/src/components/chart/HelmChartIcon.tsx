import { PackageIcon, cn } from "@litelens/design-system";
import { FC, useState } from "react";

export const HelmChartIcon: FC<{ src: string; size?: string }> = ({ src, size = "size-10" }) => {
  const [errored, setErrored] = useState(false);

  if (!src || errored)
    return (
      <div
        className={cn("bg-muted flex shrink-0 items-center justify-center rounded-md", size)}
        aria-hidden
      >
        <PackageIcon className="text-muted-foreground size-5" />
      </div>
    );

  return (
    <img
      src={src}
      alt=""
      className={cn("shrink-0 rounded-md object-contain", size)}
      // Icon load may fail due to external CDN issues (TLS cert errors, stale paths from Bitnami/ArtifactHub repos); fallback placeholder is handled above.
      onError={() => setErrored(true)}
    />
  );
};
