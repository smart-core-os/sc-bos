import {pullBootState} from '@/api/sc/traits/boot.js';
import {getResourceUse} from '@/api/sc/traits/resource-use.js';
import {closeResource, newResourceValue} from '@/api/resource.js';
import {timestampToDate} from '@/api/convpb.js';
import {getServiceMetadata} from '@/api/ui/services.js';
import {useCohortHealthStore} from '@/stores/cohort.js';
import {decomposeDuration} from '@/util/date.js';
import {downloadCSVRows} from '@/util/downloadCSV.js';
import {reactive, toValue, watch} from 'vue';

const csvHeaders = [
  'Name',
  'Address',
  'Role',
  'Connected',
  'Automations',
  'Drivers',
  'Systems',
  'CPU %',
  'Memory %',
  'Uptime',
  'Last Reboot Reason'
];

/**
 * Starts a pullBootState stream and resolves with the first value received, or null after 5s.
 *
 * @param {string} name
 * @return {Promise<BootState.AsObject|null>}
 */
function getBootStateOnce(name) {
  return new Promise((resolve) => {
    const resource = reactive(newResourceValue());
    pullBootState({name}, resource);
    const stop = watch(
        () => resource.value,
        (v) => {
          if (v != null) {
            stop();
            closeResource(resource);
            resolve(v);
          }
        },
        {immediate: true}
    );
    setTimeout(() => {
      stop();
      closeResource(resource);
      resolve(null);
    }, 5000);
  });
}

/**
 * Formats a boot state's bootTime as a human-readable uptime string relative to now.
 *
 * @param {BootState.AsObject|null} bootState
 * @return {string}
 */
function formatUptime(bootState) {
  const bt = bootState?.bootTime;
  if (!bt) return '';
  const {days, hours, minutes, seconds} = decomposeDuration(Date.now() - timestampToDate(bt));
  if (days > 0) return `${days}d ${hours}h ${minutes}m`;
  if (hours > 0) return `${hours}h ${minutes}m ${seconds}s`;
  if (minutes > 0) return `${minutes}m ${seconds}s`;
  return `${seconds}s`;
}

/**
 * Returns a human-readable connectivity string for a node using the health store's
 * latest poll result. 'Yes' means reachable with no error, 'No' means an error was
 * recorded, 'Unknown' means the health check has not completed yet.
 *
 * @param {string} nodeName
 * @param {import('@/stores/cohort.js').ReturnType<typeof useCohortHealthStore>} healthStore
 * @return {string}
 */
function nodeConnectedLabel(nodeName, healthStore) {
  const result = healthStore.resultsByName[nodeName];
  if (!result) return 'Unknown';
  if (toValue(result.pending)) return 'Unknown';
  return toValue(result.error) ? 'No' : 'Yes';
}

/**
 * Downloads a CSV snapshot of all cohort nodes and their current status.
 *
 * @param {import('@/stores/cohort.js').CohortNode[]} nodes
 * @return {Promise<void>}
 */
export async function downloadComponentsReport(nodes) {
  const healthStore = useCohortHealthStore();

  const rows = await Promise.all(nodes.map(async (node) => {
    const [ruResult, autoResult, drvResult, sysResult, bootResult] = await Promise.allSettled([
      getResourceUse({name: node.name}),
      getServiceMetadata({name: node.name + '/automations'}),
      getServiceMetadata({name: node.name + '/drivers'}),
      getServiceMetadata({name: node.name + '/systems'}),
      getBootStateOnce(node.name)
    ]);

    const ru = ruResult.status === 'fulfilled' ? ruResult.value : null;
    const auto = autoResult.status === 'fulfilled' ? autoResult.value : null;
    const drv = drvResult.status === 'fulfilled' ? drvResult.value : null;
    const sys = sysResult.status === 'fulfilled' ? sysResult.value : null;
    const boot = bootResult.status === 'fulfilled' ? bootResult.value : null;

    return [
      node.name ?? '',
      node.grpcAddress ?? '',
      node.role ?? '',
      nodeConnectedLabel(node.name, healthStore),
      auto?.totalActiveCount ?? '',
      drv?.totalActiveCount ?? '',
      sys?.totalActiveCount ?? '',
      ru?.cpu?.utilization != null ? ru.cpu.utilization.toFixed(1) : '',
      ru?.memory?.utilization != null ? ru.memory.utilization.toFixed(1) : '',
      formatUptime(boot),
      boot?.lastRebootReason ?? ''
    ];
  }));

  const now = new Date();
  const filename = `components_${now.getFullYear()}-${now.getMonth() + 1}-${now.getDate()}.csv`;
  downloadCSVRows(filename, [csvHeaders, ...rows]);
}
