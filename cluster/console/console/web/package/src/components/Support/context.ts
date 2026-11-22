import * as React from "react";

export const supportProbeKey = ["support", "probe"];

export const SupportAvailabilityContext = React.createContext(false);

export const useSupportAvailable = () =>
  React.useContext(SupportAvailabilityContext);

export interface SupportReport {
  subject?: string;
  message?: string;
  time?: string;
  level?: string;
  component?: { namespace?: string; type?: string; uid?: string };
}

let pendingReport: { id: string; subject?: string; message?: string } | null =
  null;

export const getSupportReport = (id: string | null) =>
  id && pendingReport?.id === id
    ? { subject: pendingReport.subject, message: pendingReport.message }
    : undefined;

export const clearSupportReport = (id: string | null) => {
  if (id && pendingReport?.id === id) pendingReport = null;
};

export const getSupportReportPath = (report: SupportReport) => {
  const params = new URLSearchParams();
  if (report.subject || report.message) {
    const id = crypto.randomUUID();
    pendingReport = {
      id,
      ...(report.subject ? { subject: report.subject.slice(0, 1000) } : {}),
      ...(report.message ? { message: report.message.slice(0, 4000) } : {}),
    };
    params.set("report", id);
  }
  if (report.time) params.set("time", report.time);
  if (report.level) params.set("level", report.level);
  if (report.component?.namespace)
    params.set("component.namespace", report.component.namespace);
  if (report.component?.type)
    params.set("component.type", report.component.type);
  if (report.component?.uid) params.set("component.uid", report.component.uid);
  const query = params.toString();
  return `/support/new${query ? `?${query}` : ""}`;
};
