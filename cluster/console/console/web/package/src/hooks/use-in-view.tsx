import * as React from "react";

const ROOT_MARGIN = "400px 0px";

export function useInView<T extends HTMLElement>() {
  const ref = React.useRef<T | null>(null);
  const [inView, setInView] = React.useState(false);

  React.useEffect(() => {
    const element = ref.current;
    if (!element || inView) return;

    if (typeof IntersectionObserver === "undefined") {
      setInView(true);
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) setInView(true);
      },
      { rootMargin: ROOT_MARGIN },
    );
    observer.observe(element);

    return () => observer.disconnect();
  }, [inView]);

  return { ref, inView };
}
