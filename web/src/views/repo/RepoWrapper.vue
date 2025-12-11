<template>
  <Scaffold v-if="repo && repoPermissions && route.meta.repoHeader" enable-tabs>
    <template #title>
      <span class="flex">
        <router-link :to="{ name: 'org', params: { orgId: repo.org_id } }" class="hover:underline">{{
          repo.owner
          /* eslint-disable-next-line @intlify/vue-i18n/no-raw-text */
        }}</router-link>
        &nbsp;/
        {{ repo.name }}
      </span>
    </template>
    <template #headerActions>
      <a v-if="badgeUrl" :href="badgeUrl" target="_blank">
        <img class="w-28" :src="badgeUrl" />
      </a>
      <IconButton :href="repo.forge_url" :title="$t('repo.open_in_forge')" :icon="forgeIcon" class="forge h-8 w-8" />
      <IconButton
        v-if="repoPermissions.admin"
        :to="{ name: 'repo-settings' }"
        :title="$t('settings')"
        icon="settings"
      />
    </template>

    <template #tabActions>
      <div v-if="repoPermissions.push && route.name !== 'repo-manual'" class="flex flex-wrap gap-2">
        <Button :text="$t('repo.manual_pipeline.trigger')" start-icon="manual-pipeline" :to="{ name: 'repo-manual' }" />
        <Button
          v-for="action in manualActions"
          :key="action.id"
          :text="action.title"
          start-icon="manual-pipeline"
          :is-loading="manualActionLoading[action.id]"
          @click="triggerManualAction(action.id)"
        />
      </div>
      <Button
        v-else-if="repoPermissions.push"
        :text="$t('repo.manual_pipeline.show_pipelines')"
        start-icon="back"
        :to="{ name: 'repo' }"
      />
    </template>

    <Tab icon="list-group" :to="{ name: 'repo' }" :title="$t('repo.activity')" />
    <Tab icon="branch" :to="{ name: 'repo-branches' }" match-children :title="$t('repo.branches')" />
    <Tab
      v-if="repo.pr_enabled && repo.allow_pr"
      icon="pull-request"
      :to="{ name: 'repo-pull-requests' }"
      match-children
      :title="$t('repo.pull_requests')"
    />

    <router-view />
  </Scaffold>
  <router-view v-else-if="repo && repoPermissions" />
</template>

<script lang="ts" setup>
import type { Ref } from 'vue';
import { computed, onMounted, ref, toRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import Button from '~/components/atomic/Button.vue';
import type { IconNames } from '~/components/atomic/Icon.vue';
import IconButton from '~/components/atomic/IconButton.vue';
import Scaffold from '~/components/layout/scaffold/Scaffold.vue';
import Tab from '~/components/layout/scaffold/Tab.vue';
import useApiClient from '~/compositions/useApiClient';
import useAuthentication from '~/compositions/useAuthentication';
import useConfig from '~/compositions/useConfig';
import { useForgeStore } from '~/compositions/useForgeStore';
import { provide } from '~/compositions/useInjectProvide';
import useNotifications from '~/compositions/useNotifications';
import useRepos from '~/compositions/useRepos';
import type { Forge, ManualActionDefinition, Repo, RepoPermissions } from '~/lib/api/types';
import { usePipelineStore } from '~/store/pipelines';
import { useRepoStore } from '~/store/repos';

const props = defineProps<{
  repoId: string;
}>();

const _repoId = toRef(props, 'repoId');
const repositoryId = computed(() => Number.parseInt(_repoId.value, 10));
const repoStore = useRepoStore();
const pipelineStore = usePipelineStore();
const apiClient = useApiClient();
const notifications = useNotifications();
const { isAuthenticated } = useAuthentication();
const route = useRoute();
const router = useRouter();
const i18n = useI18n();
const config = useConfig();
const forgeStore = useForgeStore();
const { updateLastAccess } = useRepos();

const repo = repoStore.getRepo(repositoryId);
const repoPermissions = ref<RepoPermissions>();
const pipelines = pipelineStore.getRepoPipelines(repositoryId);
provide('repo', repo as Ref<Repo>); // can't be undefined because of v-if in template
provide('repo-permissions', repoPermissions as Ref<RepoPermissions>); // can't be undefined because of v-if in template
provide('pipelines', pipelines);
const forge = ref<Forge>();
const forgeIcon = computed<IconNames>(() => {
  if (forge.value && forge.value.type !== 'addon') {
    return forge.value.type;
  }
  return 'repo';
});

async function loadRepo() {
  repoPermissions.value = await apiClient.getRepoPermissions(repositoryId.value);
  if (!repoPermissions.value.pull) {
    notifications.notify({ type: 'error', title: i18n.t('repo.not_allowed') });
    // no access and not authenticated, redirect to login
    if (!isAuthenticated) {
      await router.replace({ name: 'login', query: { url: route.fullPath } });
      return;
    }
    await router.replace({ name: 'home' });
    return;
  }

  await repoStore.loadRepo(repositoryId.value);
  await pipelineStore.loadRepoPipelines(repositoryId.value);
  await loadManualActions();

  if (repo.value) {
    forge.value = (await forgeStore.getForge(repo.value?.forge_id)).value;
  }
  updateLastAccess(repositoryId.value);
}

onMounted(() => {
  loadRepo();
});

watch([repositoryId], () => {
  loadRepo();
});

const badgeUrl = computed(() => repo.value && `${config.rootPath}/api/badges/${repo.value.id}/status.svg`);

const manualActions = ref<ManualActionDefinition[]>([]);
const manualActionLoading = ref<Record<string, boolean>>({});

async function loadManualActions() {
  try {
    manualActions.value = await apiClient.getManualActions(repositoryId.value);
    manualActionLoading.value = {};
    manualActions.value.forEach((action) => {
      manualActionLoading.value[action.id] = false;
    });
  } catch (error) {
    console.error('Failed to load manual actions', error);
    manualActions.value = [];
    manualActionLoading.value = {};
  }
}

async function triggerManualAction(actionId: string) {
  if (manualActionLoading.value[actionId]) {
    return;
  }
  manualActionLoading.value[actionId] = true;
  const action = manualActions.value.find((entry) => entry.id === actionId);
  try {
    await apiClient.triggerManualAction(repositoryId.value, actionId, {
      repo_id: String(repositoryId.value),
      action_id: actionId,
    });
    notifications.notify({
      type: 'success',
      title: i18n.t('repo.manual_pipeline.webhooks.success', { action: action?.title ?? actionId }),
    });
  } catch (error) {
    console.error('Failed to trigger webhook', error);
    notifications.notify({
      type: 'error',
      title: i18n.t('repo.manual_pipeline.webhooks.error', { action: action?.title ?? actionId }),
    });
  } finally {
    manualActionLoading.value[actionId] = false;
  }
}
</script>
