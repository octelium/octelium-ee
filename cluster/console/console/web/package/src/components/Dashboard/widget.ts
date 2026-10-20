import * as React from "react";

export interface WidgetControls {
  id: string;
  locked: boolean;
  collapse: () => void;
  hide: () => void;
  customize: () => void;
}

export const WidgetContext = React.createContext<WidgetControls | undefined>(
  undefined,
);

export const useWidget = () => React.useContext(WidgetContext);
