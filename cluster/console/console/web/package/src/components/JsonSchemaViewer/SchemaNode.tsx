import React from "react";
import { JSONSchema } from "./Schema";
import { useSchemaContext } from "./SchemaContext";

interface SchemaNodeProps {
  schema: JSONSchema | null;
  name?: string;
  isRequired?: boolean;
  level?: number;
  isRefDefinition?: boolean;
  refPath?: string;
}

const getRefName = (refPath: string): string => {
  const parts = refPath.split("/");
  return parts[parts.length - 1] || refPath;
};

const SchemaNode: React.FC<SchemaNodeProps> = ({
  schema: currentSchema,
  name,
  isRequired = false,
  level = 0,
  isRefDefinition = false,
  refPath: originalRefPath,
}) => {
  const { resolveRef, renderedRefs, addRenderedRef } = useSchemaContext();
  const indent = { paddingLeft: `${level * 20}px` };

  if (currentSchema && currentSchema.$ref) {
    const refPath = currentSchema.$ref;
    const resolvedSchema = resolveRef(refPath);

    if (!resolvedSchema) {
      return (
        <div style={indent} className="schema-node schema-error">
          <strong>{name}: </strong>
          <span style={{ color: "red" }}>
            Error: Could not resolve $ref: {refPath}
          </span>
        </div>
      );
    }

    if (renderedRefs.has(refPath)) {
      return (
        <div style={indent} className="schema-node schema-ref-link">
          {name && (
            <strong>
              {name}
              {isRequired && <span style={{ color: "red" }}>*</span>}:{" "}
            </strong>
          )}
          <span style={{ fontStyle: "italic", color: "#666" }}>
            Ref: <code title={refPath}>{getRefName(refPath)}</code> (see
            definition below or above)
          </span>
          {currentSchema.description && (
            <div
              style={{
                marginLeft: "15px",
                fontSize: "0.9em",
                color: "#555",
                fontStyle: "italic",
              }}
            >
              {currentSchema.description}
            </div>
          )}
        </div>
      );
    }

    addRenderedRef(refPath);
    return (
      <SchemaNode
        schema={resolvedSchema}
        name={name}
        isRequired={isRequired}
        level={level}
        isRefDefinition={true}
        refPath={refPath}
      />
    );
  }

  if (!currentSchema) {
    return (
      <div style={indent} className="schema-node schema-error">
        {name && <strong>{name}: </strong>}
        <span style={{ color: "red" }}>
          Invalid schema or unresolved reference.
        </span>
      </div>
    );
  }

  const {
    type,
    description,
    properties,
    items,
    required,
    enum: enumValues,
    default: defaultValue,
    format,
    title: schemaTitle,
    oneOf,
    allOf,
    anyOf,
  } = currentSchema;

  const displayType = Array.isArray(type) ? type.join(" | ") : type;

  const renderDetails = (detailsSchema: JSONSchema = currentSchema) => (
    <div style={{ marginLeft: "15px", fontSize: "0.9em", color: "#555" }}>
      {detailsSchema.description && (
        <p style={{ margin: "2px 0" }}>{detailsSchema.description}</p>
      )}
      {detailsSchema.format && (
        <p style={{ margin: "2px 0" }}>
          <em>Format:</em> {format}
        </p>
      )}
      {detailsSchema.enum && (
        <p style={{ margin: "2px 0" }}>
          <em>Enum:</em>{" "}
          <code>
            {detailsSchema.enum.map((e) => JSON.stringify(e)).join(", ")}
          </code>
        </p>
      )}
      {detailsSchema.default !== undefined && (
        <p style={{ margin: "2px 0" }}>
          <em>Default:</em> <code>{JSON.stringify(detailsSchema.default)}</code>
        </p>
      )}
    </div>
  );

  const renderComposition = (
    keyword: "oneOf" | "allOf" | "anyOf",
    subSchemas: JSONSchema[],
  ) => (
    <div className={`schema-composition schema-${keyword}`}>
      <strong style={{ textTransform: "capitalize" }}>
        {keyword.replace("Of", " of")}:
      </strong>
      <div
        style={{
          borderLeft: "2px dotted #ccc",
          marginLeft: "5px",
          paddingLeft: "15px",
        }}
      >
        {subSchemas.map((subSchema, index) => (
          <SchemaNode key={index} schema={subSchema} level={level + 1} />
        ))}
      </div>
    </div>
  );

  return (
    <div
      style={indent}
      className={`schema-node ${
        isRefDefinition ? "schema-ref-definition" : ""
      }`}
    >
      <div style={{ marginBottom: "5px" }}>
        {isRefDefinition && originalRefPath && (
          <div
            style={{ fontSize: "0.8em", color: "#888", marginBottom: "3px" }}
          >
            (Definition for{" "}
            <code title={originalRefPath}>{getRefName(originalRefPath)}</code>)
          </div>
        )}
        {name && (
          <strong>
            {name}
            {isRequired && <span style={{ color: "red" }}>*</span>}:{" "}
          </strong>
        )}
        {displayType && (
          <span style={{ fontStyle: "italic" }}>{displayType}</span>
        )}
        {schemaTitle && ` (${schemaTitle})`}
        {!displayType &&
          !oneOf &&
          !allOf &&
          !anyOf &&
          !properties &&
          !items && (
            <span style={{ fontStyle: "italic", color: "#888" }}>
              (Type not specified)
            </span>
          )}
      </div>

      {renderDetails()}

      {oneOf && renderComposition("oneOf", oneOf)}
      {allOf && renderComposition("allOf", allOf)}
      {anyOf && renderComposition("anyOf", anyOf)}

      {type === "object" && properties && (
        <div
          className="schema-properties"
          style={{
            borderLeft: "1px solid #eee",
            marginLeft: "5px",
            paddingLeft: "10px",
            marginTop: "5px",
          }}
        >
          {Object.entries(properties).map(([propName, propSchema]) => (
            <SchemaNode
              key={propName}
              schema={propSchema}
              name={propName}
              isRequired={required?.includes(propName)}
              level={level + 1}
            />
          ))}
        </div>
      )}
      {type === "object" && !properties && !oneOf && !allOf && !anyOf && (
        <span style={{ fontStyle: "italic", color: "#888" }}>
          {" "}
          (Object has no properties defined)
        </span>
      )}

      {type === "array" && items && (
        <div
          className="schema-array-items"
          style={{
            borderLeft: "1px solid #eee",
            marginLeft: "5px",
            paddingLeft: "10px",
            marginTop: "5px",
          }}
        >
          {!Array.isArray(items) && (
            <>
              <em>Items:</em>
              <SchemaNode schema={items} level={level + 1} />
            </>
          )}
          {Array.isArray(items) && ( // Tuple validation
            <>
              <em>Items (Tuple):</em>
              {items.map((itemSchema, index) => (
                <SchemaNode
                  key={index}
                  schema={itemSchema}
                  name={`[${index}]`}
                  level={level + 1}
                />
              ))}
            </>
          )}
        </div>
      )}
      {type === "array" && !items && !oneOf && !allOf && !anyOf && (
        <span style={{ fontStyle: "italic", color: "#888" }}>
          {" "}
          (Array has no item type defined)
        </span>
      )}
    </div>
  );
};

export default SchemaNode;
