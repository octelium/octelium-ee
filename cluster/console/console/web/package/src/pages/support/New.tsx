import {
  CreateSupportRequestRequest,
  SupportInfo_State,
  SupportRequest_Attachment_Purpose,
  SupportRequest_Priority,
} from "@/apis/enterprisev1/enterprisev1";
import Meta from "@/components/Meta";
import { PageTitleAddon } from "@/components/PageHeader/context";
import RpcError from "@/components/RpcError";
import { twMerge } from "@/lib/utils";
import { onError } from "@/utils";
import { getClientSupport } from "@/utils/client";
import { useAppSelector } from "@/utils/hooks";
import {
  Alert,
  Button,
  Collapse,
  Skeleton,
  Textarea,
  TextInput,
  UnstyledButton,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion";
import {
  AlertTriangle,
  ChevronDown,
  Info,
  Send,
  Server,
  Stethoscope,
  UserRound,
  X,
} from "lucide-react";
import * as React from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import {
  DiagnosticsButton,
  EnvironmentList,
  FileButton,
  PendingFiles,
  StateBadge,
  StateNotice,
  ToneBadge,
} from "./components";
import Diagnostics, { DiagnosticsPreset } from "./Diagnostics";
import {
  PendingFile,
  supportKeys,
  toPendingFile,
  uploadFiles,
  useSupportInfo,
} from "./queries";
import {
  checkFile,
  DESCRIPTION_TEMPLATE,
  newIdempotencyKey,
  PRIORITIES,
  supportRequestPath,
  TONE_DOT,
} from "./utils";

const Card = (props: {
  step: number;
  title: string;
  description?: string;
  children: React.ReactNode;
}) => (
  <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
    <header className="flex items-start gap-3 border-b border-slate-100 px-4 py-3.5 sm:px-5">
      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-slate-50 text-micro font-bold tabular-nums text-slate-600">
        {props.step}
      </span>
      <div className="min-w-0">
        <h2 className="text-body font-semibold text-slate-800">
          {props.title}
        </h2>
        {props.description && (
          <p className="text-micro font-normal text-slate-500">
            {props.description}
          </p>
        )}
      </div>
    </header>
    <div className="flex flex-col gap-4 px-4 py-4 sm:px-5">
      {props.children}
    </div>
  </section>
);

const PriorityPicker = (props: {
  value: SupportRequest_Priority;
  disabled?: boolean;
  onChange: (value: SupportRequest_Priority) => void;
}) => (
  <div
    role="radiogroup"
    aria-label="Priority"
    className="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-4"
  >
    {PRIORITIES.map((item) => {
      const selected = props.value === item.value;
      return (
        <UnstyledButton
          key={item.code}
          role="radio"
          aria-checked={selected}
          disabled={props.disabled}
          onClick={() => props.onChange(item.value)}
          className={twMerge(
            "flex min-w-0 flex-col gap-1 rounded-lg border px-3 py-2.5 text-left outline-none transition-[border-color,background-color,box-shadow] duration-150 focus-visible:ring-2 focus-visible:ring-slate-400 disabled:cursor-not-allowed disabled:opacity-60",
            selected
              ? "border-slate-800 bg-slate-50 shadow-[0_0_0_1px_var(--color-slate-800)]"
              : "border-slate-200 bg-white hover:border-slate-300",
          )}
        >
          <span className="flex items-center gap-2">
            <span
              aria-hidden="true"
              className={twMerge("h-2 w-2 rounded-full", TONE_DOT[item.tone])}
            />
            <span className="text-xs font-bold tabular-nums text-slate-900">
              {item.code}
            </span>
            <span className="text-xs font-semibold text-slate-700">
              {item.label}
            </span>
          </span>
          <span className="text-micro font-normal leading-[1.5] text-slate-500">
            {item.description}
          </span>
        </UnstyledButton>
      );
    })}
  </div>
);

const getPreset = (params: URLSearchParams): DiagnosticsPreset | undefined => {
  const component = {
    namespace: params.get("component.namespace") ?? undefined,
    type: params.get("component.type") ?? undefined,
    uid: params.get("component.uid") ?? undefined,
  };
  const time = params.get("time") ?? undefined;
  if (!time && !component.type) return undefined;
  return {
    ...(time && Number.isFinite(Date.parse(time)) ? { time } : {}),
    ...(params.get("level") ? { level: params.get("level")! } : {}),
    ...(component.type ? { component } : {}),
  };
};

const getInitialDescription = (params: URLSearchParams) => {
  const message = params.get("message");
  if (!message) return DESCRIPTION_TEMPLATE;
  const lines = [
    "What happened?",
    "",
    `The ${params.get("component.type") || "component"} logged the following ${(params.get("level") || "error").toLowerCase()}${params.get("time") ? ` at ${params.get("time")}` : ""}:`,
    "",
    message,
    "",
  ];
  return `${lines.join("\n")}\n${DESCRIPTION_TEMPLATE.slice(DESCRIPTION_TEMPLATE.indexOf("What did you expect"))}`;
};

const New = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const info = useSupportInfo();
  const settings = useAppSelector((state) => state.settings);
  const preset = React.useMemo(() => getPreset(searchParams), [searchParams]);

  const [priority, setPriority] = React.useState(SupportRequest_Priority.P3);
  const [subject, setSubject] = React.useState(
    () => searchParams.get("subject") ?? "",
  );
  const [description, setDescription] = React.useState(() =>
    getInitialDescription(searchParams),
  );
  const [files, setFiles] = React.useState<PendingFile[]>([]);
  const [progress, setProgress] = React.useState<
    Record<string, "uploading" | "done" | "failed">
  >({});
  const [diagnosticsOpened, setDiagnosticsOpened] = React.useState(false);
  const [environmentOpened, setEnvironmentOpened] = React.useState(false);
  const [idempotencyKey, setIdempotencyKey] = React.useState(newIdempotencyKey);
  const created = React.useRef<string | null>(null);
  const uploaded = React.useRef(new Map<string, string>());

  const limits = info.data?.limits;
  const available = info.data?.state === SupportInfo_State.AVAILABLE;
  const maxAttachments = limits?.maxAttachmentsPerMessage ?? 10;
  const maxBody = limits?.maxBodyLength ?? 16000;
  const maxSubject = limits?.maxSubjectLength ?? 200;

  const draftKey = `${priority}|${subject.trim()}|${description.trim()}`;
  const lastDraft = React.useRef(draftKey);
  React.useEffect(() => {
    if (lastDraft.current === draftKey || created.current) return;
    lastDraft.current = draftKey;
    setIdempotencyKey(newIdempotencyKey());
  }, [draftKey]);

  const addFiles = (items: PendingFile[]) => {
    const errors: string[] = [];
    const next = [...files];
    for (const item of items) {
      const error = checkFile(limits, item.file);
      if (error) {
        errors.push(error);
        continue;
      }
      if (next.some((existing) => existing.key === item.key)) continue;
      if (next.length >= maxAttachments) {
        errors.push(`A request can include at most ${maxAttachments} files`);
        break;
      }
      next.push(item);
    }
    setFiles(next);
    for (const error of new Set(errors)) {
      notifications.show({ color: "red", message: error });
    }
  };

  const mutation = useMutation({
    mutationFn: async () => {
      let id = created.current;
      if (!id) {
        const { response } = await getClientSupport().createSupportRequest(
          CreateSupportRequestRequest.create({
            subject: subject.trim(),
            description: description.trim(),
            priority,
            idempotencyKey,
          }),
        );
        id = response.id;
        created.current = id;
      }

      setProgress(
        Object.fromEntries(
          files
            .filter((item) => !uploaded.current.has(item.key))
            .map((item) => [item.key, "uploading" as const]),
        ),
      );
      const result = await uploadFiles(id, files, uploaded.current, () => {
        setProgress((current) =>
          Object.fromEntries(
            Object.entries(current).map(([key, value]) => [
              key,
              uploaded.current.has(key) ? "done" : value,
            ]),
          ),
        );
      });
      setProgress(
        Object.fromEntries(
          result.failed.map(({ item }) => [item.key, "failed" as const]),
        ),
      );
      return { id, failed: result.failed };
    },
    onSuccess: ({ id, failed }) => {
      queryClient.invalidateQueries({ queryKey: supportKeys.requests });
      if (failed.length > 0) {
        notifications.show({
          color: "red",
          title: "The request was opened, but some files were not attached",
          message: failed.map(({ item }) => item.file.name).join(", "),
        });
      } else {
        notifications.show({
          color: "green",
          message: "Your request was sent to Octelium Support",
        });
      }
      navigate(supportRequestPath(id));
    },
    onError,
  });

  const subjectError =
    subject.trim().length > 0 && subject.trim().length < 3
      ? "Use at least 3 characters"
      : subject.includes("\n")
        ? "Use a single line"
        : undefined;
  const valid =
    available &&
    subject.trim().length >= 3 &&
    !subjectError &&
    description.trim().length > 0 &&
    description.trim() !== DESCRIPTION_TEMPLATE.trim();
  const pending = mutation.isPending;
  const reporterName =
    settings.status?.user?.metadata?.displayName?.trim() ||
    settings.status?.user?.metadata?.name;
  const reporterEmail = settings.status?.user?.spec?.email?.trim();

  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: "easeOut" }}
      className="flex w-full flex-col gap-4 py-4"
    >
      <Meta title="New support request" />

      <PageTitleAddon>
        {info.data && <StateBadge state={info.data.state} />}
      </PageTitleAddon>

      {info.isError ? (
        <Alert color="red" title="Could not check the support of the Cluster">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <RpcError error={info.error} />
            <Button
              size="compact-xs"
              variant="outline"
              onClick={() => info.refetch()}
            >
              Try again
            </Button>
          </div>
        </Alert>
      ) : (
        info.data && <StateNotice info={info.data} />
      )}

      {info.isLoading ? (
        <div className="flex flex-col gap-4">
          <Skeleton height={150} radius="lg" />
          <Skeleton height={360} radius="lg" />
        </div>
      ) : (
        <form
          noValidate
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            if (valid && !pending) mutation.mutate();
          }}
        >
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,340px)]">
            <div className="flex min-w-0 flex-col gap-4">
              <Card
                step={1}
                title="Priority"
                description="How severely the issue affects your use of Octelium"
              >
                <PriorityPicker
                  value={priority}
                  disabled={!available || pending}
                  onChange={setPriority}
                />
                {priority === SupportRequest_Priority.P1 && (
                  <div className="flex items-start gap-2.5 rounded-lg border border-red-200 bg-red-50/60 px-3 py-2.5">
                    <AlertTriangle
                      size={14}
                      strokeWidth={2.4}
                      className="mt-0.5 shrink-0 text-red-600"
                    />
                    <p className="text-xs font-normal leading-5 text-red-800">
                      Use P1 only when Octelium is down or a core function is
                      unavailable without a workaround. Octelium Support may
                      change the priority with a written reason.
                    </p>
                  </div>
                )}
              </Card>

              <Card
                step={2}
                title="Describe the issue"
                description="What happens, what you expected and how to reproduce it"
              >
                <TextInput
                  label="Subject"
                  placeholder="e.g. Gateways restart after the upgrade to 0.48"
                  value={subject}
                  maxLength={maxSubject}
                  disabled={!available || pending}
                  error={subjectError}
                  onChange={(event) =>
                    setSubject(
                      event.currentTarget.value.replace(/[\r\n]+/g, " "),
                    )
                  }
                  rightSection={
                    <span className="pr-2 text-micro tabular-nums text-slate-400">
                      {subject.length}/{maxSubject}
                    </span>
                  }
                  rightSectionWidth={64}
                />
                <Textarea
                  label="Description"
                  autosize
                  minRows={10}
                  maxRows={24}
                  value={description}
                  maxLength={maxBody}
                  disabled={!available || pending}
                  onChange={(event) =>
                    setDescription(event.currentTarget.value)
                  }
                  description="Do not include passwords, tokens or other secrets."
                  styles={{ input: { lineHeight: 1.55 } }}
                />
              </Card>

              <Card
                step={3}
                title="Files and diagnostics"
                description="Logs, screenshots and the component logs of the Cluster"
              >
                {preset &&
                  files.every(
                    (item) =>
                      item.purpose !==
                      SupportRequest_Attachment_Purpose.DIAGNOSTIC,
                  ) && (
                    <div className="flex flex-col gap-2 rounded-lg border border-blue-200 bg-blue-50/60 px-3 py-2.5 sm:flex-row sm:items-center sm:justify-between">
                      <div className="flex min-w-0 items-start gap-2.5">
                        <Stethoscope
                          size={14}
                          strokeWidth={2.4}
                          className="mt-0.5 shrink-0 text-blue-600"
                        />
                        <p className="text-xs font-normal leading-5 text-blue-900">
                          Attach the component logs around the reported entry
                          {preset.component?.type
                            ? ` of ${preset.component.type}`
                            : ""}
                          .
                        </p>
                      </div>
                      <Button
                        size="compact-xs"
                        variant="default"
                        disabled={!available || pending}
                        onClick={() => setDiagnosticsOpened(true)}
                      >
                        Review and attach
                      </Button>
                    </div>
                  )}

                <div className="flex flex-wrap items-center gap-2">
                  <DiagnosticsButton
                    disabled={!available || pending}
                    onClick={() => setDiagnosticsOpened(true)}
                  />
                  <FileButton
                    disabled={!available || pending}
                    accept={limits?.attachmentExtensions}
                    onFiles={(items) =>
                      addFiles(items.map((file) => toPendingFile(file)))
                    }
                  />
                </div>

                <PendingFiles
                  files={files}
                  progress={progress}
                  disabled={pending}
                  onRemove={(key) =>
                    setFiles((current) =>
                      current.filter((item) => item.key !== key),
                    )
                  }
                />

                <p className="text-micro font-normal leading-5 text-slate-500">
                  Up to {maxAttachments} files
                  {limits?.maxAttachmentSize
                    ? ` of ${Math.floor(Number(limits.maxAttachmentSize) / 1048576)} MB each`
                    : ""}
                  . Larger files can be attached from the customer portal once
                  the request is open.
                </p>
              </Card>
            </div>

            <aside className="flex min-w-0 flex-col gap-4">
              <section className="@container overflow-hidden rounded-xl border border-slate-200 bg-white">
                <header className="flex items-center gap-3 border-b border-slate-100 px-4 py-3.5">
                  <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
                    <Info size={15} strokeWidth={2.2} />
                  </span>
                  <div className="min-w-0">
                    <h2 className="truncate text-body font-semibold text-slate-800">
                      Sent with the request
                    </h2>
                    <p className="truncate text-micro font-normal text-slate-500">
                      Added automatically to help the engineers
                    </p>
                  </div>
                </header>
                <div className="flex flex-col gap-3 px-4 py-4">
                  <div className="flex items-start gap-2.5">
                    <UserRound
                      size={13}
                      strokeWidth={2.3}
                      className="mt-0.5 shrink-0 text-slate-400"
                    />
                    <div className="min-w-0 text-xs leading-5 text-slate-600">
                      <span className="font-semibold text-slate-800">
                        {reporterName || "You"}
                      </span>{" "}
                      is recorded as the reporter.{" "}
                      {reporterEmail ? (
                        <>
                          If{" "}
                          <span className="break-all font-semibold text-slate-800">
                            {reporterEmail}
                          </span>{" "}
                          is a member of your organization in the customer
                          portal, Octelium emails a link to follow the request.
                        </>
                      ) : (
                        "Follow the request here or in the customer portal."
                      )}
                    </div>
                  </div>
                  <UnstyledButton
                    onClick={() => setEnvironmentOpened((value) => !value)}
                    aria-expanded={environmentOpened}
                    className="flex items-center justify-between gap-2 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2 text-left outline-none transition-colors hover:bg-slate-100/70 focus-visible:ring-2 focus-visible:ring-slate-400"
                  >
                    <span className="flex min-w-0 items-center gap-2 text-xs font-semibold text-slate-700">
                      <Server
                        size={13}
                        strokeWidth={2.3}
                        className="text-slate-400"
                      />
                      Cluster details
                      <ToneBadge tone="neutral">
                        {Object.keys(info.data?.environment ?? {}).length}
                      </ToneBadge>
                    </span>
                    <ChevronDown
                      size={14}
                      strokeWidth={2.4}
                      className={twMerge(
                        "shrink-0 text-slate-400 transition-transform duration-150",
                        environmentOpened && "rotate-180",
                      )}
                    />
                  </UnstyledButton>
                  <Collapse expanded={environmentOpened}>
                    <EnvironmentList environment={info.data?.environment} />
                  </Collapse>
                  <p className="text-micro font-normal leading-5 text-slate-500">
                    The request is opened for the organization of the Cluster's
                    license and is visible to its Owner and Technical members in
                    the customer portal.
                  </p>
                </div>
              </section>
            </aside>
          </div>

          <div className="sticky bottom-0 z-10 -mx-1 flex flex-col-reverse gap-2 rounded-xl border border-slate-200 bg-white/95 px-4 py-3 shadow-card backdrop-blur sm:flex-row sm:items-center sm:justify-between">
            <span className="text-micro font-normal text-slate-500">
              {files.length > 0
                ? `${files.length} file${files.length === 1 ? "" : "s"} will be attached`
                : "No files attached"}
            </span>
            <div className="flex items-center justify-end gap-2">
              <Button
                component={Link}
                to="/support"
                variant="default"
                size="sm"
                leftSection={<X size={13} strokeWidth={2.5} />}
                disabled={pending}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant="filled"
                color="ink"
                size="sm"
                leftSection={<Send size={13} strokeWidth={2.5} />}
                disabled={!valid}
                loading={pending}
              >
                Send request
              </Button>
            </div>
          </div>
        </form>
      )}

      <Diagnostics
        opened={diagnosticsOpened}
        onClose={() => setDiagnosticsOpened(false)}
        onAttach={(file) =>
          addFiles([
            toPendingFile(file, SupportRequest_Attachment_Purpose.DIAGNOSTIC),
          ])
        }
        maxBytes={Number(limits?.maxAttachmentSize ?? 3 * 1048576)}
        clusterDomain={info.data?.clusterDomain}
        preset={preset}
      />
    </motion.div>
  );
};

export default New;
