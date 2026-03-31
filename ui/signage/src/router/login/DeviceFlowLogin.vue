<template>
  <div>
    <template v-if="!code">
      <v-card-title>Setting up device authentication...</v-card-title>
    </template>
    <template v-else>
      <v-card-text class="instructions">
        <div class="instructions--title text-h4 mb-6">Log in using your device</div>
        <div class="activation-link--title">Go to:</div>
        <div class="activation-link--a font-weight-bold mb-4">
          <a :href="code.verification_uri" class="text-decoration-none" target="_blank">
            {{ code.verification_uri }}
          </a>
        </div>
        <div class="activation-code--title">Enter your unique activation code:</div>
        <div class="activation-code text-h1 d-flex align-center">
          <span>{{ code.user_code }}</span>
          <v-progress-circular
              v-if="showCodeExpiresPercentage"
              class="ml-4"
              color="warning"
              :size="28"
              :width="3"
              :model-value="codeExpiresInPercentage"/>
        </div>
      </v-card-text>
      <v-card-text class="qr d-flex flex-column align-center">
        <a :href="code.verification_uri_complete" target="_blank">
          <img :src="qrCodeDataUrl" alt="QR Code containing the url to login using your device" class="qr--container">
        </a>
        <div class="qr--text mt-2">Or scan this QR Code</div>
      </v-card-text>
    </template>
  </div>
</template>
<script setup>
import {SECOND, useNow} from '@/components/now';
import {useAccountStore} from '@/stores/account';
import QRCode from 'qrcode';
import {computed, onBeforeUnmount, onMounted, ref, watch} from 'vue';

const props = defineProps({
  scopes: {
    type: Array,
    default: () => undefined
  }
});

const accountStore = useAccountStore();

const context = ref(
    /** @type {import('@/composables/authentication/useDeviceFlow').Context | null} */
    null
);
const cancelContext = () => {
  if (context.value) {
    context.value.cancel();
  }
};
const code = computed(() => /** @type {CodeResponse} */ context.value?.code);

const {now} = useNow(SECOND / 4); // this has a direct impact on how smooth the progress bar looks
// at what time will the current user code expire.
const expiresAt = computed(() => {
  if (!code.value) {
    return false;
  }
  return new Date(code.value.timestamp + code.value.expires_in * 1000);
});

// renew the user code whenever the old one expires - assuming the user hasn't completed the flow yet
let oldBeginAgainHandle = 0;
watch(expiresAt, (expiresAt) => {
  clearTimeout(oldBeginAgainHandle);
  if (expiresAt === false) {
    return;
  }
  const expiresInMillis = expiresAt.getTime() - now.value.getTime();
  oldBeginAgainHandle = setTimeout(async () => {
    cancelContext();
    context.value = await accountStore.beginDeviceFlow(props.scopes);
  }, expiresInMillis);
});

// props that tell the user if their code is about to expire
const expiresInSecs = computed(() => {
  const at = expiresAt.value;
  if (!at) {
    return false;
  }
  return (at.getTime() - now.value.getTime()) / 1000;
});
const showCodeExpiresPercentageThreshold = ref(30); // in secs
const showCodeExpiresPercentage = computed(() => expiresInSecs.value < showCodeExpiresPercentageThreshold.value);
// counts down from 100 to 0 starting with 100$% being showCodeExpiresPercentageThreshold away from expiry.
const codeExpiresInPercentage = computed(() => (expiresInSecs.value / showCodeExpiresPercentageThreshold.value) * 100);

// QR code generation
const qrCodeContent = computed(() => code.value?.verification_uri_complete);
const qrCodeDataUrl = ref(';');
watch(qrCodeContent, (qrCodeContent) => {
  if (qrCodeContent) {
    const opts = {
      errorCorrectionLevel: 'M',
      width: 200
    };
    QRCode.toDataURL(qrCodeContent, opts, (err, url) => {
      if (err) {
        console.error('Error generating QR code', err);
      } else {
        qrCodeDataUrl.value = url;
      }
    });
  }
});

// kick off the flow
onMounted(async () => {
  context.value = await accountStore.beginDeviceFlow(props.scopes);
});
onBeforeUnmount(() => {
  cancelContext();
  clearTimeout(oldBeginAgainHandle);
});
</script>
<style scoped>
.qr--container {
  vertical-align: middle;
}
</style>
