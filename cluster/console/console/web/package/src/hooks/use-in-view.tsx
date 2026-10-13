import * as React from "react";

const ROOT_MARGIN = "400px 0px";

export function useInView<T extends HTMLElement>(
  once = true,
  initialInView = false,
) {
  const ref = React.useRef<T | null>(null);
  const [inView, setInView] = React.useState(initialInView);
  const done = once && inView;

  React.useEffect(() => {
    const element = ref.current;
    if (!element || done) return;

    if (typeof IntersectionObserver === "undefined") {
      setInView(true);
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries.some((entry) => entry.isIntersecting);
        if (visible || !once) setInView(visible);
      },
      { rootMargin: ROOT_MARGIN },
    );
    observer.observe(element);

    return () => observer.disconnect();
  }, [once, done]);

  return { ref, inView };
}
