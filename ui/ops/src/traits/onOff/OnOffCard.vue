<template>
  <v-card elevation="0" tile>
    <v-card-title class="d-flex text-title-caps-large text-neutral-lighten-3">
      <span>On Off</span>
    </v-card-title>
    <v-list tile class="ma-0 pa-0">
      <v-list-item class="py-1">
        <v-list-item-title class="text-body-small text-capitalize">
          State
        </v-list-item-title>

        <template #append>
          <v-list-item-subtitle class="text-body-1">
            {{ onOffState }}
          </v-list-item-subtitle>
        </template>
      </v-list-item>
    </v-list>
    <v-card-actions class="px-4">
      <v-spacer/>
      <v-btn
          size="small"
          variant="tonal"
          @click="setOnOff(OnOff.State.OFF)">
        Off
      </v-btn>
      <v-btn
          size="small"
          variant="tonal"
          @click="setOnOff(OnOff.State.ON)">
        On
      </v-btn>
    </v-card-actions>
  </v-card>
</template>

<script setup>

import {onOffToString} from '@/traits/onOff/onOff.js';
import {OnOff} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/onoff/v1/on_off_pb.js';
import {computed, toValue} from 'vue';

const props = defineProps({
  value: {
    type: Object, // of OnOff.AsObject
    default: () => ({})
  },
  loading: {
    type: Boolean,
    default: false
  },
  name: {
    type: String,
    required: true
  }
});
const emit = defineEmits([
  'updateOnOff' // UpdateOnOffRequest.AsObject
]);

const onOffState = computed(() => {
  const v = toValue(props.value);
  if (v && v.state !== undefined) {
    return onOffToString(v.state);
  }
  return onOffToString(OnOff.State.STATE_UNSPECIFIED);
});

/**
 * Set OnOff state
 *
 * @param {OnOff.State} s
 */
function setOnOff(s) {
  emit('updateOnOff', { name: props.name, onOff: {state: s} })
}

</script>
