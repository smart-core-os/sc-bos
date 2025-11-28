<template>
  <span>
    <span class="current">{{ currentValueStr }}</span>
    <span v-if="cmpStr" class="cmp">{{ cmpStr }}</span>
    <span v-if="rhsStr" class="rhs">{{ rhsStr }}</span>
  </span>
</template>

<script setup>
import {valueToString} from '@/traits/health/health.js';
import {hasOneOf} from '@/util/proto.js';
import {HealthCheck} from '@smart-core-os/sc-bos-ui-gen/proto/health_pb';
import {computed} from 'vue';

const props = defineProps({
  modelValue: {
    /** @type {import('vue').PropType<import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.AsObject>} */
    type: Object,
    default: null
  }
});

const isNormal = computed(() => props.modelValue?.normality === HealthCheck.Normality.NORMAL);
const boundsOneOf = computed(() => {
  /** @type {import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.Bounds.AsObject} */
  const b = props.modelValue?.bounds;
  const options = ['normalValue', 'abnormalValue', 'normalValues', 'abnormalValues', 'normalRange', 'abnormalRange'];
  for (const opt of options) {
    if (hasOneOf(b, opt)) {
      return opt;
    }
  }
  return null;
})
const currentValueUnit = computed(() => {
  return null
});
const currentValueStr = computed(() => valueToString(props.modelValue?.bounds?.currentValue, currentValueUnit.value));

const cmps = {
  'abnormal': {
    'normalValue': '≠',
    'abnormalValue': '=',
    'normalValues': ' not in ',
    'abnormalValues': ' in ',
    'normalRange': {
      [HealthCheck.Normality.LOW]: '<',
      [HealthCheck.Normality.HIGH]: '>',
      '': ' out of bounds'
    }
  },
  'normal': {
    'normalValue': '=',
    'abnormalValue': '≠',
    'normalValues': ' in ',
    'abnormalValues': ' not in ',
    'normalRange': {
      [HealthCheck.Normality.NORMAL]: ' within '
    }
  }
}
const cmpStr = computed(() => {
  const currentCmps = isNormal.value ? cmps.normal : cmps.abnormal;
  switch (boundsOneOf.value) {
    case 'normalValue':
      return currentCmps.normalValue;
    case 'abnormalValue':
      return currentCmps.abnormalValue;
    case 'normalValues':
      return currentCmps.normalValues;
    case 'abnormalValues':
      return currentCmps.abnormalValues;
    case 'normalRange': {
      const c = currentCmps.normalRange[props.modelValue?.normality];
      if (c) {
        return c;
      }
      return currentCmps.normalRange[''];
    }
  }
  return '';
});
const rhsStr = computed(() => {
  /** @type {import('@smart-core-os/sc-bos-ui-gen/proto/health_pb').HealthCheck.Bounds.AsObject} */
  const b = props.modelValue?.bounds;
  switch (boundsOneOf.value) {
    case 'normalValue':
      return valueToString(b.normalValue, b.displayUnit);
    case 'abnormalValue':
      return valueToString(b.abnormalValue, b.displayUnit);
    case 'normalValues':
      return b.normalValues.valuesList.map(v => valueToString(v)).join(', ');
    case 'abnormalValues':
      return b.abnormalValues.valuesList.map(v => valueToString(v)).join(', ');
    case 'normalRange': {
      const r = b.normalRange;
      const db = r.deadband ? ` (±${valueToString(r.deadband)})` : '';
      switch (props.modelValue?.normality) {
        case HealthCheck.Normality.LOW:
          return valueToString(r.low, b.displayUnit) + db;
        case HealthCheck.Normality.HIGH:
          return valueToString(r.high, b.displayUnit) + db;
        case HealthCheck.Normality.NORMAL:
          return `${valueToString(r.low)}—${valueToString(r.high, b.displayUnit)}`;
      }
      return '';
    }
  }
  return '';
})
</script>

<style scoped>

</style>