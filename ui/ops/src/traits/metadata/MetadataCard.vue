<template>
  <v-card elevation="0" tile>
    <v-list tile class="ma-0 pa-0" lines="two">
      <v-list-subheader class="text-title-caps-large text-neutral-lighten-3">
        Information
      </v-list-subheader>

      <v-col v-for="(val, key) in deviceInfo[0]" :key="key" class="pa-0" cols="align-self">
        <v-list-item class="py-1">
          <v-list-item-title class="text-body-small text-capitalize">
            {{ camelToSentence(key) }}
          </v-list-item-title>
          <v-list-item-subtitle class="text-subtitle-1 py-1 font-weight-medium text-wrap ml-2 metadata-value">
            <structured-value :value="val"/>
          </v-list-item-subtitle>
        </v-list-item>
      </v-col>

      <v-col v-for="(subValue, subKey) in deviceInfo[1]" :key="subKey" class="pa-0" cols="align-self">
        <v-list-item class="py-1">
          <v-list-item-title class="text-body-small text-capitalize">
            {{ camelToSentence(subKey) }}
          </v-list-item-title>
          <v-row class="py-0 flex-column">
            <v-col v-for="(val, key) in subValue" :key="key" cols="12" class="py-0">
              <v-list-item-subtitle class="py-1 mx-2">
                <v-col cols="align-self">
                  <v-row class="text-capitalize text-caption">
                    {{ camelToSentence(key) }}
                  </v-row>
                  <v-row class="text-subtitle-1">
                    <structured-value :value="val"/>
                  </v-row>
                </v-col>
              </v-list-item-subtitle>
            </v-col>
          </v-row>
        </v-list-item>
      </v-col>
    </v-list>
  </v-card>
</template>

<script setup>
import StructuredValue from '@/components/StructuredValue.vue';
import {useSidebarStore} from '@/stores/sidebar';
import {camelToSentence} from '@/util/string';
import {computed} from 'vue';

const sidebar = useSidebarStore();

const deviceInfo = computed(() => {
  // Initialize variables for info and subInfo
  const info = {};
  const subInfo = {};

  // Check if data has metadata property
  if (sidebar.data?.metadata) {
    const deviceData = sidebar.data?.metadata;

    // Get all properties of metadata as an array of [key, value] pairs
    const data = Object.entries(deviceData);

    // Filter out properties that are traitsList or membership or empty arrays or undefined values
    const filtered = data.filter(([key, value]) => {
      const notIncluded = !['traitsList', 'membership'].includes(key);
      const hasValue = Array.isArray(value) ? value.length > 0 : value !== undefined;
      return notIncluded && hasValue;
    });

    // Flatten out the filtered data
    filtered.forEach(([key, value]) => {
      // If value is not empty
      if (value) {
        // If value is not an object
        if (typeof value !== 'object') {
          info[key] = value;
        } else {
          // If key is location
          if (key === 'location') {
            // Set zone value if it exists
            info['zone'] = value?.zone ? value.zone : value.title;
            // Add any properties from moreMap array to info object
            if (value.moreMap.length) {
              for (const more of deviceData.location.moreMap) {
                info[more[0]] = more[1];
              }
            }
          } else {
            // Loop through the object for inner values
            for (const subValue in value) {
              // If subValue and is not moreMap and has a value
              if (subValue && value[subValue] && subValue !== 'moreMap') {
                // If subInfo[key] does not exist, create it
                if (!subInfo[key]) {
                  subInfo[key] = {};
                }
                // Add subValue and its value to subInfo[key]
                subInfo[key][subValue] = value[subValue];
              }
            }
          }
        }
      }
    });
  }

  // Return info and subInfo objects within an array
  return [info, subInfo];
});

</script>

<style scoped>
.v-list-item {
  min-height: auto;
}

/* allow structured (JSON) values to expand beyond the subtitle's default line clamp */
.metadata-value {
  -webkit-line-clamp: unset;
  -webkit-box-orient: unset;
  display: block;
  overflow: visible;
  text-overflow: clip;
  white-space: normal;
}
</style>
