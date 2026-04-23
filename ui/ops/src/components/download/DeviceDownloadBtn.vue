<template>
  <v-btn-group class="menu-btn">
    <v-btn v-bind="{...dlBtnProps, ...dlBtnLinkProps}"
           @click="dlBtnClicked"
           :disabled="!!dlBtnDisabled"
           v-tooltip:bottom="dlBtnTooltip">
      <template v-if="dlBtnHasPreIcon">
        <v-icon size="24" :start="dlBtnHasText">mdi-file-download</v-icon>
      </template>
      <template v-if="dlBtnHasText">
        Download CSV...
      </template>
      <template v-if="dlBtnHasPostIcon">
        <v-icon size="24" end>mdi-file-download</v-icon>
      </template>
    </v-btn>
    <v-btn icon="true" width="1.4em">
      <v-icon size="20">mdi-chevron-down</v-icon>
      <v-menu activator="parent" :close-on-content-click="false">
        <v-card width="380px">
          <v-card-title>Download Options</v-card-title>
          <v-expansion-panels flat tile v-model="expandedSections" multiple variant="accordion">
            <v-expansion-panel value="history">
              <v-expansion-panel-title>
                <span>Historical data</span>
                <v-divider class="flex-1-1-0"/>
                <template #actions="{ disabled, expanded, readonly }">
                  <v-checkbox
                      :readonly="readonly"
                      :disabled="disabled"
                      :model-value="expanded"
                      @click.stop="toggleSection('history')"
                      hide-details/>
                </template>
              </v-expansion-panel-title>
              <v-expansion-panel-text>
                <v-list-item title="Duration" class="px-0">
                  <template #append>
                    <v-btn-toggle
                        v-model="historyDurationSelected"
                        density="compact"
                        variant="outlined"
                        divided
                        mandatory>
                      <v-btn
                          v-for="item in historyDurationSelections"
                          :key="item.text ?? item.icon"
                          :value="item"
                          width="50px"
                          size="small"
                          :text="item.text"
                          :icon="item.icon"
                          v-tooltip:bottom="item.tooltip"/>
                    </v-btn-toggle>
                  </template>
                </v-list-item>
                <v-expand-transition>
                  <v-list-item v-if="historyDurationShowCustomDuration" class="px-0">
                    <v-date-input
                        v-model="historyDurationCustomDates" multiple="range"
                        label="Custom date range" placeholder="from - to" persistent-placeholder
                        hide-details/>
                  </v-list-item>
                </v-expand-transition>
              </v-expansion-panel-text>
            </v-expansion-panel>
          </v-expansion-panels>
          <v-card-actions>
            <div v-tooltip:bottom="dlBtnTooltip" style="min-width: 100%">
              <v-btn block
                     color="primary"
                     variant="flat"
                     v-bind="dlBtnLinkProps"
                     @click="dlBtnClicked"
                     :disabled="!!dlBtnDisabled">
                <v-icon size="24" start>mdi-file-download</v-icon>
                Download CSV...
              </v-btn>
            </div>
          </v-card-actions>
        </v-card>
      </v-menu>
    </v-btn>
  </v-btn-group>
</template>

<script setup>
import {useDownloadLink} from '@/components/download/download.js';
import {useNow} from '@/components/now.js';
import {DAY, HOUR, MINUTE, SECOND} from '@/util/date.js';
import {addDays, startOfDay} from 'date-fns';
import {computed, onScopeDispose, ref} from 'vue';
import {VDateInput} from 'vuetify/labs/components';

const props = defineProps({
  content: {
    type: String,
    default: 'icon+text'
  },
  query: {
    type: Object,
    default: undefined
  },
});

const dlBtnProps = computed(() => {
  const p = {};
  if (props.content === 'icon') {
    p.icon = 'true';
  }
  return p;
});
const dlBtnHasText = computed(() => props.content === 'icon+text' || props.content === 'text' || props.content === 'text+icon');
const dlBtnHasPreIcon = computed(() => props.content === 'icon+text' || props.content === 'icon');
const dlBtnHasPostIcon = computed(() => props.content === 'text+icon');

const expandedSections = ref(/** @type {string[]} */ []);
const isSectionExpanded = (section) => expandedSections.value.includes(section);
const toggleSection = (section) => {
  if (isSectionExpanded(section)) {
    expandedSections.value = expandedSections.value.filter(s => s !== section);
  } else {
    expandedSections.value.push(section);
  }
}

const historyDurationSelections = [
  {text: '24h', value: 24 * HOUR, tooltip: '24 hours'},
  {text: '1w', value: 7 * DAY, tooltip: '1 week'},
  {text: '30d', value: 30 * DAY, tooltip: '30 days'},
  {icon: 'mdi-calendar', tooltip: 'Custom date range'},
];
const historyDurationSelected = ref(historyDurationSelections[0]);
const historyDurationShowCustomDuration = computed(() => !Object.hasOwn(historyDurationSelected.value, 'value'));
const historyDurationCustomDates = ref([]);
const {now: historyNow} = useNow(10 * MINUTE);
const historyDurationPeriod = computed(() => {
  if (historyDurationShowCustomDuration.value) {
    const dates = historyDurationCustomDates.value;
    if (dates.length === 0) {
      return undefined;
    }
    return {
      startTime: startOfDay(dates[0]),
      endTime: startOfDay(addDays(dates[dates.length - 1], 1)),
    };
  }

  const duration = historyDurationSelected.value.value;
  const now = historyNow.value;
  return {
    startTime: startOfDay(new Date(now - duration)),
    endTime: startOfDay(addDays(now, 1)),
  }
})

const dlBtnRecentlyClicked = ref(false);
let dlBtnRecentlyClickedHandle = 0;
onScopeDispose(() => clearTimeout(dlBtnRecentlyClickedHandle));
const dlBtnClicked = () => {
  dlBtnRecentlyClicked.value = true;
  dlBtnRecentlyClickedHandle = setTimeout(() => dlBtnRecentlyClicked.value = false, 5 * SECOND);
}
const dlBtnDisabled = computed(() => {
  if (isSectionExpanded('history') && !historyDurationPeriod.value) return 'Please select a historical date range';
  if (dlBtnRecentlyClicked.value) return `Your download should begin shortly...`;
  return undefined;
});
const dlBtnTooltip = computed(() => {
  if (dlBtnDisabled.value) return dlBtnDisabled.value;
  if (isSectionExpanded('history')) {
    let dateRange = 'past';
    if (historyDurationShowCustomDuration.value) {
      dateRange = `${historyDurationPeriod.value.startTime.toLocaleDateString()} - ${addDays(historyDurationPeriod.value.endTime, -1).toLocaleDateString()}`;
    } else {
      dateRange = 'last ' + historyDurationSelected.value.tooltip;
    }
    return `Download ${dateRange} data as CSV...`;
  }
  return 'Download data as CSV...';
})
const dlBtnHistory = computed(() => {
  if (!isSectionExpanded('history')) return undefined;
  return historyDurationPeriod.value;
})
const {downloadBtnProps: dlBtnLinkProps} = useDownloadLink(() => props.query, dlBtnHistory)
</script>

<style lang="scss" scoped>
.v-expansion-panel-title,
::v-deep(.v-expansion-panel-text__wrapper) {
  padding-inline: 1rem;

  .v-divider {
    margin-inline: 1rem;
  }
}

.menu-btn {
  &:hover {
    --old-hover-opacity: var(--v-hover-opacity);

    > * {
      --v-hover-opacity: calc(var(--old-hover-opacity) * 0.6);
    }

    ::v-deep(.v-btn__overlay) {
      opacity: calc(var(--v-hover-opacity) * var(--v-theme-overlay-multiplier));
    }

    ::v-deep(.v-btn:hover) {
      --v-hover-opacity: var(--old-hover-opacity);
    }
  }

  ::v-deep(.v-btn--icon.v-btn--density-default) {
    width: var(--v-btn-height);
  }
}
</style>
