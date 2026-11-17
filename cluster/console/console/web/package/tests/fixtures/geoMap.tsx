import "@/index.css";
import "@mantine/core/styles.layer.css";
import "@fontsource/plus-jakarta-sans/500.css";
import "@fontsource/plus-jakarta-sans/600.css";
import "@fontsource/plus-jakarta-sans/700.css";
import { Session } from "@/apis/corev1/corev1";
import { GeoActivityBody, type GeoView } from "@/components/GeoActivity";
import SessionLocations from "@/pages/core/Session/Location";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import { MantineProvider } from "@mantine/core";
import { MotionConfig } from "framer-motion";
import React from "react";
import { createRoot } from "react-dom/client";
import { createMemoryRouter, RouterProvider, useLocation } from "react-router-dom";

const params = new URLSearchParams(window.location.search);
const scenario = params.get("scenario") ?? "panel";

const view: GeoView = {
  rows: [
    {
      code: "US",
      name: "United States",
      value: 1200,
      segments: [
        { label: "allowed", value: 1100, tone: "primary" },
        { label: "denied", value: 100, tone: "danger" },
      ],
      meta: ["12 users"],
    },
    { code: "DE", name: "Germany", value: 300, meta: ["4 users"] },
    { code: "BR", name: "Brazil", value: 40 },
    { code: "SG", name: "Singapore", value: 7 },
  ],
  points: [
    { id: "US/San Francisco", lat: 37.77, lng: -122.42, value: 900, label: "San Francisco, United States", countryCode: "US" },
    { id: "DE/Berlin", lat: 52.52, lng: 13.4, value: 300, label: "Berlin, Germany", countryCode: "DE" },
    { id: "BR/São Paulo", lat: -23.55, lng: -46.63, value: 40, tone: "danger", label: "São Paulo, Brazil", countryCode: "BR" },
  ],
  unknown: 53,
  totalCountries: 4,
  totalCities: 3,
  tone: "primary",
  noun: "sign-ins",
  note: "Resolved from GeoIP at authentication time",
  to: (code) => `/visibility/authenticationlogs?countryCode=${code}`,
};

const now = Date.now();
const geoip = (code: string, name: string, city: string, latitude: number, longitude: number, ip: string) => ({
  ip,
  country: { code, name },
  city: { name: city },
  coordinates: { latitude, longitude, accuracyRadius: 20 },
});
const auth = (tokenID: string, minutesAgo: number, value: ReturnType<typeof geoip>) => ({
  tokenID,
  setAt: new Date(now - minutesAgo * 60_000).toISOString(),
  info: { type: "IDENTITY_PROVIDER", geoip: value },
});

const session = Session.fromJson({
  apiVersion: "core/v1",
  kind: "Session",
  metadata: { name: "usr1-trail", uid: "5a0b4c64-56d4-4c8f-9a3e-1a1a7b1b2c3d" },
  spec: { state: "ACTIVE" },
  status: {
    type: "CLIENTLESS",
    authentication: auth("t3", 2, geoip("US", "United States", "San Francisco", 37.77, -122.42, "203.0.113.10")),
    lastAuthentications: [
      auth("t2", 62, geoip("FR", "France", "Paris", 48.86, 2.35, "198.51.100.7")),
      auth("t1", 900, geoip("US", "United States", "San Francisco", 37.78, -122.41, "203.0.113.11")),
    ],
    initialAuthentication: auth("t1", 900, geoip("US", "United States", "San Francisco", 37.78, -122.41, "203.0.113.11")),
  },
});

const Location = () => {
  const location = useLocation();
  return <output data-testid="location">{`${location.pathname}${location.search}`}</output>;
};

const Fixture = () => (
  <div style={{ padding: 16 }}>
    {scenario === "session" ? (
      <SessionLocations item={session} />
    ) : (
      <GeoActivityBody
        view={scenario === "empty" ? { ...view, rows: [], points: [], totalCountries: 0, totalCities: 0, unknown: 5 } : view}
        rangeLabel="1h"
      />
    )}
    <Location />
  </div>
);

const router = createMemoryRouter([{ path: "*", element: <Fixture /> }], {
  initialEntries: ["/"],
});

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MotionConfig reducedMotion="always">
      <MantineProvider
        theme={themeMantine}
        cssVariablesResolver={cssVariablesResolver}
        forceColorScheme={params.get("theme") === "dark" ? "dark" : "light"}
      >
        <RouterProvider router={router} />
      </MantineProvider>
    </MotionConfig>
  </React.StrictMode>,
);
