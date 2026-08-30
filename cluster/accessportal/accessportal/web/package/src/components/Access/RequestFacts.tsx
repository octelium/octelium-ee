import * as React from "react";
import * as AccessP from "@/apis/accessv1/accessv1";
import { CalendarClock, FileText, Gauge, Hourglass, ScrollText, Timer } from "lucide-react";

import TimeAgo, { AbsoluteTime, Countdown, Elapsed } from "@/components/TimeAgo";
import {
  CopyValue,
  Eyebrow,
  MonoValue,
  StatusBadge,
  UrgencyBadge,
} from "@/ui";
import { formatDuration, isPendingRequest, tsToDate } from "@/utils";

const Row = (props: {
  label: string;
  icon?: React.ReactNode;
  children: React.ReactNode;
}) => (
  <div className="flex items-start justify-between gap-3 py-2.5">
    <div className="flex items-center gap-1.5 pt-[1px]">
      {props.icon}
      <Eyebrow>{props.label}</Eyebrow>
    </div>
    <div className="min-w-0 text-right text-[0.78rem] font-semibold text-slate-700">
      {props.children}
    </div>
  </div>
);

const RequestFacts = (props: {
  request: AccessP.Request;
  showPolicy?: boolean;
}) => {
  const { request } = props;
  const pending = isPendingRequest(request);
  const deadline = tsToDate(request.spec?.deadline);

  return (
    <div className="divide-y divide-slate-100">
      <Row label="Status">
        <StatusBadge status={request.status?.state?.status} withHint />
      </Row>

      <Row label="Urgency" icon={<Gauge size={11} className="text-slate-300" />}>
        <UrgencyBadge urgency={request.spec?.urgency} />
      </Row>

      <Row label="Duration" icon={<Hourglass size={11} className="text-slate-300" />}>
        {formatDuration(request.spec?.duration)}
      </Row>

      <Row label="Created" icon={<CalendarClock size={11} className="text-slate-300" />}>
        <TimeAgo rfc3339={request.status?.state?.createdAt ?? request.metadata?.createdAt} />
      </Row>

      {pending && (
        <Row label="Waiting" icon={<Timer size={11} className="text-slate-300" />}>
          <Elapsed
            rfc3339={request.status?.approvalStartAt ?? request.metadata?.createdAt}
            suffix=" so far"
          />
        </Row>
      )}

      {!pending && request.status?.approvalEndAt && (
        <Row label="Decided" icon={<CalendarClock size={11} className="text-slate-300" />}>
          <TimeAgo rfc3339={request.status.approvalEndAt} />
        </Row>
      )}

      {deadline && (
        <Row label="Deadline" icon={<Timer size={11} className="text-slate-300" />}>
          {pending ? (
            <Countdown date={deadline} suffix=" left" endedLabel="Passed" tone="amber" />
          ) : (
            <AbsoluteTime rfc3339={request.spec?.deadline} />
          )}
        </Row>
      )}

      {props.showPolicy && request.status?.policyRef?.name && (
        <Row label="Policy" icon={<ScrollText size={11} className="text-slate-300" />}>
          <MonoValue>{request.status.policyRef.name}</MonoValue>
        </Row>
      )}

      {props.showPolicy && request.status?.rule?.name && (
        <Row label="Matched rule" icon={<FileText size={11} className="text-slate-300" />}>
          <MonoValue>{request.status.rule.name}</MonoValue>
        </Row>
      )}

      <Row label="Request ID">
        <CopyValue value={request.metadata?.name ?? ""} />
      </Row>
    </div>
  );
};

export default RequestFacts;
