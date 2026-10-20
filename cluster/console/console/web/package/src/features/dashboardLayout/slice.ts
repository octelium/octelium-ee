import { PayloadAction, createSlice } from "@reduxjs/toolkit";

export interface StoredLayout {
  order: string[];
  hidden: string[];
  collapsed: string[];
}

export interface DashboardLayoutState {
  key?: string;
  layouts: Record<string, StoredLayout>;
}

const STORAGE_PREFIX = "octelium.console.layout.v1";
const MAX_IDS = 64;
const MAX_ID_LENGTH = 64;

export const layoutStorageKey = (domain: string, userUID: string) =>
  `${STORAGE_PREFIX}:${domain}:${userUID}`;

const isIDs = (value: unknown): value is string[] =>
  Array.isArray(value) &&
  value.length <= MAX_IDS &&
  value.every(
    (itm) =>
      typeof itm === "string" && itm.length > 0 && itm.length <= MAX_ID_LENGTH,
  );

export const parseLayouts = (raw: string | null) => {
  const ret: Record<string, StoredLayout> = {};
  if (!raw) return ret;

  let value: unknown;
  try {
    value = JSON.parse(raw);
  } catch {
    return ret;
  }
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return ret;
  }

  for (const [page, layout] of Object.entries(value)) {
    if (typeof layout !== "object" || layout === null) continue;
    const { order, hidden, collapsed } = layout as Record<string, unknown>;
    if (isIDs(order) && isIDs(hidden) && isIDs(collapsed)) {
      ret[page] = { order, hidden, collapsed };
    }
  }

  return ret;
};

export const loadLayouts = (key: string) => {
  try {
    return parseLayouts(localStorage.getItem(key));
  } catch {
    return {};
  }
};

export const saveLayouts = (
  key: string,
  layouts: Record<string, StoredLayout>,
) => {
  try {
    if (Object.keys(layouts).length === 0) localStorage.removeItem(key);
    else localStorage.setItem(key, JSON.stringify(layouts));
  } catch {
    return;
  }
};

export const slice = createSlice({
  name: "dashboardLayout",
  initialState: { layouts: {} } as DashboardLayoutState,
  reducers: {
    hydrateLayouts: (
      state,
      action: PayloadAction<{
        key: string;
        layouts: Record<string, StoredLayout>;
      }>,
    ) => {
      state.key = action.payload.key;
      state.layouts = action.payload.layouts;
    },

    setLayout: (
      state,
      action: PayloadAction<{ page: string; layout?: StoredLayout }>,
    ) => {
      if (action.payload.layout) {
        state.layouts[action.payload.page] = action.payload.layout;
      } else {
        delete state.layouts[action.payload.page];
      }
    },
  },
});

export const { hydrateLayouts, setLayout } = slice.actions;

export default slice.reducer;
