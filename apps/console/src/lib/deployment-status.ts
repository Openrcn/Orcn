type StatusLikeNode = any;

/**
 * Display status for a deployment, derived from node liveness.
 *
 * The backend intentionally keeps node InfraStatus = PENDING while the agent
 * is unreachable (providers/local/client.go) and only flips deployment.Status
 * to STOPPED when nodes report a definitive stop. Until the controller marks
 * unreachable nodes OFFLINE, the UI derives "STOPPED" here: if the deployment
 * was active (READY/RUNNING) but no node reports a live status anymore.
 */
export function getDerivedDeploymentStatus(
  deployment: any,
  nodes: StatusLikeNode[],
): string {
  const nodeStatuses = (nodes || []).map((n) =>
    String(n?.InfraStatus || "").toUpperCase(),
  );
  const liveCount = nodeStatuses.filter((s) =>
    ["RUNNING", "READY", "HEALTHY"].includes(s),
  ).length;
  const depStatus = String(deployment?.Status || "").toUpperCase();

  if (nodeStatuses.length === 0) {
    if (depStatus === "RUNNING" || depStatus === "READY") return "RUNNING";
    if (depStatus === "STOPPED") return "STOPPED";
    return depStatus || "PENDING";
  }

  const allStopped = nodeStatuses.every(
    (s) =>
      s === "STOPPED" || s === "FAILED" || s === "OFFLINE" || s === "ERROR",
  );
  const allRunning = nodeStatuses.every(
    (s) => s === "RUNNING" || s === "READY" || s === "HEALTHY",
  );
  const hasError = nodeStatuses.some((s) => s === "ERROR" || s === "FAILED");

  if (allStopped) return "STOPPED";
  if (allRunning) return "RUNNING";
  if (hasError) return "ERROR";

  if (
    ["READY", "RUNNING", "PARTIAL", "SCALING"].includes(depStatus) &&
    liveCount === 0
  ) {
    return "STOPPED";
  }
  return deployment?.Status || "PENDING";
}
