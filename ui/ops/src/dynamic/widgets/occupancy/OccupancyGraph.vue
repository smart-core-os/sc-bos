<template>
  <div>
    <bar-chart :chart-data="chartData" :chart-options="chartOptions"/>
  </div>
</template>

<script setup>
import {timestampToDate} from '@/api/convpb.js';
import {listOccupancySensorHistory} from '@/api/sc/traits/occupancy.js';
import BarChart from '@/components/charts/BarChart.vue';
import {computed, onMounted, onUnmounted, reactive, ref, watch} from 'vue';

const props = defineProps({
  source: {
    type: String,
    required: true
  },
  span: { // how wide the bars of the histogram are / group interval
    type: Number,
    default: 15 * 60 * 1000 // in ms
  },
  showYAxisLabel: {
    type: Boolean,
    default: false
  },
});

const pollDelay = computed(() => props.span / 10);
const now = ref(Date.now());
const nowHandle = ref(0);

const seriesMap = reactive({
  occupancy: {
    baseRequest: computed(() => {
      return baseRequest(props.source);
    }),
    data: [],
    handle: 0,
    records: /** @type {OccupancyRecord.AsObject[]} */ []
  }
});

//
//
// Computed
// Formatting the data for the bar chart
const series = computed(() => {
  return Object.entries(seriesMap).map(([seriesName]) => {
    const data = seriesMap[seriesName].data;

    if (data && data.length > 0) {
      return {name: 'Occupancy', data};
    } else {
      return null;
    }
  }).filter(obj => obj !== null);
});


//
//
// Methods
/**
 *
 * @param {OccupancyRecord.AsObject[]} records
 * @return {Array<{x: Date, y: number}>}
 */
const data = (records) => {
  const intervalsMap = [];
  const currentDate = new Date();

  // Adjusting current date to nearest half-hour mark
  currentDate.setMinutes(currentDate.getMinutes() - (currentDate.getMinutes() % 30), 0, 0);

  // Populate the bar chart with data
  // 24 hour divided into 30 min intervals
  for (let i = 0; i < 48; i++) {
    const dataPoint = {
      x: new Date(currentDate.getTime() - (i * 30 * 60 * 1000)),
      y: 0
    };

    intervalsMap.unshift(dataPoint); // Update the array of objects depending on the currentDate
  }

  // Split each hour into 30min intervals and group the records while finding the highest value
  for (const record of records) {
    const recordTime = new Date(timestampToDate(record.recordTime));
    const minute = recordTime.getMinutes();
    const intervalStart = new Date(recordTime); // Separating the hours into 30 min intervals
    intervalStart.setMinutes(minute < 30 ? 0 : 30, 0, 0); // Start of the interval

    // Looking for the existing interval record
    const existingInterval = intervalsMap.find(
        intrvl => intrvl.x.getTime() === intervalStart.getTime()
    );
    // Updating the interval record if higher peopleCount comes in
    if (existingInterval && record.occupancy.peopleCount > existingInterval.y) {
      existingInterval.y = record.occupancy.peopleCount;
    }
  }

  return intervalsMap;
};


// Return an array of object with request details
const baseRequest = (name) => {
  if (!name) return undefined;

  const period = {
    startTime: new Date(now.value - 24 * 60 * 60 * 1000)
  };

  return {
    name,
    period,
    pageSize: 1000,
    pageToken: ''
  };
};

/**
 *
 * @param {*} req
 * @param {string} type
 */
async function pollReadings(req, type) {
  const all = [];
  try {
    while (true) {
      const page = await listOccupancySensorHistory(req, {});

      all.push(...page.occupancyRecordsList);
      req.pageToken = page.nextPageToken;
      if (!req.pageToken) {
        break;
      }
    }
  } catch (e) {
    console.error('error getting occupancy readings', e);
  }

  seriesMap[type].records = all;
  seriesMap[type].handle = setTimeout(() => pollReadings(req, type), pollDelay.value);
}

//
//
// Bar graph styling and other options
const chartOptions = {
  animation: {
    duration: 500
  },
  layout: {
    padding: {
      top: 0,
      right: 10,
      bottom: 20,
      left: 10
    }
  },
  maintainAspectRatio: false,
  mode: 'none',
  responsive: true,
  plugins: {
    title: {
      display: true,
      text: '',
      color: '#f0f0f0',
      font: {
        size: 16
      },
      padding: {
        bottom: 0
      }
    },
    legend: {
      display: false
    },
    tooltip: {
      backgroundColor: 'rgba(0, 0, 0, 1)',
      padding: 12,
      cornerRadius: 5,
      borderColor: '#000',
      borderWidth: 2,
      titleColor: '#fff',
      bodyColor: '#fff',
      displayColors: false
    }
  },
  scales: {
    y: {
      border: {
        display: false
      },
      grid: {
        color: 'rgba(100, 100, 100, 0.35)',
        drawTicks: false,
      },
      ticks: {
        color: '#fff',
        display: true,
        font: {
          size: 12// Specify the desired font size
        }
      },
      title: (() => {
        if (props.showYAxisLabel) {
          return {
            display: true,
            text: 'People Count'
          };
        }
        return {
          display: false,
          text: ''
        }
      })(),
      min: 0
    },
    x: {
      border: {
        display: false
      },
      ticks: {
        align: 'center',
        color: '#fff',
        display: true,
        font: {
          size: 10// Specify the desired font size
        },
        // Limit xAxis label rotation to 0 degrees
        maxRotation: 0,
        minRotation: 0
      },
      grid: {
        color: '',
        drawTicks: false,
      }
    }
  }
};

const chartData = computed(() => {
  let labels = [];
  let data = [];
  const dataset = series.value[0].data;

  labels = dataset.map((data) => {
    const newDate = new Date(data.x);
    // removing seconds from the time
    const hour = newDate.getHours().toString().padStart(2, '0');
    const minute = newDate.getMinutes().toString().padStart(2, '0');
    const formattedTime = hour + ':' + minute;

    return formattedTime;
  });
  data = dataset.map((data) => data.y);

  // returning restructured data for the bar chart
  return {
    labels, // collection of time intervals
    datasets: [
      {
        label: 'Occupancy', // tooltip label
        maxBarThickness: 20,
        data,
        backgroundColor: '#C5CC3CBF',
        borderColor: '#fff',
        borderRadius: 5
      }
    ]
  };
});

//
//
// Watcher
/**
 * This function takes two arrays of records and combines them by matching the x values and updating the y values.
 *
 * @param {*} newRecords
 * @param {*} existingRecords
 * @return {Array} combinedRecords
 *
 * Map over the newRecords array and find the corresponding record in the existingRecords array.
 * If a match is found, update the y value with the larger of the two values.
 * If no match is found, return the new record.
 */
const handleRecords = (newRecords, existingRecords) => {
  const combinedRecords = newRecords.map((newRecord) => {
    const existingRecord = existingRecords.find(
        (record) => record.x.getTime() === newRecord.x.getTime()
    );

    if (existingRecord) {
      return {
        x: newRecord.x,
        y: newRecord.y > existingRecord.y ? newRecord.y : existingRecord.y
      };
    } else {
      return newRecord;
    }
  });

  return combinedRecords;
};

// Watch for changes to the seriesMap object
Object.entries(seriesMap).forEach(([name, series]) => {
  // watch for new base request
  watch(() => series.baseRequest, (request) => {
    clearTimeout(series.handle);
    series.records = [];

    // create new stream
    if (request) {
      pollReadings(request, name);
    }
  }, {immediate: true, deep: true, flush: 'sync'});

  // watch for new records
  watch(() => series.records, (records) => {
    const newRecords = data(records);

    if (!series.data.length) {
      series.data = data(records);
    } else {
      const existingRecords = series.data;
      series.data = handleRecords(newRecords, existingRecords);
    }
  }, {immediate: true, deep: true, flush: 'sync'});
});

//
//
// Lifecycle
onMounted(() => {
  nowHandle.value = setInterval(() => {
    now.value = Date.now();
  }, pollDelay.value);
});

onUnmounted(() => {
  clearInterval(nowHandle.value);
});
</script>

<style lang="scss" scoped>
</style>
