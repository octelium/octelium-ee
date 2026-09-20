export type ErrorContext = {
  source: string;
  route?: string;
  componentStack?: string;
};

export type ErrorReport = {
  name: string;
  message: string;
  stack?: string;
  componentStack?: string;
  source: string;
  route: string;
  at: string;
  userAgent: string;
};

const currentRoute = () =>
  typeof window === "undefined"
    ? ""
    : `${window.location.pathname}${window.location.search}`;

const isRouteErrorResponse = (
  value: unknown,
): value is { status: number; statusText: string; data: unknown } =>
  typeof value === "object" &&
  value !== null &&
  "status" in value &&
  "statusText" in value;

export const describeError = (error: unknown) => {
  if (error instanceof Error) {
    return {
      name: error.name || "Error",
      message: error.message || String(error),
      stack: error.stack,
    };
  }

  if (isRouteErrorResponse(error)) {
    return {
      name: `HTTP ${error.status}`,
      message:
        error.statusText ||
        (typeof error.data === "string" ? error.data : "Route error"),
      stack: undefined,
    };
  }

  if (typeof error === "object" && error !== null) {
    const record = error as Record<string, unknown>;
    return {
      name: typeof record.name === "string" ? record.name : "Error",
      message:
        typeof record.message === "string" ? record.message : String(error),
      stack: typeof record.stack === "string" ? record.stack : undefined,
    };
  }

  return { name: "Error", message: String(error), stack: undefined };
};

export const buildErrorReport = (
  error: unknown,
  context: ErrorContext,
): ErrorReport => {
  const described = describeError(error);

  return {
    ...described,
    componentStack: context.componentStack,
    source: context.source,
    route: context.route ?? currentRoute(),
    at: new Date().toISOString(),
    userAgent:
      typeof navigator === "undefined" ? "unknown" : navigator.userAgent,
  };
};

export const formatErrorReport = (report: ErrorReport) =>
  [
    `Octelium Console error report`,
    `at:        ${report.at}`,
    `route:     ${report.route}`,
    `source:    ${report.source}`,
    `error:     ${report.name}: ${report.message}`,
    `userAgent: ${report.userAgent}`,
    ``,
    `Stack:`,
    report.stack ?? "(none)",
    ``,
    `Component stack:`,
    report.componentStack?.trim() ?? "(none)",
  ].join("\n");

export const logAppError = (error: unknown, context: ErrorContext) => {
  const report = buildErrorReport(error, context);

  console.error(
    `[octelium-console] ${report.name}: ${report.message} (${report.source} @ ${report.route})`,
    error,
  );

  if (report.componentStack) {
    console.error(
      `[octelium-console] component stack:${report.componentStack}`,
    );
  }

  return report;
};

let installed = false;

export const installGlobalErrorLogging = () => {
  if (installed || typeof window === "undefined") return;
  installed = true;

  window.addEventListener("error", (event) => {
    logAppError(event.error ?? event.message, { source: "window.error" });
  });

  window.addEventListener("unhandledrejection", (event) => {
    logAppError(event.reason, { source: "unhandledrejection" });
  });
};
