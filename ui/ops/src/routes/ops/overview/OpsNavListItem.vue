<template>
  <v-list-group v-if="hasChildren" :value="props.item.title">
    <template #activator="{props: _props, isOpen: _isOpen}">
      <!--
      Slightly different behaviour for containers that have their own pages, vs those that don't.
      The first we expand on button click only, clicking the activator navs to the page.
      The second we expand on activator click, the button is just there for visual consistency.
      -->
      <template v-if="props.item.layout">
        <v-list-item
            :to="toAreaLink"
            :active="active"
            :class="{activeExact}">
          <template #prepend>
            <v-icon v-if="!props.miniVariant || !props.item.shortTitle">{{ props.item.icon }}</v-icon>
            <v-list-item-title v-else class="v-icon text-center text-truncate" style="width: 24px">
              {{ props.item.shortTitle }}
            </v-list-item-title>
          </template>
          <v-list-item-title>{{ props.item.title }}</v-list-item-title>
          <template #append>
            <v-btn
                @click.prevent.stop="_props.onClick"
                variant="text"
                size="x-small"
                style="font-size: 120%"
                :icon="_isOpen ? 'mdi-chevron-down' : 'mdi-chevron-left'"/>
          </template>
        </v-list-item>
      </template>
      <template v-else>
        <v-list-item
            @click.prevent.stop="_props.onClick"
            :active="active"
            :class="{activeExact}">
          <template #prepend>
            <v-icon v-if="!props.miniVariant || !props.item.shortTitle">{{ props.item.icon }}</v-icon>
            <v-list-item-title v-else class="v-icon text-center text-truncate" style="width: 24px">
              {{ props.item.shortTitle }}
            </v-list-item-title>
          </template>
          <v-list-item-title>{{ props.item.title }}</v-list-item-title>
          <template #append>
            <v-btn
                variant="text"
                size="x-small"
                style="font-size: 120%"
                :icon="_isOpen ? 'mdi-chevron-down' : 'mdi-chevron-left'"/>
          </template>
        </v-list-item>
      </template>
    </template>
    <ops-nav-list-items
        :items="props.item.children"
        :depth="props.depth + 1"
        :mini-variant="props.miniVariant"
        :parent-path="currentPath"/>
  </v-list-group>
  <v-list-item
      v-else
      :to="toAreaLink"
      :active="active"
      :class="{activeExact}">
    <template #prepend>
      <v-icon v-if="!props.miniVariant || !props.item.shortTitle">{{ props.item.icon }}</v-icon>
      <v-list-item-title v-else class="v-icon text-center text-truncate" style="width: 24px;">
        {{ props.item.shortTitle }}
      </v-list-item-title>
    </template>
    <v-list-item-title>{{ props.item.title }}</v-list-item-title>
  </v-list-item>
</template>

<script setup>
import OpsNavListItems from '@/routes/ops/overview/OpsNavListItems.vue';
import {computed} from 'vue';
import {useRoute} from 'vue-router';

const props = defineProps({
  item: {
    type: Object,
    required: true
  },
  items: {
    type: Array,
    required: true
  },
  depth: {
    type: Number,
    default: 0
  },
  miniVariant: {
    type: Boolean,
    default: false
  },
  parentPath: {
    type: String,
    default: ''
  }
});
const route = useRoute();

/**
 * Computed checker to see if the item has children
 *
 * @type {import('vue').ComputedRef<boolean>}
 */
const hasChildren = computed(() => props.item.children && props.item.children.length > 0);

/**
 * Computed checker to return the current path
 *
 * @type {import('vue').ComputedRef<string>}
 */
const currentPath = computed(() => {
  const pathSegments = props.parentPath ? [props.parentPath] : [];
  pathSegments.push(encodeURIComponent(props.item.path ?? props.item.title));
  return pathSegments.join('/');
});

const activeExact = computed(() => route.path === toAreaLink.value);
const active = computed(() => route.path === toAreaLink.value || route.path.startsWith(toAreaLink.value + '/'));

/**
 * Computed checker to return the link to the area
 *
 * @type {import('vue').ComputedRef<string>}
 */
const toAreaLink = computed(() => `/ops/overview/${currentPath.value}`);
</script>

<style scoped>
.v-list-item--active:not(.activeExact) :deep(.v-list-item__overlay) {
  background-color: transparent;
}
</style>
