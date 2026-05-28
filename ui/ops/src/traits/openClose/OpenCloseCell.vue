<template>
  <status-alert v-if="props.streamError" icon="mdi-cancel" :resource="props.streamError"/>

  <v-tooltip v-else location="left">
    <template #activator="{ props: _props }">
      <v-icon :class="doorState.class" size="20" v-bind="_props">{{ doorState.icon }}</v-icon>
    </template>
    <span class="text-capitalize">{{ doorState?.text }}</span>
  </v-tooltip>
</template>

<script setup>
import StatusAlert from '@/components/StatusAlert.vue';
import {useOpenClose} from '@/traits/openClose/openClose.js';
import {computed} from 'vue';

const props = defineProps({
  value: {
    type: Object, // of OpenClosePositions.AsObject
    default: () => {}
  },
  loading: {
    type: Boolean,
    default: false
  },
  streamError: {
    type: Object,
    default: null
  }
});

const {openStr, openIcon, openClass} = useOpenClose(() => props.value);
const doorState = computed(() => {
  return {
    icon: openIcon.value,
    class: openClass.value,
    text: openStr.value
  };
});
</script>

<style scoped>
.open, .moving {
  color: rgb(var(--v-theme-success));
}

.closed {
  color: rgb(var(--v-theme-warning));
}

.unknown {
  color: grey;
}
</style>
