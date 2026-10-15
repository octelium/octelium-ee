import * as React from "react";
import * as AccessP from "@/apis/accessv1/accessv1";

import TimeAgo, { AbsoluteTime } from "@/components/TimeAgo";
import { CopyValue, MonoValue } from "@/ui";
import {
  formatDuration,
  isEvaluatingRequest,
  isPendingRequest,
  requestCreatedAt,
} from "@/utils";

const Row = (props: { label: string; children: React.ReactNode }) => (
  <div className="flex items-start justify-between gap-3 py-2.5">
    <span className="pt-px text-xs font-normal text-slate-500">
      {props.label}
    </span>
    <div className="flex min-w-0 justify-end text-right text-xs font-medium text-slate-800">
      {props.children}
    </div>
  </div>
);

const RequestFacts = (props: {
  request: AccessP.Request;
  showPolicy?: boolean;
}) => {
  const { request } = props;
  const open = isPendingRequest(request) || isEvaluatingRequest(request);

  return (
    <div className="divide-y divide-slate-100">
      <Row label="Requested duration">
        {formatDuration(request.spec?.duration)}
      </Row>

      <Row label="Created">
        <TimeAgo rfc3339={requestCreatedAt(request)} />
      </Row>

      {!open && request.status?.approvalEndAt && (
        <Row label="Decided">
          <TimeAgo rfc3339={request.status.approvalEndAt} />
        </Row>
      )}

      {!open && request.spec?.deadline && (
        <Row label="Deadline">
          <AbsoluteTime rfc3339={request.spec.deadline} />
        </Row>
      )}

      {props.showPolicy && request.status?.policyRef?.name && (
        <Row label="Policy">
          <MonoValue title={request.status.policyRef.name}>
            {request.status.policyRef.name}
          </MonoValue>
        </Row>
      )}

      {props.showPolicy && request.status?.rule?.name && (
        <Row label="Matched rule">
          <MonoValue title={request.status.rule.name}>
            {request.status.rule.name}
          </MonoValue>
        </Row>
      )}

      <Row label="Request ID">
        <CopyValue
          value={request.metadata?.name ?? ""}
          copyLabel="Copy request ID"
        />
      </Row>
    </div>
  );
};

export default RequestFacts;
