<template>
  <span :class="color">
    {{ str }}
    <v-menu v-if="showErrorIcon" :close-on-content-click="false">
      <template #activator="{ props: _props }">
        <v-progress-circular
            v-if="showNextAttemptTime"
            v-bind="_props"
            :size="20"
            :width="2"
            :model-value="progressToNextAttempt"
            color="error-lighten-2"
            class="ml-2"/>
        <v-icon v-else v-bind="_props" class="text-error-lighten-2" end>
          mdi-alert-circle-outline
        </v-icon>
      </template>
      <v-card max-width="400px">
        <v-card-title class="text-h5">
          Error
        </v-card-title>
        <v-card-subtitle v-if="showAttemptCount">
          {{ props.service.failedAttempts }} attempts since {{ attemptingSinceStr }}
        </v-card-subtitle>
        <v-card-text>{{ errStr }}</v-card-text>
      </v-card>
    </v-menu>
  </span>
</template>
<script setup>
import {timestampToDate} from '@/api/convpb';
import {useNow} from '@/components/now';
import {SECOND} from '@/util/date.js';
import {computed, ref, watch} from 'vue';

const props = defineProps({
  service: {
    type: Object, // Service.AsObject
    default: () => ({})
  }
});

const str = computed(() => {
  const s = props.service ?? {};
  if (s.active === false) return 'Stopped';
  if (s.loading && s.error) return 'Failed, restarting';
  if (s.loading) return 'Starting';
  return 'Running';
});
const color = computed(() => {
  const s = props.service ?? {};
  if (s.active === false && s.error) return 'text-error-lighten-2';
  if (s.active === false) return '';
  if (s.loading && s.error) return 'text-error-lighten-2';
  if (s.loading) return 'text-info-lighten-2';
  return 'text-success-lighten-2';
});

const showErrorIcon = computed(() => props.service?.error &&
    (props.service?.active === false || props.service?.loading));
const errStr = computed(() => props.service?.error);

const showAttemptCount = computed(() => showErrorIcon.value && props.service?.failedAttempts);
const attemptingSinceStr = computed(() => {
  const d = timestampToDate(props.service?.lastLoadingStartTime);
  if (!d) return '';
  return d.toLocaleString();
});

const {now} = useNow(SECOND);
const showNextAttemptTime = computed(() => {
  const s = props.service ?? {};
  return s.active && s.loading && s.nextAttemptTime;
});
const nextAttemptDate = computed(() => timestampToDate(props.service?.nextAttemptTime));

const attemptTimeStartDate = ref(new Date());
watch(nextAttemptDate, () => {
  attemptTimeStartDate.value = new Date();
}, {immediate: true});
const progressToNextAttempt = computed(() => {
  return (now.value.getTime() - attemptTimeStartDate.value.getTime()) /
      (nextAttemptDate.value.getTime() - attemptTimeStartDate.value.getTime()) * 100;
});
</script>
