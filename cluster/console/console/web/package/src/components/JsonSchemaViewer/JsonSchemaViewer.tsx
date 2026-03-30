import React from "react";
import "./JsonSchemaViewer.css";
import { JSONSchema } from "./Schema";
import { SchemaProvider } from "./SchemaContext";
import SchemaNode from "./SchemaNode";

interface JsonSchemaViewerProps {
  schema: JSONSchema;
  title?: string;
}

const JsonSchemaViewer: React.FC<JsonSchemaViewerProps> = ({
  schema,
  title,
}) => {
  if (!schema) {
    return <div className="json-schema-viewer">No schema provided.</div>;
  }

  const viewerTitle = title || schema.title || "Schema Documentation";

  return (
    <SchemaProvider schema={schema}>
      <div className="json-schema-viewer">
        <h2>{viewerTitle}</h2>
        {schema.description && !schema.$ref && (
          <p className="schema-main-description">{schema.description}</p>
        )}
        <SchemaNode schema={schema} level={0} />
      </div>
    </SchemaProvider>
  );
};

export default JsonSchemaViewer;
