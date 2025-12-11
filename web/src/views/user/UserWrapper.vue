<template>
  <Scaffold enable-tabs>
    <template #title>{{ $t('user.settings.settings') }}</template>
    <template #headerActions><Button :text="$t('logout')" @click="logout" /></template>

    <Tab icon="settings-outline" :to="{ name: 'user' }" :title="$t('user.settings.general.general')" />
    <Tab icon="secret" :to="{ name: 'user-secrets' }" :title="$t('secrets.secrets')" />
    <Tab icon="docker" :to="{ name: 'user-registries' }" :title="$t('registries.registries')" />
    <Tab icon="console" :to="{ name: 'user-cli-and-api' }" :title="$t('user.settings.cli_and_api.cli_and_api')" />
    <Tab
      v-if="userRegisteredAgents"
      icon="agent"
      :to="{ name: 'user-agents' }"
      :title="$t('admin.settings.agents.agents')"
    />

    <router-view />
  </Scaffold>
</template>

<script lang="ts" setup>
import Button from '~/components/atomic/Button.vue';
import Scaffold from '~/components/layout/scaffold/Scaffold.vue';
import Tab from '~/components/layout/scaffold/Tab.vue';
import useConfig from '~/compositions/useConfig';
import useAuthentication from '~/compositions/useAuthentication';

const config = useConfig();
const { userRegisteredAgents } = config;
const authentication = useAuthentication();
const logout = () => authentication.logout();
</script>
