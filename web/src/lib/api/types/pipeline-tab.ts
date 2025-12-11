export interface PipelineTabComponentMeta {
  type: string;
  props?: Record<string, unknown>;
}

export interface PipelineTabDefinition {
  id: string;
  title: string;
  icon?: string;
  requires_push?: boolean;
  components: PipelineTabComponentMeta[];
}
