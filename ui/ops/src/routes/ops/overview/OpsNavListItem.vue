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
                v-if="showExternalButton"
                @click.prevent.stop="openExternal"
                variant="text"
                size="x-small"
                class="text-medium-emphasis"
                :aria-label="`Open ${props.item.title} (external)`"
                :title="props.item.href"
                icon="mdi-open-in-new"/>
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
    <template v-if="showExternalButton" #append>
      <v-btn
          @click.prevent.stop="openExternal"
          variant="text"
          size="x-small"
          class="text-medium-emphasis"
          :aria-label="`Open ${props.item.title} (external)`"
          :title="props.item.href"
          icon="mdi-open-in-new"/>
    </template>
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
 * Whether the item has an in-app dashboard layout.
 *
 * @type {import('vue').ComputedRef<boolean>}
 */
const hasLayout = computed(() => Boolean(props.item.layout));

/**
 * Whether the item has its own primary destination (an in-app layout or an external href),
 * as opposed to only expanding to reveal children. When true, the row navigates on click and
 * any children collapse to an appended chevron button.
 *
 * @type {import('vue').ComputedRef<boolean>}
 */
const hasOwnPage = computed(() => hasLayout.value || isExternal.value);

/**
 * The anchor attributes for an external link (href + target + rel), used when the row itself
 * is the external anchor (href with no layout).
 *
 * @type {import('vue').ComputedRef<{href: string, target: string, rel: string}>}
 */
const externalAttrs = computed(() => ({
  href: props.item.href,
  target: props.item.target ?? '_blank',
  rel: 'noopener noreferrer'
}));

/**
 * Opens the item's external href. Used by the appended button when a layout owns the row - a
 * button rather than a nested anchor, as an <a> can't validly contain another <a> (the same
 * pattern as the chevron, which is also a button inside the row).
 */
function openExternal() {
  if ((props.item.target ?? '_blank') === '_self') {
    window.location.assign(props.item.href);
  } else {
    window.open(props.item.href, '_blank', 'noopener,noreferrer');
  }
}

/**
 * The link attributes to bind to the row's list item. An in-app layout takes precedence and
 * owns the row; a lone href (no layout) makes the row itself the external anchor. When both a
 * layout and an href are present the layout owns the row and the href is surfaced as an appended
 * button (see showExternalButton), so neither destination is silently unreachable.
 *
 * @type {import('vue').ComputedRef<Object>}
 */
const linkAttrs = computed(() => {
  if (isExternal.value && !hasLayout.value) return externalAttrs.value;
  return {to: toAreaLink.value};
});

/**
 * Whether to surface the external link as an appended button. Happens when an item has both an
 * in-app layout (which owns the row) and an href, so the external destination stays reachable.
 *
 * @type {import('vue').ComputedRef<boolean>}
 */
const showExternalButton = computed(() => isExternal.value && hasLayout.value);

// Warn about a configured href that is unsafe/invalid so the misconfiguration is visible rather
// than silently degrading to a dead in-app link.
if (props.item.href && !isSafeHref(props.item.href)) {
  console.warn(
      `Ops nav item "${props.item.title ?? props.item.path}" has an unsafe or invalid href and ` +
      `will be ignored: ${props.item.href}`);
}

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
