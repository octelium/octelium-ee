import {
  Button,
  SegmentedControl,
  useMantineColorScheme,
  type MantineColorScheme,
} from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { ExternalLink, LockKeyhole, SunMoon, UserRound } from "lucide-react";

import {
  Avatar,
  BackLink,
  CopyValue,
  DocumentTitle,
  ErrorState,
  Field,
  InfoGrid,
  KeyValue,
  Note,
  PageHeader,
  SectionCard,
} from "@/ui";
import { getDomain, isDev } from "@/utils";
import { getUserMainClient } from "@/utils/client";
import { COLOR_SCHEMES } from "@/utils/theme/colorScheme";

const Appearance = () => {
  const { colorScheme, setColorScheme } = useMantineColorScheme();

  return (
    <SectionCard
      title="Appearance"
      description="How this portal looks on this device"
      icon={<SunMoon size={14} strokeWidth={2.25} />}
    >
      <Field
        label="Color scheme"
        description="System follows the color scheme of your operating system. The choice is kept in this browser only."
      >
        <SegmentedControl
          value={colorScheme}
          onChange={(value) => setColorScheme(value as MantineColorScheme)}
          data={COLOR_SCHEMES}
          size="sm"
          className="w-full max-w-xs"
        />
      </Field>
    </SectionCard>
  );
};

const Identity = () => {
  const statusQry = useQuery({
    queryKey: ["user", "getStatus"],
    queryFn: async () => {
      const { response } = await getUserMainClient().getStatus({});
      return response;
    },
    staleTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
  });

  if (statusQry.isLoading) {
    return (
      <SectionCard
        title="Signed-in user"
        icon={<UserRound size={14} strokeWidth={2.25} />}
      >
        <div aria-hidden className="flex items-center gap-3">
          <div className="h-12 w-12 rounded-full bg-slate-100 motion-safe:animate-pulse" />
          <div className="flex-1 space-y-2">
            <div className="h-3 w-1/3 rounded bg-slate-100 motion-safe:animate-pulse" />
            <div className="h-2.5 w-1/2 rounded bg-slate-50 motion-safe:animate-pulse" />
          </div>
        </div>
      </SectionCard>
    );
  }

  if (!statusQry.data) {
    return (
      <ErrorState
        title="Could not load your identity"
        error={statusQry.error}
        onRetry={() => statusQry.refetch()}
      />
    );
  }

  const user = statusQry.data.user;
  const displayName = user?.metadata?.displayName || user?.metadata?.name || "—";
  const email = user?.spec?.email;

  return (
    <SectionCard
      title="Signed-in user"
      description="The identity every request you create is attributed to"
      icon={<UserRound size={14} strokeWidth={2.25} />}
    >
      <div className="flex flex-col gap-4">
        <div className="flex min-w-0 items-center gap-3">
          <Avatar src={user?.metadata?.picURL} name={displayName} size="lg" />
          <div className="min-w-0">
            <p className="truncate text-base font-semibold text-slate-900">
              {displayName}
            </p>
            {email && (
              <p className="truncate text-xs font-normal text-slate-500">
                {email}
              </p>
            )}
          </div>
        </div>

        <InfoGrid className="border-t border-slate-100 pt-4">
          <KeyValue label="Username">
            <CopyValue
              value={user?.metadata?.name ?? ""}
              copyLabel="Copy username"
            />
          </KeyValue>
          <KeyValue label="Cluster domain">
            {statusQry.data.domain || getDomain()}
          </KeyValue>
        </InfoGrid>
      </div>
    </SectionCard>
  );
};

const Settings = () => {
  const authenticatorsHref = isDev()
    ? `${window.location.origin}/authenticators`
    : `https://${getDomain()}/authenticators`;

  return (
    <div className="w-full max-w-3xl">
      <DocumentTitle title="Settings" />
      <BackLink to="/user/requests">My Requests</BackLink>

      <PageHeader
        title="Settings"
        description="Your identity in this Cluster and how you authenticate to it."
      />

      <div className="flex flex-col gap-4">
        <Identity />

        <SectionCard
          title="Authentication"
          description="Passkeys and other authenticators for your account"
          icon={<LockKeyhole size={14} strokeWidth={2.25} />}
        >
          <div className="flex flex-col gap-3">
            <Note>
              Authenticators are managed in the Cluster portal. Changes apply to
              every application you sign in to.
            </Note>
            <Button
              component="a"
              href={authenticatorsHref}
              variant="default"
              className="self-start"
              leftSection={<LockKeyhole size={14} strokeWidth={2.25} />}
              rightSection={<ExternalLink size={12} strokeWidth={2.25} />}
            >
              Manage authenticators
            </Button>
          </div>
        </SectionCard>

        <Appearance />
      </div>
    </div>
  );
};

export default Settings;
