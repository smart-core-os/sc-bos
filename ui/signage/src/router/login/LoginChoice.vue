<template>
  <div>
    <template v-if="canChooseKeyCloak">
      <v-card-text class="text-center mx-auto" style="max-width: 320px;">
        Please sign in to setup this panel.
      </v-card-text>
      <v-card-actions class="justify-center mt-4">
        <v-btn
            @click="emit('choose', 'keyCloakAuth')"
            color="primary"
            variant="flat"
            block
            size="large"
            class="text-body-1 font-weight-bold">
          Sign in
        </v-btn>
      </v-card-actions>
      <template v-if="canChooseDevice">
        <v-card-actions class="justify-center">
          <a
              @click="emit('choose', 'deviceFlow')"
              class="text-body-1">
            Or sign in using your device
          </a>
        </v-card-actions>
      </template>
    </template>

    <template v-else-if="canChooseDevice">
      <v-card-text class="text-center mx-auto" style="max-width: 320px;">
        Please sign in to setup this panel.
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
      <v-card-actions class="d-flex flex-column align-center justify-center">
        <v-btn
            block
            size="large"
            elevation="0"
            class="text-body-2 ma-0"
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
