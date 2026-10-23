import * as React from "react";
import { KeepAliveContext, KeepAliveStore } from "./keepAlive";

const KeepAliveScope = (props: { children: React.ReactNode }) => {
  const [store] = React.useState(() => new KeepAliveStore());

  React.useEffect(() => {
    store.retain();
    return () => store.scheduleDispose();
  }, [store]);

  return (
    <KeepAliveContext.Provider value={store}>
      {props.children}
    </KeepAliveContext.Provider>
  );
};

export default KeepAliveScope;
