<template>
  <div class="d-flex flex-column align-center my-auto">
    <v-card class="ma-auto pa-4" min-width="450px" max-width="500px">
      <LocalLogin v-if="displayLoginForm"/>
      <device-flow-login v-else-if="displayDeviceLogin" :scopes="scopes"/>
      <login-choice v-else @choose="chooseProvider"/>

      <v-card-actions v-if="choiceExists && !displayChoice" class="d-flex justify-center mt-8">
        <a @click="showChoice" class="text-center">Use a different sign in method</a>
      </v-card-actions>
    </v-card>
    <v-btn
        v-if="uiConfig.auth.disabled || configStore.isReconfiguring"
        class="mt-4"
        color="neutral"
        elevation="0"
        @click="configStore.abortReconfigure()">
      <v-icon class="ml-n2">mdi-chevron-left</v-icon>
      Return to home
    </v-btn>

    <v-snackbar v-model="snackbar.visible">
      {{ snackbar.message }}

      <template #actions="{ attrs }">
        <v-btn color="pink" variant="text" v-bind="attrs" @click="snackbar.visible = false">
          Close
        </v-btn>
      </template>
    </v-snackbar>
  </div>
</template>
<script setup>
import DeviceFlowLogin from '@/router/login/DeviceFlowLogin.vue';
import LocalLogin from '@/router/login/LocalLogin.vue';
import LoginChoice from '@/router/login/LoginChoice.vue';
import {useAccountStore} from '@/stores/account.js';
import {useConfigStore} from '@/stores/config.js';
import {useUiConfigStore} from '@/stores/ui-config';
import {storeToRefs} from 'pinia';
import {computed, ref} from 'vue';

const uiConfig = useUiConfigStore();
const configStore = useConfigStore();
const accountStore = useAccountStore();
const {snackbar} = storeToRefs(accountStore);
const scopes = ref(['offline_access']);

const manualDisplayLoginForm = ref(false);
const displayLoginForm = computed(() => {
  return manualDisplayLoginForm.value || accountStore.isOnlyProvider('localAuth');
});

const manualDisplayDeviceLogin = ref(false);
const displayDeviceLogin = computed(() => {
  return manualDisplayDeviceLogin.value || accountStore.isOnlyProvider('deviceFlow');
});

// don't need a display ref for keycloak because it uses redirect, aka there is no page for it

const choiceExists = computed(() => {
  return accountStore.availableAuthProviders.length > 1;
});
const displayChoice = computed(() => {
  return !displayLoginForm.value && !displayDeviceLogin.value;
});

const showChoice = () => {
  manualDisplayLoginForm.value = false;
  manualDisplayDeviceLogin.value = false;
};
const chooseProvider = (p) => {
  switch (p) {
    case 'localAuth':
      manualDisplayLoginForm.value = true;
      break;
    case 'deviceFlow':
      manualDisplayDeviceLogin.value = true;
      break;
    case 'keyCloakAuth': {
      // no page to display, redirect instead
      const opts = /** @type {import('keycloak-js').KeycloakLoginOptions } */ {};
      if (scopes.value.length > 0) {
        opts.scope = scopes.value.join(' ');
      }
      accountStore.loginWithKeyCloak(opts);
      break;
    }
  }
};

</script>
