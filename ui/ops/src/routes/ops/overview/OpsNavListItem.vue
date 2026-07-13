<template>
  <v-list-group v-if="hasChildren" :value="props.item.title">
    <template #activator="{props: _props, isOpen: _isOpen}">
      <!--
      Slightly different behaviour for containers that have their own destination (an in-app
      layout or an external href), vs those that only expand.
      The first we expand on button click only, clicking the activator navs to the destination.
      The second we expand on activator click, the button is just there for visual consistency.
      -->
      <template v-if="hasOwnPage">
        <v-list-item
            v-bind="linkAttrs"
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
      v-bind="linkAttrs"
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

/**
 * Whether the item is an external link, i.e. it has a safe href.
 *
 * @type {import('vue').ComputedRef<boolean>}
 */
const isExternal = computed(() => isSafeHref(props.item.href));

/**
 * Whether the item has its own primary destination (an external href or an in-app layout),
 * as opposed to only expanding to reveal children. When true, the row navigates on click and
 * any children collapse to an appended chevron button.
 *
 * @type {import('vue').ComputedRef<boolean>}
 */
const hasOwnPage = computed(() => isExternal.value || Boolean(props.item.layout));

/**
 * The link attributes to bind to the row's list item: an external anchor when the item has a
 * safe href, otherwise an in-app router link to the area.
 *
 * @type {import('vue').ComputedRef<Object>}
 */
const linkAttrs = computed(() => {
  if (isExternal.value) {
    return {
      href: props.item.href,
      target: props.item.target ?? '_blank',
      rel: 'noopener noreferrer'
    };
  }
  return {to: toAreaLink.value};
});

/**
 * Checks whether an href is safe to render as an external link.
 *
 * Only http(s) URLs and root-relative paths are permitted. This rejects dangerous schemes
 * such as javascript: and data:, which would otherwise execute in the Ops UI origin when
 * clicked (the href comes from config, so this is defence-in-depth against a bad config).
 *
 * @param {string} href
 * @return {boolean}
 */
function isSafeHref(href) {
  if (typeof href !== 'string' || !href) return false;
  // Root-relative paths (e.g. /-/ooh-ac-request/) are same-origin and safe.
  if (href.startsWith('/')) return true;
  try {
    const url = new URL(href, window.location.origin);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}
</script>

<style scoped>
.v-list-item--active:not(.activeExact) :deep(.v-list-item__overlay) {
  background-color: transparent;
}
</style>
