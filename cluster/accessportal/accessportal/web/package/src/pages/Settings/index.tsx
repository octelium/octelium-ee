import { Button } from "@mantine/core";
import { ArrowLeft, ExternalLink, LockKeyhole } from "lucide-react";
import { Link } from "react-router-dom";

import { getDomain, isDev } from "../../utils";
import { useAppSelector } from "../../utils/hooks";
import { Avatar, Badge, Card, KeyValue, PageHeader, SectionTitle } from "../../ui";

const Settings = () => {
  const settings = useAppSelector((state) => state.settings);
  const user = settings.status?.user;
  const displayName = user?.metadata?.displayName || user?.metadata?.name || "User";
  const email = user?.spec?.email;
  const authenticatorsHref = isDev()
    ? `${window.location.origin}/authenticators`
    : `https://${getDomain()}/authenticators`;

  return (
    <div className="w-full max-w-3xl">
      <Link
        to="/user/requests"
        className="inline-flex items-center gap-1.5 text-[0.75rem] font-bold text-slate-400 hover:text-slate-700 transition-colors duration-150 mb-4"
      >
        <ArrowLeft size={13} strokeWidth={2.5} />
        Back to requests
      </Link>

      <PageHeader
        eyebrow="Account"
        title="Settings"
        description="Manage your account and authentication settings."
      />

      <Card className="p-5">
        <SectionTitle>Signed-in user</SectionTitle>
        <div className="flex items-center gap-3 mb-5">
          <Avatar
            src={user?.metadata?.picURL}
            name={displayName}
          />
          <div className="min-w-0">
            <p className="text-[0.95rem] font-bold text-slate-800 truncate">{displayName}</p>
            {email && <p className="text-[0.78rem] font-medium text-slate-500 truncate">{email}</p>}
          </div>
          <Badge className="ml-auto" tone="slate">Current session</Badge>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 border-t border-slate-100 pt-4">
          <KeyValue label="User ID">
            <span className="font-mono break-all">{user?.metadata?.name || "—"}</span>
          </KeyValue>
          <KeyValue label="Domain">{settings.status?.domain || getDomain()}</KeyValue>
        </div>

        <div className="border-t border-slate-100 mt-5 pt-5">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-[0.82rem] font-bold text-slate-700">Authenticators</p>
              <p className="text-[0.72rem] font-medium text-slate-400 mt-1">
                Add or remove passkeys and other authentication methods.
              </p>
            </div>
            <Button
              component="a"
              href={authenticatorsHref}
              variant="outline"
              leftSection={<LockKeyhole size={14} strokeWidth={2.5} />}
              rightSection={<ExternalLink size={12} strokeWidth={2.5} />}
            >
              Manage
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
};

export default Settings;
