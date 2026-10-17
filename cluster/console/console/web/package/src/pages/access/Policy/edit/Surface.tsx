import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { ListIntegrationOptions } from "@/apis/visibilityv1/access/vaccessv1";
import { CommonListOptions } from "@/apis/visibilityv1/meta/vmetav1";
import { twMerge } from "@/lib/utils";
import { getClientVisibilityAccess } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { ActionIcon, Button, Select, Tooltip } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  Hash,
  MousePointerClick,
  Plus,
  SquareArrowOutUpRight,
  Trash2,
  User,
  UserRoundSearch,
  Users,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Link } from "react-router-dom";
import { getProviderKind, PROVIDER_LABEL, ProviderMark } from "../../common";
import { getSharedDestination, hasCapability } from "../../Integration/utils";

const Audience = AccessP.Policy_Spec_Rule_Surface_Destination_Audience;
const MAX_INTEGRATION_CHIPS = 6;
const Capability = AccessP.Integration_Status_Capability;

export const useIntegrationChoices = () =>
  useQuery({
    queryKey: ["access", "policy-edit", "integrations"],
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) =>
      (
        await getClientVisibilityAccess().listIntegration(
          ListIntegrationOptions.create({
            common: CommonListOptions.create({ itemsPerPage: 200 }),
          }),
          { abort: signal },
        )
      ).response.items,
  });

type AudienceOption = {
  value: AccessP.Policy_Spec_Rule_Surface_Destination_Audience;
  label: string;
  description: string;
  icon: LucideIcon;
};

const SHARED_OPTION: AudienceOption = {
  value: Audience.SHARED,
  label: "Shared destination",
  description: "The Integration's channel, project or URL",
  icon: Hash,
};

const REVIEW_AUDIENCES: AudienceOption[] = [
  SHARED_OPTION,
  {
    value: Audience.REVIEWERS,
    label: "Each reviewer",
    description: "A direct message to every eligible reviewer",
    icon: Users,
  },
];

const NOTIFICATION_AUDIENCES: AudienceOption[] = [
  SHARED_OPTION,
  {
    value: Audience.REQUESTER,
    label: "Requester",
    description: "A direct message to whoever asked",
    icon: UserRoundSearch,
  },
  {
    value: Audience.SUBJECT,
    label: "Subject",
    description: "A direct message to whoever gets the access",
    icon: User,
  },
];

export const newSurface = (isReview: boolean) =>
  AccessP.Policy_Spec_Rule_Surface.create({
    destination: AccessP.Policy_Spec_Rule_Surface_Destination.create({
      integrationRef: MetaP.ObjectReference.create(),
      audience: Audience.SHARED,
    }),
    interactionMode: isReview
      ? AccessP.Policy_Spec_Rule_Surface_InteractionMode.INTERACTIVE
      : AccessP.Policy_Spec_Rule_Surface_InteractionMode.INTERACTION_MODE_UNSET,
  });

const getAudienceProblem = (
  integration: AccessP.Integration | undefined,
  audience: AccessP.Policy_Spec_Rule_Surface_Destination_Audience,
): string | undefined => {
  if (!integration) return undefined;
  if (audience === Audience.SHARED) {
    if (!hasCapability(integration, Capability.NOTIFICATION))
      return "This Integration cannot post to a shared destination";
    if (!getSharedDestination(integration))
      return "This Integration has no shared destination set";
    return undefined;
  }
  if (!hasCapability(integration, Capability.DIRECT_USER_DELIVERY))
    return "This Integration cannot message individual Users";
  return undefined;
};

const SurfaceRow = (props: {
  surface: AccessP.Policy_Spec_Rule_Surface;
  isReview: boolean;
  integrations: AccessP.Integration[];
  onUpdate: () => void;
  onRemove: () => void;
}) => {
  const { surface, isReview, integrations, onUpdate } = props;
  const destination = () => {
    if (!surface.destination)
      surface.destination = AccessP.Policy_Spec_Rule_Surface_Destination.create();
    return surface.destination;
  };
  const selectedName = surface.destination?.integrationRef?.name;
  const selected = integrations.find((x) => x.metadata?.name === selectedName);
  const audience = surface.destination?.audience ?? Audience.AUDIENCE_UNSET;
  const options = isReview ? REVIEW_AUDIENCES : NOTIFICATION_AUDIENCES;
  const problem = getAudienceProblem(selected, audience);
  const interactive =
    surface.interactionMode === AccessP.Policy_Spec_Rule_Surface_InteractionMode.INTERACTIVE;
  const interactiveUnsupported =
    interactive && !!selected && !hasCapability(selected, Capability.INTERACTIVE_REVIEW);

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-3">
      <div className="flex items-start gap-2">
        <div className="flex min-w-0 flex-1 flex-col gap-1.5">
          <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
            Integration
          </span>
          {integrations.length === 0 ? (
            <span className="text-xs text-slate-500">
              No Integration yet.{" "}
              <Link
                to="/access/integrations/create"
                className="font-semibold text-blue-700 hover:text-blue-800"
              >
                Connect Slack, Jira or a webhook
              </Link>
            </span>
          ) : integrations.length > MAX_INTEGRATION_CHIPS ? (
            <Select
              aria-label="Integration"
              placeholder="Choose an Integration"
              searchable
              nothingFoundMessage="No matching Integration"
              className="max-w-md"
              value={selected ? selectedName : null}
              data={integrations.map((integration) => {
                const provider = getProviderKind(integration);
                return {
                  value: integration.metadata?.name ?? "",
                  label: `${integration.metadata?.displayName || integration.metadata?.name} · ${provider ? PROVIDER_LABEL[provider] : "Not configured"}`,
                };
              })}
              leftSection={
                selected ? <ProviderMark provider={getProviderKind(selected)} size={13} /> : undefined
              }
              onChange={(value) => {
                const integration = integrations.find((x) => x.metadata?.name === value);
                if (!integration) return;
                destination().integrationRef = getResourceRef(integration);
                onUpdate();
              }}
            />
          ) : (
            <div role="radiogroup" aria-label="Integration" className="flex flex-wrap gap-1.5">
              {integrations.map((integration) => {
                const name = integration.metadata?.name;
                const active = name === selectedName;
                const provider = getProviderKind(integration);
                const label = integration.metadata?.displayName || name || "";
                const providerLabel = provider ? PROVIDER_LABEL[provider] : "Not configured";
                return (
                  <button
                    key={integration.metadata?.uid}
                    type="button"
                    role="radio"
                    aria-checked={active}
                    onClick={() => {
                      if (active) return;
                      destination().integrationRef = getResourceRef(integration);
                      onUpdate();
                    }}
                    className={twMerge(
                      "inline-flex cursor-pointer items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs font-medium outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
                      active
                        ? "border-slate-800 bg-slate-800 text-white"
                        : "border-slate-200 bg-white text-slate-700 hover:border-slate-300",
                      integration.spec?.isDisabled && !active && "opacity-60",
                    )}
                  >
                    <ProviderMark provider={provider} size={13} />
                    {label}
                    {!label.toLowerCase().includes(providerLabel.toLowerCase()) && (
                      <span className={twMerge("text-micro", active ? "text-slate-300" : "text-slate-500")}>
                        {providerLabel}
                      </span>
                    )}
                  </button>
                );
              })}
            </div>
          )}
          {selectedName && !selected && integrations.length > 0 && (
            <span className="flex items-center gap-1 text-micro text-amber-700">
              <AlertTriangle size={11} /> The Integration {selectedName} does not exist anymore
            </span>
          )}
        </div>
        <Tooltip label="Remove" withArrow>
          <ActionIcon variant="subtle" color="red" size="sm" aria-label="Remove delivery" onClick={props.onRemove}>
            <Trash2 size={13} />
          </ActionIcon>
        </Tooltip>
      </div>

      <div className="flex flex-col gap-1.5">
        <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
          Delivered to
        </span>
        <div
          role="radiogroup"
          aria-label="Delivered to"
          className={twMerge("grid grid-cols-1 gap-1.5", isReview ? "sm:grid-cols-2" : "sm:grid-cols-3")}
        >
          {options.map((option) => {
            const active = audience === option.value;
            const Icon = option.icon;
            const optionProblem = getAudienceProblem(selected, option.value);
            return (
              <button
                key={option.value}
                type="button"
                role="radio"
                aria-checked={active}
                onClick={() => {
                  if (active) return;
                  destination().audience = option.value;
                  onUpdate();
                }}
                className={twMerge(
                  "flex min-w-0 cursor-pointer items-start gap-2 rounded-lg border px-2.5 py-2 text-left outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
                  active
                    ? "border-slate-800 bg-slate-800 text-white"
                    : "border-slate-200 bg-white text-slate-700 hover:border-slate-300",
                )}
              >
                <Icon size={13} className={twMerge("mt-0.5 shrink-0", active ? "text-slate-300" : "text-slate-500")} />
                <span className="flex min-w-0 flex-col">
                  <span className="text-xs font-semibold">{option.label}</span>
                  <span className={twMerge("text-micro leading-4", active ? "text-slate-300" : "text-slate-500")}>
                    {optionProblem && !active ? "Not supported by this Integration" : option.description}
                  </span>
                </span>
              </button>
            );
          })}
        </div>
        {problem && (
          <span className="flex items-center gap-1 text-micro font-medium text-amber-700">
            <AlertTriangle size={11} /> {problem}. The Cluster will refuse to save it.
          </span>
        )}
        {audience !== Audience.SHARED && audience !== Audience.AUDIENCE_UNSET && (
          <span className="text-micro leading-4 text-slate-500">
            Users are messaged once the Integration links them to their
            account in the provider, by email by default.
          </span>
        )}
      </div>

      {isReview && (
        <label className="flex cursor-pointer items-start gap-2.5 rounded-lg bg-slate-50 px-2.5 py-2">
          <input
            type="checkbox"
            className="mt-0.5 h-3.5 w-3.5 cursor-pointer accent-slate-800"
            checked={interactive}
            onChange={(event) => {
              surface.interactionMode = event.target.checked
                ? AccessP.Policy_Spec_Rule_Surface_InteractionMode.INTERACTIVE
                : AccessP.Policy_Spec_Rule_Surface_InteractionMode.DEEP_LINK_ONLY;
              onUpdate();
            }}
          />
          <span className="flex min-w-0 flex-col gap-0.5">
            <span className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-700">
              {interactive ? <MousePointerClick size={12} /> : <SquareArrowOutUpRight size={12} />}
              Reviewers can approve or reject from the message
            </span>
            <span className="text-micro leading-4 text-slate-500">
              {interactive
                ? "Approve and Reject buttons are shown. Every decision is still authorized by the Cluster exactly like one made in the access portal."
                : "The message only links to the access portal, where the reviewer decides."}
            </span>
            {interactiveUnsupported && (
              <span className="flex items-center gap-1 text-micro font-medium text-amber-700">
                <AlertTriangle size={11} /> This Integration does not support interactive
                reviews: only the link is shown.
              </span>
            )}
          </span>
        </label>
      )}
    </div>
  );
};

export const SurfacesEdit = (props: {
  surfaces: AccessP.Policy_Spec_Rule_Surface[];
  isReview: boolean;
  keyAt: (index: number) => string;
  removeKeyAt: (index: number) => void;
  onUpdate: () => void;
  addLabel: string;
  emptyLabel: string;
}) => {
  const integrations = useIntegrationChoices();
  const items = integrations.data ?? [];

  return (
    <div className="flex flex-col gap-2">
      {props.surfaces.length === 0 && (
        <span className="text-xs text-slate-500">{props.emptyLabel}</span>
      )}
      {props.surfaces.map((surface, index) => (
        <SurfaceRow
          key={props.keyAt(index)}
          surface={surface}
          isReview={props.isReview}
          integrations={items}
          onUpdate={props.onUpdate}
          onRemove={() => {
            props.removeKeyAt(index);
            props.surfaces.splice(index, 1);
            props.onUpdate();
          }}
        />
      ))}
      <Button
        variant="default"
        size="compact-xs"
        className="w-fit"
        leftSection={<Plus size={12} />}
        onClick={() => {
          const surface = newSurface(props.isReview);
          if (items.length === 1 && surface.destination)
            surface.destination.integrationRef = getResourceRef(items[0]);
          props.surfaces.push(surface);
          props.onUpdate();
        }}
      >
        {props.addLabel}
      </Button>
    </div>
  );
};
