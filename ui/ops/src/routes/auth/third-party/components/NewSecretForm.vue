<template>
  <v-dialog v-model="dialog" max-width="512" persistent>
    <v-card class="pa-5">
      <v-card-title class="px-0 pt-0 pb-3 text-h4 font-weight-bold">
        {{ accountName !== '' ? accountName + ':' : '' }} New Token
      </v-card-title>
      <v-divider/>
      <!-- Form to create secret -->
      <v-form
          v-if="creatingSecret"
          @submit.prevent="saveSecret"
          v-model="formValid"
          class="pa-0"
          ref="form">
        <v-card-text class="px-0">
          <v-text-field
              label="Note"
              v-model="newSecret.note"
              required
              variant="filled"
              hide-details="auto"
              hint="Easily recognisable (e.g. 'Read only', 'AV System')"
              :rules="noteRules"/>
          <div class="d-flex justify-start align-baseline mt-5">
            <v-select
                v-model="newSecret.expiresIn"
                :items="Object.values(suggestedExpiresIn)"
                hide-details
                variant="filled"
                class="expires-in"
                label="Expires"/>
            <v-menu
                v-if="newSecret.expiresIn === suggestedExpiresIn.custom"
                v-model="customExpiryMenuVisible"
                :close-on-content-click="false">
              <template #activator="{props: _props}">
                <v-text-field
                    v-model="expiresAtString"
                    placeholder="yyyy-mm-dd"
                    readonly
                    v-bind="_props"
                    class="expires-at"
                    hide-details="auto"
                    :rules="expiresAtRules"/>
              </template>
              <v-date-picker v-model="newSecret.expiresAt" @input="customExpiryMenuVisible = false" no-title/>
            </v-menu>
            <span
                v-else-if="newSecret.expiresIn === suggestedExpiresIn.noExpiry"
                class="expires-in-message never">This token will never expire</span>
            <span v-else class="expires-in-message">This token will expire <relative-date
                :date="computedExpiresAt"
                no-relative/></span>
          </div>
        </v-card-text>
        <v-card-actions class="justify-end">
          <v-btn type="cancel" variant="text" @click.prevent="cancelAddSecret">Cancel</v-btn>
          <v-btn color="primary" type="submit" variant="flat" :disabled="!formValid">Create Secret</v-btn>
        </v-card-actions>
        <v-progress-linear color="primary" indeterminate :active="createSecretTracker.loading"/>
      </v-form>
      <!-- Display secret details -->
      <v-list v-else class="pb-0" lines="two">
        <v-list-item
            class="banner info-banner my-4"
            variant="tonal"
            v-if="createSecretTracker.response"
            prepend-icon="mdi-information">
          Make sure to copy your secret token now. You won't be able to see it again.
        </v-list-item>
        <v-list-item class="banner error-banner" v-if="createSecretTracker.error" prepend-icon="mdi-alert-circle">
          {{ createSecretTracker.error.name }}: {{ createSecretTracker.error.message }}
        </v-list-item>
        <v-list-item class="banner secret-banner" v-if="createSecretTracker.response" prepend-icon="mdi-key">
          {{ createdSecret.secret }}
          <template #append>
            <v-list-item-action class="mr-n2">
              <v-btn icon="mdi-content-copy" variant="text" @click="copySecret">
                <v-icon size="24"/>
              </v-btn>
            </v-list-item-action>
          </template>
        </v-list-item>
        <v-card-actions class="justify-end pt-4 pb-0 pr-0">
          <v-btn variant="outlined" @click="creatingSecret=true" v-if="createSecretTracker.error">Back</v-btn>
          <v-btn color="primary" variant="flat" @click="finished">Done</v-btn>
        </v-card-actions>
      </v-list>
      <v-snackbar v-model="copyConfirm" timeout="2000" color="success">
        <span class="text-body-large align-baseline"><v-icon start>mdi-check-circle</v-icon>Secret copied</span>
      </v-snackbar>
    </v-card>
    <template #activator="attrs">
      <slot name="activator" v-bind="attrs"/>
    </template>
  </v-dialog>
</template>

<script setup>
import {newActionTracker} from '@/api/resource';
import {createSecret, secretToObject} from '@/api/ui/tenant';
import {useNow} from '@/components/now.js';
import RelativeDate from '@/components/RelativeDate.vue';
import {useErrorStore} from '@/components/ui-error/error';
import {DAY} from '@/util/date.js';
import {add} from 'date-fns';
import {computed, onMounted, onUnmounted, reactive, ref} from 'vue';
import {useDate} from 'vuetify';

const emit = defineEmits(['finished']);
const props = defineProps({
  accountName: {
    type: String,
    default: ''
  },
  accountId: {
    type: String,
    default: ''
  }
});

const dialog = ref(false);
const form = ref(null);
// track stage of secret creation
const creatingSecret = ref(true);

const createSecretTracker = reactive(/** @type {ActionTracker<Secret.AsObject>} */ newActionTracker());
const createdSecret = computed(() => secretToObject(createSecretTracker.response));

const suggestedExpiresIn = {
  week: '7 days',
  month: '30 days',
  twoMonths: '60 days',
  quarter: '90 days',
  custom: 'Custom...',
  noExpiry: 'No expiration'
};

const newSecret = reactive({
  note: '',
  expiresIn: suggestedExpiresIn.month,
  expiresAt: /** @type {Date} */ null
});
const date = useDate();
const expiresAtString = computed({
  get: () => newSecret.expiresAt ? date.format(newSecret.expiresAt, 'keyboardDate') : undefined,
  set: v => {
    newSecret.expiresAt = date.date(v);
  }
});
const customExpiryMenuVisible = ref(false);

const {now} = useNow(DAY);
const computedExpiresAt = computed(() => {
  const duration = {};
  switch (newSecret.expiresIn) {
    case suggestedExpiresIn.week:
      duration.days = 7;
      break;
    case suggestedExpiresIn.month:
      duration.days = 30;
      break;
    case suggestedExpiresIn.twoMonths:
      duration.days = 60;
      break;
    case suggestedExpiresIn.quarter:
      duration.days = 90;
      break;
    case suggestedExpiresIn.custom:
      return newSecret.expiresAt;
    default:
      return null;
  }

  return add(now.value, duration);
});

// form validation
const formValid = ref(false);
const noteRules = [
  v => Boolean(v) || Boolean(v.trim()) || 'Notes can\'t be blank'
];
const expiresAtRules = [
  v => Boolean(v) || 'Expiry date can\'t be blank'
];

// UI error handling
const errorStore = useErrorStore();
let unwatchErrors;
onMounted(() => {
  unwatchErrors = errorStore.registerTracker(createSecretTracker);
});
onUnmounted(() => {
  if (unwatchErrors) unwatchErrors();
});

/**
 *
 */
function addSecretReset() {
  newSecret.note = '';
  newSecret.expiresIn = suggestedExpiresIn.month;
  newSecret.expiresAt = null;
  customExpiryMenuVisible.value = false;
  form.value?.resetValidation();
  creatingSecret.value = true;
  // clear out the secret for security
  createSecretTracker.response = {};
}


/**
 *
 */
function cancelAddSecret() {
  addSecretReset();
  dialog.value = false;
}

/**
 *
 */
async function saveSecret() {
  creatingSecret.value = false;
  const secret = {
    note: newSecret.note,
    expireTime: computedExpiresAt.value,
    tenant: {id: props.accountId}
  };
  await createSecret({secret}, createSecretTracker);
}

/**
 *
 */
function finished() {
  creatingSecret.value = true;
  dialog.value = false;
  addSecretReset();
  emit('finished');
}

const copyConfirm = ref(false);

/**
 *
 */
function copySecret() {
  navigator.clipboard.writeText(createdSecret.value.secret);
  copyConfirm.value = true;
}
</script>

<style scoped>
.expires-in {
  width: 180px;
  flex-grow: 0;
}

.expires-at {
  min-width: 150px;
  flex-grow: 0;
  margin-left: 8px;
}

.expires-in-message {
  margin-left: 8px;
  opacity: .7;
}

.banner:before {
  position: absolute;
  content: '';
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: -1;
}

.banner {
  z-index: 1;
}

.banner.secret-banner:before {
  background-color: rgb(var(--v-theme-secondary));
  opacity: 0.2;
}

.banner.info-banner:before {
  background-color: rgb(var(--v-theme-secondaryTeal-darken-1));
  color: white;
  opacity: 1;
}

.banner.error-banner:before {
  background-color: rgb(var(--v-theme-error));
  opacity: 0.5;
}
</style>
