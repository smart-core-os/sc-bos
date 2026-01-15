<template>
  <v-card class="zone-notifications" :style="props.style" :class="props.class">
    <notifications-table overview-page :force-query="forceQuery" v-bind="$attrs"/>
  </v-card>
</template>
<script setup>
import NotificationsTable from '@/routes/ops/notifications/NotificationsTable.vue';
import {computed} from 'vue';

const props = defineProps({
  forceQuery: {
    type: Object, /** @type {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alert/v1/alerts_pb.js').Alert.Query.AsObject} */
    default: null
  },
  // legacy, use forceQuery instead. Here because it's driven directly from json config.
  zone: {
    type: String,
    default: ''
  },
  style: {
    type: [String, Object, Array],
    default: ''
  },
  class: {
    type: [String, Object, Array],
    default: ''
  }
});
defineOptions({
  inheritAttrs: false
});

const forceQuery = computed(() => {
  const q = props.forceQuery ?? {};
  if (props.zone) {
    q.zone = props.zone;
  }
  return q;
});
</script>
