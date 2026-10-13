import * as React from "react";
import { createPortal } from "react-dom";

interface PageHeaderSlots {
  actions: HTMLElement | null;
  addon: HTMLElement | null;
}

interface PageHeaderSetters {
  setActions: (element: HTMLElement | null) => void;
  setAddon: (element: HTMLElement | null) => void;
  setDetails: (details: Record<string, string> | undefined) => void;
}

const SlotsContext = React.createContext<PageHeaderSlots>({
  actions: null,
  addon: null,
});

const DetailsContext = React.createContext<Record<string, string> | undefined>(
  undefined,
);

const SettersContext = React.createContext<PageHeaderSetters>({
  setActions: () => {},
  setAddon: () => {},
  setDetails: () => {},
});

export const PageHeaderProvider = (props: { children: React.ReactNode }) => {
  const [actions, setActions] = React.useState<HTMLElement | null>(null);
  const [addon, setAddon] = React.useState<HTMLElement | null>(null);
  const [details, setDetails] = React.useState<Record<string, string>>();

  const slots = React.useMemo(() => ({ actions, addon }), [actions, addon]);
  const setters = React.useMemo(
    () => ({ setActions, setAddon, setDetails }),
    [],
  );

  return (
    <SettersContext.Provider value={setters}>
      <SlotsContext.Provider value={slots}>
        <DetailsContext.Provider value={details}>
          {props.children}
        </DetailsContext.Provider>
      </SlotsContext.Provider>
    </SettersContext.Provider>
  );
};

export const usePageHeaderSetters = () => React.useContext(SettersContext);

export const usePageDetailsValue = () => React.useContext(DetailsContext);

export const usePageDetails = (details?: Record<string, string>) => {
  const { setDetails } = React.useContext(SettersContext);
  const serialized = details ? JSON.stringify(details) : "";

  React.useEffect(() => {
    setDetails(serialized ? JSON.parse(serialized) : undefined);
    return () => setDetails(undefined);
  }, [serialized, setDetails]);
};

export const PageActions = (props: { children: React.ReactNode }) => {
  const { actions } = React.useContext(SlotsContext);
  return actions ? createPortal(props.children, actions) : null;
};

export const PageTitleAddon = (props: { children: React.ReactNode }) => {
  const { addon } = React.useContext(SlotsContext);
  return addon ? createPortal(props.children, addon) : null;
};
