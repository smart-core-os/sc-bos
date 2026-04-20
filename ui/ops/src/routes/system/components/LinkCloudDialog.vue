<template>
  <v-dialog v-model="dialog" max-width="480">
    <v-card>
      <v-card-title>{{ isLinked ? 'Manage Cloud Link' : 'Link to Cloud' }}</v-card-title>

      <!-- Connected state: show current link info + Unlink button -->
      <template v-if="isLinked">
        <v-card-text>
          <v-list density="compact">
            <v-list-item>
              <v-list-item-subtitle>Client ID</v-list-item-subtitle>
              <v-list-item-title class="text-body-2 font-weight-medium">
                {{ cloudConnection.clientId || '—' }}
              </v-list-item-title>
            </v-list-item>
            <v-list-item>
              <v-list-item-subtitle>Server</v-list-item-subtitle>
              <v-list-item-title class="text-body-2 font-weight-medium">
                {{ cloudConnection.bosapiRoot || '—' }}
              </v-list-item-title>
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

      <!-- Enrollment form with method tabs -->
      <template v-else>
        <v-tabs v-model="tab" density="compact">
          <v-tab value="code">Enrollment Code</v-tab>
          <v-tab value="manual">Manual</v-tab>
        </v-tabs>
        <v-divider/>

        <v-form @submit.prevent="doRegister">
          <v-card-text>
            <v-tabs-window v-model="tab">
              <!-- Enrollment code tab -->
              <v-tabs-window-item value="code">
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
              </v-tabs-window-item>

              <!-- Manual credentials tab -->
              <v-tabs-window-item value="manual">
                <v-text-field
                    v-model="clientId"
                    label="Client ID"
                    variant="filled"
                    class="mb-3 mt-1"
                    autocomplete="off"/>
                <v-text-field
                    v-model="clientSecret"
                    label="Client secret"
                    type="password"
                    autocomplete="off"
                    variant="filled"
                    class="mb-3"/>
                <v-text-field
                    v-model="bosapiRoot"
                    label="BOS API root"
                    hint="e.g. https://bosapi.example.com"
                    persistent-hint
                    variant="filled"/>
              </v-tabs-window-item>
            </v-tabs-window>
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
  CloudErrMessage, getCloudConnectionDefaults, registerCloudConnection, testCloudConnection, unlinkCloudConnection
} from '@/api/ui/cloud-connection.js';
import {newActionTracker} from '@/api/resource.js';
import {DAY, HOUR, MINUTE, SECOND, useNow} from '@/components/now.js';
import {CloudConnection} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/ops/cloud/v1alpha/cloud_connection_pb';
import {computed, reactive, ref, watch} from 'vue';
import {formatTimeAgoRounded} from '@/util/date.js';

const props = defineProps({
  nodeName: {type: String, required: true},
  cloudConnection: {type: Object, default: () => null}
});
const emit = defineEmits(['linked', 'unlinked']);

const dialog = defineModel({type: Boolean, default: false});
const unlinkDialog = ref(false);

const tab = ref('code');

// Enrollment code tab
const enrollmentCode = ref('');
const registerUrlOverride = ref('');

// Manual tab
const clientId = ref('');
const clientSecret = ref('');
const bosapiRoot = ref('');

const registerTracker = reactive(newActionTracker());
const unlinkTracker = reactive(newActionTracker());
const testTracker = reactive(newActionTracker());
const testSuccess = ref(false);

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
  return formatTimeAgoRounded(lastCheckInDate.value, now.value, MINUTE, MINUTE, HOUR, DAY);
});

// null = not yet loaded; '' = loaded but empty; string = loaded with value
const defaultRegisterUrl = ref(null);
const advancedExpanded = ref(false);

// reset dialog on open
watch(dialog, async (open) => {
  if (!open) {
    testSuccess.value = false;
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
  if (tab.value === 'code') {
    if (!enrollmentCode.value) return false;
    if (!defaultRegisterUrl.value && !registerUrlOverride.value) return false;
    return true;
  }
  return !!(clientId.value && clientSecret.value && bosapiRoot.value);
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
  const request = {name: props.nodeName};
  if (tab.value === 'code') {
    request.enrollmentCode = {
      code: enrollmentCode.value.trim(),
      registerUrl: registerUrlOverride.value.trim()
    };
  } else {
    request.manual = {
      clientId: clientId.value.trim(),
      clientSecret: clientSecret.value,
      bosapiRoot: bosapiRoot.value.trim()
    };
  }
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
  clientId.value = '';
  clientSecret.value = '';
  bosapiRoot.value = '';
  advancedExpanded.value = false;
}
</script>
