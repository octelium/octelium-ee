import { Button } from "@mantine/core";
import { ExternalLink, Globe, LockKeyhole, UserRound } from "lucide-react";

import {
  Avatar,
  Badge,
  BackLink,
  CopyValue,
  InfoGrid,
  KeyValue,
  Note,
  PageHeader,
  SectionCard,
} from "@/ui";
import { getDomain, isDev } from "@/utils";
import { useAppSelector } from "@/utils/hooks";

const Settings = () => {
  const settings = useAppSelector((state) => state.settings);
  const user = settings.status?.user;
  const displayName =
    user?.metadata?.displayName || user?.metadata?.name || "User";
  const email = user?.spec?.email;
  const authenticatorsHref = isDev()
    ? `${window.location.origin}/authenticators`
    : `https://${getDomain()}/authenticators`;

  return (
    <div className="w-full max-w-3xl">
      <BackLink to="/user/requests">My Requests</BackLink>

      <PageHeader
        eyebrow="Account"
        title="Settings"
        description="Your identity in this Cluster and how you authenticate to it."
      />

      <div className="flex flex-col gap-4">
        <SectionCard
          title="Signed-in user"
          description="The identity every request you create is attributed to"
          icon={<UserRound size={14} strokeWidth={2.4} />}
          tone="blue"
        >
          <div className="flex flex-col gap-4">
            <div className="flex items-center gap-3">
              <Avatar src={user?.metadata?.picURL} name={displayName} size="lg" />
              <div className="min-w-0">
                <p className="truncate text-[1rem] font-bold text-slate-900">
                  {displayName}
                </p>
                {email && (
                  <p className="truncate text-[0.78rem] font-medium text-slate-500">
                    {email}
                  </p>
                )}
              </div>
              <Badge tone="emerald" dot className="ml-auto">
                Current session
              </Badge>
            </div>

            <InfoGrid className="border-t border-slate-100 pt-4">
              <KeyValue label="User ID">
                <CopyValue value={user?.metadata?.name ?? ""} />
              </KeyValue>
              <KeyValue label="Cluster domain" icon={<Globe size={12} className="text-slate-300" />}>
                {settings.status?.domain || getDomain()}
              </KeyValue>
            </InfoGrid>
          </div>
        </SectionCard>

        <SectionCard
          title="Authentication"
          description="Passkeys and other authenticators for your account"
          icon={<LockKeyhole size={14} strokeWidth={2.4} />}
          tone="violet"
        >
          <div className="flex flex-col gap-3">
            <Note tone="slate">
              Authenticators are managed in the Cluster portal. Changes apply to
              every application you sign in to.
            </Note>
            <Button
              component="a"
              href={authenticatorsHref}
              variant="default"
              className="self-start"
              leftSection={<LockKeyhole size={14} strokeWidth={2.5} />}
              rightSection={<ExternalLink size={12} strokeWidth={2.5} />}
            >
              Manage authenticators
            </Button>
          </div>
        </SectionCard>
      </div>
    </div>
  );
};

export default Settings;
