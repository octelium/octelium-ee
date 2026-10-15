import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, SegmentedControl, Textarea, Tooltip } from "@mantine/core";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Check,
  CircleAlert,
  Layers,
  PanelRightOpen,
  Send,
  ServerCog,
  UserRound,
} from "lucide-react";
import * as React from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "@/lib/utils";

import { useCatalogs, useService } from "@/components/Access/hooks";
import UrgencyPicker from "@/components/Access/UrgencyPicker";
import CatalogDetailsDrawer from "@/components/Catalog/CatalogDetailsDrawer";
import DurationInput from "@/components/DurationInput";
import TimestampPicker from "@/components/TimestampPicker";
import {
  Avatar,
  Badge,
  ConfirmDialog,
  DocumentTitle,
  EmptyState,
  ErrorState,
  Eyebrow,
  Field,
  IconTile,
  LengthCounter,
  Note,
  PageHeader,
  SearchInput,
  SectionCard,
  SkeletonRows,
  SummaryRow,
  isNotFoundError,
} from "@/ui";
import {
  MAX_JUSTIFICATION_BYTES,
  formatDuration,
  tsToMillis,
  urgencyMeta,
  utf8Length,
} from "@/utils";
import { getUserClient } from "@/utils/client";
import { useAppSelector } from "@/utils/hooks";

import ServicePicker from "./ServicePicker";
import SubjectPicker from "./SubjectPicker";

type Selection =
  | { kind: "service"; name: string; label: string }
  | { kind: "catalog"; name: string; label: string };

const StepBadge = (props: { index: number }) => (
  <span
    aria-label={`Step ${props.index}`}
    className="flex h-5 w-5 items-center justify-center rounded-full border border-slate-300 text-micro font-semibold text-slate-600"
  >
    {props.index}
  </span>
);

const CatalogCard = (props: {
  title: string;
  subtitle?: string;
  serviceCount: number;
  namespaceCount: number;
  selected: boolean;
  onClick: () => void;
  onDetails: () => void;
}) => (
  <div
    className={twMerge(
      "flex w-full items-center gap-2 rounded-lg border px-3 py-2.5 transition-[border-color,box-shadow,background-color] duration-150",
      props.selected
        ? "border-slate-400 bg-slate-50 ring-1 ring-slate-300"
        : "border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50",
    )}
  >
    <button
      type="button"
      onClick={props.onClick}
      aria-pressed={props.selected}
      className="flex min-w-0 flex-1 items-center gap-3 rounded-md text-left outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
    >
      <div
        className={twMerge(
          "flex h-9 w-9 shrink-0 items-center justify-center rounded-lg",
          props.selected ? "bg-slate-900 text-white" : "bg-slate-100 text-slate-500",
        )}
      >
        {props.selected ? (
          <Check size={16} strokeWidth={2.75} />
        ) : (
          <Layers size={16} strokeWidth={2} />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <div className="truncate text-body font-semibold text-slate-800">
          {props.title}
        </div>
        {props.subtitle && (
          <div className="truncate font-mono text-xs font-normal text-slate-500">
            {props.subtitle}
          </div>
        )}
        <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
          {props.serviceCount > 0 && (
            <Badge>
              {props.serviceCount} listed service{props.serviceCount === 1 ? "" : "s"}
            </Badge>
          )}
          {props.namespaceCount > 0 && (
            <Badge>
              {props.namespaceCount} namespace{props.namespaceCount === 1 ? "" : "s"}
            </Badge>
          )}
        </div>
      </div>
    </button>
    <Tooltip label="View the contents of this Catalog">
      <button
        type="button"
        onClick={props.onDetails}
        aria-label={`View the contents of ${props.title}`}
        className="flex h-8 shrink-0 items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 text-xs font-medium text-slate-600 outline-none transition-[border-color,background-color,color] hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
      >
        <PanelRightOpen size={14} strokeWidth={2.25} />
        <span className="hidden sm:inline">Contents</span>
      </button>
    </Tooltip>
  </div>
);

const NewRequest = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const status = useAppSelector((state) => state.settings.status);
  const [searchParams] = useSearchParams();
  const deepLinkApplied = React.useRef(false);
  const [tab, setTab] = React.useState<"service" | "catalog">("service");
  const [catalogQuery, setCatalogQuery] = React.useState("");
  const [detailsCatalog, setDetailsCatalog] = React.useState<AccessP.Catalog>();
  const [selection, setSelection] = React.useState<Selection | undefined>();
  const [requestFor, setRequestFor] = React.useState<"self" | "subject">("self");
  const [subject, setSubject] = React.useState<AccessP.SubjectUser | undefined>();
  const [deadline, setDeadline] = React.useState<Timestamp>();
  const [submitOpen, setSubmitOpen] = React.useState(false);
  const [urgency, setUrgency] = React.useState<AccessP.Request_Spec_Urgency>(
    AccessP.Request_Spec_Urgency.NORMAL,
  );
  const [justification, setJustification] = React.useState("");
  const [duration, setDuration] = React.useState<MetaP.Duration>(
    MetaP.Duration.create({ type: { oneofKind: "hours", hours: 1 } as any }),
  );

  const catalogsQry = useCatalogs();
  const catalogs = catalogsQry.data ?? [];
  const serviceQry = useService(
    selection?.kind === "service" ? selection.name : undefined,
  );

  React.useEffect(() => {
    if (deepLinkApplied.current) return;

    const svcName = searchParams.get("serviceRef.name");
    const catName = searchParams.get("catalogRef.name");

    if (svcName) {
      deepLinkApplied.current = true;
      setTab("service");
      setSelection({ kind: "service", name: svcName, label: svcName });
    } else if (catName) {
      deepLinkApplied.current = true;
      setTab("catalog");
      setSelection({ kind: "catalog", name: catName, label: catName });
    }
  }, [searchParams]);

  const selectedCatalog =
    selection?.kind === "catalog"
      ? catalogs.find((c) => c.metadata?.name === selection.name)
      : undefined;

  const selectionLabel =
    selection?.kind === "service"
      ? serviceQry.data?.metadata?.displayName || selection.label
      : selection?.kind === "catalog"
        ? selectedCatalog?.metadata?.displayName || selection.label
        : undefined;

  const selectionError =
    selection?.kind === "service" && serviceQry.isError
      ? isNotFoundError(serviceQry.error)
        ? `The Service ${selection.name} does not exist or is not visible to you`
        : "The selected Service could not be checked. Try again."
      : selection?.kind === "catalog" && catalogsQry.isSuccess && !selectedCatalog
        ? `The Catalog ${selection.name} does not exist or is not visible to you`
        : undefined;

  const createMutation = useMutation({
    mutationFn: async () => {
      if (!selection) throw new Error("Select a resource to request access to");
      if (requestFor === "subject" && !subject) {
        throw new Error("Select the user who should receive access");
      }

      const resource: AccessP.Request_Spec_Resource =
        selection.kind === "service"
          ? AccessP.Request_Spec_Resource.create({
              type: {
                oneofKind: "serviceRef",
                serviceRef: MetaP.ObjectReference.create({
                  name: selection.name,
                }),
              },
            })
          : AccessP.Request_Spec_Resource.create({
              type: {
                oneofKind: "catalog",
                catalog: {
                  catalogRef: MetaP.ObjectReference.create({
                    name: selection.name,
                  }),
                },
              },
            });

      const req = AccessP.Request.create({
        apiVersion: "access/v1",
        kind: "Request",
        metadata: {},
        spec: {
          urgency,
          justification,
          duration,
          deadline,
          resource,
          ...(requestFor === "subject" && subject
            ? {
                subject: AccessP.Request_Spec_Subject.create({
                  type: {
                    oneofKind: "userRef",
                    userRef: MetaP.ObjectReference.create({
                      name: subject.userRef?.name,
                    }),
                  },
                }),
              }
            : {}),
        },
      });

      const { response } =
        requestFor === "subject"
          ? await getUserClient().createRequestForSubject(req)
          : await getUserClient().createRequest(req);
      return response;
    },
    onSuccess: (response) => {
      toast.success("Access request submitted");
      setSubmitOpen(false);
      queryClient.invalidateQueries({ queryKey: ["user", "listRequest"] });
      navigate(`/user/requests/${response.metadata!.name}`);
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to submit request");
    },
  });

  const cq = catalogQuery.toLowerCase().trim();
  const filteredCatalogs = catalogs.filter(
    (c) =>
      !cq ||
      c.metadata?.name.toLowerCase().includes(cq) ||
      c.metadata?.displayName?.toLowerCase().includes(cq),
  );

  const selfName =
    status?.user?.metadata?.displayName || status?.user?.metadata?.name;
  const recipientLabel =
    requestFor === "subject"
      ? subject?.displayName || subject?.userRef?.name
      : selfName || "You";

  const justificationBytes = utf8Length(justification);
  const tooLong = justificationBytes > MAX_JUSTIFICATION_BYTES;
  const deadlinePassed =
    !!deadline && (tsToMillis(deadline) ?? 0) <= Date.now();

  const blocker = !selection
    ? `Select a ${tab === "service" ? "Service" : "Catalog"} to request access to`
    : selectionError
      ? selectionError
      : requestFor === "subject" && !subject
        ? "Select the user who should receive the access"
        : tooLong
          ? "The justification is too long"
          : deadlinePassed
            ? "The deadline is in the past. Choose a later time or clear it."
            : undefined;

  return (
    <div className="w-full min-w-0">
      <DocumentTitle title="New Request" />
      <PageHeader
        title="New Request"
        description="Request just-in-time access to a single Service or to an entire Catalog of resources."
      />

      <div className="grid min-w-0 grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className="flex min-w-0 flex-col gap-4">
          <SectionCard
            title="Who needs the access"
            description="The user the access is granted to"
            icon={<UserRound size={14} strokeWidth={2.25} />}
            actions={<StepBadge index={1} />}
          >
            <div className="flex flex-col gap-3">
              <SegmentedControl
                fullWidth
                aria-label="Request access for"
                value={requestFor}
                onChange={(value) => {
                  if (value === "self") {
                    setRequestFor("self");
                    setSubject(undefined);
                  } else {
                    setRequestFor("subject");
                  }
                }}
                data={[
                  { value: "self", label: "Myself" },
                  { value: "subject", label: "Another user" },
                ]}
              />

              {requestFor === "self" ? (
                <div className="flex items-center gap-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2.5">
                  <Avatar
                    src={status?.user?.metadata?.picURL}
                    name={selfName}
                    size="sm"
                  />
                  <div className="min-w-0">
                    <p className="truncate text-body font-semibold text-slate-800">
                      {selfName || "Your account"}
                    </p>
                    <p className="truncate text-xs font-normal text-slate-500">
                      {status?.user?.spec?.email || "The access is granted to you"}
                    </p>
                  </div>
                  <Badge className="ml-auto">You</Badge>
                </div>
              ) : (
                <SubjectPicker value={subject} onChange={setSubject} />
              )}
            </div>
          </SectionCard>

          <SectionCard
            title="What to request"
            description="Pick a single Service or a whole Catalog"
            icon={<ServerCog size={14} strokeWidth={2.25} />}
            actions={<StepBadge index={2} />}
          >
            <div className="flex flex-col gap-3">
              <SegmentedControl
                fullWidth
                aria-label="Resource type"
                value={tab}
                onChange={(value) => {
                  setTab(value as "service" | "catalog");
                  setSelection(undefined);
                }}
                data={[
                  { value: "service", label: "Services" },
                  { value: "catalog", label: "Catalogs" },
                ]}
              />

              {tab === "service" ? (
                <ServicePicker
                  value={selection?.kind === "service" ? selection.name : undefined}
                  onChange={(svc) =>
                    setSelection({
                      kind: "service",
                      name: svc.metadata!.name,
                      label: svc.metadata!.displayName || svc.metadata!.name,
                    })
                  }
                />
              ) : (
                <div className="flex flex-col gap-3">
                  <SearchInput
                    value={catalogQuery}
                    onChange={setCatalogQuery}
                    placeholder="Search catalogs by name"
                    ariaLabel="Search catalogs"
                  />

                  {catalogsQry.isLoading ? (
                    <SkeletonRows rows={3} />
                  ) : catalogsQry.isError && !catalogsQry.data ? (
                    <ErrorState
                      title="Could not load catalogs"
                      error={catalogsQry.error}
                      onRetry={() => catalogsQry.refetch()}
                    />
                  ) : filteredCatalogs.length === 0 ? (
                    <EmptyState
                      icon={<Layers size={18} strokeWidth={2} />}
                      title={
                        catalogs.length ? "No matching catalogs" : "No catalogs available"
                      }
                      description={
                        catalogs.length
                          ? "Try a different search term."
                          : "There are no Catalogs you can currently request access to."
                      }
                    />
                  ) : (
                    <div className="flex max-h-[460px] flex-col gap-2 overflow-y-auto overscroll-contain pr-0.5">
                      {filteredCatalogs.map((c) => {
                        const collection = c.spec?.resourceCollection?.service;
                        return (
                          <CatalogCard
                            key={c.metadata!.uid || c.metadata!.name}
                            title={c.metadata!.displayName || c.metadata!.name}
                            subtitle={c.metadata!.name}
                            serviceCount={collection?.services.length ?? 0}
                            namespaceCount={collection?.namespaces.length ?? 0}
                            selected={
                              selection?.kind === "catalog" &&
                              selection.name === c.metadata!.name
                            }
                            onClick={() =>
                              setSelection({
                                kind: "catalog",
                                name: c.metadata!.name,
                                label:
                                  c.metadata!.displayName || c.metadata!.name,
                              })
                            }
                            onDetails={() => setDetailsCatalog(c)}
                          />
                        );
                      })}
                    </div>
                  )}
                </div>
              )}
            </div>
          </SectionCard>
        </div>

        <SectionCard
          title="Access details"
          description="How long and how urgent"
          icon={<Send size={14} strokeWidth={2.25} />}
          actions={<StepBadge index={3} />}
          className="lg:sticky lg:top-20"
        >
          <div className="flex flex-col gap-4">
            <div className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2.5">
              <Eyebrow>Summary</Eyebrow>
              {selection ? (
                <div className="mt-1.5 flex items-center gap-2">
                  <IconTile size="sm" className="bg-white">
                    {selection.kind === "catalog" ? (
                      <Layers size={13} strokeWidth={2.25} />
                    ) : (
                      <ServerCog size={13} strokeWidth={2.25} />
                    )}
                  </IconTile>
                  <span className="min-w-0 flex-1 truncate text-body font-semibold text-slate-800">
                    {selectionLabel}
                  </span>
                </div>
              ) : (
                <p className="mt-1.5 text-xs font-normal text-slate-500">
                  Nothing selected yet
                </p>
              )}
              <p className="mt-2 flex flex-wrap items-center gap-x-1.5 text-xs font-normal text-slate-600">
                <UserRound size={11} aria-hidden className="text-slate-500" />
                {recipientLabel || "No recipient"}
                <span aria-hidden className="text-slate-300">·</span>
                {formatDuration(duration)}
              </p>
            </div>

            <Field
              label="Urgency"
              description="Higher urgency helps reviewers triage the queue"
            >
              <UrgencyPicker value={urgency} onChange={setUrgency} />
            </Field>

            <Field
              label="Duration"
              description="How long the access lasts once it is approved"
            >
              <DurationInput value={duration} onChange={setDuration} />
            </Field>

            <TimestampPicker
              label="Deadline"
              description="Optional. Expire the request if it is not decided in time"
              placeholder="No deadline"
              value={deadline}
              isFuture
              onChange={setDeadline}
            />

            <Textarea
              label={
                <span className="flex w-full items-baseline justify-between gap-3">
                  Justification
                  <LengthCounter
                    length={justificationBytes}
                    max={MAX_JUSTIFICATION_BYTES}
                  />
                </span>
              }
              description="Reviewers decide faster with a clear reason"
              error={tooLong ? "The justification is too long" : undefined}
              autosize
              minRows={3}
              maxRows={6}
              placeholder="I need access to ..."
              value={justification}
              onChange={(e) => setJustification(e.currentTarget.value)}
              styles={{ label: { display: "flex", width: "100%" } }}
            />

            {blocker && (
              <Note icon={<CircleAlert size={13} strokeWidth={2.25} />}>
                {blocker}
              </Note>
            )}

            <Button
              fullWidth
              variant="filled"
              color="ink"
              leftSection={<Send size={14} strokeWidth={2.25} />}
              disabled={!!blocker}
              loading={createMutation.isPending}
              onClick={() => setSubmitOpen(true)}
            >
              Submit Request
            </Button>
          </div>
        </SectionCard>
      </div>

      <ConfirmDialog
        opened={submitOpen}
        onClose={() => setSubmitOpen(false)}
        onConfirm={() => createMutation.mutate()}
        title="Submit this access request?"
        description="The Cluster evaluates the access policies and either decides immediately or routes the request to its reviewers."
        details={
          <div className="flex flex-col">
            <SummaryRow label={selection?.kind === "catalog" ? "Catalog" : "Service"}>
              {selection?.name ?? "—"}
            </SummaryRow>
            <SummaryRow label="For">{recipientLabel ?? "—"}</SummaryRow>
            <SummaryRow label="Duration">{formatDuration(duration)}</SummaryRow>
            <SummaryRow label="Urgency">{urgencyMeta(urgency).label}</SummaryRow>
            {deadline && (
              <SummaryRow label="Deadline">
                {Timestamp.toDate(deadline).toLocaleString()}
              </SummaryRow>
            )}
            <SummaryRow label="Justification">
              {justification.trim() || "None"}
            </SummaryRow>
          </div>
        }
        confirmLabel="Submit request"
        loading={createMutation.isPending}
      />

      <CatalogDetailsDrawer
        catalog={detailsCatalog}
        opened={!!detailsCatalog}
        onClose={() => setDetailsCatalog(undefined)}
      />
    </div>
  );
};

export default NewRequest;
