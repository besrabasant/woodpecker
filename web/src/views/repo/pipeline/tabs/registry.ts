import type { Component } from 'vue';
import ReportsSection from './ReportsSection.vue';

export const pipelineTabComponentRegistry: Record<string, Component> = {
  'pipeline-reports-section': ReportsSection,
};
