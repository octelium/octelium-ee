import { Button, Group } from "@mantine/core";
import { notifications } from "@mantine/notifications";

export function showActionNotification(
  message: string,
  action: { label: string; onClick: () => void },
) {
  const id = notifications.show({
    message: (
      <Group gap="sm" justify="space-between" wrap="nowrap">
        <span className="min-w-0 flex-1">{message}</span>
        <Button
          size="compact-xs"
          variant="light"
          onClick={() => {
            notifications.hide(id);
            action.onClick();
          }}
        >
          {action.label}
        </Button>
      </Group>
    ),
  });

  return id;
}
