<template>
  <div
      class="pa-12 align-center my-auto"
      @mousemove="resetTimer"
      @click="resetTimer"
      @keydown="resetTimer"
      @touchstart="resetTimer"
      @touchmove="resetTimer">
    <list-selector
        v-if="step === 1"
        title="Select your room"
        @zone-selected="onSingleZone"
        @should-auto-logout="shouldAutoLogout"/>
    <list-selector
        v-else
        title="Select combined room"
        :show-skip="true"
        @zone-selected="onJoinedZone"
        @zone-skipped="skipJoinedZone"
        @should-auto-logout="shouldAutoLogout"/>
    <v-snackbar
        class="d-flex flex-row justify-center mb-4 elevation-0"
        color="transparent"
        :timeout="ALERT_TIME"
        v-model="autoLogoutAlert">
      <span class="ml-9 font-weight-bold">Automatic {{ actionType }} in {{ countdown }} seconds</span>
    </v-snackbar>
  </div>
</template>

<script setup>
import ListSelector from '@/routes/setup/components/ListSelector.vue';
import {useAccountStore} from '@/stores/account';
import {useConfigStore} from '@/stores/config';
import {useUiConfigStore} from '@/stores/ui-config';
import {computed, onUnmounted, ref, watch} from 'vue';
import {useRouter} from 'vue-router';

const router = useRouter();
const uiConfig = useUiConfigStore();
const configStore = useConfigStore();
const {logout} = useAccountStore();

const step = ref(1);

const onSingleZone = async (zone) => {
  await configStore.setZone(zone.id, zone.metadata);
  step.value = 2;
};

const onJoinedZone = async (zone) => {
  await configStore.setJoinedZone(zone.id, zone.metadata);
  await router.push({name: 'home'});
};

const skipJoinedZone = async () => {
  await configStore.setJoinedZone('', null);
  await router.push({name: 'home'});
};
const autoLogoutAlert = ref(false);
const autoLogoutOn = ref(false);
const disableAuthentication = computed(() => uiConfig.auth.disabled);
const actionType = computed(() => {
  if (disableAuthentication.value) return 'redirect';
  else return 'logout';
});
const countdown = ref(60); // 60 seconds = 1 minute


let logoutTimer;
let countdownTimer;
const TIMEOUT_DURATION = 5 * 60 * 1000; // 5 minutes in milliseconds
const ALERT_TIME = 1 * 60 * 1000; // 1 minute in milliseconds

// Reset the timer when the user interacts with the page
const resetTimer = () => {
  clearTimeout(logoutTimer);
  clearTimeout(countdownTimer);
  countdown.value = 60; // Reset the countdown
  autoLogoutAlert.value = false; // Hide the snackbar
  startTimer();
};

// Start the timer
const startTimer = () => {
  countdownTimer = setTimeout(() => {
    autoLogoutAlert.value = true; // Show the snackbar
    startCountdown();
  }, TIMEOUT_DURATION - ALERT_TIME);

  // Logout the user after the timeout duration
  logoutTimer = setTimeout(() => {
    if (!disableAuthentication.value) {
      logout(); // if auth is enabled, logout the user
    } else if (disableAuthentication.value && (configStore.zoneId || configStore.zoneName)) {
      router.push({name: 'home'}).catch(() => {}); // if auth is disabled, redirect to home
    } else console.log('No zone selected');
  }, TIMEOUT_DURATION);
};

// Start the countdown timer for the snackbar
const startCountdown = () => {
  const interval = setInterval(() => {
    countdown.value--;
    if (countdown.value <= 0) {
      clearInterval(interval);
    }
  }, 1000);
};

// Emit an event to the ListSelector component to auto logout the user
const shouldAutoLogout = (value) => {
  autoLogoutOn.value = value;
};

// Watch for changes to the autoLogoutOn value
// If the value is true, start the timer
// If the value is false, clear the timers
watch(autoLogoutOn, value => {
  const noAuthWithSelection = disableAuthentication.value && (configStore.zoneId || configStore.zoneName);
  const authEnabled = !disableAuthentication.value;

  clearTimeout(logoutTimer);
  clearTimeout(countdownTimer);

  if (value) {
    if (noAuthWithSelection || authEnabled) {
      startTimer();
    } else return;
  }
});

onUnmounted(() => {
  clearTimeout(logoutTimer);
  clearTimeout(countdownTimer);
  autoLogoutOn.value = false;
});
</script>
