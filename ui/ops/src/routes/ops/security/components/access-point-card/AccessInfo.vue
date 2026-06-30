<template>
  <div>
    <!-- Status bar -->
    <status-bar
        :title="props.device.title"
        :status-bar-color="statusColor"
        :door-status-text="doorStatusText"
        :show-close="props.showClose"
        @click:close="emit('click:close')"/>

    <!-- Grant area -->
    <v-card-text class="text-h6 text-white font-weight-regular d-flex flex-row pt-4 pb-0">
      <span>Last access:</span>
      <span :class="[`text-${color}`, 'ml-auto font-weight-bold']">
        {{ formatString(grantStates) }}
      </span>
    </v-card-text>
    <v-card-text class="text-h6 text-white d-flex flex-column pt-1" style="max-width: 350px">
      <span class="text-subtitle-1"> {{ user.name }} {{ user.cardId }} </span>
    </v-card-text>

    <!-- Alert/Acknowledge area -->
    <v-card-text v-if="hasAlerts" class="mt-4 px-0">
      <v-list-item lines="two">
        <v-tooltip location="bottom">
          <template #activator="{ props: _props }">
            <v-list-item-title class="text-uppercase flex-shrink-1" v-bind="_props">
              {{ alert.description }}
            </v-list-item-title>
          </template>
          {{ alert.description }}
        </v-tooltip>
        <v-list-item-subtitle>
          {{ alert.createTime }}
        </v-list-item-subtitle>
        <template #append>
          <acknowledgement-btn
              :ack="alert.acknowledgement"
              @acknowledge="setAcknowledged(true, alert, hubName)"
              @unacknowledge="setAcknowledged(false, alert, hubName)"/>
        </template>
      </v-list-item>
    </v-card-text>
  </div>
</template>

<script setup>
import {closeResource} from '@/api/resource';
import {grantNamesByID} from '@/api/sc/traits/access';
import {alertToObject} from '@/api/ui/alerts.js';
import {useAlertsCollection} from '@/composables/alerts.js';
import {useAcknowledgement} from '@/composables/notifications.js';
import AcknowledgementBtn from '@/routes/ops/notifications/AcknowledgementBtn.vue';
import StatusBar from '@/routes/ops/security/components/access-point-card/StatusBar.vue';
import {useStatus} from '@/routes/ops/security/components/access-point-card/useStatus';
import {useCohortStore} from '@/stores/cohort.js';
import {computed, onBeforeUnmount} from 'vue';

const props = defineProps({
  accessAttempt: {
    type: Object,
    default: () => {
    }
  },
  openClose: {
    type: Object,
    default: () => {
    }
  },
  device: {
    type: Object,
    default: () => {
    }
  },
  paused: {
    type: Boolean,
    default: false
  },
  showClose: {
    type: Boolean,
    default: false
  }
});
const {setAcknowledged} = useAcknowledgement();
const emit = defineEmits(['click:close']);

const {color} = useStatus(() => props.accessAttempt);

const statusColor = computed(() => {
  if (props.openClose) {
    const percentage = props.openClose.statesList[0].openPercent;

    if (percentage === 0) return 'success'; // closed
    if (percentage > 0 && percentage <= 100) return 'warning'; // moving and open
    else return 'grey'; // unknown
  } else return color.value;
});

const doorStatusText = computed(() => {
  if (props.openClose) {
    const percentage = props.openClose.statesList[0].openPercent;

    if (percentage === 0) return 'closed'; // closed
    if (percentage > 0 && percentage < 100) return 'moving'; // moving
    if (percentage === 100) return 'open'; // open
    else return 'unknown'; // unknown
  } else return undefined;
});

// ----------------- Access Attempt ----------------- //
const grantId = computed(() => props.accessAttempt?.grant);
const grantStates = computed(() => {
  return grantNamesByID[grantId.value || 0].toLowerCase();
});
const formatString = (str) => {
  return str.split('_').join(' ').charAt(0).toUpperCase() + str.split('_').join(' ').slice(1);
};

// ----------------- Alerts ----------------- //
const cohort = useCohortStore();
const hubName = computed(() => cohort.hubNode?.name ?? '');
const query = computed(() => ({
  createdNotBefore: undefined,
  createdNotAfter: undefined,
  severityNotAbove: undefined,
  severityNotBelow: undefined,
  floor: undefined,
  zone: undefined,
  subsystem: undefined,
  source: props.device.source === '' ? undefined : props.device.source,
  acknowledged: undefined,
  resolved: false,
  resolvedNotBefore: undefined,
  resolvedNotAfter: undefined
}));

const alertsRequest = computed(() => ({
  name: hubName.value,
  query: query.value
}));
const alertsOptions = computed(() => ({
  paused: props.paused,
  wantCount: 1
}));
const alertsCollection = useAlertsCollection(alertsRequest, alertsOptions);
const hasAlerts = computed(() => alertsCollection.items.value.length > 0);
const alert = computed(() => {
  if (alertsCollection.items.value.length === 0) {
    return {};
  }
  return alertToObject(alertsCollection.items.value[0]) ?? {};
});

const user = computed(() => {
  if (!props.accessAttempt?.actor?.displayName && !props.accessAttempt?.actor?.idsMap?.[0]?.[1]) {
    return {
      name: 'No card details provided'
    };
  }

  return {
    name: props.accessAttempt?.actor?.displayName ?? 'Unknown',
    cardId: `(${props.accessAttempt?.actor?.idsMap?.[0]?.[1] ?? 'Unknown'})`
  };
});

onBeforeUnmount(() => {
  closeResource(props.accessAttempt);
});
</script>
<style lang="scss" scoped>
.granted {
  color: rgb(var(--v-theme-success));
  transition: color 0.5s ease-in-out;
}

.denied,
.forced,
.failed {
  color: rgb(var(--v-theme-error));
}

.pending,
.aborted,
.tailgate {
  color: rgb(var(--v-theme-warning));
}

.grant_unknown {
  color: grey;
}
</style>
