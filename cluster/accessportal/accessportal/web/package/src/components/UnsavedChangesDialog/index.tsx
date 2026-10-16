import { ConfirmDialog } from "@/ui";
import { useUnsavedChanges } from "@/utils/hooks/useUnsavedChanges";

const UnsavedChangesDialog = (props: {
  guard: ReturnType<typeof useUnsavedChanges>;
}) => (
  <ConfirmDialog
    opened={props.guard.opened}
    onClose={props.guard.onClose}
    onConfirm={props.guard.onConfirm}
    title="Discard unsaved changes?"
    description="This form has changes that have not been saved. Leaving now discards them."
    cancelLabel="Keep editing"
    confirmLabel="Discard changes"
    danger
  />
);

export default UnsavedChangesDialog;
