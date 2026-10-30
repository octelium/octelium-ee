import "./index.css";
import "@mantine/core/styles.layer.css";

import React from "react";
import ReactDOM from "react-dom/client";
import { MotionConfig } from "framer-motion";

import { MantineProvider } from "@mantine/core";

import { Provider } from "react-redux";
import { RouterProvider } from "react-router-dom";

import Notifications from "@/components/Notifications";
import store from "@/store";

import router from "@/router";
import { queryClient } from "@/utils";
import { installGlobalErrorLogging } from "@/utils/errors";
import {
  colorSchemeManager,
  DEFAULT_COLOR_SCHEME,
} from "@/utils/theme/colorScheme";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import { QueryClientProvider } from "@tanstack/react-query";

installGlobalErrorLogging();

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MotionConfig reducedMotion="user">
      <Provider store={store}>
        <MantineProvider
          theme={themeMantine}
          colorSchemeManager={colorSchemeManager}
          defaultColorScheme={DEFAULT_COLOR_SCHEME}
          cssVariablesResolver={cssVariablesResolver}
        >
          <Notifications />
          <QueryClientProvider client={queryClient}>
            <RouterProvider router={router} />
          </QueryClientProvider>
        </MantineProvider>
      </Provider>
    </MotionConfig>
  </React.StrictMode>,
);
