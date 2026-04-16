<template>
  <div>
    <template v-if="canChooseKeyCloak">
      <v-card-text class="text-center mx-auto" style="max-width: 320px;">
        Sign in with your organisation account (SSO) — includes Azure Active Directory, Keycloak,
        and other identity providers.
      </v-card-text>
      <v-card-actions class="justify-center mt-4">
        <v-btn
            @click="emit('choose', 'keyCloakAuth')"
            color="primary"
            variant="elevated"
            block
            size="large"
            class="text-body-1 font-weight-bold">
          Sign in with SSO
        </v-btn>
      </v-card-actions>
      <template v-if="canChooseDevice">
        <v-card-actions class="justify-center">
          <v-btn
              @click="emit('choose', 'deviceFlow')"
              variant="text"
              class="text-body-1">
            Or sign in using your device
          </v-btn>
        </v-card-actions>
      </template>
    </template>

    <template v-else-if="canChooseDevice">
      <v-card-text class="text-center mx-auto" style="max-width: 320px;">
        Sign in to the Smart Core Operator App to unlock all features.
      </v-card-text>
      <v-card-actions class="justify-center mt-4">
        <v-btn
            @click="emit('choose', 'deviceFlow')"
            color="primary"
            block
            size="large"
            class="text-body-1 font-weight-bold">
          Sign in using your device
        </v-btn>
      </v-card-actions>
    </template>

    <template v-if="canChooseLocal">
      <v-card-text class="text-body-2 text-center mt-10 mx-auto" style="max-width: 350px;">
        If you are an administrator or need to setup your building, please sign in with a local account.
      </v-card-text>
      <v-card-actions class="d-flex flex-column align-center justify-center mt-n2">
        <v-btn
            block
            class="text-body-2 ma-0"
            variant="text"
            @click="emit('choose', 'localAuth')">
          Sign in with local Account
        </v-btn>
      </v-card-actions>
    </template>
  </div>
</template>

<script setup>
import {useAccountStore} from '@/stores/account';
import {computed} from 'vue';

const emit = defineEmits(['choose']);
const accountStore = useAccountStore();

const canChooseKeyCloak = computed(() => accountStore.hasProvider('keyCloakAuth'));
const canChooseDevice = computed(() => accountStore.hasProvider('deviceFlow'));
const canChooseLocal = computed(() => accountStore.hasProvider('localAuth'));
</script>
