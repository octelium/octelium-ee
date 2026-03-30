import React, { createContext, ReactNode, useContext, useState } from "react";
import { JSONSchema } from "./Schema";

interface SchemaContextValue {
  rootSchema: JSONSchema | null;
  renderedRefs: Set<string>;
  addRenderedRef: (ref: string) => void;
  resolveRef: (ref: string) => JSONSchema | null;
}

const SchemaContext = createContext<SchemaContextValue | undefined>(undefined);

export const useSchemaContext = () => {
  const context = useContext(SchemaContext);
  if (!context) {
    throw new Error("useSchemaContext must be used within a SchemaProvider");
  }
  return context;
};

interface SchemaProviderProps {
  schema: JSONSchema;
  children: ReactNode;
}

export const SchemaProvider: React.FC<SchemaProviderProps> = ({
  schema,
  children,
}) => {
  const [renderedRefs, setRenderedRefs] = useState<Set<string>>(new Set());

  const addRenderedRef = (ref: string) => {
    setRenderedRefs((prev) => new Set(prev).add(ref));
  };

  const resolveRef = (ref: string): JSONSchema | null => {
    if (!ref || !ref.startsWith("#/")) {
      console.warn(`Unsupported or invalid $ref format: ${ref}`);
      return null;
    }
    const path = ref.substring(2).split("/");
    let current: any = schema;
    try {
      for (const segment of path) {
        if (current === null || typeof current !== "object") return null;
        const decodedSegment = decodeURIComponent(
          segment.replace(/~1/g, "/").replace(/~0/g, "~"),
        );
        if (!(decodedSegment in current)) {
          console.warn(
            `Could not resolve $ref segment: ${decodedSegment} in path ${ref}`,
          );
          return null;
        }
        current = current[decodedSegment];
      }
      return current as JSONSchema;
    } catch (error) {
      console.error(`Error resolving $ref: ${ref}`, error);
      return null;
    }
  };

  const value = {
    rootSchema: schema,
    renderedRefs,
    addRenderedRef,
    resolveRef,
  };

  return (
    <SchemaContext.Provider value={value}>{children}</SchemaContext.Provider>
  );
};
