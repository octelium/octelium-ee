import * as React from "react";

const listeners = new Set<() => void>();
const dirtyForms = new Set<string>();
let snapshot = false;

const emit = () => {
  const next = dirtyForms.size > 0;
  if (next === snapshot) return;
  snapshot = next;
  listeners.forEach((notify) => notify());
};

const subscribe = (listener: () => void) => {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
};

const getSnapshot = () => snapshot;

export const useRegisterDirtyForm = (dirty: boolean) => {
  const id = React.useId();

  React.useEffect(() => {
    if (dirty) dirtyForms.add(id);
    else dirtyForms.delete(id);
    emit();
  }, [dirty, id]);

  React.useEffect(
    () => () => {
      dirtyForms.delete(id);
      emit();
    },
    [id],
  );
};

export const useHasDirtyForm = () =>
  React.useSyncExternalStore(subscribe, getSnapshot, getSnapshot);

export const RESOURCE_NAME_PATTERN = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/;

export const validateResourceName = (name: string): string | undefined => {
  const short = name.split(".").at(0) ?? "";
  if (short.length === 0) return "A name is required";
  if (short.length > 63) return "Names are limited to 63 characters";
  if (!RESOURCE_NAME_PATTERN.test(short))
    return "Use lowercase letters, digits and dashes, starting and ending with a letter or digit";
  return undefined;
};

const newRowKey = () =>
  globalThis.crypto?.randomUUID?.() ??
  `row-${Date.now()}-${Math.random().toString(36).slice(2)}`;

export const useListKeys = () => {
  const store = React.useRef(new Map<string, string[]>());

  return React.useMemo(
    () => ({
      keyAt: (path: string, index: number) => {
        let keys = store.current.get(path);
        if (!keys) {
          keys = [];
          store.current.set(path, keys);
        }
        while (keys.length <= index) keys.push(newRowKey());
        return keys[index];
      },
      removeAt: (path: string, index: number) => {
        store.current.get(path)?.splice(index, 1);
      },
    }),
    [],
  );
};
