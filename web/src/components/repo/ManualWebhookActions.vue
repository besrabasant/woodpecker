<template>
  <Panel>
    <div class="flex flex-col gap-4">
      <div>
        <span class="text-wp-text-100 text-xl">{{ $t('repo.manual_pipeline.webhooks.title') }}</span>
        <p class="text-sm text-wp-text-alt-100">
          {{ $t('repo.manual_pipeline.webhooks.desc') }}
        </p>
      </div>
      <div v-if="loading" class="text-wp-text-100 flex justify-center">
        <Icon name="spinner" />
      </div>
      <div v-else-if="manualActions.length === 0" class="text-wp-text-alt-100">
        {{ $t('repo.manual_pipeline.webhooks.empty') }}
      </div>
      <div v-else class="space-y-6">
        <div
          v-for="action in manualActions"
          :key="action.id"
          class="rounded border border-wp-border-default p-4"
        >
          <div class="mb-3 flex flex-col gap-1">
            <span class="text-lg text-wp-text-100">{{ action.title }}</span>
            <span class="text-sm text-wp-text-alt-100">{{ action.description }}</span>
            <span class="text-xs text-wp-text-alt-100 uppercase">
              {{ action.request.method }} · {{ action.request.url }}
            </span>
          </div>
          <div class="space-y-3">
            <div v-if="action.request.query?.length">
              <p class="mb-1 text-sm font-semibold text-wp-text-100">
                {{ $t('repo.manual_pipeline.webhooks.sections.query') }}
              </p>
              <div class="grid gap-3 md:grid-cols-2">
                <InputField v-for="field in action.request.query" :key="field.key" :label="field.label">
                  <TextField
                    :model-value="getManualActionValue(action.id, 'query', field.key)"
                    :placeholder="field.placeholder"
                    :required="field.required"
                    @update:model-value="setManualActionValue(action.id, 'query', field.key, $event)"
                  />
                </InputField>
              </div>
            </div>
            <div v-if="action.request.headers?.length">
              <p class="mb-1 text-sm font-semibold text-wp-text-100">
                {{ $t('repo.manual_pipeline.webhooks.sections.headers') }}
              </p>
              <div class="grid gap-3 md:grid-cols-2">
                <InputField v-for="field in action.request.headers" :key="field.key" :label="field.label">
                  <TextField
                    :model-value="getManualActionValue(action.id, 'headers', field.key)"
                    :placeholder="field.placeholder"
                    :required="field.required"
                    @update:model-value="setManualActionValue(action.id, 'headers', field.key, $event)"
                  />
                </InputField>
              </div>
            </div>
            <div v-if="action.request.body?.length">
              <p class="mb-1 text-sm font-semibold text-wp-text-100">
                {{ $t('repo.manual_pipeline.webhooks.sections.body') }}
              </p>
              <div class="grid gap-3 md:grid-cols-2">
                <InputField v-for="field in action.request.body" :key="field.key" :label="field.label">
                  <TextField
                    :model-value="getManualActionValue(action.id, 'body', field.key)"
                    :placeholder="field.placeholder"
                    :required="field.required"
                    @update:model-value="setManualActionValue(action.id, 'body', field.key, $event)"
                  />
                </InputField>
              </div>
            </div>
          </div>
          <div class="mt-4">
            <Button
              :text="$t('repo.manual_pipeline.webhooks.trigger', { action: action.title })"
              :is-loading="manualActionStates[action.id]?.isLoading"
              :disabled="!isManualActionValid(action)"
              @click="triggerManualAction(action)"
            />
          </div>
        </div>
      </div>
    </div>
  </Panel>
</template>

<script lang="ts" setup>
import { useNotification } from '@kyvg/vue3-notification';
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import Button from '~/components/atomic/Button.vue';
import Icon from '~/components/atomic/Icon.vue';
import InputField from '~/components/form/InputField.vue';
import TextField from '~/components/form/TextField.vue';
import Panel from '~/components/layout/Panel.vue';
import useApiClient from '~/compositions/useApiClient';
import { requiredInject } from '~/compositions/useInjectProvide';
import type { ManualActionDefinition, ManualActionField, ManualActionTriggerPayload } from '~/lib/api/types';

const apiClient = useApiClient();
const notifications = useNotification();
const i18n = useI18n();

const repo = requiredInject('repo');

const manualActions = ref<ManualActionDefinition[]>([]);
const manualActionStates = ref<
  Record<
    string,
    {
      values: ManualActionTriggerPayload;
      isLoading: boolean;
    }
  >
>({});
const loading = ref(true);

onMounted(async () => {
  await loadManualActions();
  loading.value = false;
});

function buildDefaultSectionValues(fields?: ManualActionField[]) {
  const defaults: Record<string, string> = {};
  fields?.forEach((field) => {
    defaults[field.key] = field.default ?? '';
  });
  return defaults;
}

async function loadManualActions() {
  try {
    manualActions.value = await apiClient.getManualActions(repo.value.id);
    manualActions.value.forEach((action) => {
      manualActionStates.value[action.id] = {
        values: {
          query: buildDefaultSectionValues(action.request.query),
          headers: buildDefaultSectionValues(action.request.headers),
          body: buildDefaultSectionValues(action.request.body),
        },
        isLoading: false,
      };
    });
  } catch (error) {
    console.error('Failed to load manual actions', error);
  }
}

type ManualActionSection = keyof ManualActionTriggerPayload;

function ensureSection(actionId: string, section: ManualActionSection) {
  if (!manualActionStates.value[actionId]) {
    manualActionStates.value[actionId] = {
      values: {},
      isLoading: false,
    };
  }
  if (!manualActionStates.value[actionId].values[section]) {
    manualActionStates.value[actionId].values[section] = {};
  }
}

function setManualActionValue(actionId: string, section: ManualActionSection, key: string, value: string) {
  ensureSection(actionId, section);
  manualActionStates.value[actionId].values[section]![key] = value;
}

function getManualActionValue(actionId: string, section: ManualActionSection, key: string) {
  return manualActionStates.value[actionId]?.values?.[section]?.[key] ?? '';
}

function isManualActionValid(action: ManualActionDefinition) {
  const state = manualActionStates.value[action.id];
  if (!state) {
    return false;
  }
  const sections: ManualActionSection[] = ['query', 'headers', 'body'];
  for (const section of sections) {
    const fields = action.request[section];
    if (!fields || fields.length === 0) {
      continue;
    }
    for (const field of fields) {
      if (!field.required) {
        continue;
      }
      const value = state.values[section]?.[field.key] ?? field.default ?? '';
      if (value.trim() === '') {
        return false;
      }
    }
  }
  return true;
}

async function triggerManualAction(action: ManualActionDefinition) {
  const state = manualActionStates.value[action.id];
  if (!state) {
    return;
  }
  state.isLoading = true;
  try {
    await apiClient.triggerManualAction(repo.value.id, action.id, state.values);
    notifications.notify({
      type: 'success',
      title: i18n.t('repo.manual_pipeline.webhooks.success', { action: action.title }),
    });
  } catch (error) {
    console.error('Failed to trigger manual action', error);
    notifications.notify({
      type: 'error',
      title: i18n.t('repo.manual_pipeline.webhooks.error', { action: action.title }),
    });
  } finally {
    state.isLoading = false;
  }
}
</script>
