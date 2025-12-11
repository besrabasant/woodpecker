<template>
  <div v-if="tab">
    <component
      v-for="(componentMeta, index) in tab.components"
      :is="resolveComponent(componentMeta.type)"
      v-bind="componentMeta.props"
      :key="`${componentMeta.type}-${index}`"
    />
  </div>
  <Panel v-else>
    <p class="text-sm text-wp-text-alt-100">{{ $t('repo.pipeline.tabs.empty') }}</p>
  </Panel>
</template>

<script lang="ts" setup>
import type { Ref } from 'vue';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import Panel from '~/components/layout/Panel.vue';
import { requiredInject } from '~/compositions/useInjectProvide';
import type { PipelineTabDefinition } from '~/lib/api/types';
import { pipelineTabComponentRegistry } from '~/views/repo/pipeline/tabs/registry';

const route = useRoute();
useI18n();

const pipelineTabs = requiredInject('pipeline-tabs') as Ref<PipelineTabDefinition[]>;

const tab = computed(() => pipelineTabs.value.find((entry) => entry.id === route.params.tabId));

function resolveComponent(type: string) {
  return pipelineTabComponentRegistry[type] ?? Panel;
}
</script>
