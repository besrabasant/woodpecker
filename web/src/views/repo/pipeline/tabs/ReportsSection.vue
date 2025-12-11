<template>
  <Panel>
    <div class="flex flex-col gap-4">
      <div>
        <h2 class="text-lg font-semibold text-wp-text-100">{{ $t('repo.pipeline.reports.title') }}</h2>
        <p class="text-sm text-wp-text-alt-100">
          {{ description || $t('repo.pipeline.reports.desc') }}
        </p>
      </div>
      <div class="rounded border border-wp-border-default bg-white">
        <iframe
          :src="iframeSrc"
          class="h-96 w-full"
          frameborder="0"
          sandbox="allow-same-origin allow-scripts"
        />
      </div>
    </div>
  </Panel>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import Panel from '~/components/layout/Panel.vue';
import useConfig from '~/compositions/useConfig';
import { requiredInject } from '~/compositions/useInjectProvide';
import { useScaffoldLayoutConsumer } from '~/compositions/useScaffoldLayout';

const props = defineProps<{
  description?: string;
  endpoint?: string;
}>();

useI18n();

const repo = requiredInject('repo');
const pipeline = requiredInject('pipeline');
const config = useConfig();
useScaffoldLayoutConsumer({ fluid: true });

const iframeSrc = computed(() => {
  const base = config.rootPath ?? '';
  if (props.endpoint) {
    return props.endpoint;
  }
  return `${base}/api/repos/${repo.value.id}/pipelines/${pipeline.value.number}/reports`;
});
</script>
