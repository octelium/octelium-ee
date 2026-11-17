import assert from "node:assert/strict";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

const root = fileURLToPath(new URL("../", import.meta.url));
const world = JSON.parse(await readFile(join(root, "src/components/GeoMap/world.json"), "utf8"));

describe("Geography helpers", () => {
  let modules;
  let geo;
  let map;
  let activity;
  let location;

  before(async () => {
    modules = await createConsoleModules();
    [geo, map, activity, location] = await Promise.all([
      modules.load("components/GeoMap/geo.ts"),
      modules.load("components/GeoMap/WorldMap.tsx"),
      modules.load("components/GeoActivity/index.tsx"),
      modules.load("pages/core/Session/Location.tsx"),
    ]);
  });

  after(async () => {
    await modules?.close();
  });

  it("ships a projected world map keyed by ISO 3166-1 alpha-2 codes", () => {
    assert.equal(world.width, 1000);
    assert.equal(world.height, 440);
    const codes = world.countries.map(([code]) => code);
    assert.equal(new Set(codes).size, codes.length);
    assert.ok(codes.length > 230);
    for (const code of codes) assert.match(code, /^[A-Z]{2}$/);
    assert.ok(!codes.includes("AQ"));
    for (const code of ["US", "DE", "BR", "IN", "XK"]) {
      const entry = world.countries.find(([itm]) => itm === code);
      assert.ok(entry[1].startsWith("M"), code);
    }
    for (const code of ["SG", "HK", "BH", "MT"]) {
      const entry = world.countries.find(([itm]) => itm === code);
      assert.equal(entry[1], "", code);
      assert.ok(entry[2] > 0 && entry[2] < world.width && entry[3] > 0 && entry[3] < world.height, code);
    }
  });

  it("projects coordinates with the same Equal Earth projection as the map", () => {
    assert.deepEqual(geo.projectEqualEarth(world, { lat: 0, lng: 0 }), world.translate);
    for (const [code, lat, lng] of [["FR", 46.6, 2.4], ["AU", -25.3, 134.7], ["SG", 1.35, 103.82], ["BR", -10.3, -53.2]]) {
      const [, , cx, cy] = world.countries.find(([itm]) => itm === code);
      const [x, y] = geo.projectEqualEarth(world, { lat, lng });
      assert.ok(Math.hypot(x - cx, y - cy) < 25, `${code} ${x},${y} vs ${cx},${cy}`);
    }
    const [west] = geo.projectEqualEarth(world, { lat: 0, lng: -180 });
    const [east] = geo.projectEqualEarth(world, { lat: 0, lng: 180 });
    assert.ok(Math.abs((west + east) / 2 - world.translate[0]) < 1e-6);
    assert.ok(east - west >= world.width - 1);
    assert.deepEqual(
      geo.projectEqualEarth(world, { lat: 120, lng: 400 }),
      geo.projectEqualEarth(world, { lat: 90, lng: 180 }),
    );
  });

  it("measures great-circle distances and formats them", () => {
    const paris = { lat: 48.8566, lng: 2.3522 };
    const newYork = { lat: 40.7128, lng: -74.006 };
    assert.ok(Math.abs(geo.distanceKm(paris, newYork) - 5837) < 30);
    assert.equal(geo.distanceKm(paris, paris), 0);
    assert.ok(Math.abs(geo.distanceKm({ lat: 0, lng: 179.5 }, { lat: 0, lng: -179.5 }) - 111.2) < 1);
    assert.equal(geo.formatDistance(0.4), "<1 km");
    assert.equal(geo.formatDistance(42.4), "42 km");
    assert.equal(geo.formatDistance(8955.2), "8,955 km");
    assert.equal(geo.formatLatLng({ lat: -33.8688, lng: 151.2093 }), "33.87°S, 151.21°E");
  });

  it("normalizes country codes and resolves their names", () => {
    assert.equal(geo.normalizeCountryCode(" de "), "DE");
    for (const code of [undefined, "", "USA", "1A", "U'"]) {
      assert.equal(geo.normalizeCountryCode(code), undefined);
    }
    assert.equal(geo.getCountryName("us"), "United States");
    assert.equal(geo.getCountryName("GB"), "United Kingdom");
    assert.equal(geo.getCountryName("ZZ", "Unknown Region"), "Unknown Region");
    assert.equal(geo.getCountryName("bad", "Fallback"), "Fallback");
    assert.equal(geo.getCountryName(undefined), "Unknown");
    assert.equal(geo.hasCoordinates({ latitude: 0, longitude: 0 }), false);
    assert.equal(geo.hasCoordinates({ latitude: 0, longitude: 12 }), true);
    assert.equal(geo.hasCoordinates(undefined), false);
  });

  it("scales the country intensity logarithmically", () => {
    assert.equal(map.getIntensity(0, 100), 0);
    assert.equal(map.getIntensity(5, 0), 0);
    assert.equal(map.getIntensity(100, 100), 1);
    const low = map.getIntensity(1, 1000);
    const mid = map.getIntensity(30, 1000);
    assert.ok(low >= 0.22 && low < mid && mid < 1);
  });

  it("fits the view to the points without leaving the map", () => {
    assert.deepEqual(map.getFitViewBox(world, []), { x: 0, y: 0, w: 1000, h: 440 });
    const single = map.getFitViewBox(world, [{ lat: 48.86, lng: 2.35 }]);
    assert.ok(single.w < 300);
    assert.ok(Math.abs(single.w / single.h - 1000 / 440) < 1e-6);
    const [x, y] = geo.projectEqualEarth(world, { lat: 48.86, lng: 2.35 });
    assert.ok(x > single.x && x < single.x + single.w && y > single.y && y < single.y + single.h);
    const edge = map.getFitViewBox(world, [{ lat: 70, lng: -170 }]);
    assert.ok(edge.x >= 0 && edge.y >= 0);
    assert.deepEqual(
      map.getFitViewBox(world, [{ lat: 0, lng: -179 }, { lat: 0, lng: 179 }]),
      { x: 0, y: 0, w: 1000, h: 440 },
    );
  });

  it("builds sorted country rows and city points", () => {
    const rows = activity.toCountryRows([
      { country: { code: "de", name: "Germany" }, count: 3 },
      { country: { code: "US", name: "United States" }, count: 9, lastSeenAt: { seconds: 1n, nanos: 0 } },
      { country: { code: "", name: "" }, count: 40 },
      { country: { code: "FR", name: "France" }, count: 3 },
    ]);
    assert.deepEqual(rows.map((row) => [row.code, row.value]), [["US", 9], ["FR", 3], ["DE", 3]]);
    assert.equal(rows[0].name, "United States");

    const points = activity.toCityPoints([
      { country: { code: "BR", name: "Brazil" }, city: { name: "São Paulo" }, coordinates: { latitude: -23.5, longitude: -46.6 }, count: 5, countFailure: 4, countSuccess: 1, countUser: 1 },
      { country: { code: "DE", name: "Germany" }, city: { name: "" }, coordinates: { latitude: 51, longitude: 10 }, count: 2, countFailure: 0, countSuccess: 2, countUser: 2 },
      { country: { code: "SG", name: "Singapore" }, coordinates: { latitude: 0, longitude: 0 }, count: 9, countFailure: 0, countSuccess: 9, countUser: 0 },
    ], "primary");
    assert.equal(points.length, 2);
    assert.equal(points[0].tone, "danger");
    assert.equal(points[0].label, "São Paulo, Brazil");
    assert.deepEqual(points[0].meta, ["4 failed", "1 user"]);
    assert.equal(points[1].tone, "primary");
    assert.equal(points[1].label, "Germany");
    assert.deepEqual(points[1].meta, ["2 users"]);
  });

  it("orders a Session's sign-ins and flags impossible travel", () => {
    const at = (minutes) => ({ seconds: BigInt(Math.floor((Date.UTC(2026, 10, 17, 12) + minutes * 60_000) / 1000)), nanos: 0 });
    const auth = (tokenID, minutes, code, latitude, longitude) => ({
      tokenID,
      setAt: at(minutes),
      info: { geoip: { country: { code, name: code }, city: { name: code }, coordinates: { latitude, longitude, accuracyRadius: 5 } } },
    });
    const current = auth("t4", 0, "US", 37.77, -122.42);
    const item = {
      status: {
        authentication: current,
        lastAuthentications: [
          auth("t3", -60, "FR", 48.86, 2.35),
          auth("t2", -20 * 60, "US", 40.71, -74),
          auth("t1", -21 * 60, "US", 42.36, -71.06),
          { tokenID: "t0", setAt: at(-30 * 60), info: {} },
        ],
        initialAuthentication: auth("t1", -21 * 60, "US", 42.36, -71.06),
      },
    };

    const trail = location.getSessionTrail(item);
    assert.deepEqual(trail.map((entry) => entry.auth.tokenID), ["t1", "t2", "t3", "t4"]);
    assert.equal(trail[3].isCurrent, true);
    assert.equal(trail[0].fromPrevious, undefined);
    assert.equal(trail[1].fromPrevious.isImpossible, false);
    assert.ok(trail[1].fromPrevious.km < 500);
    assert.equal(trail[2].fromPrevious.isImpossible, false);
    assert.equal(trail[2].fromPrevious.isNewCountry, true);
    assert.equal(trail[3].fromPrevious.isImpossible, true);
    assert.ok(trail[3].fromPrevious.kmh > location.MAX_TRAVEL_KMH);
    assert.deepEqual(location.getImpossibleTravel(item).map((entry) => entry.auth.tokenID), ["t4"]);
    assert.deepEqual(location.getSessionTrail({ status: {} }), []);
  });
});

describe("World map", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-geo-"));
    server = await createServer({
      root,
      cacheDir,
      server: { host: "localhost", port: 0, watch: null },
    });
    await server.listen();
    origin = server.resolvedUrls.local[0];
    browser = await chromium.launch();
  });

  after(async () => {
    await browser?.close();
    await server?.close();
    if (cacheDir) await rm(cacheDir, { recursive: true, force: true });
  });

  beforeEach(async () => {
    context = await browser.newContext({
      locale: "en-US",
      timezoneId: "UTC",
      viewport: { width: 1440, height: 1000 },
    });
    page = await context.newPage();
    page.setDefaultTimeout(30_000);
    errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
  });

  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  const load = async (extra = {}) => {
    await page.goto(
      `${origin}tests/fixtures/geoMap.html?${new URLSearchParams(extra)}`,
    );
    await page.locator("svg[role=group] path").first().waitFor();
  };

  const currentLocation = () => page.getByTestId("location").textContent();

  const waitLocation = async (expected) => {
    await page.waitForFunction(
      (value) => document.querySelector("[data-testid=location]")?.textContent === value,
      expected,
    );
    assert.equal(await currentLocation(), expected);
  };

  const centroidOf = async (code) => {
    const [, , cx, cy] = world.countries.find(([itm]) => itm === code);
    return page.evaluate(([x, y]) => {
      const svg = document.querySelector("svg[role=group]");
      const point = svg.createSVGPoint();
      point.x = x;
      point.y = y;
      const ret = point.matrixTransform(svg.getScreenCTM());
      return { x: ret.x, y: ret.y };
    }, [cx, cy]);
  };

  it("shades the active countries and marks the cities", async () => {
    await load();
    const svg = page.getByRole("group", { name: "Map of sign-ins by country in the last 1h" });
    assert.equal(await svg.locator("path[data-code]").count(), 3);
    assert.equal(await svg.locator("circle[data-code='SG']").count(), 1);
    assert.equal(await svg.locator("[data-point]").count(), 3);
    const us = Number(await svg.locator("path[data-code='US']").getAttribute("fill-opacity"));
    const br = Number(await svg.locator("path[data-code='BR']").getAttribute("fill-opacity"));
    assert.ok(us > br);
    assert.equal(await svg.locator("path[data-code='US']").getAttribute("aria-label"), "United States: 1,200 sign-ins");
    for (const label of ["Countries", "Top country", "Located", "No location"]) {
      await page.getByText(label, { exact: true }).waitFor();
    }
    await page.getByText("75% of sign-ins").waitFor();
    await page.getByText("97%", { exact: true }).waitFor();
    await page.getByText("1,547 of 1,600 sign-ins").waitFor();
    await page.getByText("3 cities").waitFor();
  });

  it("shows tooltips for countries and cities", async () => {
    await load();
    const us = await centroidOf("US");
    await page.mouse.move(us.x, us.y);
    const tooltip = page.getByRole("tooltip");
    await tooltip.waitFor();
    assert.match(await tooltip.textContent(), /United States.*1,200 sign-ins.*1,100 allowed.*100 denied.*12 users/);
    await page.mouse.move(5, 5);
    await tooltip.waitFor({ state: "detached" });

    await page.locator("[data-point='DE/Berlin']").hover({ force: true });
    assert.match(await page.getByRole("tooltip").textContent(), /Berlin, Germany.*300 sign-ins/);
  });

  it("highlights the country of a hovered list row and links to the filtered logs", async () => {
    await load();
    const list = page.getByRole("list");
    const links = list.getByRole("link");
    assert.equal(await links.count(), 4);
    assert.equal(
      await links.nth(1).getAttribute("href"),
      "/visibility/authenticationlogs?countryCode=DE",
    );
    const outlines = page.locator("svg[role=group] > path[fill='none']");
    assert.equal(await outlines.count(), 0);
    await links.nth(1).hover();
    await outlines.first().waitFor();
    assert.equal(await outlines.count(), 1);
    await links.nth(1).click();
    await waitLocation("/visibility/authenticationlogs?countryCode=DE");
  });

  it("navigates by clicking a country or with the keyboard", async () => {
    await load();
    const br = await centroidOf("BR");
    await page.mouse.click(br.x, br.y);
    await waitLocation("/visibility/authenticationlogs?countryCode=BR");

    await load();
    await page.locator("path[data-code='DE']").focus();
    assert.match(await page.getByRole("tooltip").textContent(), /Germany.*300 sign-ins/);
    await page.keyboard.press("Escape");
    await page.getByRole("tooltip").waitFor({ state: "detached" });
    await page.keyboard.press("Enter");
    await waitLocation("/visibility/authenticationlogs?countryCode=DE");
  });

  it("shows why the map is empty", async () => {
    await load({ scenario: "empty" });
    await page.getByText("No located sign-ins in the last 1h").waitFor();
    await page.getByText("5 sign-ins without GeoIP information.").waitFor();
    assert.equal(await page.locator("[data-code]").count(), 0);
    assert.equal(await page.getByRole("list").count(), 0);
  });

  it("traces a Session's sign-ins and flags impossible travel", async () => {
    await load({ scenario: "session" });
    const svg = page.getByRole("group", { name: "Map of the Session's sign-in locations" });
    const [, , w, h] = (await svg.getAttribute("viewBox")).split(" ").map(Number);
    assert.ok(w < 1000);
    assert.ok(Math.abs(w / h - 1000 / 440) < 1e-3);
    assert.equal(await svg.locator("[data-point]").count(), 3);
    assert.equal(await svg.locator("path[stroke-dasharray]").count(), 2);
    await page.getByText("Impossible travel", { exact: true }).waitFor();
    await page.getByText("3 locations").waitFor();
    await page.getByText("2 countries").waitFor();
    const items = page.getByRole("list", { name: "Sign-in locations, newest first" }).getByRole("listitem");
    assert.equal(await items.count(), 3);
    assert.match(await items.nth(0).textContent(), /San Francisco, United States.*Current.*203\.0\.113\.10.*from previous in 1h.*faster than any flight/);
    assert.match(await items.nth(1).textContent(), /Paris, France.*198\.51\.100\.7/);
    assert.doesNotMatch(await items.nth(1).textContent(), /faster than any flight/);
  });

  for (const theme of ["light", "dark"]) {
    for (const width of [320, 768, 1440]) {
      it(`fits ${width}px in the ${theme} theme`, async () => {
        await page.setViewportSize({ width, height: 1000 });
        for (const scenario of ["panel", "session"]) {
          await load({ theme, scenario });
          const overflow = await page.evaluate(
            () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
          );
          assert.ok(overflow <= 0, `${scenario} overflows by ${overflow}px`);
          const box = await page.locator("svg[role=group]").boundingBox();
          assert.ok(box.width > 200 && box.width <= width);
          assert.ok(Math.abs(box.width / box.height - 1000 / 440) < 0.05);
        }
      });
    }
  }

  it("follows the color scheme", async () => {
    const colors = {};
    for (const theme of ["light", "dark"]) {
      await load({ theme });
      colors[theme] = await page.evaluate(() => {
        const land = document.querySelector("svg[role=group] > g path");
        const active = document.querySelector("path[data-code='US']");
        return {
          land: getComputedStyle(land).fill,
          active: active.getAttribute("fill"),
          body: getComputedStyle(document.body).backgroundColor,
        };
      });
    }
    assert.notEqual(colors.light.land, colors.dark.land);
    assert.notEqual(colors.light.active, colors.dark.active);
    assert.notEqual(colors.light.land, colors.light.body);
  });
});
