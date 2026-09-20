import {
  buildErrorReport,
  ErrorReport,
  formatErrorReport,
  logAppError,
} from "@/utils/errors";
import { Button } from "@mantine/core";
import {
  Check,
  ChevronDown,
  Copy,
  Home,
  RefreshCw,
  RotateCcw,
  TriangleAlert,
} from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate, useRouteError } from "react-router-dom";
import { twMerge } from "tailwind-merge";

const CopyReport = (props: { report: ErrorReport }) => {
  const [copied, setCopied] = React.useState(false);

  React.useEffect(() => {
    if (!copied) return;
    const timer = window.setTimeout(() => setCopied(false), 1800);
    return () => window.clearTimeout(timer);
  }, [copied]);

  return (
    <Button
      variant="default"
      size="compact-sm"
      leftSection={
        copied ? (
          <Check size={12} strokeWidth={2.5} />
        ) : (
          <Copy size={12} strokeWidth={2.5} />
        )
      }
      onClick={() => {
        navigator.clipboard
          ?.writeText(formatErrorReport(props.report))
          .then(() => setCopied(true))
          .catch(() => setCopied(false));
      }}
    >
      {copied ? "Copied" : "Copy report"}
    </Button>
  );
};

export const ErrorScreen = (props: {
  report: ErrorReport;
  onRetry?: () => void;
}) => {
  const [expanded, setExpanded] = React.useState(false);
  const navigate = useNavigate();

  return (
    <div className="flex w-full flex-col gap-4 py-4" role="alert">
      <section className="overflow-hidden rounded-xl border border-red-200 bg-white shadow-card">
        <header className="flex items-start gap-3 border-b border-red-100 bg-red-50/60 px-4 py-4 sm:px-5">
          <span className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-red-200 bg-white text-red-600">
            <TriangleAlert size={18} strokeWidth={2.2} />
          </span>
          <div className="min-w-0 flex-1">
            <h1 className="text-base font-bold tracking-[-0.01em] text-slate-950">
              This page hit an unexpected error
            </h1>
            <p className="mt-1 text-xs font-normal leading-5 text-slate-600">
              The rest of the Console still works. The full report below is also
              printed to the browser console.
            </p>
          </div>
        </header>

        <div className="flex flex-col gap-3 px-4 py-4 sm:px-5">
          <div className="rounded-lg border border-slate-200 bg-slate-50/70 px-3.5 py-3">
            <div className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
              {props.report.name}
            </div>
            <p className="mt-1 font-mono text-xs leading-5 break-words text-slate-800">
              {props.report.message}
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {props.onRetry && (
              <Button
                variant="filled"
                color="ink"
                size="compact-sm"
                leftSection={<RotateCcw size={12} strokeWidth={2.5} />}
                onClick={props.onRetry}
              >
                Try again
              </Button>
            )}
            <Button
              variant="default"
              size="compact-sm"
              leftSection={<RefreshCw size={12} strokeWidth={2.5} />}
              onClick={() => window.location.reload()}
            >
              Reload
            </Button>
            <Button
              variant="default"
              size="compact-sm"
              leftSection={<Home size={12} strokeWidth={2.5} />}
              onClick={() => navigate("/")}
            >
              Overview
            </Button>
            <CopyReport report={props.report} />
          </div>

          <button
            type="button"
            onClick={() => setExpanded((value) => !value)}
            aria-expanded={expanded}
            className="flex w-fit cursor-pointer items-center gap-1.5 rounded px-1 py-0.5 text-micro font-semibold text-slate-500 outline-none transition-colors duration-150 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
          >
            {expanded ? "Hide details" : "Show details"}
            <ChevronDown
              size={12}
              strokeWidth={2.5}
              className={twMerge(
                "transition-transform duration-150",
                expanded && "rotate-180",
              )}
            />
          </button>

          {expanded && (
            <pre className="max-h-96 overflow-auto rounded-lg border border-slate-200 bg-slate-50/70 px-3.5 py-3 font-mono text-micro leading-5 whitespace-pre-wrap text-slate-700">
              {formatErrorReport(props.report)}
            </pre>
          )}
        </div>
      </section>
    </div>
  );
};

type BoundaryProps = {
  resetKey: string;
  children: React.ReactNode;
};

type BoundaryState = {
  report?: ErrorReport;
};

class Boundary extends React.Component<BoundaryProps, BoundaryState> {
  state: BoundaryState = {};

  static getDerivedStateFromError(error: unknown): BoundaryState {
    return { report: buildErrorReport(error, { source: "react" }) };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    this.setState({
      report: logAppError(error, {
        source: "react",
        componentStack: info.componentStack ?? undefined,
      }),
    });
  }

  componentDidUpdate(previous: BoundaryProps) {
    if (this.state.report && previous.resetKey !== this.props.resetKey) {
      this.setState({ report: undefined });
    }
  }

  render() {
    if (this.state.report) {
      return (
        <ErrorScreen
          report={this.state.report}
          onRetry={() => this.setState({ report: undefined })}
        />
      );
    }

    return this.props.children;
  }
}

export const AppErrorBoundary = (props: { children: React.ReactNode }) => {
  const location = useLocation();

  return (
    <Boundary resetKey={`${location.pathname}${location.search}`}>
      {props.children}
    </Boundary>
  );
};

export const RouteErrorBoundary = () => {
  const error = useRouteError();

  const report = React.useMemo(
    () => buildErrorReport(error, { source: "react-router" }),
    [error],
  );

  React.useEffect(() => {
    logAppError(error, { source: "react-router" });
  }, [error]);

  return (
    <div className="mx-auto w-full max-w-[var(--page-max-width)] px-4">
      <ErrorScreen report={report} />
    </div>
  );
};
