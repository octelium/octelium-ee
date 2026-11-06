import type { DeviceManager_Spec } from "@/apis/enterprisev1/enterprisev1";
import type { GetDeviceManagerSummaryResponse } from "@/apis/visibilityv1/enterprise/venterprisev1";
import { twMerge } from "@/lib/utils";
import * as React from "react";

export type ManagerKind = NonNullable<DeviceManager_Spec["type"]["oneofKind"]>;

export type ManagerCategory = "edr" | "mdm" | "trust";

export const CATEGORY_LABEL: Record<ManagerCategory, string> = {
  edr: "Endpoint security",
  mdm: "Device management",
  trust: "Device trust",
};

export interface ManagerProvider {
  kind: ManagerKind;
  statusType: string;
  summaryField: keyof GetDeviceManagerSummaryResponse;
  label: string;
  description: string;
  category: ManagerCategory;
  keywords: string[];
  tint: string;
  reportsOwners: boolean;
  publishesProbes: boolean;
}

export const PROVIDERS: ManagerProvider[] = [
  {
    kind: "crowdStrike",
    statusType: "CROWDSTRIKE",
    summaryField: "totalCrowdStrike",
    label: "CrowdStrike Falcon",
    description: "Hosts, sensors and Zero Trust Assessment scores from Falcon.",
    category: "edr",
    keywords: ["crowdstrike", "falcon", "edr", "zta", "sensor"],
    tint: "#FA0001",
    reportsOwners: false,
    publishesProbes: true,
  },
  {
    kind: "sentinelOne",
    statusType: "SENTINELONE",
    summaryField: "totalSentinelOne",
    label: "SentinelOne",
    description: "Agents and their threat status from a management console.",
    category: "edr",
    keywords: ["sentinelone", "s1", "singularity", "edr", "xdr"],
    tint: "#6B0AEA",
    reportsOwners: false,
    publishesProbes: true,
  },
  {
    kind: "huntress",
    statusType: "HUNTRESS",
    summaryField: "totalHuntress",
    label: "Huntress",
    description: "Agents of a Huntress account and its organizations.",
    category: "edr",
    keywords: ["huntress", "edr", "mdr"],
    tint: "#1FBFCC",
    reportsOwners: false,
    publishesProbes: false,
  },
  {
    kind: "microsoftIntune",
    statusType: "MICROSOFT_INTUNE",
    summaryField: "totalMicrosoftIntune",
    label: "Microsoft Intune",
    description: "Managed devices and their compliance from Microsoft Graph.",
    category: "mdm",
    keywords: ["microsoft", "intune", "endpoint manager", "entra", "mdm"],
    tint: "#1169DA",
    reportsOwners: true,
    publishesProbes: true,
  },
  {
    kind: "jamf",
    statusType: "JAMF_PRO",
    summaryField: "totalJamfPro",
    label: "Jamf Pro",
    description: "Computers and smart groups of a Jamf Pro server.",
    category: "mdm",
    keywords: ["jamf", "apple", "macos", "mdm"],
    tint: "#056AE6",
    reportsOwners: true,
    publishesProbes: true,
  },
  {
    kind: "iru",
    statusType: "IRU",
    summaryField: "totalIru",
    label: "Iru",
    description: "Devices of an Iru tenant, formerly Kandji.",
    category: "mdm",
    keywords: ["iru", "kandji", "apple", "macos", "mdm"],
    tint: "#4A9BDA",
    reportsOwners: true,
    publishesProbes: false,
  },
  {
    kind: "fleetDM",
    statusType: "FLEETDM",
    summaryField: "totalFleetDM",
    label: "Fleet",
    description: "Hosts of a Fleet server, or of a single team.",
    category: "mdm",
    keywords: ["fleet", "fleetdm", "osquery", "mdm"],
    tint: "#5CABDF",
    reportsOwners: false,
    publishesProbes: true,
  },
  {
    kind: "onePassword",
    statusType: "ONEPASSWORD",
    summaryField: "totalOnePassword",
    label: "1Password Device Trust",
    description:
      "Devices registered in 1Password Device Trust, formerly Kolide.",
    category: "trust",
    keywords: ["1password", "onepassword", "kolide", "device trust", "xam"],
    tint: "#3B66BC",
    reportsOwners: true,
    publishesProbes: true,
  },
];

export const getProvider = (kind?: string) =>
  PROVIDERS.find((provider) => provider.kind === kind);

type Mark = {
  viewBox: string;
  paths: { fill: string; rule?: "evenodd"; d: string }[];
};

const MARKS: Record<ManagerKind, Mark> = {
  crowdStrike: {
    viewBox: "0 0 512 512",
    paths: [
      {
        fill: "#FA0001",
        d: "M503.2 512c-17.7-38.8-53-90-190.7-162.4c-63.6-35.3-171.3-88.3-268.4-190.7c8.9 37.1 54.8 118.3 249 218.9C346 406.1 437.8 434.3 503.2 512m-17.7-70.6c-15.9-47.7-44.1-107.7-181.8-196C236.6 201.3 137.7 144.8 8.8 0c8.8 38.8 49.4 139.5 256 270.1c67.1 47.7 155.4 75.9 220.7 171.3",
      },
    ],
  },
  sentinelOne: {
    viewBox: "-9.5 0 96 96",
    paths: [
      { fill: "#6B0AEA", d: "M32.08 0H45v77.25H32.08z" },
      {
        fill: "#6B0AEA",
        rule: "evenodd",
        d: "m48.13 95.88l12.91-8V21a32.2 32.2 0 0 0-12.91-5.72ZM16 87.92l12.92 8v-80.6A32.2 32.2 0 0 0 16 21ZM64.17 3.67v82.81l6-3.72a15.3 15.3 0 0 0 6.89-13V30.65c.03-11.28-12.89-26.98-12.89-26.98M0 69.73a15.27 15.27 0 0 0 6.89 13l6 3.72V3.67S0 19.37 0 30.65Z",
      },
    ],
  },
  microsoftIntune: {
    viewBox: "0.69 5.4 400 390.62",
    paths: [
      {
        fill: "#1169DA",
        d: "M0.685165 65.8523C0.685165 14.7648 67.4855 -13.2029 111.616 19.4083L207.551 90.2535L288.514 44.373C331.305 21.0394 399.874 37.7748 400.685 101.559L400.685 336.622C400.685 387.71 335.845 414.343 291.715 381.732L194.031 309.34L112.973 355.23C69.7783 378.784 0.752625 360.714 0.752625 298.637C0.752654 224.124 0.685165 252.734 0.685165 65.8523Z",
      },
      {
        fill: "#0151BD",
        d: "M111.616 19.4083C67.4855 -13.2029 0.685165 14.7648 0.685165 65.8523V295.742C0.685165 290.153 3.5327 279.026 3.5327 279.026C16.6954 236.001 72.6487 219.869 111.36 248.223L291.715 381.732C335.845 414.343 400.685 387.71 400.685 336.622V101.559C400.685 145.201 343.643 189.011 289.755 151.051L111.616 19.4083Z",
      },
      {
        fill: "#23C0FE",
        d: "M111.616 149.691C67.4855 117.08 0.685165 145.048 0.685165 196.135L0.685179 65.8523C0.685183 14.7648 67.4855 -13.2029 111.616 19.4083C170.996 63.2888 230.374 107.171 289.755 151.051C343.643 189.011 400.685 145.201 400.685 101.559V234.89C400.685 285.978 333.885 313.946 289.755 281.334C230.374 237.454 170.996 193.572 111.616 149.691Z",
      },
    ],
  },
  jamf: {
    viewBox: "0 0 43 43",
    paths: [
      {
        fill: "#056AE6",
        d: "M33.6,7 C28.1,7 24.8,9.3 23.3,14.3 L19.5,26.2 C18.1,30.1 15.8,31.7 11.7,31.7 L2.2,31.7 C1,31.7 0,32.7 0,33.9 L0,40 C0,41.1 0.9,42 2.1,42 L40.7,42 C41.9,42 42.9,41 42.9,39.8 L42.9,9.1 C42.9,7.9 42,7 40.8,7 L33.6,7 L33.6,7 Z M2.3,0 C1.1,0 0,1.1 0,2.3 L0,18.5 C0,19.8 1,20.8 2.3,20.8 L10.5,20.8 C14.2,20.8 19.4,20 20.8,13.4 C20.8,13.4 22.2,6.7 23,2.8 C23.3,1.3 22.2,-7.10542736e-15 20.7,-7.10542736e-15 L2.3,-7.10542736e-15 L2.3,0 Z",
      },
    ],
  },
  onePassword: {
    viewBox: "0 0 24 24",
    paths: [
      {
        fill: "#3B66BC",
        d: "M12 0c6.627 0 12 5.373 12 12 0 6.628-5.373 12-12 12S0 18.628 0 12C0 5.373 5.373 0 12 0m-.893 4.86c-.485 0-.727.001-.913.095a.87.87 0 0 0-.378.379c-.094.185-.095.428-.095.912v2.747c0 .12 0 .182.016.238q.02.075.065.138a1 1 0 0 0 .175.162l.695.564c.113.092.17.139.19.194a.22.22 0 0 1 0 .15c-.02.056-.077.102-.19.194l-.695.564a1 1 0 0 0-.175.162.4.4 0 0 0-.065.138 1 1 0 0 0-.016.238v6.019c0 .485 0 .728.095.913a.87.87 0 0 0 .378.378c.186.094.428.094.913.094h1.786c.485 0 .727 0 .913-.094a.87.87 0 0 0 .378-.378c.095-.185.095-.428.095-.913v-2.747c0-.12 0-.182-.016-.238a.4.4 0 0 0-.065-.138 1 1 0 0 0-.175-.162l-.695-.564c-.113-.092-.17-.138-.191-.193a.22.22 0 0 1 0-.152c.02-.055.078-.1.19-.193l.696-.564a1 1 0 0 0 .175-.162.4.4 0 0 0 .065-.138 1 1 0 0 0 .016-.238V6.246c0-.484 0-.727-.095-.912a.87.87 0 0 0-.378-.379c-.186-.094-.428-.094-.913-.094Z",
      },
    ],
  },
  fleetDM: {
    viewBox: "0 0 512 512",
    paths: [
      {
        fill: "#63C740",
        d: "M65.4 128c35.1 0 63.5-28.7 63.5-64S100.5 0 65.4 0S1.9 28.7 1.9 64s28.5 64 63.5 64",
      },
      {
        fill: "#5CABDF",
        d: "M256 128c35.1 0 63.5-28.7 63.5-64S291.1 0 256 0s-63.5 28.7-63.5 64s28.4 64 63.5 64",
      },
      {
        fill: "#D66C7B",
        d: "M446.6 128c35.1 0 63.5-28.7 63.5-64S481.7 0 446.6 0S383 28.7 383 64s28.5 64 63.6 64",
      },
      {
        fill: "#C98DEF",
        d: "M65.4 320c35.1 0 63.5-28.7 63.5-64s-28.4-64-63.5-64s-63.5 28.7-63.5 64s28.5 64 63.5 64",
      },
      {
        fill: "#FAA669",
        d: "M256 320c35.1 0 63.5-28.7 63.5-64s-28.4-64-63.5-64s-63.5 28.7-63.5 64s28.4 64 63.5 64",
      },
      {
        fill: "#3AEFC4",
        d: "M65.4 512c35.1 0 63.5-28.7 63.5-64s-28.4-64-63.5-64s-63.5 28.7-63.5 64s28.5 64 63.5 64",
      },
    ],
  },
  huntress: {
    viewBox: "0 0 24 24",
    paths: [
      {
        fill: "#1FBFCC",
        rule: "evenodd",
        d: "M7.1 0.04L7.14 1.49L7.37 2.17L7.9 3.12L9.05 4.27L11.41 6.02L11.87 6.48L12.44 7.28L12.86 8.34L12.86 8.69L12.97 8.91L12.97 9.71L12.74 10.63L12.78 10.93L13.09 10.4L13.31 9.49L13.31 8.11L12.97 6.74L12.51 5.71L12.63 5.6L13.81 6.48L14.8 8L14.91 7.89L14.84 6.21L15.18 6.36L15.87 7.16L16.59 8.61L16.7 8.65L16.86 8.34L16.86 7.54L16.9 7.47L17.01 7.5L17.58 8.3L18.57 10.4L18.76 11.12L18.69 11.2L17.77 11.2L16.29 10.86L16.1 10.93L16.74 11.31L17.54 11.54L18.8 11.54L18.95 11.62L18.91 13.14L19.03 13.37L19.03 13.94L19.68 15.85L19.22 15.47L18.69 14.51L18.46 13.71L18.42 12.72L17.43 12.69L16.51 12.46L15.56 11.92L15.1 11.35L14.91 11.54L14.8 12.8L14.91 13.03L15.03 13.94L15.68 15.35L16.17 16L16.17 16.23L15.83 16.8L15.6 17.49L15.6 19.09L16.06 20.23L16.29 21.14L16.29 22.06L16.17 22.29L16.17 22.74L15.79 23.96L15.6 23.66L15.6 23.09L15.49 22.74L14.95 21.68L14.04 20.65L12.9 19.85L11.37 19.09L10 18.63L9.58 18.59L9.62 18.48L10.84 17.75L11.64 17.07L11.79 16.76L12.1 16.5L12.55 15.58L12.67 15.62L13.12 16.3L13.73 16.84L14.57 17.26L14.65 17.18L13.73 16.15L13.09 14.97L12.63 13.49L12.59 12.72L12.48 12.76L12.29 13.26L12.29 14.63L12.4 14.86L11.87 15.81L11.52 16.27L10.38 17.3L9.54 17.71L9.09 17.83L8.86 17.83L8.74 17.71L8.74 17.26L8.86 16.91L9.16 16.42L10.04 15.7L10.19 15.35L9.24 16.15L7.94 16.8L7.1 16.99L6.91 16.8L7.1 16.08L7.68 15.5L8.9 14.78L8.93 14.67L7.14 15.43L6.8 15.43L6.57 15.54L5.77 15.54L5.66 15.43L5.96 14.82L6.65 14.13L7.94 13.49L9.43 13.03L9.73 12.84L8.17 13.26L6.91 13.37L6.69 13.49L5.31 13.49L5.09 13.37L4.4 13.37L3.49 13.14L3.3 12.99L3.45 12.76L4.25 12.19L5.66 11.54L6.11 11.43L9.16 11.39L9.09 11.31L7.03 11.31L5.54 10.97L4.59 10.55L3.9 10.1L3.33 9.52L3.03 9.03L3.07 8.84L3.83 8.57L5.09 8.46L5.31 8.57L6.23 8.69L8.17 9.26L8.59 9.22L7.14 8.8L5.5 7.92L4.59 7.01L4.13 6.32L3.49 4.8L3.49 4.57L3.6 4.46L3.83 4.57L4.4 4.57L5.31 4.8L6.15 5.22L7.45 6.21L7.68 6.25L6.88 5.3L6.46 4.46L6.23 3.66L6.23 2.17L6.57 1.03ZM11.71 3.89L13.66 4L15.6 4.57L16.67 5.1L17.7 5.79L19.07 7.16L19.75 8.19L20.51 9.83L20.86 11.2L20.86 12L20.97 12.23L20.86 14.06L20.44 15.47L20.32 15.5L19.94 14.51L19.71 13.49L19.71 13.03L19.6 11.31L19.26 10.06L18.5 8.53L17.81 7.62L17.39 7.24L16.78 6.13L16.67 6.1L16.63 6.4L16.51 6.51L14.88 5.64L14.19 4.95L14.11 5.03L14.19 5.33L13.77 5.37L13.54 5.26L12.97 5.26L11.64 4.61L11.6 4.69L11.83 5.03L11.75 5.18L11.49 5.26L9.92 4.27L10.8 4L11.49 4ZM4.86 7.77L5.28 8.15L5.09 8.23L4.67 8.19L4.63 8.11ZM3.9 10.67L5.05 11.24L4.36 11.62L3.71 12L3.6 11.89L3.71 10.86ZM15.49 12.57L16.51 13.03L17.81 13.22L17.89 13.49L17.77 13.83L18.11 14.06L17.89 14.29L18 14.97L18.3 15.58L18.65 16.04L19.03 16.34L18.95 16.72L18.27 17.3L18.61 17.68L18.69 18.06L18.61 18.21L18.04 18.44L18.27 18.7L18.34 18.97L18.27 19.12L17.85 19.28L17.66 19.66L17.66 19.89L17.66 20.8L17.47 21.18L16.97 21.49L16.74 21.03L16.74 20.57L16.29 19.43L16.17 18.51L16.06 18.29L16.17 18.06L16.17 17.49L16.29 17.14L16.86 16L16.13 15.01L15.71 14.17L15.37 12.91L15.37 12.69ZM3.71 13.71L4.93 13.98L5.2 15.09L5.96 16.61L6.42 17.3L7.45 18.32L8.25 18.9L9.77 19.66L11.14 20L12.06 20L13.58 20.88L13.73 21.07L13.58 21.18L13.43 21.26L11.03 21.26L10.8 21.14L10.46 21.14L8.97 20.69L7.68 20.04L6.99 19.58L5.73 18.44L4.82 17.18L4.17 15.89L3.83 14.86L3.6 13.83Z",
      },
    ],
  },
  iru: {
    viewBox: "0 0 24 24",
    paths: [
      {
        fill: "#4A9BDA",
        d: "M11.27 0.06L13.18 0.1L14.53 0.33L15.57 0.62L16.67 1.05L17.52 1.48L18.55 2.13L19.41 2.79L19.95 3.29L20.64 4.08L21.36 5.08L21.79 5.83L22.27 6.93L22.57 7.81L22.94 9.81L22.99 11.93L22.91 12.39L22.69 12.86L22.46 13.17L22.19 13.34L12.79 13.36L1.95 13.36L1.62 13.19L1.36 12.86L1.14 12.34L1.06 11.88L1.11 9.81L1.25 8.86L1.53 7.67L2.01 6.37L2.45 5.48L2.99 4.64L3.66 3.78L4.64 2.79L5.5 2.13L6.53 1.48L7.28 1.1L9.13 0.43L9.97 0.24Z",
      },
      {
        fill: "#9CCBEC",
        d: "M5.17 13.97L18.91 13.99L19 14.39L19 15.33L18.79 17.95L20.01 18.55L20.79 18.71L21.3 18.89L21.82 19.21L22.22 19.6L22.53 20.07L22.76 20.68L22.84 21.09L22.76 22L22.5 22.67L21.92 23.37L21.4 23.74L20.79 23.96L19.89 24.05L19.08 23.92L18.29 23.68L16.84 23.05L15.95 22.56L14.7 23.25L13.95 23.54L12.7 23.86L11.21 24.05L10.47 24.05L9.6 23.84L9.14 23.61L8.73 23.27L8.16 22.55L6.91 23.2L5.91 23.63L4.97 23.92L4.16 24.05L3.77 24.05L3.11 23.92L2.59 23.7L2.13 23.37L1.65 22.82L1.33 22.15L1.21 21.54L1.21 21.14L1.33 20.53L1.5 20.11L1.88 19.55L2.49 19.02L3.26 18.71L4.04 18.55L5.26 17.95L5.1 16.36L5.05 14.88L5.1 14.09Z",
      },
    ],
  },
};

export const ManagerMark = (props: {
  kind?: ManagerKind;
  size?: number | string;
  mono?: boolean;
  className?: string;
}) => {
  const mark = props.kind ? MARKS[props.kind] : undefined;
  const size = props.size ?? 18;
  if (!mark) {
    return (
      <span
        aria-hidden="true"
        style={{ width: size, height: size }}
        className={twMerge("rounded-md bg-slate-200", props.className)}
      />
    );
  }
  return (
    <svg
      viewBox={mark.viewBox}
      width={size}
      height={size}
      aria-hidden="true"
      className={twMerge("shrink-0", props.className)}
    >
      {mark.paths.map((path) => (
        <path
          key={path.d}
          fill={props.mono ? "currentColor" : path.fill}
          fillRule={path.rule}
          clipRule={path.rule}
          d={path.d}
        />
      ))}
    </svg>
  );
};

export const managerIcon =
  (kind: ManagerKind) =>
  (props: {
    size?: number | string;
    strokeWidth?: number | string;
    className?: string;
  }) => (
    <ManagerMark
      mono
      kind={kind}
      size={props.size}
      className={props.className}
    />
  );

export const ManagerTile = (props: {
  kind?: ManagerKind;
  size?: "sm" | "md" | "lg";
  className?: string;
}) => {
  const size = props.size ?? "md";
  const tint = getProvider(props.kind)?.tint ?? "#64748B";
  return (
    <span
      aria-hidden="true"
      style={{
        backgroundColor: `color-mix(in srgb, ${tint} 9%, var(--color-white))`,
        borderColor: `color-mix(in srgb, ${tint} 22%, var(--color-slate-200))`,
      }}
      className={twMerge(
        "flex shrink-0 items-center justify-center border",
        size === "sm" && "h-7 w-7 rounded-md",
        size === "md" && "h-9 w-9 rounded-lg",
        size === "lg" && "h-11 w-11 rounded-xl",
        props.className,
      )}
    >
      <ManagerMark
        kind={props.kind}
        size={size === "sm" ? 15 : size === "md" ? 19 : 23}
      />
    </span>
  );
};
