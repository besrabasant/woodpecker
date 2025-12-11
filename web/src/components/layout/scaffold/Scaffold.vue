<template>
  <Header
    :go-back="goBack"
    :enable-tabs="enableTabs"
    :search="search"
    :full-width="fullWidthHeader"
    @update:search="(value) => $emit('update:search', value)"
  >
    <template #title><slot name="title" /></template>
    <template v-if="$slots.headerActions" #headerActions><slot name="headerActions" /></template>
    <template v-if="$slots.tabActions" #tabActions><slot name="tabActions" /></template>
  </Header>

  <slot v-if="computedFluidContent" />
  <Container v-else>
    <slot />
  </Container>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import Container from '~/components/layout/Container.vue';
import { useScaffoldLayoutProvider } from '~/compositions/useScaffoldLayout';
import { useTabsProvider } from '~/compositions/useTabs';

import Header from './Header.vue';

const props = defineProps<{
  // Header
  goBack?: () => void;
  search?: string;
  fullWidthHeader?: boolean;

  // Tabs
  enableTabs?: boolean;

  // Content
  fluidContent?: boolean;
}>();

defineEmits<{
  (event: 'update:search', value: string): void;
}>();

const fluidLayout = useScaffoldLayoutProvider();
const computedFluidContent = computed(() => props.fluidContent || fluidLayout.fluidOverride.value);

if (props.enableTabs) {
  useTabsProvider();
}
</script>
