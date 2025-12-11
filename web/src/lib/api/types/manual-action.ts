export interface ManualActionField {
  key: string;
  label: string;
  required: boolean;
  placeholder?: string;
  default?: string;
  type?: string;
}

export interface ManualActionRequestDefinition {
  url: string;
  method: string;
  bodyType?: string;
  query?: ManualActionField[];
  headers?: ManualActionField[];
  body?: ManualActionField[];
}

export interface ManualActionDefinition {
  id: string;
  title: string;
  description?: string;
  request: ManualActionRequestDefinition;
}

export interface ManualActionTriggerPayload {
  query?: Record<string, string>;
  headers?: Record<string, string>;
  body?: Record<string, string>;
  [key: string]: string | Record<string, string> | undefined;
}
