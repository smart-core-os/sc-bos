<template>
  <v-chip v-tooltip:bottom="tooltipStr">
    <template #prepend v-if="prependStr">{{ prependStr }}</template>
    {{ valueStr }}
  </v-chip>
</template>

<script setup>
import {ResourceTypeById} from '@/api/ui/account.js';
import {RoleAssignment} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/account/v1/account_pb';
import {computed} from 'vue';

const props = defineProps({
  title: {
    type: String,
    default: undefined
  },
  value: {
    type: String,
    default: undefined
  },
  type: {
    type: Number,
    default: RoleAssignment.ResourceType.RESOURCE_TYPE_UNSPECIFIED
  }
});

const prependStr = computed(() => {
  switch (props.type) {
    case RoleAssignment.ResourceType.NAMED_RESOURCE:
      return 'Device';
    case RoleAssignment.ResourceType.NAMED_RESOURCE_PATH_PREFIX:
      return 'Device+';
    case RoleAssignment.ResourceType.NODE:
      return 'On Node';
    case RoleAssignment.ResourceType.SUBSYSTEM:
      return 'System';
    case RoleAssignment.ResourceType.ZONE:
      return 'Zone';
    case RoleAssignment.ResourceType.RESOURCE_TYPE_UNSPECIFIED:
      return '';
    default:
      return ResourceTypeById[props.type] ?? 'Unknown';
  }
});
const valueStr = computed(() => {
  return props.title;
});
const tooltipStr = computed(() => {
  return props.value ?? props.title;
})
</script>

<style scoped lang="scss">
.v-chip {
  // This calc is taken from VChip/_mixins.scss in Vuetify and is used
  // by Vuetify to calculate the padding for chips at different sizes.
  // Vuetify does this is SCSS so we can't reuse it directly.
  --pad-start: calc(var(--v-chip-height, 32px) / (2 + 2 / 3));
  padding: 0 var(--pad-start)
}
:deep(.v-chip__prepend) {
  height: 100%;
  margin-left: calc(var(--pad-start)*-1);
  padding-left: calc(var(--pad-start));
  margin-right: .5em;
  padding-right: .5em;
  background: rgba(var(--v-theme-primary), 0.4);
}
</style>