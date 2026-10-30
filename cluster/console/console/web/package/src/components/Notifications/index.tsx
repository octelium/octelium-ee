import "@mantine/notifications/styles.layer.css";

import { Notifications as MantineNotifications } from "@mantine/notifications";

export default function Notifications() {
  return (
    <MantineNotifications position="bottom-center" limit={3} autoClose={4000} />
  );
}
