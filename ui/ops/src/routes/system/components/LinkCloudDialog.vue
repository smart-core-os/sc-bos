<template>
  <v-dialog v-model="dialog" max-width="480">
    <v-card>
      <v-card-title>{{ isLinked ? 'Manage Cloud Link' : 'Link to Cloud' }}</v-card-title>

      <!-- Connected state: show current link info + Unlink button -->
      <template v-if="isLinked">
        <v-card-text>
          <v-list density="compact">
            <v-list-item>
              <v-list-item-subtitle>Node ID</v-list-item-subtitle>
              <v-list-item-title class="text-body-2 font-weight-medium">
                {{ cloudConnection.nodeId || '—' }}
              </v-list-item-title>
            </v-list-item>
            <v-list-item>
              <v-list-item-subtitle>API endpoint</v-list-item-subtitle>
              <v-list-item-title class="text-body-2 font-weight-medium">
                {{ cloudConnection.apiEndpoint || '—' }}
              </v-list-item-title>
            </v-list-item>
            <v-divider/>
            <v-list-item>
              <v-list-item-subtitle>Certificate issued</v-list-item-subtitle>
              <v-list-item-title class="text-body-2 font-weight-medium">
                {{ formatDate(cloudConnection.certificateIssuedTime) }}
              </v-list-item-title>
            </v-list-item>
            <v-list-item>
              <v-list-item-subtitle>Certificate expires</v-list-item-subtitle>
              <v-list-item-title class="text-body-2 font-weight-medium">
                {{ formatDate(cloudConnection.certificateExpiryTime) }}
              </v-list-item-title>
            </v-list-item>
            <v-list-item>
              <v-list-item-subtitle>Auto-renews</v-list-item-subtitle>
              <v-list-item-title class="text-body-2 font-weight-medium">
                {{ formatDate(cloudConnection.nextRenewalTime) }}
              </v-list-item-title>
              <template #append>
                <v-btn size="small" variant="text" :loading="renewTracker.loading" @click="doRenew">
                  Renew now
                </v-btn>
              </template>
            </v-list-item>
            <v-divider/>
            <v-list-item>
              <v-list-item-subtitle>Status</v-list-item-subtitle>
              <v-list-item-title>
                <v-chip :color="connectionState.color" size="small" variant="tonal" label>
                  {{ connectionState.label }}
                </v-chip>
              </v-list-item-title>
            </v-list-item>
            <v-list-item>
              <v-list-item-subtitle>Last checked in</v-list-item-subtitle>
              <v-list-item-title class="text-body-2 font-weight-medium">
                {{ lastCheckInText }}
              </v-list-item-title>
            </v-list-item>
            <v-list-item v-if="cloudConnection.lastError">
              <v-list-item-subtitle>Last error</v-list-item-subtitle>
              <v-list-item-title class="text-body-2 text-error text-wrap">
                {{ lastErrorMessage }}
              </v-list-item-title>
            </v-list-item>
          </v-list>
        </v-card-text>
        <v-alert v-if="testSuccess" type="success" class="mx-4 mb-2" density="compact" closable>
          Connection successful
        </v-alert>
        <v-alert v-if="testTracker.error" type="error" class="mx-4 mb-2" density="compact" closable>
          {{ testErrorMessage }}
        </v-alert>
        <v-alert v-if="renewSuccess" type="success" class="mx-4 mb-2" density="compact" closable>
          Certificate renewed
        </v-alert>
        <v-alert v-if="renewTracker.error" type="error" class="mx-4 mb-2" density="compact" closable>
          {{ renewErrorMessage }}
        </v-alert>
        <v-card-actions>
          <v-btn @click="dialog = false">Close</v-btn>
          <v-spacer/>
          <v-btn :loading="testTracker.loading" @click="doTest">Test Connection</v-btn>
          <v-btn color="error" @click="unlinkDialog = true">Unlink</v-btn>
        </v-card-actions>

        <v-dialog v-model="unlinkDialog" max-width="400">
          <v-card>
            <v-card-title>Confirm Unlink</v-card-title>
            <v-card-text>
              This cannot be undone. Saved cloud credentials will be deleted.
            </v-card-text>
            <v-alert v-if="unlinkTracker.error" type="error" class="mx-4 mb-2" density="compact" closable>
              {{ unlinkErrorMessage }}
            </v-alert>
            <v-card-actions>
              <v-spacer/>
              <v-btn @click="unlinkDialog = false">Cancel</v-btn>
              <v-btn color="error" :loading="unlinkTracker.loading" @click="doUnlink">Unlink</v-btn>
            </v-card-actions>
          </v-card>
        </v-dialog>
      </template>

      <!-- Enrollment form -->
      <template v-else>
        <v-divider/>

        <v-form @submit.prevent="doRegister">
          <v-card-text>
            <v-text-field
                v-model="enrollmentCode"
                label="Enrollment code"
                hint="6-character code from Smart Core Connect"
                persistent-hint
                variant="filled"
                class="mb-3 mt-1"
                autofocus/>
            <v-checkbox
                v-model="advancedExpanded"
                label="Show advanced options"
                hide-details/>
            <v-expand-transition>
              <div v-show="advancedExpanded">
                <v-text-field
                    v-model="registerUrlOverride"
                    label="Register URL"
                    :hint="defaultRegisterUrl ? `Default: ${defaultRegisterUrl}` : 'Required - no default is configured'"
                    persistent-hint
                    variant="filled"/>
              </div>
            </v-expand-transition>
          </v-card-text>

          <v-alert v-if="registerTracker.error" type="error" class="mx-4 mb-2" density="compact">
            {{ registerErrorMessage }}
          </v-alert>

          <v-card-actions>
            <v-btn @click="dialog = false">Cancel</v-btn>
            <v-spacer/>
            <v-btn
                color="primary"
                type="submit"
                :loading="registerTracker.loading"
                :disabled="!canSubmit">
              Link
            </v-btn>
          </v-card-actions>
        </v-form>
      </template>
    </v-card>
  </v-dialog>
</template>

<script setup>
import {
  CloudErrMessage, getCloudConnectionDefaults, registerCloudConnection, renewCloudConnection, testCloudConnection,
  unlinkCloudConnection
} from '@/api/ui/cloud-connection.js';
import {newActionTracker} from '@/api/resource.js';
import {useNow} from '@/components/now.js';
import {CloudConnection} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/ops/cloud/v1alpha/cloud_connection_pb';
import {computed, reactive, ref, watch} from 'vue';
import {formatTimeAgoRounded, MINUTE, SECOND} from '@/util/date.js';

const props = defineProps({
  nodeName: {type: String, required: true},
  cloudConnection: {type: Object, default: () => null}
});
const emit = defineEmits(['linked', 'unlinked']);

const dialog = defineModel({type: Boolean, default: false});
const unlinkDialog = ref(false);

// Enrollment code
const enrollmentCode = ref('');
const registerUrlOverride = ref('');

const registerTracker = reactive(newActionTracker());
const unlinkTracker = reactive(newActionTracker());
const testTracker = reactive(newActionTracker());
const testSuccess = ref(false);
const renewTracker = reactive(newActionTracker());
const renewSuccess = ref(false);

const {now} = useNow(SECOND);

const connectionState = computed(() => {
  switch (props.cloudConnection?.state) {
    case CloudConnection.State.CONNECTING:   return {label: 'Connecting…', color: undefined};
    case CloudConnection.State.CONNECTED:    return {label: 'Connected',   color: 'success'};
    case CloudConnection.State.FAILED:       return {label: 'Disconnected', color: 'warning'};
    default:                                 return {label: 'Unknown',      color: undefined};
  }
});

const lastCheckInDate = computed(() => {
  const ts = props.cloudConnection?.lastCheckInTime;
  if (!ts?.seconds) return null;
  return new Date(ts.seconds * 1000);
});

const lastCheckInText = computed(() => {
  if (!lastCheckInDate.value) return 'never';
  if (now.value - lastCheckInDate.value < MINUTE) {
    return 'less than a minute ago';
  }
  return formatTimeAgoRounded(lastCheckInDate.value, now.value, MINUTE);
});

// null = not yet loaded; '' = loaded but empty; string = loaded with value
const defaultRegisterUrl = ref(null);
const advancedExpanded = ref(false);

// reset dialog on open
watch(dialog, async (open) => {
  if (!open) {
    testSuccess.value = false;
    renewSuccess.value = false;
    return;
  }

  // clear previous inputs
  resetFields();

  if (defaultRegisterUrl.value !== null) return;

  // find out what the default register URL is
  const resp = await getCloudConnectionDefaults({name: props.nodeName}).catch(() => null);
  defaultRegisterUrl.value = resp?.defaults?.registerUrl ?? '';
  if (!defaultRegisterUrl.value) advancedExpanded.value = true;
});

const canSubmit = computed(() => {
  if (!enrollmentCode.value) return false;
  if (!defaultRegisterUrl.value && !registerUrlOverride.value) return false;
  return true;
});

const registerErrorMessage = computed(() => {
  const err = registerTracker.error?.error;
  if (!err) return '';
  return CloudErrMessage[err.message] || err.message || 'Registration failed';
});

const testErrorMessage = computed(() => {
  const err = testTracker.error?.error;
  if (!err) return '';
  return CloudErrMessage[err.message] || err.message || 'Connection test failed';
});

const renewErrorMessage = computed(() => {
  const err = renewTracker.error?.error;
  if (!err) return '';
  return CloudErrMessage[err.message] || err.message || 'Renewal failed';
});

const unlinkErrorMessage = computed(() => {
  const err = unlinkTracker.error?.error;
  if (!err) return '';
  return err.message || 'Unlink failed';
})

const lastErrorMessage = computed(() => {
  return CloudErrMessage[props.cloudConnection?.lastError] || props.cloudConnection?.lastError || '';
})

const isLinked = computed(() =>
  props.cloudConnection?.state != null &&
  props.cloudConnection.state !== CloudConnection.State.UNCONFIGURED &&
  props.cloudConnection.state !== CloudConnection.State.STATE_UNSPECIFIED
);

async function doRegister() {
  const request = {
    name: props.nodeName,
    enrollmentCode: {
      code: enrollmentCode.value.trim(),
      registerUrl: registerUrlOverride.value.trim()
    }
  };
  await registerCloudConnection(request, registerTracker).catch(() => {});
  if (!registerTracker.error) {
    resetFields();
    emit('linked');
  }
}

async function doTest() {
  testSuccess.value = false;
  await testCloudConnection({name: props.nodeName}, testTracker).catch(() => {});
  if (!testTracker.error) testSuccess.value = true;
}

async function doRenew() {
  renewSuccess.value = false;
  await renewCloudConnection({name: props.nodeName}, renewTracker).catch(() => {});
  if (!renewTracker.error) renewSuccess.value = true;
}

/** Format a protobuf Timestamp ({seconds, nanos}) as a short date, or '—' if absent. */
function formatDate(ts) {
  if (!ts?.seconds) return '—';
  return new Date(ts.seconds * 1000).toLocaleDateString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric'
  });
}

async function doUnlink() {
  await unlinkCloudConnection({name: props.nodeName}, unlinkTracker).catch(() => {});
  if (!unlinkTracker.error) {
    unlinkDialog.value = false;
    dialog.value = false;
    emit('unlinked');
  }
}

/** restore all fields to what they should be when the dialog opens */
function resetFields() {
  enrollmentCode.value = '';
  registerUrlOverride.value = '';
  advancedExpanded.value = false;
}
</script>
