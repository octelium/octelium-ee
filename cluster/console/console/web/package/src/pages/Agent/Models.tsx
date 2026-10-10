import type {
  AgentInfo,
  AuthProvider,
  LoginEvent,
  LoginSession,
  ModelInfo,
} from "@/apis/consoleagent/protocol";
import {
  Badge,
  Button,
  Combobox,
  Drawer,
  Loader,
  PasswordInput,
  Select,
  TextInput,
  useCombobox,
} from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Brain,
  Check,
  ChevronDown,
  CircleAlert,
  CircleCheck,
  ExternalLink,
  Image as ImageIcon,
  KeyRound,
  Sparkles,
} from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import { AgentClient } from "./client";
import { formatCount, matchesTerms, searchTerms } from "./utils";

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

const useModels = (client: AgentClient, enabled: boolean) => {
  const queryClient = useQueryClient();
  const modelsKey = ["agent", client.baseUrl, "models"];

  const query = useQuery({
    queryKey: modelsKey,
    queryFn: () => client.listModels(),
    enabled,
  });

  const mutation = useMutation({
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

  return { query, mutation };
};

const ModelOption = (props: { model: ModelInfo; selected: boolean }) => {
  const { model } = props;
  return (
    <div className="flex items-center gap-2.5">
      <div className="min-w-0 flex-1">
        <p className="truncate text-body font-semibold">
          {model.name ?? model.id}
        </p>
        <p className="flex items-center gap-1.5 truncate text-micro opacity-70">
          <span className="truncate font-mono">{model.id}</span>
          {model.contextWindow ? (
            <span className="shrink-0">
              · {formatCount(model.contextWindow)}
            </span>
          ) : null}
        </p>
      </div>
      {model.reasoning && (
        <Brain
          size={12}
          className="shrink-0 opacity-60"
          aria-label="Reasoning"
        />
      )}
      {model.input?.includes("image") && (
        <ImageIcon
          size={12}
          className="shrink-0 opacity-60"
          aria-label="Images"
        />
      )}
      <Check
        size={14}
        strokeWidth={2.75}
        className={twMerge("shrink-0", props.selected ? "" : "invisible")}
      />
    </div>
  );
};

export const ModelPicker = (props: {
  client: AgentClient;
  info: AgentInfo;
  onManage: () => void;
}) => {
  const { client, info } = props;
  const [query, setQuery] = React.useState("");
  const { query: modelsQuery, mutation } = useModels(
    client,
    info.capabilities.models,
  );
  const combobox = useCombobox({
    onDropdownClose: () => {
      combobox.resetSelectedOption();
      setQuery("");
    },
    onDropdownOpen: () => combobox.focusSearchInput(),
  });

  const models = modelsQuery.data?.models ?? [];
  const current = modelsQuery.data?.current ?? info.model;
  const currentKey = current ? modelKey(current) : undefined;

  const groups = React.useMemo(() => {
    const terms = searchTerms(query);
    const ret = new Map<string, ModelInfo[]>();
    for (const model of models) {
      if (
        terms.length > 0 &&
        !matchesTerms(
          `${model.provider} ${model.id} ${model.name ?? ""}`,
          terms,
        )
      ) {
        continue;
      }
      ret.set(model.provider, [...(ret.get(model.provider) ?? []), model]);
    }
    return [...ret.entries()];
  }, [models, query]);

  if (!info.capabilities.models) {
    return (
      <span className="flex h-8 min-w-0 items-center gap-1.5 px-2 text-xs font-semibold text-slate-500">
        <Sparkles size={13} className="shrink-0" />
        <span className="truncate">{modelLabel(current)}</span>
      </span>
    );
  }

  return (
    <Combobox
      store={combobox}
      width={320}
      position="top-start"
      offset={8}
      shadow="md"
      radius="lg"
      withinPortal
      transitionProps={{ transition: "pop", duration: 150 }}
      onOptionSubmit={(value) => {
        const model = models.find((m) => modelKey(m) === value);
        if (model && value !== currentKey) {
          mutation.mutate({ provider: model.provider, id: model.id });
        }
        combobox.closeDropdown();
      }}
      styles={{
        dropdown: {
          border: "1px solid var(--color-slate-200)",
          boxShadow: "var(--shadow-overlay)",
          padding: 0,
          overflow: "hidden",
        },
      }}
    >
      <Combobox.Target withAriaAttributes={false}>
        <button
          type="button"
          aria-label="Choose the model"
          aria-haspopup="listbox"
          aria-expanded={combobox.dropdownOpened}
          onClick={() => combobox.toggleDropdown()}
          className={twMerge(
            "flex h-8 min-w-0 max-w-[220px] cursor-pointer items-center gap-1.5 rounded-full px-2.5 transition-colors duration-150 hover:bg-slate-100 hover:text-slate-900",
            current ? "text-slate-600" : "text-amber-700",
            combobox.dropdownOpened && "bg-slate-100 text-slate-900",
          )}
        >
          {mutation.isPending ? (
            <Loader size={11} color="gray" />
          ) : (
            <Sparkles size={13} className="shrink-0" />
          )}
          <span className="truncate text-xs font-semibold">
            {current ? modelLabel(current) : "Choose a model"}
          </span>
          <ChevronDown size={12} strokeWidth={2.5} className="shrink-0" />
        </button>
      </Combobox.Target>

      <Combobox.Dropdown>
        <div className="border-b border-slate-100 p-2">
          <Combobox.Search
            value={query}
            onChange={(event) => setQuery(event.currentTarget.value)}
            placeholder="Search models"
            styles={{
              input: {
                border: "none",
                background: "var(--color-slate-50)",
                borderRadius: "8px",
                minHeight: "34px",
                margin: 0,
                width: "100%",
              },
            }}
          />
        </div>
        {modelsQuery.data?.error && (
          <p className="mx-2 mt-2 flex items-start gap-1.5 rounded-lg border border-amber-200 bg-amber-50 px-2.5 py-2 text-micro text-amber-800">
            <CircleAlert size={13} className="mt-px shrink-0" />
            {modelsQuery.data.error.message}
          </p>
        )}
        <Combobox.Options className="max-h-[280px] overflow-y-auto p-1.5">
          {modelsQuery.isPending ? (
            <div className="flex justify-center py-6">
              <Loader size="xs" color="gray" />
            </div>
          ) : groups.length === 0 ? (
            <Combobox.Empty className="px-3 py-6 text-xs text-slate-500">
              {models.length === 0
                ? "No available models. Sign in to a provider first."
                : "No models match your search"}
            </Combobox.Empty>
          ) : (
            groups.map(([provider, items]) => (
              <Combobox.Group key={provider} label={provider}>
                {items.map((model) => (
                  <Combobox.Option
                    key={modelKey(model)}
                    value={modelKey(model)}
                    active={modelKey(model) === currentKey}
                  >
                    <ModelOption
                      model={model}
                      selected={modelKey(model) === currentKey}
                    />
                  </Combobox.Option>
                ))}
              </Combobox.Group>
            ))
          )}
        </Combobox.Options>
        {(current?.reasoning || info.capabilities.login) && (
          <div className="flex items-center gap-2 border-t border-slate-100 bg-slate-50/70 px-3 py-2">
            {current?.reasoning && (
              <Select
                size="xs"
                className="w-32"
                aria-label="Thinking level"
                leftSection={<Brain size={12} />}
                data={THINKING_LEVELS}
                value={modelsQuery.data?.thinkingLevel ?? null}
                disabled={mutation.isPending}
                allowDeselect={false}
                comboboxProps={{ withinPortal: false }}
                onChange={(value) =>
                  value &&
                  mutation.mutate({
                    provider: current.provider,
                    id: current.id,
                    thinkingLevel: value,
                  })
                }
              />
            )}
            {info.capabilities.login && (
              <Button
                size="compact-xs"
                variant="subtle"
                color="gray"
                className="ml-auto"
                leftSection={<KeyRound size={12} />}
                onClick={() => {
                  combobox.closeDropdown();
                  props.onManage();
                }}
              >
                Subscriptions
              </Button>
            )}
          </div>
        )}
      </Combobox.Dropdown>
    </Combobox>
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

  const providersQuery = useQuery({
    queryKey: providersKey,
    queryFn: () => client.listAuthProviders(),
    enabled: props.opened,
  });

  const logout = useMutation({
    mutationFn: (provider: string) => client.logout(provider),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: providersKey });
      void queryClient.invalidateQueries({ queryKey: modelsKey });
    },
    onError: (err: Error) => toast.error(err.message),
  });

  return (
    <Drawer
      opened={props.opened}
      onClose={props.onClose}
      position="right"
      size="md"
      overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
      title={
        <span className="flex items-center gap-2 text-sm font-bold text-slate-800">
          <KeyRound size={15} />
          Subscriptions
        </span>
      }
    >
      <div className="flex flex-col gap-4">
        <p className="text-xs leading-5 text-slate-500">
          Use your own subscription instead of the Cluster's LLM. The
          credentials are stored only inside your agent Workspace and never
          leave it.
        </p>
        <ul className="flex flex-col gap-2">
          {(providersQuery.data?.items ?? []).map((provider) => (
            <li
              key={provider.id}
              className="flex items-center gap-3 rounded-xl border border-slate-200 bg-white px-3 py-2.5 shadow-card"
            >
              <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
                <KeyRound size={14} />
              </span>
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
                  loading={logout.isPending && logout.variables === provider.id}
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
        {providersQuery.isPending && (
          <div className="flex justify-center py-4">
            <Loader size="xs" color="gray" />
          </div>
        )}
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
      </div>
    </Drawer>
  );
};

export default ModelsDrawer;
