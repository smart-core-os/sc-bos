import {timestampToDate} from '@/api/convpb';
import {listAirQualitySensorHistory} from '@/api/sc/traits/air-quality-sensor';
import {DAY, useNow} from '@/components/now';
import {useDevices} from '@/composables/devices';
import useTimePeriod from '@/composables/useTimePeriod';
import {hasTrait} from '@/util/devices';
import {csvDownload} from '@/util/downloadCSV';
import {AirQuality} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/airqualitysensor/v1/air_quality_sensor_pb';
import {computed, onMounted, onUnmounted, reactive, ref, watch, watchEffect} from 'vue';

/**
 *
 * @param {{
 *   name: import('vue').GetterOrRef<string>,
 *   span: import('vue').GetterOrRef<number>,
 *   timeFrame: import('vue').GetterOrRef<number>,
 *   pollDelay: import('vue').GetterOrRef<number>,
 *   filter: (v: Device.AsObject) => boolean
 * }} props
 * @return {{
 *   airQualitySensorHistoryValues: {},
 *   deviceOptions: import('vue').ComputedRef<{label: string, value: string}[]>,
 *   downloadCSV: (function(): void),
 *   isMounted: import('vue').Ref<boolean>,
 *   mappedDeviceNames: import('vue').ComputedRef<string[]>,
 *   periodEnd: import('vue').ComputedRef<Date>,
 *   periodStart: import('vue').ComputedRef<Date>,
 *   queryEnd: import('vue').ComputedRef<Date>,
 *   queryStart: import('vue').ComputedRef<Date>,
 *   setUpRequest: (function(string, number, string): void)
 * }}
 */
export default function(props) {
  // Pull in devices to check if any available
  const useDevicesOpts = computed(() => {
    return {
      conditions: [
        // todo: match against `field: 'metadata.traits.name'` when SC-618 is fixed
        {stringEqualFold: 'smartcore.traits.AirQualitySensor'}
      ],
      // todo: support either infinite scroll or search for selecting the IAQ sensor
      wantCount: 1000, // we have to define some limit here
      filter: props.filter
    };
  });
  const {items: devicesData} = useDevices(useDevicesOpts);

  // Filter zones to only those with air quality sensors
  const zonesWithAirQualitySensors = computed(() => {
    return devicesData.value
        .filter(zone => hasTrait(zone, 'smartcore.traits.AirQualitySensor'))
        .map(zone => zone.name);
  });
  // Mapping the device or zone names to an array
  const mappedDeviceNames = computed(() => {
    // Check if zoneList is available
    if (zonesWithAirQualitySensors.value) {
      // Return only the zone.id in alphabetical order
      return zonesWithAirQualitySensors.value.map(zone => zone).sort();

      // Check if props data has been passed in
    } else if (props.name.length) {
      if (typeof props.name === 'string') {
        return [props.name];
      } else if (Array.isArray(props.name)) {
        return props.name;
      }
    }

    // If all else fails, return the device names
    return devicesData.value.map(device => device.name);
  });
  // Mapping the device names to an array of objects for the dropdown
  const deviceOptions = computed(() => mappedDeviceNames.value.map(
      device => {
        return {
          label: device,
          value: device
        };
      }
  ));
  const airDevice = ref('');
  const previousAirDevice = ref(''); // Store the previous device name

  // Time period and now
  const {now} = useNow(() => props.span);
  const {periodStart: downloadPeriodStart, periodEnd: downloadPeriodEnd} = useTimePeriod(
      now, () => props.timeFrame * 30 * DAY, () => props.span
  );
  const {periodStart, periodEnd} = useTimePeriod(now, () => props.timeFrame, () => props.span);
  const queryStart = computed(() => new Date(periodStart.value.getTime() - props.span * 2));
  const queryEnd = computed(() => periodEnd.value);

  // Flag to indicate if the component is still mounted
  const isMounted = ref(false);
  const isFetching = ref(false);

  // Data and polling state
  const airQualitySensorHistoryValues = reactive({});

  // Initialize data and polling state for each device
  const initializeAirQualityData = (deviceName) => {
    if (!airQualitySensorHistoryValues[deviceName]) {
      airQualitySensorHistoryValues[deviceName] = {
        data: [],
        fetching: false,
        lastSuccessfulFetchTime: 0,
        pollHandler: 0,
        records: [],
        request: {}
      };
    }
    setUpRequest(deviceName);
  };


  // Setting up the request object
  const setUpRequest = (deviceName, pageSize, pageToken) => {
    const request = {
      name: deviceName,
      pageSize: pageSize || 1000,
      pageToken: pageToken || '',
      timePeriod: {
        startTime: periodStart.value,
        endTime: periodEnd.value
      }
    };
    // Ensuring reactivity when setting up the request
    airQualitySensorHistoryValues[deviceName].request = request;
  };

  // Watch airDevice.value to trigger re-initialization and data fetch
  watch(airDevice, (newDevice) => {
    if (newDevice && !airQualitySensorHistoryValues[newDevice]) {
      initializeAirQualityData(newDevice);
    }
  });

  // Function to fetch data
  // This function will fetch all pages of data and store it in the records array
  // It will also set the fetching flag to true while fetching
  const fetchData = async (device) => {
    const currentTime = Date.now();
    const deviceData = airQualitySensorHistoryValues[device];

    if (deviceData.fetching || isFetching.value) return;

    // Check if enough time has passed since the last fetch
    // Separate variable to track the time since the last successful fetch
    const timeSinceLastFetch = currentTime - (deviceData.lastSuccessfulFetchTime || 0);

    // Check if enough time has passed since the last successful fetch
    if (timeSinceLastFetch < props.pollDelay) {
      console.warn('Not enough time has passed since last fetch for device:', device);
      return;
    }

    deviceData.fetching = true;
    isFetching.value = true;

    try {
      const listAction = listAllPages(deviceData.request, deviceData.records);
      deviceData.records = await listAction;
      deviceData.lastSuccessfulFetchTime = Date.now(); // Update last successful fetch time
    } catch (e) {
      console.error('Error fetching air quality data:', e);
    } finally {
      deviceData.fetching = false;
      isFetching.value = false;
    }

    // Initialize polling for each device
    deviceData.pollHandler = setInterval(
        () => fetchData(device), props.pollDelay);
  };


  // If the airQualitySensorHistoryValues initialized and the request object is set up,
  // loop through each device and fetch data
  watchEffect(async () => {
    // Check if there are any device names available
    if (mappedDeviceNames.value) {
      // If airDevice is not set, set it to the first device
      if (!airDevice.value) {
        airDevice.value = mappedDeviceNames.value[0];
      }

      const currentDevice = airDevice.value;

      // Check if the current device is different from the previous one
      if (previousAirDevice.value !== currentDevice) {
        // If there was a previous device, delete its data
        if (previousAirDevice.value && airQualitySensorHistoryValues[previousAirDevice.value]) {
          delete (airQualitySensorHistoryValues[previousAirDevice.value]);
        }

        // Initialize data for the current device
        if (!airQualitySensorHistoryValues[currentDevice]) {
          await initializeAirQualityData(currentDevice);
        }

        // Fetch data for the current device if it hasn't been fetched yet
        if (airQualitySensorHistoryValues[currentDevice]?.request &&
            !airQualitySensorHistoryValues[currentDevice].lastFetchTime) {
          await fetchData(currentDevice);
        }

        // Update the previousDevice value
        previousAirDevice.value = currentDevice;
      }
    }
  });


  // ---- Lifecycle hooks ---- //
  onMounted(async () => {
    isMounted.value = true;
  });

  onUnmounted(() => {
    // Clear the pollHandler
    clearPollHandler();
    isMounted.value = false;
  });


  // ---- Helper functions ---- //
  // Function to fetch all pages of data
  /**
   *
   * @param {ListAirQualityHistoryRequest.AsObject} request
   * @param {AirQualityRecord.AsObject[]} existingRecords
   * @param {string} type
   * @return {Promise<AirQualityRecord.AsObject[]>}
   */
  async function listAllPages(request, existingRecords = [], type = '') {
    if (!request) return [];

    const allRecords = [...existingRecords];
    let pageToken = '';

    do {
      request.pageToken = pageToken;

      try {
        const page = await listAirQualitySensorHistory(request, {});
        allRecords.push(...page.airQualityRecordsList);
        pageToken = page.nextPageToken;

        // Check if component is still mounted before continuing
        if (!page.nextPageToken) break;
      } catch (e) {
        console.error('Error in listAllPages:', e);
        break; // Stop fetching on error
      }
    } while (pageToken);

    const removeDuplicates = (records) => {
      const seen = new Set();
      return records.filter(record => {
        const duplicate = seen.has(record.recordTime);
        seen.add(record.recordTime);
        return !duplicate;
      });
    };

    // Filter records to include only those within the specified time frame
    const filteredRecords = filterRecordsByTimeFrame(allRecords);

    // Remove duplicates and set data for graph
    const uniqueRecords = removeDuplicates(filteredRecords);


    // Set the records and data for the graph
    if (!type) {
      airQualitySensorHistoryValues[request.name].records = uniqueRecords;
      airQualitySensorHistoryValues[request.name].data = processRecordsForGraph(uniqueRecords);
    }

    return uniqueRecords;
  }

  // Function to read the comfort value
  const readComfortValue = (level) => {
    switch (level) {
      case AirQuality.Comfort.COMFORT_UNSPECIFIED:
        return 'Unspecified';
      case AirQuality.Comfort.COMFORTABLE:
        return 'Comfortable';
      case AirQuality.Comfort.UNCOMFORTABLE:
        return 'Uncomfortable';
      default:
        return;
    }
  };


  // ---- Acronyms ---- //
  const acronyms = {
    carbonDioxideLevel: {label: 'CO₂', unit: 'ppm'},
    volatileOrganicCompounds: {label: 'VOC', unit: 'ppm'},
    airPressure: {label: 'Air Pressure', unit: 'hPa'},
    comfort: {label: 'Comfort', unit: ''},
    infectionRisk: {label: 'Infection Risk', unit: '%'},
    score: {label: 'Air Quality Score', unit: '%'}
  };

  // ---- Graph data ---- //
  const processRecordsForGraph = (records) => {
    return records.map(record => {
      return {
        y: {...record.airQuality},
        x: timestampToDate(record.recordTime)
      };
    });
  };

  // ---- CSV download ---- //
  const flatteningRecords = (records, deviceName) => {
    return records.map(record => {
      const recordDate = timestampToDate(record.recordTime);
      return {
        deviceName,
        ...record.airQuality,
        comfort: readComfortValue(record.airQuality.comfort),
        recordTime: `${recordDate.toLocaleDateString()} ${recordDate.toLocaleTimeString()}`
      };
    });
  };

  const fetchRecordsForCSV = async (deviceName) => {
    const request = {
      name: deviceName,
      pageSize: 1000,
      pageToken: '',
      timePeriod: {
        startTime: downloadPeriodStart.value,
        endTime: downloadPeriodEnd.value
      }
    };

    return await listAllPages(request, [], 'csv');
  };

  // Using a utility which handles the CSV download after setting up the correct parameters
  const downloadCSV = async () => {
    csvDownload({
      acronyms,
      docType: 'Air Quality',
      flattenRecords: (records) => flatteningRecords(records, airDevice.value),
      records: async () => await fetchRecordsForCSV(airDevice.value),
      deviceName: airDevice
    });
  };


  // ---- Cleanup ---- //
  // Function to clear the pollHandler
  const clearPollHandler = () => {
    if (!airDevice.value) return;

    clearInterval(airQualitySensorHistoryValues[airDevice.value].pollHandler);
    delete (airQualitySensorHistoryValues[airDevice.value]);
  };

  // Function to remove records outside the time frame
  const filterRecordsByTimeFrame = (records) => {
    const timeFrameStart = new Date(now.value.getTime() - props.timeFrame);
    return records.filter(record => {
      const recordTime = timestampToDate(record.recordTime);
      return recordTime >= timeFrameStart && recordTime <= now.value;
    });
  };

  // ---- Return ---- //
  return {
    acronyms,
    airQualitySensorHistoryValues,
    deviceOptions,
    airDevice,
    downloadCSV,
    isMounted,
    isFetching,
    mappedDeviceNames,
    periodEnd,
    periodStart,
    queryEnd,
    queryStart,
    setUpRequest,
    readComfortValue,
    clearPollHandler
  };
}
