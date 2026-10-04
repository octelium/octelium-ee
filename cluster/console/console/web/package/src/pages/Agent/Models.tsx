import type {
  AuthProvider,
  LoginEvent,
  LoginSession,
  ModelInfo,
} from "@/apis/consoleagent/protocol";
import {
  Badge,
  Button,
  Drawer,
  PasswordInput,
  Select,
  TextInput,
} from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CircleAlert,
  CircleCheck,
  ExternalLink,
  KeyRound,
  Sparkles,
} from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { AgentClient } from "./client";

const THINKING_LEVELS = [
  "off",
  "minimal",
  "low",
  "medium",
  "high",
  "xhigh",
  "max",
];

const modelKey = (model: ModelInfo) => `${model.provider}/${model.id}`;

export const modelLabel = (model?: ModelInfo) =>
  model ? (model.name ?? model.id) : "No model";

const lastEvent = <T extends LoginEvent["type"]>(
  session: LoginSession,
  type: T,
) =>
  session.events.filter((e) => e.type === type).at(-1) as
    Extract<LoginEvent, { type: T }> | undefined;

const LoginFlow = (props: {
  client: AgentClient;
  provider: AuthProvider;
  onDone: (session: LoginSession) => void;
  onCancel: () => void;
}) => {
  const { client, provider } = props;
  const [session, setSession] = React.useState<LoginSession>();
  const [error, setError] = React.useState<string>();
  const [value, setValue] = React.useState("");
  const [submitting, setSubmitting] = React.useState(false);
  const sessionRef = React.useRef<LoginSession | undefined>(undefined);
  const onDone = React.useRef(props.onDone);

  React.useEffect(() => {
    onDone.current = props.onDone;
  });

  React.useEffect(() => {
    let active = true;
    let timer: number | undefined;

    const update = (next: LoginSession) => {
      sessionRef.current = next;
      setSession(next);
      if (next.status !== "pending") {
        onDone.current(next);
        return;
      }
      timer = window.setTimeout(poll, 1000);
    };

    const poll = () => {
      const id = sessionRef.current?.id;
      if (!active || !id) return;
      client
        .getLogin(id)
        .then((next) => active && update(next))
        .catch((err: Error) => active && setError(err.message));
    };

    client
      .startLogin(provider.id)
      .then((next) => active && update(next))
      .catch((err: Error) => active && setError(err.message));

    return () => {
      active = false;
      window.clearTimeout(timer);
      const current = sessionRef.current;
      if (current?.status === "pending") {
        void client.cancelLogin(current.id).catch(() => undefined);
      }
    };
  }, [client, provider.id]);

  const prompt = session?.prompt;

  const answer = async (answerValue: string) => {
    if (!session || !prompt) return;
    setSubmitting(true);
    setError(undefined);
    try {
      const next = await client.answerLogin(session.id, prompt.id, answerValue);
      sessionRef.current = next;
      setSession(next);
      setValue("");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  const authURL = session ? lastEvent(session, "auth_url") : undefined;
  const deviceCode = session ? lastEvent(session, "device_code") : undefined;
  const progress = session ? lastEvent(session, "progress") : undefined;

  return (
    <div className="mt-3 rounded-xl border border-slate-200 bg-slate-50/70 px-3 py-3">
      <p className="text-xs font-semibold text-slate-700">
        Signing in with {provider.name}
      </p>

      {!session && !error && (
        <p className="mt-2 text-xs text-slate-500">Starting the sign-in…</p>
      )}

      {authURL && (
        <div className="mt-2.5">
          <Button
            component="a"
            href={authURL.url}
            target="_blank"
            rel="noreferrer noopener"
            size="xs"
            color="dark"
            leftSection={<ExternalLink size={13} />}
          >
            Open the sign-in page
          </Button>
          <p className="mt-1.5 text-xs leading-5 text-slate-600">
            Sign in on the opened page. Then copy the code that is shown to you,
            or the full URL of the page you were redirected to even if it fails
            to load, and paste it below.
          </p>
        </div>
      )}

      {deviceCode && (
        <div className="mt-2.5 text-xs text-slate-600">
          Open{" "}
          <a
            href={deviceCode.verificationUri}
            target="_blank"
            rel="noreferrer noopener"
            className="font-semibold text-blue-600"
          >
            {deviceCode.verificationUri}
          </a>{" "}
          and enter the code{" "}
          <code className="rounded bg-white px-1.5 py-0.5 font-mono font-bold text-slate-900">
            {deviceCode.userCode}
          </code>
        </div>
      )}

      {prompt && prompt.type === "select" && (
        <div className="mt-2.5">
          <p className="text-xs text-slate-600">{prompt.message}</p>
          <div className="mt-1.5 flex flex-wrap gap-1.5">
            {prompt.options?.map((option) => (
              <Button
                key={option.id}
                size="xs"
                variant="default"
                disabled={submitting}
                onClick={() => void answer(option.id)}
              >
                {option.label}
              </Button>
            ))}
          </div>
        </div>
      )}

      {prompt && prompt.type !== "select" && (
        <form
          className="mt-2.5 flex items-end gap-2"
          onSubmit={(event) => {
            event.preventDefault();
            void answer(value.trim());
          }}
        >
          {prompt.type === "secret" ? (
            <PasswordInput
              className="flex-1"
              size="xs"
              label={prompt.message}
              placeholder={prompt.placeholder}
              value={value}
              onChange={(event) => setValue(event.currentTarget.value)}
            />
          ) : (
            <TextInput
              className="flex-1"
              size="xs"
              label={prompt.message}
              placeholder={prompt.placeholder}
              value={value}
              onChange={(event) => setValue(event.currentTarget.value)}
            />
          )}
          <Button
            type="submit"
            size="xs"
            color="dark"
            loading={submitting}
            disabled={value.trim() === ""}
          >
            Submit
          </Button>
        </form>
      )}

      {progress && session?.status === "pending" && !prompt && (
        <p className="mt-2 text-xs text-slate-500">{progress.message}</p>
      )}

      {session?.status === "failed" && (
        <p className="mt-2 flex items-start gap-1.5 text-xs font-semibold text-red-700">
          <CircleAlert size={14} className="mt-0.5 shrink-0" />
          {session.error?.message ?? "The sign-in failed"}
        </p>
      )}

      {error && (
        <p className="mt-2 flex items-start gap-1.5 text-xs font-semibold text-red-700">
          <CircleAlert size={14} className="mt-0.5 shrink-0" />
          {error}
        </p>
      )}

      <div className="mt-3 flex justify-end">
        <Button
          size="compact-xs"
          variant="subtle"
          color="gray"
          onClick={props.onCancel}
        >
          {session?.status === "pending" || !session ? "Cancel" : "Close"}
        </Button>
      </div>
    </div>
  );
};

const ModelsDrawer = (props: {
  client: AgentClient;
  opened: boolean;
  onClose: () => void;
}) => {
  const { client } = props;
  const queryClient = useQueryClient();
  const [signingIn, setSigningIn] = React.useState<AuthProvider>();

  const modelsKey = ["agent", client.baseUrl, "models"];
  const providersKey = ["agent", client.baseUrl, "providers"];

  const modelsQuery = useQuery({
    queryKey: modelsKey,
    queryFn: () => client.listModels(),
    enabled: props.opened,
  });

  const providersQuery = useQuery({
    queryKey: providersKey,
    queryFn: () => client.listAuthProviders(),
    enabled: props.opened,
  });

  const setModel = useMutation({
    mutationFn: (req: {
      provider: string;
      id: string;
      thinkingLevel?: string;
    }) => client.setModel(req),
    onSuccess: (data) => {
      queryClient.setQueryData(modelsKey, data);
      void queryClient.invalidateQueries({
        queryKey: ["agent", client.baseUrl, "info"],
      });
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const logout = useMutation({
    mutationFn: (provider: string) => client.logout(provider),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: providersKey });
      void queryClient.invalidateQueries({ queryKey: modelsKey });
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const models = modelsQuery.data?.models ?? [];
  const current = modelsQuery.data?.current;

  const groups = React.useMemo(() => {
    const ret = new Map<string, { value: string; label: string }[]>();
    for (const model of models) {
      const items = ret.get(model.provider) ?? [];
      items.push({ value: modelKey(model), label: model.name ?? model.id });
      ret.set(model.provider, items);
    }
    return [...ret.entries()].map(([group, items]) => ({ group, items }));
  }, [models]);

  return (
    <Drawer
      opened={props.opened}
      onClose={props.onClose}
      position="right"
      size="md"
      title={
        <span className="flex items-center gap-2 text-sm font-bold text-slate-800">
          <Sparkles size={15} />
          Model and accounts
        </span>
      }
    >
      <div className="flex flex-col gap-6">
        <section>
          <p className="mb-2 text-micro font-semibold uppercase tracking-[0.08em] text-slate-500">
            Model
          </p>
          {modelsQuery.data?.error && (
            <p className="mb-2 flex items-start gap-1.5 rounded-lg border border-amber-200 bg-amber-50 px-2.5 py-2 text-xs text-amber-800">
              <CircleAlert size={14} className="mt-0.5 shrink-0" />
              {modelsQuery.data.error.message}
            </p>
          )}
          <Select
            size="sm"
            searchable
            placeholder={modelsQuery.isPending ? "Loading…" : "Choose a model"}
            nothingFoundMessage="No available models. Sign in to a provider first."
            data={groups}
            value={current ? modelKey(current) : null}
            disabled={setModel.isPending}
            onChange={(value) => {
              const model = models.find((m) => modelKey(m) === value);
              if (model)
                setModel.mutate({ provider: model.provider, id: model.id });
            }}
          />
          {current?.reasoning && (
            <Select
              className="mt-2"
              size="xs"
              label="Thinking level"
              data={THINKING_LEVELS}
              value={modelsQuery.data?.thinkingLevel ?? null}
              disabled={setModel.isPending}
              onChange={(value) =>
                value &&
                setModel.mutate({
                  provider: current.provider,
                  id: current.id,
                  thinkingLevel: value,
                })
              }
            />
          )}
        </section>

        <section>
          <p className="mb-1 text-micro font-semibold uppercase tracking-[0.08em] text-slate-500">
            Subscriptions
          </p>
          <p className="mb-3 text-xs leading-5 text-slate-500">
            Use your own subscription instead of the Cluster's LLM. The
            credentials are stored only inside your agent Workspace.
          </p>
          <ul className="flex flex-col gap-2">
            {(providersQuery.data?.items ?? []).map((provider) => (
              <li
                key={provider.id}
                className="flex items-center gap-2.5 rounded-xl border border-slate-200 bg-white px-3 py-2.5"
              >
                <KeyRound size={15} className="shrink-0 text-slate-500" />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-body font-semibold text-slate-800">
                    {provider.name}
                  </p>
                  {provider.configured ? (
                    <Badge
                      size="xs"
                      variant="light"
                      color="green"
                      leftSection={<CircleCheck size={10} />}
                    >
                      {provider.oauth ? "Signed in" : "Configured"}
                    </Badge>
                  ) : (
                    <Badge size="xs" variant="light" color="gray">
                      Not signed in
                    </Badge>
                  )}
                </div>
                {provider.oauth ? (
                  <Button
                    size="xs"
                    variant="default"
                    loading={
                      logout.isPending && logout.variables === provider.id
                    }
                    onClick={() => logout.mutate(provider.id)}
                  >
                    Sign out
                  </Button>
                ) : (
                  <Button
                    size="xs"
                    color="dark"
                    disabled={!!signingIn}
                    onClick={() => setSigningIn(provider)}
                  >
                    {provider.loginLabel ?? "Sign in"}
                  </Button>
                )}
              </li>
            ))}
          </ul>
          {providersQuery.data?.items.length === 0 && (
            <p className="text-xs text-slate-500">
              No subscription providers are enabled for this agent.
            </p>
          )}

          {signingIn && (
            <LoginFlow
              key={signingIn.id}
              client={client}
              provider={signingIn}
              onCancel={() => setSigningIn(undefined)}
              onDone={(session) => {
                void queryClient.invalidateQueries({ queryKey: providersKey });
                void queryClient.invalidateQueries({ queryKey: modelsKey });
                void queryClient.invalidateQueries({
                  queryKey: ["agent", client.baseUrl, "info"],
                });
                if (session.status === "completed") {
                  toast.success(
                    session.model
                      ? `Signed in. Now using ${modelLabel(session.model)}`
                      : "Signed in",
                  );
                  setSigningIn(undefined);
                }
              }}
            />
          )}
        </section>
      </div>
    </Drawer>
  );
};

export default ModelsDrawer;
