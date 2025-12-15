<template>
  <div class="chart__container">
    <line-chart
        :chart-options="chartOptions"
        :chart-data="chartData"
        :chart-title="props.chartTitle"
        dataset-id-key="label"
        :hide-legends="props.hideLegends">
      <template #options>
        <power-history-graph-options-menu
            v-model:duration-option="durationOption"
            v-model:show-conversion="showConversion"
            @export-csv="onDownloadClick"/>
      </template>
    </line-chart>
    <div v-if="!hasData" class="no-data-overlay">
      <no-data-graphic class="no-data-graphic"/>
    </div>
  </div>
</template>

<script setup>
import LineChart from '@/components/charts/LineChart.vue';
import {triggerDownload} from '@/components/download/download.js';
import {HOUR, MINUTE, useNow} from '@/components/now.js';
import useTimePeriod from '@/composables/useTimePeriod.js';
import PowerHistoryGraphOptionsMenu from '@/dynamic/widgets/power-history/PowerHistoryGraphOptionsMenu.vue';
import useMeterHistory from '@/dynamic/widgets/power-history/useMeterHistory.js';
import {useCarbonIntensity} from '@/stores/carbonIntensity.js';
import {computed, ref} from 'vue';
import {useTheme} from 'vuetify';
import NoDataGraphic from '@/dynamic/widgets/general/no-data-in-date-range.svg';

const props = defineProps({
  chartTitle: {
    type: String,
    default: 'Power'
  },
  hideLegends: {
    type: Boolean,
    default: false
  },
  demand: {
    type: String,
    required: true
  },
  generated: {
    type: String,
    default: null
  },
  width: {
    type: String,
    default: '100%'
  },
  height: {
    type: String,
    default: '275px'
  }
});
const theme = useTheme();
const durationOption = ref({
  id: '24H',
  span: 20 * MINUTE,
  timeFrame: 24 * HOUR
});
const {now} = useNow(() => durationOption.value.span);
const {periodStart, periodEnd} = useTimePeriod(
    now,
    () => durationOption.value.span,
    () => durationOption.value.timeFrame
);
const showConversion = ref(false);
const carbonIntensity = useCarbonIntensity();
const gramsOfCO2PerKWh = ref(86);

const themeColor = computed(() => {
  return {
    start: theme.current.value.colors.primary,
    middle: theme.current.value.colors.primary + '80',
    end: theme.current.value.colors.primary + '19'
  };
});

// Simplify co2intervals computation by mapping duration IDs to carbonIntensity properties directly
const co2intervals = computed(() => {
  const mapping = {
    '24H': carbonIntensity.last24Hours,
    '1W': carbonIntensity.last7Days,
    '30D': carbonIntensity.last30Days
  };
  return mapping[durationOption.value.id] || [];
});

// Helper function to convert kWh to grams of CO2 based on the date and the carbon intensity data
const kwhToGramsOfCO2 = date => {
  const co2source = co2intervals.value;

  // Handles empty co2source array by returning the default value if no data is available
  if (!co2source.length) return gramsOfCO2PerKWh.value;

  const first = co2source[0]; // Get the first element of the array
  const last = co2source.at(-1); // Get the last element of the array

  // Handles dates before the first range
  if (date < first.from) {
    return getIntensityValue(first);
  }

  // Handles dates after the last range
  if (date > last.to) {
    return getIntensityValue(last);
  }

  // Finds and returns intensity for a date within the ranges
  const matchingRange = co2source.find(range => date >= range.from && date <= range.to);
  return matchingRange ? getIntensityValue(matchingRange) : getIntensityValue(last);
};

// Helper function to get the actual or forecasted intensity value
const getIntensityValue = (range) => range.intensity.actual ?? range.intensity.forecast;

// Set the yAxis unit based on the `showConversion` value
const yAxisUnit = computed(() => showConversion.value ? 'Grams of CO₂ / hour' : 'kW');

// Fetch the demand and generated data based on the periodStart and periodEnd values and the durationOption's span
const {seriesData: demandSeriesData} =
    useMeterHistory(() => props.demand, periodStart, periodEnd, () => durationOption.value.span);
const {seriesData: generatedSeriesData} =
    useMeterHistory(() => props.generated, periodStart, periodEnd, () => durationOption.value.span);

// download CSV functionality
const onDownloadClick = async () => {
  const names = [];
  if (props.demand) names.push(props.demand);
  if (props.generated) names.push(props.generated);
  if (names.length === 0) return null;

  await triggerDownload(
      props.chartTitle.toLowerCase().replace(' ', '-'),
      {conditionsList: [{stringIn: {stringsList: names}}]},
      {startTime: periodStart.value, endTime: periodEnd.value},
      {
        includeColsList: [
          {name: 'timestamp', title: 'Timestamp'},
          {name: 'md.name', title: 'Device Name'},
          {name: 'meter.usage', title: 'Meter Reading (kWh)'},
        ]
      }
  )
}

// Helper function to compute the CO2 series data based on the seriesData value and the kwhToGramsOfCO2 function
const computeCO2SeriesData = seriesData => seriesData.value.map(({x, y}) => ({x, y: y * kwhToGramsOfCO2(x)}));

// Computed properties to compute the CO2 series data for the demand and generated data
const co2Demand = computed(() => computeCO2SeriesData(demandSeriesData));
const co2Generated = computed(() => computeCO2SeriesData(generatedSeriesData));

// ----------------- Chart Data and Options ----------------- //
const chartData = computed(() => {
  const datasets = [];

  // Function to create a gradient; improves readability and reusability
  const createGradient = (ctx) => {
    const canvas = ctx.chart.ctx;
    const gradient = canvas.createLinearGradient(0, 0, 0, 425);
    gradient.addColorStop(0, themeColor.value.start);
    gradient.addColorStop(0.5, themeColor.value.middle);
    gradient.addColorStop(1, themeColor.value.end); // Use predefined value
    return gradient;
  };

  // Set Generated color to success-lighten-3 (light green) by default
  const generatedColor = theme.current.value.colors['success-lighten-3'];

  // Helper function to avoid redundancy in dataset creation
  const addDataset = (data, isDemand = false) => {
    datasets.push({
      borderColor: isDemand ? themeColor.value.start : generatedColor,
      data: data,
      fill: isDemand,
      label: isDemand ? 'Demand' : 'Generated',
      backgroundColor: isDemand ? createGradient : undefined,
      pointBackgroundColor: 'rgba(0, 0, 0, 0)',
      pointBorderColor: 'rgba(0, 0, 0, 0)',
      pointHoverBackgroundColor: 'rgb(255, 255, 255)',
      pointHoverBorderColor: isDemand ? themeColor.value.start : generatedColor,
      pointStyle: 'circle',
      tension: 0.35
    });
  };

  // Decide which data to use based on `showConversion` and whether props are provided
  if (props.generated) {
    const generatedData = showConversion.value ? co2Generated.value : generatedSeriesData.value;
    addDataset(generatedData, false);
  }

  if (props.demand) {
    const demandData = showConversion.value ? co2Demand.value : demandSeriesData.value;
    addDataset(demandData, true);
  }

  return {datasets};
});

const hasData = computed(() => {
  return chartData.value.datasets.some(ds => ds.data.some(point => point.y !== 0 && point.y != null));
});

const chartOptions = computed(() => {
  return /** @type {ChartOptions<line>} */ {
    animation: {
      duration: 500
    },
    hover: {
      intersect: false
    },
    layout: {
      padding: {
        top: 0,
        right: 10,
        bottom: 0,
        left: 10
      }
    },
    maintainAspectRatio: false,
    responsive: true,
    plugins: {
      htmlLegend: {
        // ID of the container to put the legend in
        containerID: 'legend-container'
      },
      legend: {
        display: false // Legend being overwritten by htmlLegend plugin
      },
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
      tooltip: {
        backgroundColor: '#000',
        bodyColor: '#fff',
        borderColor: '#000',
        callbacks: {
          // Format the tooltip title to Month Date, 24 Hour:Minute
          title: (data) => {
            const date = new Date(data[0].parsed.x);
            return date.toLocaleString('en-GB', {
              month: 'short',
              day: 'numeric',
              hour: 'numeric',
              minute: 'numeric',
              hour12: false
            });
          },
          labelPointStyle: function() {
            return {
              pointStyle: 'line',
              rotation: 0
            };
          },
          labelColor: function(context) {
            return {
              borderColor: context.dataset.borderColor,
              backgroundColor: context.dataset.borderColor,
              borderWidth: 2,
              borderDash: [2, 2]
            };
          }
        },
        borderWidth: 2,
        cornerRadius: 5,
        displayColors: true,
        enabled: true,
        interaction: {
          axis: 'xy',
          mode: 'index'
        },
        padding: 12,
        titleColor: '#fff',
        usePointStyle: true
      }
    },
    scales: {
      y: {
        border: {
          color: 'white'
        },
        grid: {
          color: 'rgba(100, 100, 100, 0.35)'
        },
        ticks: {
          color: '#fff',
          display: true,
          font: {
            size: 12 // Specify the desired font size
          }
        },
        title: {
          display: true,
          text: yAxisUnit.value
        },
        min: 0
      },
      x: {
        border: {
          color: 'white'
        },
        ticks: {
          align: 'center',
          // autoSkip: true,
          callback: (value) => {
            // Format the xAxis label to either Month Date or 24 Hour:Minute
            const date = new Date(value);

            if (date.getHours() === 0) {
              return date.toLocaleString('en-GB', {
                day: 'numeric',
                month: 'short'
              });
            } else {
              return date.toLocaleString('en-GB', {
                hour: 'numeric',
                minute: 'numeric',
                hour12: false
              });
            }
          },
          color: '#fff',
          display: true,
          font: {
            size: 11 // Specify the desired font size
          },
          includeBounds: true, // Include the first and last ticks
          maxRotation: 0 // Limit xAxis label rotation to 0 degrees
        },
        grid: {
          color: ''
        },
        time: {
          displayFormats: {
            hour: 'HH : mm' // Format the xAxis label to 24 Hour:Minute
          },
          min: '00:00', // Set the min time to 00:00
          max: '23:59', // Set the max time to 23:59
          stepSize: 1, // Display a label for every hour
          unit: 'hour'
        },
        type: 'time'
      }
    },
    type: 'line'
  };
});
</script>

<style scoped lang="scss">
.chart__container {
  position: relative;
  height: 100%;
}
</style>

