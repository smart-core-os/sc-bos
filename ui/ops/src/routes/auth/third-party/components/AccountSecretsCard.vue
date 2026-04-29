<template>
  <v-card flat tile class="pa-0">
    <v-card-title class="text-subtitle-2 text-title-caps-large text-neutral-lighten-3">Tokens</v-card-title>
    <v-list lines="two" class="pt-0">
      <v-progress-linear color="primary" indeterminate :active="secretsTracker.loading"/>
      <v-hover v-slot="{ isHovering, props: hoverProps }" v-for="secret of secretList" :key="secret.id">
        <v-list-item class="py-0" v-bind="hoverProps">
          <v-list-item-title>{{ secret.note }}</v-list-item-title>
          <v-list-item-subtitle v-if="secret.expireTime" class="text-body-small text-neutral-lighten-5">
            Expire{{ secret.expireTime > Date.now() ? 's' : 'd' }}
            <v-tooltip location="bottom">
              <template #activator="{ props: _props }">
                <span v-bind="_props">{{ humanizeDate(secret.expireTime) }}</span>
              </template>
              <span>{{ Intl.DateTimeFormat('en-GB').format(secret.expireTime) }}</span>
            </v-tooltip>
          </v-list-item-subtitle>
          <v-list-item-subtitle v-else class="text-body-small text-neutral-lighten-5">
            This secret will not expire
          </v-list-item-subtitle>
          <template #append>
            <delete-confirmation-dialog
                title="Delete Token"
                :progress-bar="deleteSecretTracker.loading"
                @confirm="delSecret(secret.id)">
              Are you sure you want to delete the token "{{ secret.note }}"?
              <template #alert-content>
                This action will stop any integrations using this token from functioning.<br><br>
                This action cannot be undone.
              </template>
              <template #confirmBtn>Delete Token</template>
              <template #activator="{ props: _props }">
                <v-btn
                    v-show="isHovering"
                    icon="true"
                    variant="text"
                    size="small"
                    v-bind="_props"
                    :disabled="blockActions">
                  <v-icon color="neutral-lighten-5" size="24">mdi-trash-can</v-icon>
                </v-btn>
              </template>
            </delete-confirmation-dialog>
          </template>
        </v-list-item>
      </v-hover>
    </v-list>
    <v-card-actions class="px-4 pb-4">
      <new-secret-form :account-name="account.title" :account-id="account.id" @finished="refreshSecrets">
        <template #activator="{ props: _props }">
          <v-btn
              block
              color="primary"
              variant="elevated"
              class="font-weight-bold"
              v-bind="_props"
              :disabled="blockActions">
            Create new token
            <v-icon end>mdi-key</v-icon>
          </v-btn>
        </template>
      </new-secret-form>
    </v-card-actions>
  </v-card>
</template>

<script setup>
import {newActionTracker} from '@/api/resource';
import {deleteSecret, listSecrets, secretToObject} from '@/api/ui/tenant';
import {useNow} from '@/components/now.js';
import {useErrorStore} from '@/components/ui-error/error';
import useAuthSetup from '@/composables/useAuthSetup';
import DeleteConfirmationDialog from '@/routes/auth/third-party/components/DeleteConfirmationDialog.vue';
import {HOUR} from '@/util/date.js';
import {compareDesc, formatDistance} from 'date-fns';
import {computed, onMounted, onUnmounted, reactive, watch} from 'vue';
import NewSecretForm from './NewSecretForm.vue';

const secretsTracker = reactive(/** @type {ActionTracker<ListSecretsResponse.AsObject>} */ newActionTracker());
const deleteSecretTracker = reactive(/** @type {ActionTracker<DeleteSecretResponse.AsObject>} */ newActionTracker());

// UI error handling
const errorStore = useErrorStore();
let unwatchSecretErrors;
let unwatchDeleteSecretErrors;
onMounted(() => {
  unwatchSecretErrors = errorStore.registerTracker(secretsTracker);
  unwatchDeleteSecretErrors = errorStore.registerTracker(deleteSecretTracker);
});
onUnmounted(() => {
  unwatchSecretErrors();
  unwatchDeleteSecretErrors();
});

const props = defineProps({
  account: {
    type: Object,
    default: () => {}
  }
});

const secretList = computed(() => {
  // sorted by create time
  return secretsTracker.response?.secretsList
      .map((s) => secretToObject(s))
      .sort((a, b) => compareDesc(a.createTime, b.createTime));
});

// fetch secret data for account
watch(
    () => props.account,
    () => refreshSecrets(),
    {immediate: true}
);

/**
 */
function refreshSecrets() {
  if (!props.account.id) {
    secretsTracker.response = null;
    return;
  }
  listSecrets({tenantId: props.account.id}, secretsTracker).catch((err) => console.error(err));
}

const {now: day} = useNow(HOUR);

/**
 * @param {Date} date
 * @return {string}
 */
function humanizeDate(date) {
  if (date > day.value) {
    return `in ${formatDistance(date, day.value)}`;
  } else {
    return `${formatDistance(day.value, date)} ago`;
  }
}

/**
 * @param {string} id
 */
async function delSecret(id) {
  await deleteSecret({id}, deleteSecretTracker);
  refreshSecrets();
}

// ------------------------------ //
// ----- Authentication settings ----- //

const {blockActions} = useAuthSetup();
</script>

<style scoped>
.v-list--two-line .v-list-item,
.v-list-item--two-line {
  min-height: 48px;
}
</style>
