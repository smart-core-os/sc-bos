<template>
  <side-bar>
    <template v-if="canEdit" #actions>
      <v-btn :to="editLink" icon="mdi-pencil" variant="plain" size="small"/>
    </template>
    <lights-config-card v-if="automationType === 'lights'"/>
    <edit-config-card/>
    <v-divider/>
    <logs-card/>
    <v-card-actions class="justify-end px-4 pt-0" v-if="false">
      <v-btn color="primary" variant="flat" :disabled="blockActions" @click="saveConfig">Save</v-btn>
    </v-card-actions>
    <v-snackbar v-model="saveConfirm" timeout="2000" color="success" max-width="250" min-width="200">
      <span class="text-body-large align-baseline"><v-icon start>mdi-content-save-check</v-icon>Config saved</span>
    </v-snackbar>
  </side-bar>
</template>

<script setup>
import {newActionTracker} from '@/api/resource';
import {configureService, ServiceNames as ServiceTypes} from '@/api/ui/services';
import SideBar from '@/components/SideBar.vue';
import {useErrorStore} from '@/components/ui-error/error';
import useAuthSetup from '@/composables/useAuthSetup';
import {useSidebarServiceRouterLink} from '@/dynamic/route.js';
import LightsConfigCard from '@/routes/automations/components/config-cards/LightsConfigCard.vue';
import EditConfigCard from '@/routes/system/components/service-cards/EditConfigCard.vue';
import LogsCard from '@/routes/system/components/service-cards/LogsCard.vue';
import {useSidebarStore} from '@/stores/sidebar';
import {useUserConfig} from '@/stores/userConfig.js';
import {serviceName} from '@/util/gateway';
import {computed, onMounted, onUnmounted, reactive, ref} from 'vue';

const {blockActions} = useAuthSetup();

const sidebar = useSidebarStore();
const userConfig = useUserConfig();

const saveTracker = reactive(/** @type {ActionTracker<Service.AsObject>} */ newActionTracker());
const saveConfirm = ref(false);

const automationType = computed(() => {
  return sidebar.data?.config?.type ?? '';
});

const nodeName = computed(() => {
  return userConfig.node?.name;
});

// UI error handling
const errorStore = useErrorStore();
let unwatchError;
onMounted(() => {
  unwatchError = errorStore.registerTracker(saveTracker);
});
onUnmounted(() => {
  unwatchError();
});


/**
 *
 */
async function saveConfig() {
  const req = {
    name: serviceName(nodeName.value, ServiceTypes.Automations),
    id: sidebar.data?.service?.id,
    configRaw: JSON.stringify(sidebar.data.config, null, 2)
  };

  const service = await configureService(req, saveTracker);
  sidebar.data = {service, config: JSON.parse(service.configRaw ?? {})};
  saveConfirm.value = true;
}

const {hasLink: canEdit, to: editLink} = useSidebarServiceRouterLink();
</script>
