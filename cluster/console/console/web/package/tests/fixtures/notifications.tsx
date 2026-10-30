import "@/index.css";
import "@mantine/core/styles.layer.css";
import "@fontsource/ubuntu/400.css";
import "@fontsource/ubuntu/500.css";

import Notifications from "@/components/Notifications";
import { showActionNotification } from "@/utils/notifications";
import {
  colorSchemeManager,
  DEFAULT_COLOR_SCHEME,
} from "@/utils/theme/colorScheme";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import {
  Button,
  Group,
  MantineProvider,
  useMantineColorScheme,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import React from "react";
import { createRoot } from "react-dom/client";

function Controls() {
  const { toggleColorScheme } = useMantineColorScheme();
  const [actions, setActions] = React.useState(0);

  return (
    <Group p="md">
      <Button
        onClick={() =>
          notifications.show({
            color: "green",
            message: "Resource updated successfully",
            autoClose: false,
          })
        }
      >
        Show success
      </Button>
      <Button
        onClick={() =>
          notifications.show({
            color: "red",
            message: "The resource could not be updated",
            autoClose: false,
          })
        }
      >
        Show error
      </Button>
      <Button
        onClick={() =>
          showActionNotification("The section is hidden", {
            label: "Undo",
            onClick: () => setActions((current) => current + 1),
          })
        }
      >
        Show action
      </Button>
      <Button
        onClick={() =>
          notifications.show({
            message: `Could not update Service ${"service-name".repeat(30)}`,
            color: "red",
            autoClose: false,
          })
        }
      >
        Show long message
      </Button>
      <Button onClick={() => notifications.show({ message: "This will close" })}>
        Show temporary
      </Button>
      <Button onClick={() => toggleColorScheme()}>Toggle theme</Button>
      <output aria-label="Action count">{actions}</output>
    </Group>
  );
}

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MantineProvider
      theme={themeMantine}
      colorSchemeManager={colorSchemeManager}
      defaultColorScheme={DEFAULT_COLOR_SCHEME}
      cssVariablesResolver={cssVariablesResolver}
    >
      <Notifications />
      <Controls />
    </MantineProvider>
  </React.StrictMode>,
);
