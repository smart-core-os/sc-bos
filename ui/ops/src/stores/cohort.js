import {newActionTracker} from '@/api/resource.js';
import {getEnrollment, testEnrollment} from '@/api/sc/traits/enrollment.js';
import {testHubNode} from '@/api/sc/traits/hub.js';
import {useHubNodesCollection} from '@/composables/hub.js';
import {usePoll} from '@/composables/poll.js';
import {useIsGateway} from '@/composables/services.js';
import {useControllerStore} from '@/stores/controller.js';
import {isNetworkError} from '@/util/error.js';
import deepEqual from 'fast-deep-equal';
import {StatusCode} from 'grpc-web';
import {acceptHMRUpdate, defineStore, storeToRefs} from 'pinia';
import {computed, effectScope, reactive, watch} from 'vue';

export const EnrollmentStatus = {
  UNKNOWN: 'unknown',
  ENROLLED: 'enrolled',
  NOT_ENROLLED: 'not-enrolled'
};
export const HubApiStatus = {
  UNKNOWN: 'unknown',
  AVAILABLE: 'available',
  UNAVAILABLE: 'unavailable'
};
export const NodeRole = {
  UNKNOWN: 'unknown', // we don't know
  INDEPENDENT: 'independent', // an independent node, not enrolled and not a hub
  NODE: 'node', // a node that is enrolled with a hub with no other special roles
  GATEWAY: 'gateway', // an aggregating gateway
  HUB: 'hub' // the hub
};

/**
 * @typedef {Object} CohortNode
 * @property {keyof typeof NodeRole} role
 * @property {string} name
 * @property {string?} grpcAddress
 * @property {string?} grpcWebAddress
 * @property {boolean?} isServer - is this the server we are communicating with
 * @property {RpcError?} error - any error encountered fetching information about the node
 */

// A store that reports on the hub and enrolled nodes for the server the ui is connected to.
export const useCohortStore = defineStore('cohort', () => {
  // this is the information we base all our decisions about the cohort on
  const enrollmentTracker = reactive(
      /** @type {ActionTracker<Enrollment.AsObject>} */
      newActionTracker());
  const {
    items: hubNodes,
    loadingNextPage: hubNodesLoading,
    errors: hubNodesErrors,
    _listTracker: _hubListTracker,
    _pullResource: _hubPullResource
  } = useHubNodesCollection({}, {
    wantCount: -1 // we just want them all
  });

  // being enrolled is not binary, we can be enrolled or not enrolled, and we can be unsure.
  // We are unsure if either we haven't asked or our queries are failing with network errors.
  const enrollmentErr = computed(() => {
    const err = enrollmentTracker.error?.error;
    if (err) err.from = 'useCohortStore.enrollmentTracker';
    return err;
  });
  const enrollmentStatusKnown = computed(() => {
    if (enrollmentTracker.response) return true;
    const err = enrollmentErr.value;
    if (!err) return false; // no response or err means we're still waiting
    return !isNetworkError(err);
  });
  // one of EnrollmentStatus
  const enrollmentStatus = computed(() => {
    if (!enrollmentStatusKnown.value) return EnrollmentStatus.UNKNOWN;
    return enrollmentTracker.response ? EnrollmentStatus.ENROLLED : EnrollmentStatus.NOT_ENROLLED;
  });

  // Returns one of HubApiStatus.
  const hubApiStatus = computed(() => {
    if (hubNodes.value.length > 0) return HubApiStatus.AVAILABLE;
    if (hubNodesErrors.value.length > 0) {
      if (hubNodesErrors.value.every(e => isNetworkError(e.error))) return HubApiStatus.UNKNOWN;
      const notAHubError = (e) => {
        const code = e.error?.code;
        return code === StatusCode.FAILED_PRECONDITION || code === StatusCode.UNIMPLEMENTED;
      };
      if (hubNodesErrors.value.every(notAHubError)) return HubApiStatus.UNAVAILABLE;
      return HubApiStatus.AVAILABLE;
    }
    return hubNodesLoading.value ? HubApiStatus.UNKNOWN : HubApiStatus.UNAVAILABLE;
  });

  // for hub nodes we have to fetch their roles via the enabled services they have.
  const nodeGatewayChecks = reactive(
      /** @type {Record<string, {stop: () => {}, isGateway: boolean, loading: boolean, error: ResourceError}>} */
      {});
  const gatewayQueryNodes = computed(() => hubNodes.value
      .map(node => node.name)
      .filter(name => serverName.value !== name));
  watch(gatewayQueryNodes, async (nodes) => {
    const toDelete = new Set(Object.keys(nodeGatewayChecks));
    for (const name of nodes) {
      if (nodeGatewayChecks[name]) {
        // the name was and still is something we want to check
        toDelete.delete(name);
        continue;
      }
      // the gateway system used to be called the proxy system, so check for both
      const scope = effectScope();
      const task = scope.run(() => {
        return useIsGateway(name);
      });
      task.stop = () => scope.stop();
      nodeGatewayChecks[name] = task;
    }

    for (const name of toDelete) {
      nodeGatewayChecks[name].stop();
      delete nodeGatewayChecks[name];
    }
  });
  // returns the node role we think name has based on their active systems.
  const discoveredNodeRole = (name) => {
    const service = nodeGatewayChecks[name];
    if (!service) return NodeRole.UNKNOWN;
    if (service.loading) return NodeRole.UNKNOWN;
    if (service.isGateway) return NodeRole.GATEWAY;
    if (service.error?.error?.code === StatusCode.NOT_FOUND) return NodeRole.NODE;
    return NodeRole.UNKNOWN;
  };

  // Reports the NodeRole for the server the ui is connected to.
  const serverRole = computed(() => {
    const eStat = enrollmentStatus.value;
    const hStat = hubApiStatus.value;
    if (eStat === EnrollmentStatus.UNKNOWN || hStat === HubApiStatus.UNKNOWN) return NodeRole.UNKNOWN;
    if (hStat === HubApiStatus.AVAILABLE) {
      if (eStat === EnrollmentStatus.ENROLLED) return NodeRole.GATEWAY;
      return NodeRole.HUB;
    }
    if (eStat === EnrollmentStatus.ENROLLED) return NodeRole.NODE;
    return NodeRole.INDEPENDENT;
  });

  const {
    controllerName: serverName,
    hasLoaded: serverNameLoaded,
    controllerNameError: serverNameError,
    host: serverAddress
  } = storeToRefs(useControllerStore());
  // a CohortNode that represents the server we are communicating with
  const serverNode = computed(() => {
    const node = /** @type {CohortNode} */ {
      role: serverRole.value,
      isServer: true,
      grpcWebAddress: serverAddress.value,
      grpcAddress: enrollmentTracker.response?.targetAddress,
      name: serverName.value,
      error: serverNameError.value
    };
    if (node.error?.code === StatusCode.UNAUTHENTICATED) {
      // If the user isn't logged in, then we can't get the name of the server.
      // This is fine, we just don't know it.
      node.error = null;
    }
    if (!node.error && node.role === NodeRole.NODE || node.role === NodeRole.GATEWAY) {
      // we don't care about enrollment errors if the node is independent or a hub
      node.error = enrollmentErr.value;
    }
    return node;
  });
  // a CohortNode that represents the hub we are enrolled with, if any.
  const hubNode = computed(() => {
    return /** @type {CohortNode} */ {
      role: NodeRole.HUB,
      grpcAddress: enrollmentTracker.response?.managerAddress,
      name: enrollmentTracker.response?.managerName
    };
  });

  // All nodes in the cohort, including the hub.
  const cohortNodes = computed(() => {
    const res = /** @type {CohortNode[]} */ [];
    if (serverAddress.value === null) return res; // if we don't know our own server, we can't say anything
    if (!serverNameLoaded.value) return res; // technically, not universally needed, but makes the code cleaner
    switch (serverRole.value) {
      case NodeRole.UNKNOWN:
        res.push(serverNode.value);
        break;
      case NodeRole.INDEPENDENT:
        res.push(serverNode.value);
        break;
      case NodeRole.NODE:
        res.push(hubNode.value);
        res.push(serverNode.value);
        break;
      case NodeRole.HUB:
        res.push(serverNode.value);
        res.push(...hubNodes.value.map(node => ({
          role: NodeRole.UNKNOWN, // we can't ask the nodes what they are via the hub, only the gateway
          grpcAddress: node.address,
          name: node.name
        })));
        break;
      case NodeRole.GATEWAY:
        res.push(hubNode.value);
        res.push(...hubNodes.value.map(node => {
          if (node.name === serverName.value) {
            // it's us
            return serverNode.value;
          }
          return {
            role: discoveredNodeRole(node.name),
            grpcAddress: node.address,
            name: node.name
          };
        }));
    }
    return res;
  });

  // The enrollment api doesn't support pull, so setup a polling schedule
  const {nextPoll, pollNow} = usePoll(() => {
    return getEnrollment(enrollmentTracker)
        // the tracker will handle errors
        .catch(() => {});
  });

  return {
    enrollmentStatus,
    hubApiStatus,
    serverRole,
    serverNode,
    hubNode,
    cohortNodes,

    loading: computed(() => {
      return enrollmentTracker.loading || hubNodesLoading.value;
    }),

    enrollmentTracker,
    hubNodes,
    hubNodesErrors,

    nextPoll,
    pollNow,

    _hubListTracker,
    _hubPullResource
  };
});

/**
 * @typedef {Object} NodeTestResults
 * @property {boolean?} pending - are checks still in progress
 * @property {RpcError?} error - was there an error checking the node
 */

// Reports on the health of each of the nodes in the cohort.
export const useCohortHealthStore = defineStore('cohortHealth', () => {
  const cohort = useCohortStore();
  const resultsByName = reactive(/** @type {Record<string, NodeTestResults>} */ {});

  const checkHealth = async () => {
    const tasks = /** @type {PromiseLike<unknown>[]} */ []; // promises to wait on before returning

    for (const node of cohort.cohortNodes) {
      // todo: get the service status for all nodes and have them contribute to the health check

      // we perform different check depending on the relationship between the node and this ui
      if (node.isServer) {
        // we know that if we got here then comms with the server are fine, so report it as healthy
        resultsByName[node.name] = {pending: false, error: node.error};
      } else if (node.role === NodeRole.HUB) {
        const tracker = reactive(newActionTracker());
        resultsByName[node.name] = {
          pending: computed(() => tracker.loading),
          error: computed(() => {
            const commErr = node.error ?? tracker.error?.error ?? tracker.error;
            if (commErr) return commErr;
            const res = tracker.response;
            if (!res) return null; // still working
            // the testEnrollment api returns a successful code, but a payload with an error if the hub is down
            if (res.code !== StatusCode.OK) {
              return {
                code: res.code,
                message: res.error
              };
            }
            return null;
          })
        };
        tasks.push(testEnrollment(tracker));
      } else {
        const tracker = reactive(newActionTracker());
        resultsByName[node.name] = {
          pending: computed(() => tracker.loading),
          error: computed(() => node.error ?? tracker.error?.error ?? tracker.error)
        };
        tasks.push(testHubNode({address: node.grpcAddress}, tracker));
      }
    }

    try {
      await Promise.all(tasks);
    } catch {
      // errors are not important, the tracker will capture them
    }
  };

  const {lastPoll, nextPoll, pollNow, isPolling} = usePoll(checkHealth);

  watch(() => cohort.cohortNodes, (n, o) => {
    if (deepEqual(n, o)) return; // avoid unnecessary polling
    pollNow(true);
  }, {immediate: true, deep: true});

  const anyPending = computed(() => {
    for (const result of Object.values(resultsByName)) {
      if (result.pending) return true;
    }
    return false;
  });

  return {
    resultsByName,
    anyPending,
    lastPoll,
    nextPoll,
    isPolling,
    pollNow
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useCohortStore, import.meta.hot));
  import.meta.hot.accept(acceptHMRUpdate(useCohortHealthStore, import.meta.hot));
}

