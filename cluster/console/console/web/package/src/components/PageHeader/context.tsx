import * as React from "react";
import { createPortal } from "react-dom";

interface PageHeaderSlots {
  actions: HTMLElement | null;
  addon: HTMLElement | null;
}

export interface PageCondition {
  label: string;
  count: number;
  to?: string;
}

interface PageHeaderSetters {
  setActions: (element: HTMLElement | null) => void;
  setAddon: (element: HTMLElement | null) => void;
  setDetails: (id: string, details?: Record<string, string>) => void;
  setConditions: (id: string, conditions?: PageCondition[]) => void;
}

const SlotsContext = React.createContext<PageHeaderSlots>({
  actions: null,
  addon: null,
});

const DetailsContext = React.createContext<Record<string, string> | undefined>(
  undefined,
);

const ConditionsContext = React.createContext<PageCondition[]>([]);

const SettersContext = React.createContext<PageHeaderSetters>({
  setActions: () => {},
  setAddon: () => {},
  setDetails: () => {},
  setConditions: () => {},
});

const useSources = <T,>() => {
  const [sources, setSources] = React.useState<Record<string, T>>({});
  const set = React.useCallback(
    (id: string, value?: T) =>
      setSources((current) => {
        if (value === undefined) {
          if (!(id in current)) return current;
          const { [id]: _, ...rest } = current;
          return rest;
        }
        return { ...current, [id]: value };
      }),
    [],
  );
  return [sources, set] as const;
};

export const PageHeaderProvider = (props: { children: React.ReactNode }) => {
  const [actions, setActions] = React.useState<HTMLElement | null>(null);
  const [addon, setAddon] = React.useState<HTMLElement | null>(null);
  const [detailSources, setDetails] = useSources<Record<string, string>>();
  const [conditionSources, setConditions] = useSources<PageCondition[]>();

  const slots = React.useMemo(() => ({ actions, addon }), [actions, addon]);
  const setters = React.useMemo(
    () => ({ setActions, setAddon, setDetails, setConditions }),
    [setDetails, setConditions],
  );
  const details = React.useMemo(() => {
    const values = Object.values(detailSources);
    return values.length > 0 ? Object.assign({}, ...values) : undefined;
  }, [detailSources]);
  const conditions = React.useMemo(
    () =>
      Object.values(conditionSources)
        .flat()
        .sort((a, b) => b.count - a.count),
    [conditionSources],
  );

  return (
    <SettersContext.Provider value={setters}>
      <SlotsContext.Provider value={slots}>
        <DetailsContext.Provider value={details}>
          <ConditionsContext.Provider value={conditions}>
            {props.children}
          </ConditionsContext.Provider>
        </DetailsContext.Provider>
      </SlotsContext.Provider>
    </SettersContext.Provider>
  );
};

export const usePageHeaderSetters = () => React.useContext(SettersContext);

export const usePageDetailsValue = () => React.useContext(DetailsContext);

export const usePageDetails = (details?: Record<string, string>) => {
  const { setDetails } = React.useContext(SettersContext);
  const id = React.useId();
  const serialized = details ? JSON.stringify(details) : "";

  React.useEffect(() => {
    setDetails(id, serialized ? JSON.parse(serialized) : undefined);
    return () => setDetails(id, undefined);
  }, [id, serialized, setDetails]);
};

export const usePageConditionsValue = () => React.useContext(ConditionsContext);

export const usePageConditions = (conditions: PageCondition[]) => {
  const { setConditions } = React.useContext(SettersContext);
  const id = React.useId();
  const serialized = JSON.stringify(conditions);

  React.useEffect(() => {
    const value: PageCondition[] = JSON.parse(serialized);
    setConditions(id, value.length > 0 ? value : undefined);
    return () => setConditions(id, undefined);
  }, [id, serialized, setConditions]);
};

export const PageActions = (props: { children: React.ReactNode }) => {
  const { actions } = React.useContext(SlotsContext);
  return actions ? createPortal(props.children, actions) : null;
};

export const PageTitleAddon = (props: { children: React.ReactNode }) => {
  const { addon } = React.useContext(SlotsContext);
  return addon ? createPortal(props.children, addon) : null;
};
