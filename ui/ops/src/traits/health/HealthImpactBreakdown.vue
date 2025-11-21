<template>
  <div class="impact-breakdown">
    <h3 class="title">{{ props.title }}</h3>
    <div class="totals mt-2">
      <value-scaled prefix="Checks" stacked>{{ props.totalCount }}</value-scaled>
      <value-scaled prefix="Issues" stacked>
        <span :class="abnormalCountClass">{{ totalAbnormalCount }}</span>
      </value-scaled>
    </div>
    <div class="bar my-3">
      <v-progress-linear
          :model-value="props.totalCount - totalAbnormalCount"
          :max="props.totalCount"
          color="currentColor"
          :bg-color="props.badColor"
          bg-opacity="1"
          height="10"
          rounded/>
    </div>
    <div class="breakdown">
      <v-table density="compact">
        <thead>
          <tr>
            <th/>
            <th v-for="header in tableHeaders" :key="header">{{ header }}</th>
          </tr>
        </thead>
        <tbody>
          <tr class="em">
            <td>Issues</td>
            <td v-for="(count, i) in issueCounts" :key="i">{{ count }}</td>
          </tr>
          <tr v-if="!props.hideAffected">
            <td v-for="(s, i) in affectCells" :key="i">{{ s }}</td>
          </tr>
          <tr>
            <td>Change</td>
            <td v-for="(delta, i) in changeValues" :key="i">
              <value-inc :value="delta" lower-is-better/>
            </td>
          </tr>
        </tbody>
      </v-table>
    </div>
  </div>
</template>

<script setup>
import ValueInc from '@/components/ValueInc.vue';
import ValueScaled from '@/components/ValueScaled.vue';
import {computed} from 'vue';

const props = defineProps({
  title: {
    type: String,
    default: 'Occupant checks'
  },
  totalCount: {
    type: Number,
    default: 0
  },
  affectLabel: {
    type: String,
    default: 'People affected'
  },
  hideAffected: {
    type: Boolean,
    default: false
  },
  issues: {
    type: Array,
    default: () => ([
      {
        title: 'Life',
        count: 1,
        prevCount: 2,
        affect: '-',
      },
      {
        title: 'Health',
        count: 2,
        prevCount: 2,
        affect: '-',
      },
      {
        title: 'Comfort',
        count: 3,
        prevCount: 1,
        affect: 13,
      },
    ])
  },
  errorCount: {
    type: Number,
    default: 0
  },
  prevErrorCount: {
    type: Number,
    default: 0
  },
  badColor: {
    type: String,
    default: 'error'
  },
  goodColor: {
    type: String,
    default: 'success'
  }
});

const totalAbnormalCount = computed(() => props.issues.reduce((acc, issue) => acc + issue.count, 0))
const abnormalCountClass = computed(() => {
  if (totalAbnormalCount.value === 0) {
    return 'text-' + props.goodColor;
  } else {
    return 'text-' + props.badColor;
  }
})
const tableHeaders = computed(() => {
  const headers = props.issues.map(issue => issue.title);
  headers.push('Errors');
  return headers;
})
const issueCounts = computed(() => {
  const counts = props.issues.map(issue => issue.count);
  counts.push(props.errorCount);
  return counts;
});
const affectCells = computed(() => {
  const cells = [props.affectLabel];
  cells.push(...props.issues.map(issue => {
    if (issue.affect === 0) {
      return '-';
    }
    return issue.affect;
  }))
  cells.push('-'); // we can't know if the affects overlap, so can't sum them
  return cells;
});
const changeValues = computed(() => {
  const changes = props.issues.map(issue => issue.count - issue.prevCount);
  changes.push(props.errorCount - props.prevErrorCount);
  return changes;
});

</script>

<style scoped lang="scss">
.totals {
  font-size: 28px;
  font-weight: 600;
  display: flex;
  justify-content: space-between;
  gap: 1em;

  > :last-child {
    align-items: end;
  }
}

.v-table {
  background: inherit;
  --old-border-opacity: var(--v-border-opacity);
  --v-border-opacity: 0;

  .em td:not(:first-child) {
    font-size: 150%;
  }

  :deep(table) {
    table-layout: fixed;
  }

  td, th {
    border-bottom: none;

    &:not(:first-child) {
      white-space: nowrap;
      overflow: hidden;
    }

    &:first-child {
      padding-left: 0;
      min-width: min-content;
      max-width: 100%;
      font-weight: lighter;
    }

    &:last-child {
      padding-right: 0;
      border-left: thin solid rgba(var(--v-border-color), 0.12);
    }
  }
}
</style>