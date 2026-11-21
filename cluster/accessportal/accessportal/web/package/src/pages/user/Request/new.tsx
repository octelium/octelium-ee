import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import * as MetaP from "@/apis/metav1/metav1";
import {
  Button,
  Collapse,
  SegmentedControl,
  TextInput,
  Textarea,
  Tooltip,
} from "@mantine/core";
import { useDebouncedValue } from "@mantine/hooks";
import { notifications } from "@mantine/notifications";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  CalendarPlus,
  Check,
  ChevronDown,
  CircleAlert,
  CircleCheck,
  Layers,
  PanelRightOpen,
  Send,
  ServerCog,
  ShieldAlert,
  UserRound,
  Workflow,
} from "lucide-react";
import * as React from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { twMerge } from "@/lib/utils";

import {
  useCatalogs,
  useInbox,
  useRequestPreview,
  useService,
} from "@/components/Access/hooks";
import UrgencyPicker from "@/components/Access/UrgencyPicker";
import CatalogDetailsDrawer from "@/components/Catalog/CatalogDetailsDrawer";
import DurationInput from "@/components/DurationInput";
import TimestampPicker from "@/components/TimestampPicker";
import UnsavedChangesDialog from "@/components/UnsavedChangesDialog";
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
} from "@/ui";
import {
  MAX_JUSTIFICATION_LENGTH,
  MAX_TICKET_LENGTH,
  Tone,
  durationToSeconds,
  formatDuration,
  textLength,
  tsToMillis,
  urgencyMeta,
} from "@/utils";
import { getUserClient } from "@/utils/client";
import { isNotFoundError } from "@/utils/rpcError";
import { useAppSelector } from "@/utils/hooks";
import { useUnsavedChanges } from "@/utils/hooks/useUnsavedChanges";
import { useNow } from "@/utils/time";

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

const PREVIEW_OUTCOME: Record<
  AccessP.RequestPreview_Outcome,
  { label: string; tone: Tone } | undefined
> = {
  [AccessP.RequestPreview_Outcome.OUTCOME_UNSET]: undefined,
  [AccessP.RequestPreview_Outcome.NOT_REQUESTABLE]: {
    label: "Not requestable",
    tone: "red",
  },
  [AccessP.RequestPreview_Outcome.INCOMPLETE]: {
    label: "Incomplete",
    tone: "amber",
  },
  [AccessP.RequestPreview_Outcome.DENIED]: {
    label: "Would be denied",
    tone: "red",
  },
  [AccessP.RequestPreview_Outcome.NO_MATCH]: {
    label: "No matching rule",
    tone: "red",
  },
  [AccessP.RequestPreview_Outcome.AUTO_APPROVED]: {
    label: "Approved automatically",
    tone: "emerald",
  },
  [AccessP.RequestPreview_Outcome.REVIEW]: {
    label: "Goes to review",
    tone: "amber",
  },
};

const RequestPreviewPanel = (props: {
  preview?: AccessP.RequestPreview;
  loading: boolean;
  onPickCatalog: (catalog: AccessP.UserCatalog) => void;
}) => {
  const preview = props.preview;
  const outcome = preview ? PREVIEW_OUTCOME[preview.outcome] : undefined;
  if (!preview || !outcome) {
    return props.loading ? (
      <p className="text-xs font-normal text-slate-500">
        Checking what happens next…
      </p>
    ) : null;
  }

  const steps = preview.workflow?.steps ?? [];
  const existing = preview.existingRequests;

  return (
    <div
      aria-label="What happens next"
      className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2.5"
    >
      <div className="flex flex-wrap items-center gap-1.5">
        <Eyebrow>What happens next</Eyebrow>
        <Badge tone={outcome.tone}>{outcome.label}</Badge>
      </div>
      {preview.message && (
        <p className="text-xs font-normal leading-5 text-slate-700">
          {preview.message}
        </p>
      )}
      {steps.length > 0 && (
        <ol className="flex flex-col gap-1 text-xs font-normal text-slate-600">
          {steps.map((step, index) => (
            <li key={index} className="flex items-center gap-1.5">
              <Workflow
                size={11}
                strokeWidth={2.25}
                aria-hidden
                className="text-slate-400"
              />
              <span className="font-medium text-slate-700">
                {step.name || `Step ${index + 1}`}
              </span>
              {step.reviewers.length > 0 && (
                <span className="truncate">
                  {step.reviewers
                    .map((reviewer) => reviewer.displayName)
                    .join(", ")}
                </span>
              )}
            </li>
          ))}
        </ol>
      )}
      {preview.problems.length > 0 && (
        <ul className="flex flex-col gap-1">
          {preview.problems.map((problem, index) => (
            <li
              key={index}
              className="flex items-start gap-1.5 text-xs font-normal leading-5 text-amber-800"
            >
              <CircleAlert
                size={12}
                strokeWidth={2.25}
                aria-hidden
                className="mt-[3px] shrink-0"
              />
              {problem.message}
            </li>
          ))}
        </ul>
      )}
      {preview.candidateCatalogs.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <p className="text-xs font-normal text-slate-600">
            Request it through one of these Catalogs:
          </p>
          <div className="flex flex-wrap gap-1.5">
            {preview.candidateCatalogs.map((catalog) => (
              <Button
                key={catalog.metadata?.name}
                size="compact-xs"
                variant="default"
                onClick={() => props.onPickCatalog(catalog)}
              >
                {catalog.metadata?.displayName || catalog.metadata?.name}
              </Button>
            ))}
          </div>
        </div>
      )}
      {existing.length > 0 && (
        <ul className="flex flex-col gap-1">
          {existing.map((item) => (
            <li
              key={item.requestRef?.name}
              className="text-xs font-normal text-slate-600"
            >
              <CircleCheck
                size={11}
                strokeWidth={2.25}
                aria-hidden
                className="mr-1 inline text-slate-400"
              />
              You already have{" "}
              <Link
                to={`/user/requests/${item.requestRef?.name}`}
                className="font-medium text-slate-800 underline underline-offset-2"
              >
                {item.state === AccessP.Request_Status_State_Status.APPROVED
                  ? "an active request"
                  : "a pending request"}
              </Link>{" "}
              for this access.
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

const CatalogCard = (props: {
  title: string;
  subtitle?: string;
  serviceCount: number;
  levelCount: number;
  status?: string;
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
              {props.serviceCount} service{props.serviceCount === 1 ? "" : "s"}
            </Badge>
          )}
          {props.levelCount > 1 && (
            <Badge>{props.levelCount} access levels</Badge>
          )}
          {props.status && <Badge tone="emerald">{props.status}</Badge>}
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
  const now = useNow();
  const status = useAppSelector((state) => state.settings.status);
  const [searchParams] = useSearchParams();
  const [initialSelection] = React.useState<Selection | undefined>(() => {
    const svcName = searchParams.get("serviceRef.name");
    const catName = searchParams.get("catalogRef.name");
    if (svcName) return { kind: "service", name: svcName, label: svcName };
    if (catName) return { kind: "catalog", name: catName, label: catName };
    return undefined;
  });
  const [extendsName] = React.useState(() => searchParams.get("extends") ?? "");
  const [accessLevel, setAccessLevel] = React.useState(
    () => searchParams.get("accessLevel") ?? "",
  );
  const [ticket, setTicket] = React.useState("");
  const [startAt, setStartAt] = React.useState<Timestamp>();
  const [moreOpen, setMoreOpen] = React.useState(false);
  const [tab, setTab] = React.useState<"service" | "catalog">(
    initialSelection?.kind ?? "service",
  );
  const [catalogQuery, setCatalogQuery] = React.useState("");
  const [detailsCatalog, setDetailsCatalog] =
    React.useState<AccessP.UserCatalog>();
  const [selection, setSelection] = React.useState(initialSelection);
  const [requestFor, setRequestFor] = React.useState<"self" | "subject">("self");
  const [subject, setSubject] = React.useState<AccessP.SubjectUser | undefined>();
  const [deadline, setDeadline] = React.useState<Timestamp>();
  const [submitOpen, setSubmitOpen] = React.useState(false);
  const [urgency, setUrgency] = React.useState<AccessP.Request_Spec_Urgency>(
    AccessP.Request_Spec_Urgency.NORMAL,
  );
  const [justification, setJustification] = React.useState("");
  const [duration, setDuration] = React.useState<MetaP.Duration>(
    MetaP.Duration.create({ type: { oneofKind: "hours", hours: 1 } }),
  );
  const unsavedChanges = useUnsavedChanges(
    selection?.kind !== initialSelection?.kind ||
      selection?.name !== initialSelection?.name ||
      requestFor !== "self" ||
      !!deadline ||
      !!startAt ||
      ticket !== "" ||
      urgency !== AccessP.Request_Spec_Urgency.NORMAL ||
      justification !== "" ||
      durationToSeconds(duration) !== 3600,
  );

  const catalogsQry = useCatalogs();
  const catalogs = catalogsQry.data ?? [];
  const inboxQry = useInbox();
  const canRequestOnBehalf = !!inboxQry.data?.canRequestOnBehalf;
  const serviceQry = useService(
    selection?.kind === "service" ? selection.name : undefined,
  );

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

  const levels = selectedCatalog?.accessLevels ?? [];
  const selectedLevel =
    levels.find((level) => level.name === accessLevel) ?? levels.at(0);

  const selectionError =
    selection?.kind === "service" && serviceQry.isError
      ? isNotFoundError(serviceQry.error)
        ? `The Service ${selection.name} does not exist or is not visible to you`
        : "The selected Service could not be checked. Try again."
      : selection?.kind === "catalog" && catalogsQry.isSuccess && !selectedCatalog
        ? `The Catalog ${selection.name} does not exist or is not visible to you`
        : undefined;

  const buildSpec = (): AccessP.Request_Spec | undefined => {
    if (!selection) return undefined;

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
              catalog: AccessP.Request_Spec_Resource_Catalog.create({
                catalogRef: MetaP.ObjectReference.create({
                  name: selection.name,
                }),
                accessLevel: selectedLevel?.name ?? "",
              }),
            },
          });

    return AccessP.Request_Spec.create({
      urgency,
      justification,
      duration,
      deadline,
      startAt,
      ticket: ticket.trim(),
      resource,
      ...(extendsName
        ? { extendsRef: MetaP.ObjectReference.create({ name: extendsName }) }
        : {}),
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
    });
  };

  const draftSpec = buildSpec();
  const draftJson =
    draftSpec && !(requestFor === "subject" && !subject)
      ? AccessP.Request_Spec.toJsonString(draftSpec)
      : "";
  const [previewJson] = useDebouncedValue(draftJson, 400, { leading: true });
  const previewQry = useRequestPreview(previewJson);
  const preview = previewQry.data;
  const requirements = preview?.requirements;
  const ticketRequired = !!requirements?.ticket?.isRequired;
  const justificationRequired = !!requirements?.justification?.isRequired;
  const minJustification = requirements?.justification?.minLength ?? 0;

  const createMutation = useMutation({
    mutationFn: async () => {
      if (!selection) throw new Error("Select a resource to request access to");
      if (requestFor === "subject" && !subject) {
        throw new Error("Select the user who should receive access");
      }

      const req = AccessP.Request.create({
        apiVersion: "access/v1",
        kind: "Request",
        metadata: {},
        spec: buildSpec(),
      });

      const { response } =
        requestFor === "subject"
          ? await getUserClient().createRequestForSubject(req)
          : await getUserClient().createRequest(req);
      return response;
    },
    onSuccess: (response) => {
      notifications.show({ color: "green", message: "Access request submitted" });
      setSubmitOpen(false);
      queryClient.invalidateQueries({ queryKey: ["user", "listRequest"] });
      queryClient.invalidateQueries({ queryKey: ["user", "getInbox"] });
      unsavedChanges.allowNavigation(() =>
        navigate(`/user/requests/${response.metadata!.name}`),
      );
    },
    onError: (e) => {
      notifications.show({
        color: "red",
        message: e instanceof Error ? e.message : "Failed to submit request",
      });
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

  const justificationLength = textLength(justification);
  const tooLong = justificationLength > MAX_JUSTIFICATION_LENGTH;
  const ticketTooLong = textLength(ticket) > MAX_TICKET_LENGTH;
  const deadlinePassed =
    !!deadline && (tsToMillis(deadline) ?? 0) <= now;
  const startPassed = !!startAt && (tsToMillis(startAt) ?? 0) <= now;
  const notRequestable =
    preview?.outcome === AccessP.RequestPreview_Outcome.NOT_REQUESTABLE;
  const maxDuration = preview?.maxDuration ?? selectedLevel?.maxDuration;

  const blocker = !selection
    ? `Select a ${tab === "service" ? "Service" : "Catalog"} to request access to`
    : selectionError
      ? selectionError
      : requestFor === "subject" && !subject
        ? "Select the user who should receive the access"
        : tooLong
          ? "The justification is too long"
          : ticketTooLong
            ? "The ticket is too long"
            : deadlinePassed
              ? "The deadline is in the past. Choose a later time or clear it."
              : startPassed
                ? "The start time is in the past. Choose a later time or clear it."
                : justificationRequired && !justification.trim()
                  ? "A justification is required for this access"
                  : minJustification > 0 &&
                      justificationLength < minJustification
                    ? `The justification must be at least ${minJustification} characters`
                    : ticketRequired && !ticket.trim()
                      ? `${requirements?.ticket?.label || "A ticket"} is required for this access`
                      : notRequestable
                        ? preview?.problems.at(0)?.message ||
                          preview?.message ||
                          "This access cannot be requested"
                        : undefined;

  return (
    <div className="w-full min-w-0">
      <DocumentTitle title="New Request" />
      <UnsavedChangesDialog guard={unsavedChanges} />
      <PageHeader
        title={extendsName ? "Extend Access" : "New Request"}
        description="Request just-in-time access to a single Service or to an entire Catalog of resources."
      />

      {extendsName && (
        <Note
          tone="blue"
          icon={<CalendarPlus size={13} strokeWidth={2.25} />}
          className="mb-4"
        >
          Once approved, this request replaces{" "}
          <Link
            to={`/user/requests/${extendsName}`}
            className="font-medium underline underline-offset-2"
          >
            your current access
          </Link>{" "}
          without a gap.
        </Note>
      )}

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
                  {
                    value: "subject",
                    label: "Another user",
                    disabled: !canRequestOnBehalf || !!extendsName,
                  },
                ]}
              />

              {!canRequestOnBehalf && inboxQry.isSuccess && (
                <p className="text-xs font-normal text-slate-500">
                  You can only request access for yourself.
                </p>
              )}

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
                      {filteredCatalogs.map((c) => (
                        <CatalogCard
                          key={c.metadata!.uid || c.metadata!.name}
                          title={c.metadata!.displayName || c.metadata!.name}
                          subtitle={c.metadata!.name}
                          serviceCount={c.totalService}
                          levelCount={c.accessLevels.length}
                          status={
                            c.viewer?.activeRequestRef
                              ? "Active"
                              : c.viewer?.pendingRequestRef
                                ? "Pending"
                                : undefined
                          }
                          selected={
                            selection?.kind === "catalog" &&
                            selection.name === c.metadata!.name
                          }
                          onClick={() => {
                            setSelection({
                              kind: "catalog",
                              name: c.metadata!.name,
                              label:
                                c.metadata!.displayName || c.metadata!.name,
                            });
                            setAccessLevel("");
                          }}
                          onDetails={() => setDetailsCatalog(c)}
                        />
                      ))}
                    </div>
                  )}

                  {levels.length > 1 && (
                    <Field
                      label="Access level"
                      description={
                        selectedLevel?.description ||
                        "What the access allows once it is approved"
                      }
                    >
                      <SegmentedControl
                        fullWidth
                        aria-label="Access level"
                        value={selectedLevel?.name ?? ""}
                        onChange={setAccessLevel}
                        data={levels.map((level) => ({
                          value: level.name,
                          label: level.displayName || level.name,
                        }))}
                      />
                    </Field>
                  )}

                  {selectedLevel?.overridesGuardrails && (
                    <Note
                      tone="amber"
                      icon={<ShieldAlert size={13} strokeWidth={2.25} />}
                    >
                      {selectedLevel.displayName || selectedLevel.name} takes
                      precedence over the policies that would otherwise deny
                      this access.
                    </Note>
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
              description={
                maxDuration
                  ? `How long the access lasts once it is approved, up to ${formatDuration(maxDuration)}`
                  : "How long the access lasts once it is approved"
              }
            >
              <DurationInput value={duration} onChange={setDuration} />
            </Field>

            {ticketRequired && (
              <TextInput
                label={requirements?.ticket?.label || "Ticket"}
                description="Required for this access"
                withAsterisk
                value={ticket}
                error={ticketTooLong ? "The ticket is too long" : undefined}
                onChange={(event) => setTicket(event.currentTarget.value)}
              />
            )}

            <Textarea
              label={
                <span className="flex w-full items-baseline justify-between gap-3">
                  {justificationRequired
                    ? "Justification (required)"
                    : "Justification"}
                  <LengthCounter
                    length={justificationLength}
                    max={MAX_JUSTIFICATION_LENGTH}
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

            <div className="flex flex-col gap-3">
              <button
                type="button"
                aria-expanded={moreOpen}
                onClick={() => setMoreOpen((value) => !value)}
                className="flex items-center gap-1 self-start rounded-md text-xs font-medium text-slate-600 outline-none hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
              >
                <ChevronDown
                  size={13}
                  strokeWidth={2.25}
                  aria-hidden
                  className={twMerge(
                    "transition-transform",
                    moreOpen && "rotate-180",
                  )}
                />
                More options
              </button>
              <Collapse
                expanded={
                  moreOpen ||
                  !!deadline ||
                  !!startAt ||
                  (!ticketRequired && !!ticket)
                }
              >
                <div className="flex flex-col gap-4">
                  <TimestampPicker
                    label="Start at"
                    description="Optional. Schedule the access to start later"
                    placeholder="As soon as it is approved"
                    value={startAt}
                    isFuture
                    onChange={setStartAt}
                  />
                  <TimestampPicker
                    label="Deadline"
                    description="Optional. Expire the request if it is not decided in time"
                    placeholder="Cluster default"
                    value={deadline}
                    isFuture
                    onChange={setDeadline}
                  />
                  {!ticketRequired && (
                    <TextInput
                      label="Ticket"
                      description="Optional. A change ticket or incident reference"
                      value={ticket}
                      error={
                        ticketTooLong ? "The ticket is too long" : undefined
                      }
                      onChange={(event) => setTicket(event.currentTarget.value)}
                    />
                  )}
                </div>
              </Collapse>
            </div>

            <RequestPreviewPanel
              preview={selection ? preview : undefined}
              loading={previewQry.isFetching}
              onPickCatalog={(catalog) => {
                setTab("catalog");
                setAccessLevel("");
                setSelection({
                  kind: "catalog",
                  name: catalog.metadata!.name,
                  label:
                    catalog.metadata!.displayName || catalog.metadata!.name,
                });
              }}
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
            {selectedLevel && selection?.kind === "catalog" && (
              <SummaryRow label="Access level">
                {selectedLevel.displayName || selectedLevel.name}
              </SummaryRow>
            )}
            <SummaryRow label="For">{recipientLabel ?? "—"}</SummaryRow>
            <SummaryRow label="Duration">{formatDuration(duration)}</SummaryRow>
            <SummaryRow label="Urgency">{urgencyMeta(urgency).label}</SummaryRow>
            {startAt && (
              <SummaryRow label="Starts">
                {Timestamp.toDate(startAt).toLocaleString()}
              </SummaryRow>
            )}
            {deadline && (
              <SummaryRow label="Deadline">
                {Timestamp.toDate(deadline).toLocaleString()}
              </SummaryRow>
            )}
            {ticket.trim() && (
              <SummaryRow label="Ticket">{ticket.trim()}</SummaryRow>
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
