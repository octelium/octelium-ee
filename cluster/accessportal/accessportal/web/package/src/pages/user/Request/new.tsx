import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import * as MetaP from "@/apis/metav1/metav1";
import { Button, SegmentedControl, Textarea, Tooltip } from "@mantine/core";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
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
import { twMerge } from "tailwind-merge";

import { useCatalogs } from "@/components/Access/hooks";
import UrgencyPicker from "@/components/Access/UrgencyPicker";
import CatalogDetailsDrawer from "@/components/Catalog/CatalogDetailsDrawer";
import DurationInput from "@/components/DurationInput";
import TimestampPicker from "@/components/TimestampPicker";
import {
  Avatar,
  Badge,
  ConfirmDialog,
  EmptyState,
  ErrorState,
  Eyebrow,
  Field,
  IconTile,
  Loading,
  Note,
  PageHeader,
  SearchInput,
  SectionCard,
} from "@/ui";
import { formatDuration } from "@/utils";
import { getUserClient } from "@/utils/client";
import { useAppSelector } from "@/utils/hooks";

import ServicePicker from "./ServicePicker";
import SubjectPicker from "./SubjectPicker";

const MAX_JUSTIFICATION = 1500;

type Selection =
  | { kind: "service"; name: string; label: string }
  | { kind: "catalog"; name: string; label: string };

const StepBadge = (props: { index: number }) => (
  <span className="flex h-5 w-5 items-center justify-center rounded-full bg-slate-900 text-[0.62rem] font-bold text-white">
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
        ? "border-slate-900 bg-slate-50 shadow-[0_2px_8px_rgba(15,23,42,0.10)]"
        : "border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50",
    )}
  >
    <button
      type="button"
      onClick={props.onClick}
      className="flex min-w-0 flex-1 items-center gap-3 text-left"
    >
      <IconTile tone={props.selected ? "slate" : "violet"}>
        <Layers size={16} strokeWidth={2.2} />
      </IconTile>
      <div className="min-w-0 flex-1">
        <div className="truncate text-[0.84rem] font-bold text-slate-800">
          {props.title}
        </div>
        {props.subtitle && (
          <div className="truncate font-mono text-[0.7rem] font-semibold text-slate-400">
            {props.subtitle}
          </div>
        )}
        <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
          {props.serviceCount > 0 && (
            <Badge tone="slate">{props.serviceCount} services</Badge>
          )}
          {props.namespaceCount > 0 && (
            <Badge tone="slate">{props.namespaceCount} namespaces</Badge>
          )}
        </div>
      </div>
    </button>
    <Tooltip label="View the contents of this Catalog">
      <button
        type="button"
        onClick={props.onDetails}
        aria-label={`View details for ${props.title}`}
        className="flex h-8 shrink-0 items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 text-[0.68rem] font-bold text-slate-600 shadow-[0_1px_2px_rgba(15,23,42,0.05)] transition-[border-color,background-color,color] hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900"
      >
        <PanelRightOpen size={14} strokeWidth={2.4} />
        <span className="hidden sm:inline">Contents</span>
      </button>
    </Tooltip>
  </div>
);

const NewRequest = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const settings = useAppSelector((state) => state.settings);
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

  React.useEffect(() => {
    if (deepLinkApplied.current) return;

    const svcName = searchParams.get("serviceRef.name");
    const catName = searchParams.get("catalogRef.name");

    if (svcName) {
      deepLinkApplied.current = true;
      setTab("service");
      setSelection({ kind: "service", name: svcName, label: svcName });
    } else if (catName) {
      if (catalogsQry.isLoading) return;
      deepLinkApplied.current = true;
      const cat = catalogs.find((c) => c.metadata?.name === catName);
      setTab("catalog");
      setSelection({
        kind: "catalog",
        name: catName,
        label: cat?.metadata?.displayName || cat?.metadata?.name || catName,
      });
    }
  }, [searchParams, catalogsQry.isLoading, catalogs]);

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
    settings.status?.user?.metadata?.displayName ||
    settings.status?.user?.metadata?.name ||
    "My account";
  const recipientLabel =
    requestFor === "subject"
      ? subject?.displayName || subject?.userRef?.name
      : selfName;

  const tooLong = justification.length > MAX_JUSTIFICATION;
  const blocker = !selection
    ? `Select a ${tab === "service" ? "Service" : "Catalog"} to request access to`
    : requestFor === "subject" && !subject
      ? "Select the user who should receive the access"
      : tooLong
        ? "The justification is too long"
        : undefined;

  return (
    <div className="w-full min-w-0">
      <PageHeader
        eyebrow="Access"
        title="New Request"
        description="Request just-in-time access to a single Service or to an entire Catalog of resources."
      />

      <div className="grid min-w-0 grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div className="flex min-w-0 flex-col gap-4">
          <SectionCard
            title="Who needs the access"
            description="The user the access is granted to"
            icon={<UserRound size={14} strokeWidth={2.4} />}
            tone="blue"
            actions={<StepBadge index={1} />}
          >
            <div className="flex flex-col gap-3">
              <SegmentedControl
                fullWidth
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
                <div className="flex items-center gap-3 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2.5">
                  <Avatar
                    src={settings.status?.user?.metadata?.picURL}
                    name={selfName}
                    size="sm"
                  />
                  <div className="min-w-0">
                    <p className="truncate text-[0.8rem] font-bold text-slate-700">
                      {selfName}
                    </p>
                    <p className="truncate text-[0.68rem] font-medium text-slate-400">
                      {settings.status?.user?.spec?.email || "Your own access"}
                    </p>
                  </div>
                  <Badge tone="blue" className="ml-auto">
                    You
                  </Badge>
                </div>
              ) : (
                <SubjectPicker value={subject} onChange={setSubject} />
              )}
            </div>
          </SectionCard>

          <SectionCard
            title="What to request"
            description="Pick a single Service or a whole Catalog"
            icon={<ServerCog size={14} strokeWidth={2.4} />}
            tone="violet"
            actions={<StepBadge index={2} />}
          >
            <div className="flex flex-col gap-3">
              <SegmentedControl
                fullWidth
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
                    placeholder="Search catalogs..."
                    ariaLabel="Search catalogs"
                  />

                  {catalogsQry.isLoading ? (
                    <Loading label="Loading catalogs..." />
                  ) : catalogsQry.isError ? (
                    <ErrorState
                      title="Could not load catalogs"
                      onRetry={() => catalogsQry.refetch()}
                    />
                  ) : filteredCatalogs.length === 0 ? (
                    <EmptyState
                      icon={<Layers size={20} strokeWidth={2} />}
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
                    <div className="flex max-h-[460px] flex-col gap-2 overflow-y-auto pr-0.5">
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
          icon={<Send size={14} strokeWidth={2.4} />}
          tone="emerald"
          actions={<StepBadge index={3} />}
          className="lg:sticky lg:top-20"
        >
          <div className="flex flex-col gap-4">
            <div className="rounded-lg border border-slate-200 bg-slate-50/70 px-3 py-2.5">
              <Eyebrow>Summary</Eyebrow>
              {selection ? (
                <div className="mt-1.5 flex items-center gap-2">
                  <IconTile
                    size="sm"
                    tone={selection.kind === "catalog" ? "violet" : "blue"}
                  >
                    {selection.kind === "catalog" ? (
                      <Layers size={13} strokeWidth={2.4} />
                    ) : (
                      <ServerCog size={13} strokeWidth={2.4} />
                    )}
                  </IconTile>
                  <span className="min-w-0 flex-1 truncate text-[0.82rem] font-bold text-slate-800">
                    {selection.label}
                  </span>
                </div>
              ) : (
                <p className="mt-1.5 text-[0.74rem] font-semibold text-slate-400">
                  Nothing selected yet
                </p>
              )}
              <div className="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-[0.7rem] font-semibold text-slate-500">
                <span className="inline-flex items-center gap-1">
                  <UserRound size={11} className="text-slate-400" />
                  {recipientLabel || "No recipient"}
                </span>
                <span className="text-slate-300">•</span>
                <span>{formatDuration(duration)}</span>
              </div>
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

            <Field
              label="Justification"
              description="Reviewers decide faster with a clear reason"
              hint={
                <span
                  className={
                    tooLong
                      ? "text-[0.66rem] font-bold text-red-500"
                      : "text-[0.66rem] font-semibold text-slate-400"
                  }
                >
                  {justification.length}/{MAX_JUSTIFICATION}
                </span>
              }
            >
              <Textarea
                autosize
                minRows={3}
                maxRows={6}
                placeholder="I need access to ..."
                value={justification}
                onChange={(e) => setJustification(e.currentTarget.value)}
              />
            </Field>

            {blocker && (
              <Note tone="slate" icon={<CircleAlert size={13} strokeWidth={2.4} />}>
                {blocker}
              </Note>
            )}

            <Button
              fullWidth
              variant="filled"
              color="dark"
              leftSection={<Send size={14} strokeWidth={2.6} />}
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
          <div className="flex flex-col gap-1.5 text-[0.74rem] font-semibold text-slate-600">
            <div className="flex items-center justify-between gap-3">
              <span className="text-slate-400">Resource</span>
              <span className="truncate">{selection?.label ?? "—"}</span>
            </div>
            <div className="flex items-center justify-between gap-3">
              <span className="text-slate-400">For</span>
              <span className="truncate">{recipientLabel ?? "—"}</span>
            </div>
            <div className="flex items-center justify-between gap-3">
              <span className="text-slate-400">Duration</span>
              <span>{formatDuration(duration)}</span>
            </div>
          </div>
        }
        confirmLabel="Submit request"
        danger={false}
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
