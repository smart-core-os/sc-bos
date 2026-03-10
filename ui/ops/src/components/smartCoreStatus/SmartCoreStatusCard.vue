<template>
  <v-btn
      class="py-1 px-3 mr-0"
      variant="text">
    <span class="mr-1">Smart Core OS:</span>
    <span class="font-weight-bold" :class="`text-${overallStatus.color}`">
      {{ overallStatus.text }}
    </span>

    <v-menu
        activator="parent"
        location="bottom left"
        :close-on-content-click="false"
        content-class="elevation-0"
        max-height="600px"
        max-width="550px"
        min-width="400px">
      <v-card class="elevation-0 mt-4 py-1" min-width="400px">
        <v-card-title class="text-subtitle-1 d-flex align-center">
          Smart Core OS Status
          <span
              class="ml-2 font-weight-light"
              style="font-size: 10px; cursor: pointer"
              @click="showLastCheckTime = !showLastCheckTime">
            {{ checkStatusStr }}
          </span>
          <v-spacer/>
          <v-tooltip location="left">
            <template #activator="{ props }">
              <v-btn
                  v-bind="props"
                  :class="['mb-0', {'rotate-icon': rotateCheckIcon}]"
                  icon="mdi-reload"
                  variant="flat"
                  size="x-small"
                  style="padding-left: 1px; font-size: 12px"
                  @click="checkHealthNow"/>
            </template>
            <div class="d-flex flex-column">
              <span>Check Now</span>
            </div>
          </v-tooltip>
        </v-card-title>
        <v-card-text class="d-flex align-center justify-center">
          <v-defaults-provider :defaults="{
            VChip: {variant: 'flat', size: 'small'},
            VDivider: {class: 'mx-2', style: 'width: 10px; max-width: 10px;'},
            VProgressCircular: {size: 22, indeterminate: true}
          }">
            <v-tooltip
                v-if="keycloakHealth.isConfigured"
                location="bottom"
                :content-props="{'class': keycloakHealth.error ? 'bg-error' : ''}">
              <template #activator="{ props: tp }">
                <v-chip v-bind="tp" :color="keycloakHealth.chipColor">UI</v-chip>
              </template>
              <span>{{ keycloakHealth.tooltipText }}</span>
            </v-tooltip>
            <v-chip v-else color="neutral-lighten-1">UI</v-chip>
            <template v-for="link in chain" :key="link.node.name">
              <v-divider/>
              <v-progress-circular v-if="link.health.pending"/>
              <status-alert
                  v-else
                  v-bind="link.statusAttrs"/>
              <v-divider/>
              <v-chip v-bind="link.chipAttrs"/>
            </template>
          </v-defaults-provider>
        </v-card-text>
      </v-card>
    </v-menu>
  </v-btn>
</template>

<script setup>
import {DAY, HOUR, MINUTE, SECOND, useNow} from '@/components/now';
import StatusAlert from '@/components/StatusAlert.vue';
import useAuthSetup from '@/composables/useAuthSetup';
import {EnrollmentStatus, NodeRole, useCohortHealthStore, useCohortStore} from '@/stores/cohort.js';
import {useKeycloakHealthStore} from '@/stores/keycloakHealth.js';
import {formatTimeAgo} from '@/util/date';
import {formatErrorMessage, isNetworkError} from '@/util/error';
import {computed, ref} from 'vue';

const {hasNoAccess, isLoggedIn} = useAuthSetup();

const cohort = useCohortStore();
const cohortHealth = useCohortHealthStore();
const keycloakHealth = useKeycloakHealthStore();

// The chain is the list of things we show to the user, that they might be interested in the status of.
// Each link in the chain typically talks to the items later in the chain: the ui talks to the server, which talks to the hub, etc.
// Each link has a health and some attrs for the comm and chip elements we use to display this information.
const chain = computed(() => {
  const chain = [];
  const getHealth = (node) => cohortHealth.resultsByName[node.name] ?? {pending: true};
  const addLink = (link) => {
    link.statusAttrs = statusAttrs(link);
    link.chipAttrs = chipAttrs(link);
    chain.push(link);
  };

  const server = cohort.serverNode;
  addLink({
    node: server,
    health: getHealth(server)
  });

  if (cohort.enrollmentStatus === EnrollmentStatus.ENROLLED) {
    const hub = cohort.hubNode;
    addLink({
      node: hub,
      health: getHealth(hub)
    });
  }

  const nodeHealths = cohort.hubNodes.map(node => ({node, health: getHealth(node)}));
  if (nodeHealths.length > 0) {
    addLink({
      node: {name: 'Nodes'},
      health: {
        pending: nodeHealths.some(i => i.health.pending),
        error: nodeHealths.reduce((acc, i) => acc || i.health.error, undefined)
      },
      children: nodeHealths
    });
  }

  return chain;
});
const statusAttrs = (link) => {
  if (link.health.error) {
    const attrs = {
      color: 'error',
      icon: 'mdi-close',
      resource: {error: formatError(link.health.error)}
    };
    if (link.children?.length > 1) {
      const allBad = link.children.every(l => l.health.error);
      if (!allBad) {
        attrs.color = 'warning';
        attrs.icon = 'mdi-alert';
      }
      attrs.resource.errors = link.children
          .filter(c => c.health.error)
          .map(c => ({
            color: 'error',
            icon: 'mdi-close',
            name: c.node.name,
            resource: {error: formatError(c.health.error)}
          }));
      attrs.single = false;
    }
    return attrs;
  }

  let okMsg = 'Healthy';
  if (link.node.isServer) {
    okMsg = `Successfully connected to the server at ${link.node.grpcWebAddress}`;
  } else if (link.node.role === NodeRole.HUB) {
    okMsg = `Node is successfully connected to the Hub at ${link.node.grpcAddress}`;
  } else if (link.node.role === NodeRole.GATEWAY) {
    okMsg = `Node is successfully connected to the Gateway at ${link.node.grpcAddress}`;
  } else if (link.children?.length > 0) {
    okMsg = `All ${link.children.length} nodes are communicating with the Hub`;
  }
  return {
    color: 'success',
    icon: 'mdi-check',
    resource: {error: {code: 0, message: okMsg}}
  };
};
const chipAttrs = (link) => {
  const attrs = {};

  // figure out what the node is
  attrs.text = 'Server';
  attrs.color = 'neutral-lighten-1';
  switch (link.node.role) {
    case NodeRole.HUB:
      attrs.text = 'Hub';
      attrs.color = 'primary';
      break;
    case NodeRole.GATEWAY:
      attrs.text = 'Gateway';
      attrs.color = 'accent';
      break;
  }

  if (link.children?.length > 0) {
    attrs.text = 'Nodes';
    attrs.color = 'neutral-lighten-2';
    attrs.to = checkNavPermission('/system/components');
    if (link.health.error) {
      attrs.color = 'error';
    }
  } else {
    if (link.health.error || link.health.pending) {
      // This bit of code makes it possible to click on the server chip when there's an error.
      // This is super useful for those self-signed cert issues we need to approve in the browser.
      if (link.node.isServer && link.node.grpcWebAddress && link.node.grpcWebAddress !== location.host) {
        attrs.href = `https://${link.node.grpcWebAddress}`;
        attrs.target = '_blank';
      } else {
        attrs.disabled = true;
      }
    }
  }
  return attrs;
};

// code relating to the button on the toolbar and what color it should be.
const issueCount = computed(() => {
  const chainIssues = chain.value.reduce((acc, l) => {
    if (l.children) {
      return acc + l.children.reduce((acc, c) => acc + (c.health.error ? 1 : 0), 0);
    } else {
      return acc + (l.health.error ? 1 : 0);
    }
  }, 0);
  const keycloakIssues = (keycloakHealth.isConfigured && keycloakHealth.error) ? 1 : 0;
  return chainIssues + keycloakIssues;
});
const overallStatus = computed(() => {
  const res = {
    color: 'success-lighten-2',
    text: 'Healthy'
  };

  if (isNetworkError(cohort.serverNode.error)) {
    res.color = 'error';
    res.text = 'Offline';
  } else if (issueCount.value === 1) {
    res.color = 'warning';
    res.text = '1 issue';
  } else if (issueCount.value > 1) {
    res.color = 'warning';
    res.text = `${issueCount.value} issues`;
  } else if (cohort.loading || cohortHealth.isPolling || (keycloakHealth.isConfigured && keycloakHealth.pending)) {
    res.color = 'info';
    res.text = 'Checking';
  }

  return res;
});

// code related to the refresh button and the "last checked 20s ago" bit
const lastCheckTime = computed(() => cohortHealth.lastPoll);
const nextCheckTime = computed(() => cohortHealth.nextPoll);
const {now: currentTime} = useNow(SECOND / 4);
const showLastCheckTime = ref(true); // choose last check, or next check time
const checkStatusStr = computed(() => {
  if (cohortHealth.isPolling) return 'Checking now...';
  if (showLastCheckTime.value) {
    return `Last checked ${formatTimeAgo(lastCheckTime.value, currentTime.value, MINUTE, HOUR, DAY)}`;
  } else {
    return `Next check ${formatTimeAgo(nextCheckTime.value, currentTime.value, MINUTE, HOUR, DAY)}`;
  }
});
const rotateCheckIcon = ref(false);
const checkHealthNow = () => {
  rotateCheckIcon.value = true;
  cohort.pollNow();
  cohortHealth.pollNow();
  keycloakHealth.pollNow();
  setTimeout(() => {
    rotateCheckIcon.value = false;
  }, 1000);
};

// Navigate to the nodes page if the user has access
const checkNavPermission = (to) => (to && isLoggedIn && !hasNoAccess(to)) ? to : null;
const formatError = (error) => {
  if (!error) return null;
  if (isNetworkError(error)) {
    // these only match against ui-server errors, so we can be quite specific here
    return {
      code: error.code,
      message: 'Unable to communicate with the server'
    };
  }
  return {
    code: error.code,
    message: formatErrorMessage(error.message)
  };
};
</script>

<style lang="scss" scoped>
.popup {
  height: 100%;
  width: 100%;
  overflow: auto;

  &__status {
    top: 10px;
    min-height: 100%;
    max-height: 600px;
  }
}

.rotate-icon {
  animation: rotation 1s infinite linear;
}

@keyframes rotation {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
