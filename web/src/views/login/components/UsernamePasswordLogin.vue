<template>
  <form class="flex w-full flex-col gap-4" @submit.prevent="onSubmit">
    <InputField :label="$t('username')">
      <TextField v-model="username" type="text" autocomplete="username" required class="text-white" />
    </InputField>
    <InputField :label="$t('password')">
      <TextField v-model="password" type="password" autocomplete="current-password" required class="text-white" />
    </InputField>
    <p v-if="error" class="text-wp-error-200 text-sm">{{ error }}</p>
    <Button
      type="submit"
      :text="$t('login')"
      class="self-stretch justify-center"
      :disabled="loading"
      :is-loading="loading"
    />
  </form>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';

import Button from '~/components/atomic/Button.vue';
import InputField from '~/components/form/InputField.vue';
import TextField from '~/components/form/TextField.vue';

withDefaults(
  defineProps<{
    loading?: boolean;
    error?: string | null;
  }>(),
  {
    loading: false,
    error: null,
  },
);

const emit = defineEmits<{
  (event: 'submit', payload: { username: string; password: string }): void;
}>();

useI18n();

const username = ref('');
const password = ref('');

function onSubmit() {
  emit('submit', { username: username.value, password: password.value });
}
</script>
