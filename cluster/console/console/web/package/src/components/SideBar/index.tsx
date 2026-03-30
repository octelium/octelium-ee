import {
  BadgeCheck,
  BookKey,
  Boxes,
  Crown,
  DoorClosed,
  Fingerprint,
  Folder,
  Globe,
  Globe2,
  KeyRound,
  LaptopMinimal,
  Library,
  LockKeyhole,
  LockOpen,
  PanelTop,
  Scroll,
  Settings2,
  Shield,
  ShieldCheck,
  ShieldEllipsis,
  ShieldUser,
  SquareTerminal,
  Telescope,
  Terminal,
  User,
  Users,
} from "lucide-react";

import { Select } from "@mantine/core";
import * as React from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";

const itemsCore = [
  {
    title: "Services",
    url: "/core/services",
    icon: PanelTop,
  },
  {
    title: "Namespaces",
    url: "/core/namespaces",
    icon: Boxes,
  },
  {
    title: "Users",
    url: "/core/users",
    icon: User,
  },
  {
    title: "Sessions",
    url: "/core/sessions",
    icon: Terminal,
  },
  {
    title: "Devices",
    url: "/core/devices",
    icon: LaptopMinimal,
  },
  {
    title: "Groups",
    url: "/core/groups",
    icon: Users,
  },
  {
    title: "Policies",
    url: "/core/policies",
    icon: Shield,
  },
  {
    title: "Identity Providers",
    url: "/core/identityproviders",
    icon: Fingerprint,
  },
  {
    title: "Secrets",
    url: "/core/secrets",
    icon: KeyRound,
  },
  {
    title: "Credentials",
    url: "/core/credentials",
    icon: LockOpen,
  },

  {
    title: "Gateways",
    url: "/core/gateways",
    icon: DoorClosed,
  },
  {
    title: "Regions",
    url: "/core/regions",
    icon: Globe,
  },

  {
    title: "Authenticators",
    url: "/core/authenticators",
    icon: LockKeyhole,
  },

  {
    title: "ClusterConfig",
    url: "/core/clusterconfig",
    icon: Settings2,
  },
];

const itemsEnterprise = [
  {
    title: "Certificates",
    url: "/enterprise/certificates",
    icon: ShieldCheck,
  },
  {
    title: "Certificate Issuers",
    url: "/enterprise/certificateissuers",
    icon: Crown,
  },

  {
    title: "Collector Exporters",
    url: "/enterprise/collectorexporters",
    icon: Telescope,
  },
  {
    title: "Directory Providers",
    url: "/enterprise/directoryproviders",
    icon: Folder,
  },
  {
    title: "Secrets",
    url: "/enterprise/secrets",
    icon: KeyRound,
  },

  {
    title: "DNS Providers",
    url: "/enterprise/dnsproviders",
    icon: Globe2,
  },

  {
    title: "Secret Stores",
    url: "/enterprise/secretstores",
    icon: BookKey,
  },

  {
    title: "ClusterConfig",
    url: "/enterprise/clusterconfig",
    icon: Settings2,
  },
  {
    title: "Policy Tester",
    url: "/enterprise/policytester",
    icon: BadgeCheck,
  },
];

const itemsVisibility = [
  {
    title: "Access Logs",
    url: "/visibility/accesslogs",
    icon: ShieldEllipsis,
  },
  {
    title: "Authentication Logs",
    url: "/visibility/authenticationlogs",
    icon: ShieldUser,
  },
  {
    title: "Audit Logs",
    url: "/visibility/auditlogs",
    icon: Library,
  },
  {
    title: "Component Logs",
    url: "/visibility/componentlogs",
    icon: Scroll,
  },
  {
    title: "SSH Sessions",
    url: "/visibility/ssh",
    icon: SquareTerminal,
  },
];

export const IconAuthenticationLog = ShieldUser;
export const IconAccessLog = ShieldEllipsis;

export default function () {
  const loc = useLocation();
  const navigate = useNavigate();

  const [barIdx, setBarIdx] = React.useState(
    match(loc.pathname)
      .when(
        (v) => v.startsWith("/core"),
        () => 0,
      )
      .when(
        (v) => v.startsWith("/enterprise"),
        () => 1,
      )
      .when(
        (v) => v.startsWith("/clusterman"),
        () => 2,
      )
      .when(
        (v) => v.startsWith("/visibility"),
        () => 3,
      )
      .otherwise(() => 0),
  );

  React.useEffect(() => {
    setBarIdx(
      match(loc.pathname)
        .when(
          (v) => v.startsWith("/core"),
          () => 0,
        )
        .when(
          (v) => v.startsWith("/enterprise"),
          () => 1,
        )
        .when(
          (v) => v.startsWith("/clusterman"),
          () => 2,
        )
        .when(
          (v) => v.startsWith("/visibility"),
          () => 3,
        )
        .otherwise(() => 0),
    );
  }, [loc.pathname]);

  return (
    <div className="min-h-full w-full">
      <div>
        <div>
          <div
            className={twMerge("font-extrabold text-slate-800 text-md mb-4")}
          >
            <Select
              className="shadow-2xl mb-16"
              data={[
                {
                  label: "Core",
                  value: "0",
                },
                {
                  label: "Enterprise",
                  value: "1",
                },

                {
                  label: "Cluster Management",
                  value: "2",
                },
                {
                  label: "Visibility",
                  value: "3",
                },
              ]}
              value={`${barIdx}`}
              allowDeselect={false}
              onChange={(val) => {
                const idx = parseInt(val ?? "0");
                const lastBarIdx = barIdx;
                setBarIdx(idx);

                match(idx)
                  .with(1, () => navigate(`/enterprise/certificates`))
                  .with(0, () => navigate(`/core`))
                  .with(2, () => navigate(`/clusterman`))
                  .with(3, () => navigate(`/visibility`))
                  .otherwise(() => navigate(`/core`));
              }}
            />
          </div>
          <div>
            <div>
              {match(barIdx)
                .with(1, () => itemsEnterprise)
                // .with(2, () => itemsClusterMan)
                .with(0, () => itemsCore)
                .with(3, () => itemsVisibility)
                .otherwise(() => undefined)
                ?.map((item) => (
                  <div key={item.title}>
                    <div>
                      <Link
                        viewTransition
                        className={twMerge(
                          "transition-all duration-500 hover:bg-slate-200 font-extrabold",
                          "flex w-full items-center justify-center",
                          "py-1 px-2 rounded-md my-1",
                          "text-sm",
                          loc.pathname === item.url ||
                            loc.pathname.startsWith(`${item.url}/`)
                            ? `!text-white bg-zinc-800 hover:bg-black shadow-md`
                            : `text-zinc-600 hover:text-zinc-800`,
                        )}
                        to={item.url}
                      >
                        <item.icon />
                        <span className="flex-1 ml-2">{item.title}</span>
                      </Link>
                    </div>
                  </div>
                ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
