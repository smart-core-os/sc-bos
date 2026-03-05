import {newActionTracker} from '@/api/resource';
import {startService, stopService} from '@/api/ui/services';
import {useErrorStore} from '@/components/ui-error/error';
import {useServicesCollection} from '@/composables/services.js';
import {useCohortStore} from '@/stores/cohort.js';
import {useUserConfig} from '@/stores/userConfig.js';
import {useSidebarStore} from '@/stores/sidebar';
import {computed, onMounted, onUnmounted, reactive, ref, toValue, watch, watchEffect} from 'vue';

/**
 * @param {{
 * name: MaybeRefOrGetter<string>,
 * type: MaybeRefOrGetter<string>
 * }} props
 * @return {{
 * serviceList: ComputedRef<Service.AsObject[]>,
 * serviceCollection: UseCollectionResponse<Service.AsObject>,
 * loading: Ref<boolean>,
 * showService: function(Service.AsObject): void,
 * startStopTracker: ActionTracker,
 * search: Ref<string>,
 * node: ComputedRef<Node.AsObject>,
 * nodesListValues: ComputedRef<Node.AsObject[]>,
 * _startService: function(Service.AsObject): Promise<void>,
 * _stopService: function(Service.AsObject): Promise<void>
 * }}
 */
export default function(props) {
  const sidebar = useSidebarStore();
  const errors = useErrorStore();
  const cohort = useCohortStore();

  const startStopTracker = reactive(
      /** @type {ActionTracker<Service.AsObject>} */
      newActionTracker()
  );

  // query fields
  const search = ref('');

  const userConfig = useUserConfig();
  const serviceName = computed(() => `${userConfig.node?.name}/${props.name}`);
  const serviceCollection = useServicesCollection(serviceName, computed(() => ({
    paused: !userConfig.node?.name,
    wantCount: -1 // there's no server search features, so we have to get them all and do it client side
  })));

  // Available services to select from
  const serviceList = computed(() => {
    return serviceCollection.items.value.filter(service => {
      const type = toValue(props.type);
      return type === '' || type === 'all' || service.type === type;
    });
  });

  // Available nodes to select from
  const nodesListValues = computed(() => cohort.cohortNodes);

  // Make sure there's a node selected
  watchEffect(() => {
    if (!userConfig.node?.name) {
      if (nodesListValues.value.length > 0) {
        userConfig.node = nodesListValues.value[0];
      }
    }
  });

  /**
   *
   * @param {Service.AsObject} service
   */
  function showService(service) {
    sidebar.visible = true;
    sidebar.title = service.id;
    sidebar.data = {service, config: service.configRaw ? JSON.parse(service.configRaw) : {}, nodeName: userConfig.node?.name};
  }

  /**
   *
   * @param {Service.AsObject} service
   */
  async function _startService(service) {
    // Update the data if the sidebar is open and the service is being started
    if (sidebar.visible && sidebar.data?.service?.id !== service.id) {
      sidebar.title = service.id;
      sidebar.data = {service, config: service.configRaw ? JSON.parse(service.configRaw) : {}, nodeName: userConfig.node?.name};
    }

    await startService({
      name: serviceName.value,
      id: service.id
    }, startStopTracker);
  }

  /**
   *
   * @param {Service.AsObject} service
   */
  async function _stopService(service) {
    // Update the data if the sidebar is open and the service is being stopped
    if (sidebar.visible && sidebar.data?.service?.id !== service.id) {
      sidebar.title = service.id;
      sidebar.data = {service, config: service.configRaw ? JSON.parse(service.configRaw) : {}, nodeName: userConfig.node?.name};
    }

    await stopService({
      name: serviceName.value,
      id: service.id
    }, startStopTracker);
  }

  // Watch for changes in the serviceList and update the sidebar data if needed.
  // This is necessary if we want to update the status details in the sidebar
  // simultaneously with the status details in the service list.
  // Mainly when the sidebar is open and the service is being started/stopped.
  watch(
      serviceList,
      (newServiceList, oldServiceList) => {
        if (!sidebar.data?.service?.id) return;

        // Find the service in the new list that matches the id in data
        const updatedService = newServiceList.find(s => s.id === sidebar.data.service.id);

        if (updatedService) {
          // Check if the service has been updated by comparing it with the old list
          const oldService = oldServiceList.find(s => s.id === updatedService.id);

          // Perform a deep comparison if necessary, for now, we just check if the old service exists
          if (!oldService || JSON.stringify(updatedService) !== JSON.stringify(oldService)) {
            // Update the data with the new service data
            // Ensuring to parse the config if it's in a raw JSON string format
            sidebar.data = {
              service: updatedService,
              config: updatedService.configRaw ? JSON.parse(updatedService.configRaw) : {},
              nodeName: userConfig.node?.name
            };
          }
        }
      },
      {immediate: true, deep: true} // Watch for nested changes within the serviceList array
  );

  //
  //
  // UI error handling
  let unwatchStartStopErrors;
  onMounted(() => {
    unwatchStartStopErrors = errors.registerTracker(startStopTracker);
  });
  onUnmounted(() => {
    if (unwatchStartStopErrors) unwatchStartStopErrors();
  });

  return {
    serviceName,
    serviceList,
    loading: serviceCollection.loading,
    serviceCollection,
    showService,
    startStopTracker,
    search,
    nodesListValues,
    _startService,
    _stopService
  };
}
