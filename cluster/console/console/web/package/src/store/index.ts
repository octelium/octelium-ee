import { configureStore } from "@reduxjs/toolkit";

import dashboardLayoutReducer from "../features/dashboardLayout/slice";
import settingsReducer from "../features/settings/slice";

const store = configureStore({
  reducer: {
    settings: settingsReducer,
    dashboardLayout: dashboardLayoutReducer,
  },
});

export default store;

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
