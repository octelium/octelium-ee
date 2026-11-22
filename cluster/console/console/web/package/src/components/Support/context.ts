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

export const getSupportReportPath = (report: SupportReport) => {
  const params = new URLSearchParams();
  if (report.subject) params.set("subject", report.subject.slice(0, 200));
  if (report.message) params.set("message", report.message.slice(0, 2000));
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
