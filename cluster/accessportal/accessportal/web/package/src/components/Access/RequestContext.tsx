import * as AccessP from "@/apis/accessv1/accessv1";
import * as CoreP from "@/apis/corev1/corev1";
import * as MetaP from "@/apis/metav1/metav1";
import * as UserP from "@/apis/userv1/userv1";
import { useQuery } from "@tanstack/react-query";
import { Clock3, Globe2, Network, ShieldCheck } from "lucide-react";

import {
  Avatar,
  Badge,
  Card,
  KeyValue,
  SectionTitle,
} from "../../ui";
import {
  durationToParts,
  namespaceFromName,
  requestResourceLabel,
  requestSubjectName,
  serviceModeMeta,
  shortName,
  urgencyMeta,
  userTypeMeta,
} from "../../utils";
import { getUserClient, getUserMainClient } from "../../utils/client";

const RequestContext = (props: {
  request: AccessP.Request;
  heading?: string;
}) => {
  const { request } = props;
  const resource = requestResourceLabel(request);
  const serviceName = resource.kind === "Service" ? resource.name : "";
  const serviceNamespace = namespaceFromName(serviceName);
  const subjectName = requestSubjectName(request);
  const subjectQuery = useQuery({
    queryKey: ["access", "getSubjectUser", subjectName],
    enabled: !!subjectName,
    queryFn: async () => {
      const { response } = await getUserClient().getSubjectUser({
        userRef: MetaP.ObjectReference.create({ name: subjectName }),
      });
      return response;
    },
  });
  const serviceQuery = useQuery({
    queryKey: ["userapi", "getService", serviceName],
    enabled: !!serviceName,
    queryFn: async () => {
      let page = 0;
      for (;;) {
        const { response } = await getUserMainClient().listService(
          UserP.ListServiceOptions.create({
            common: { page, itemsPerPage: 500 },
            namespace: serviceNamespace,
          }),
        );
        const service = response.items.find(
          (item) => item.metadata?.name === serviceName,
        );
        if (service) return service;
        if (!response.listResponseMeta?.hasMore || page > 1000) return undefined;
        page += 1;
      }
    },
  });

  const service = serviceQuery.data;
  const mode = serviceModeMeta(service?.spec?.type);
  const urgency = urgencyMeta(request.spec?.urgency);
  const duration = durationToParts(request.spec?.duration);
  const subject = subjectQuery.data;
  const recipientLabel = subjectName
    ? subject?.displayName || shortName(subjectName)
    : "Requester (self)";
  const recipientType = subject ? userTypeMeta(subject.type as CoreP.User_Spec_Type) : undefined;
  const requester = shortName(request.status?.userRef?.name);

  return (
    <Card className="p-5">
      <SectionTitle>{props.heading ?? "Access details"}</SectionTitle>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <KeyValue label="Resource">
          <div className="flex min-w-0 items-center gap-2">
            {resource.kind === "Service" ? (
              <Network size={14} className="shrink-0 text-slate-400" />
            ) : (
              <Globe2 size={14} className="shrink-0 text-slate-400" />
            )}
            <span className="truncate font-mono">{resource.name || "—"}</span>
            <Badge tone="slate">{resource.kind}</Badge>
          </div>
        </KeyValue>
        <KeyValue label="Namespace">
          <span className="font-mono">
            {service?.status?.namespace || serviceNamespace || "—"}
          </span>
        </KeyValue>

        <KeyValue label="Access recipient">
          <div className="flex items-center gap-2">
            {subject ? (
              <Avatar src={subject.picURL} name={recipientLabel} size="sm" />
            ) : (
              <Avatar name={recipientLabel} size="sm" />
            )}
            <div className="min-w-0">
              <p className="truncate">{recipientLabel}</p>
              {subject?.email && (
                <p className="truncate text-[0.68rem] font-medium text-slate-400">
                  {subject.email}
                </p>
              )}
            </div>
            {recipientType && <Badge tone={recipientType.tone}>{recipientType.label}</Badge>}
          </div>
        </KeyValue>
        <KeyValue label="Requester">
          <span className="font-mono">{requester || "—"}</span>
        </KeyValue>

        {resource.kind === "Service" && (
          <>
            <KeyValue label="Service type">
              {serviceQuery.isLoading ? (
                <span className="text-slate-400">Loading…</span>
              ) : (
                <Badge tone={mode.tone}>{mode.label}</Badge>
              )}
            </KeyValue>
            <KeyValue label="Connection">
              <span className="break-all font-mono">
                {service?.status?.primaryHostname ||
                  service?.status?.addresses.join(", ") ||
                  "—"}
              </span>
            </KeyValue>
            <KeyValue label="Port">
              {service?.spec?.port || "—"}
            </KeyValue>
            <KeyValue label="Exposure">
              <div className="flex flex-wrap gap-1.5">
                <Badge tone={service?.spec?.isTLS ? "emerald" : "slate"}>
                  {service?.spec?.isTLS ? "TLS" : "Plain"}
                </Badge>
                <Badge tone={service?.spec?.isPublic ? "amber" : "slate"}>
                  {service?.spec?.isPublic ? "Public" : "Private"}
                </Badge>
              </div>
            </KeyValue>
          </>
        )}

        <KeyValue label="Urgency">
          <Badge tone={urgency.tone}>{urgency.label}</Badge>
        </KeyValue>
        <KeyValue label="Duration">
          <div className="flex items-center gap-1.5">
            <Clock3 size={13} className="text-slate-400" />
            {duration.amount} {duration.unit}
          </div>
        </KeyValue>
        {request.spec?.deadline && (
          <KeyValue label="Deadline">
            <span className="font-mono">
              {new Date(
                request.spec.deadline.seconds * 1000 +
                  request.spec.deadline.nanos / 1_000_000,
              ).toLocaleString()}
            </span>
          </KeyValue>
        )}
        {request.spec?.justification && (
          <KeyValue label="Justification" full>
            <span className="whitespace-pre-wrap font-normal text-slate-600">
              {request.spec.justification}
            </span>
          </KeyValue>
        )}
      </div>

      {resource.kind === "Catalog" && (
        <div className="mt-4 flex items-start gap-2 rounded-lg border border-blue-100 bg-blue-50/60 px-3 py-2.5 text-[0.72rem] font-medium text-blue-700">
          <ShieldCheck size={14} className="mt-0.5 shrink-0" />
          This request grants access to the services collected by this Catalog.
        </div>
      )}
    </Card>
  );
};

export default RequestContext;
