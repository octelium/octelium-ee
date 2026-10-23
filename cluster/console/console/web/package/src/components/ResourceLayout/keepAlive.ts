import * as React from "react";

export interface Disposable {
  dispose: () => void;
}

export class KeepAliveStore {
  private items = new Map<string, Disposable>();
  private timer = 0;

  ensure<T extends Disposable>(key: string, create: () => T): T {
    let ret = this.items.get(key) as T | undefined;
    if (!ret) {
      ret = create();
      this.items.set(key, ret);
    }
    return ret;
  }

  release(key: string) {
    this.items.get(key)?.dispose();
    this.items.delete(key);
  }

  retain() {
    window.clearTimeout(this.timer);
  }

  scheduleDispose() {
    window.clearTimeout(this.timer);
    this.timer = window.setTimeout(() => {
      for (const item of this.items.values()) item.dispose();
      this.items.clear();
    }, 0);
  }
}

export const KeepAliveContext = React.createContext<KeepAliveStore | undefined>(
  undefined,
);

export const useKeepAliveStore = () => React.useContext(KeepAliveContext);
