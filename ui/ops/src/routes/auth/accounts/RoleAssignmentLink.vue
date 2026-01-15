<template>
  <span>
    <template v-for="(part, i) in textParts" :key="i">
      <template v-if="i > 0"> {{ ' ' }}</template>
      <template v-if="part.to && !props.noNav">
        <router-link :to="part.to" v-bind="part.props ?? {}">
          <v-tooltip v-if="part.tooltip" activator="parent" location="bottom">{{ part.tooltip }}</v-tooltip>
          {{ part.text }}
        </router-link>
      </template>
      <template v-else>
        <span v-bind="part.props ?? {}">
          <v-tooltip v-if="part.tooltip" activator="parent" location="bottom">{{ part.tooltip }}</v-tooltip>
          {{ part.text ?? part }}
        </span>
      </template>
    </template>
  </span>
</template>

<script setup>
import {ResourceTypeById} from '@/api/ui/account.js';
import {RoleAssignment} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/account/v1/account_pb';
import {computed} from 'vue';

const props = defineProps({
  roleAssignment: {
    type: Object, // type of RoleAssignment.AsObject & {role: Role.AsObject} & {device: Device.AsObject}
    required: true,
  },
  noNav: {
    type: Boolean,
    default: false,
  }
});

const textParts = computed(() => {
  const parts = [];
  const scope = props.roleAssignment.scope;
  if (!scope) {
    parts.push('Global');
  } else {
    const typeStr = (() => {
      const base = (() => {
        const resourceType = scope.resourceType;
        switch (resourceType) {
          case RoleAssignment.ResourceType.NAMED_RESOURCE:
            return 'Device';
          case RoleAssignment.ResourceType.NAMED_RESOURCE_PATH_PREFIX:
            return 'Device+';
          default:
            return ResourceTypeById[scope.resourceType];
        }
      })();
      return base[0] + base.slice(1).toLowerCase();
    })();
    parts.push({text: typeStr, props: {class: 'text-medium-emphasis'}});
    if (props.roleAssignment.device?.metadata?.appearance?.title) {
      parts.push({
        text: props.roleAssignment.device.metadata.appearance.title,
        tooltip: props.roleAssignment.scope?.resource,
      });
    } else {
      parts.push(scope.resource);
    }
  }
  parts.push({
    text: props.roleAssignment.role?.displayName ?? '...',
    to: {
      name: 'roles',
      params: {
        roleId: props.roleAssignment.roleId,
      }
    }
  });
  return parts;
})
</script>

<style scoped>

</style>