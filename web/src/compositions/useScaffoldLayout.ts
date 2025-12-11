import type { Ref } from 'vue';
import { computed, inject, onUnmounted, provide, ref } from 'vue';

const scaffoldLayoutKey = Symbol('scaffold-layout');

interface ScaffoldLayoutContext {
  requestFluid: () => void;
  releaseFluid: () => void;
  fluidOverride: Ref<boolean>;
}

export function useScaffoldLayoutProvider() {
  const overrideCount = ref(0);
  const fluidOverride = computed(() => overrideCount.value > 0);

  function requestFluid() {
    overrideCount.value += 1;
  }

  function releaseFluid() {
    overrideCount.value = Math.max(overrideCount.value - 1, 0);
  }

  provide<ScaffoldLayoutContext>(scaffoldLayoutKey, {
    requestFluid,
    releaseFluid,
    fluidOverride,
  });

  return {
    fluidOverride,
  };
}

export function useScaffoldLayoutConsumer(options?: { fluid?: boolean }) {
  const ctx = inject<ScaffoldLayoutContext | null>(scaffoldLayoutKey, null);

  if (!ctx || options?.fluid === false) {
    return;
  }

  ctx.requestFluid();
  onUnmounted(() => {
    ctx.releaseFluid();
  });
}
