export interface JSONSchema {
  type?: string | string[];
  properties?: { [key: string]: JSONSchema };
  items?: JSONSchema | JSONSchema[];
  required?: string[];
  description?: string;
  title?: string;
  enum?: any[];
  default?: any;
  format?: string;

  $ref?: string;

  oneOf?: JSONSchema[];
  allOf?: JSONSchema[];
  anyOf?: JSONSchema[];

  [key: string]: any;
}
