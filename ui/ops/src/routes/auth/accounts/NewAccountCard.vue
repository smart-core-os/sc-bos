<template>
  <v-card min-width="440">
    <v-form @submit.prevent="onSave"
            v-model="formValid"
            ref="formRef"
            :disabled="formDisabled">
      <v-card-title>{{ titleText }}</v-card-title>
      <v-card-text class="ga-2 d-flex flex-column">
        <v-btn-toggle
            v-if="props.accountTypes.length > 1"
            v-model="accountType"
            variant="outlined"
            divided
            mandatory
            density="comfortable"
            class="mb-2">
          <v-btn :value="Account.Type.USER_ACCOUNT" text="User" v-tooltip:bottom="'User account'"/>
          <v-btn :value="Account.Type.SERVICE_ACCOUNT" text="Service" v-tooltip:bottom="'Service account'"/>
        </v-btn-toggle>
        <template v-if="accountType === Account.Type.USER_ACCOUNT">
          <v-text-field label="Full Name"
                        v-model.trim="fullName"
                        :rules="fullNameRules"
                        required
                        :counter="100"
                        autocomplete="off"/>
          <v-text-field label="Username"
                        v-model.trim="username"
                        :rules="usernameRules"
                        required
                        :counter="100"
                        autocomplete="off"/>
          <v-text-field label="Password"
                        v-model.trim="password"
                        required
                        :type="showPassword ? 'text' : 'password'"
                        :append-inner-icon="showPassword ? 'mdi-eye' : 'mdi-eye-off'"
                        @click:append-inner="showPassword = !showPassword"
                        autocomplete="off"/>
        </template>
        <template v-if="accountType === Account.Type.SERVICE_ACCOUNT">
          <v-text-field label="Account Name"
                        v-model.trim="accountName"
                        :rules="accountNameRules"
                        required
                        :counter="100"
                        autocomplete="off"/>
          <v-textarea label="Description"
                      v-model.trim="description"
                      :counter="500"
                      autocomplete="off"/>
        </template>
      </v-card-text>
      <v-card-actions>
        <v-btn text="Create"
               color="primary"
               type="submit"
               variant="flat"
               :disabled="!formValid"
               :loading="saveLoading"/>
        <v-btn text="Cancel" @click="onCancel"/>
      </v-card-actions>
      <v-expand-transition>
        <div v-if="errorStr">
          <v-alert type="error" :text="errorStr" tile class="mt-4"/>
        </div>
      </v-expand-transition>
    </v-form>
  </v-card>
</template>

<script setup>
import {createAccount} from '@/api/ui/account.js';
import {useLocalProp} from '@/util/vue.js';
import {Account} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/account/v1/account_pb';
import {computed, ref, watch} from 'vue';

const props = defineProps({
  name: {
    type: String,
    default: undefined,
  },
  accountTypes: {
    type: Array,
    default: () => [Account.Type.USER_ACCOUNT, Account.Type.SERVICE_ACCOUNT],
  },
});

const emit = defineEmits(['save', 'cancel', 'error']);

const formRef = ref(null);
const formValid = ref(false);

const titleText = computed(() => {
  if (!props.accountTypes || props.accountTypes.length > 1) return 'New Account';
  switch (props.accountTypes[0]) {
    case Account.Type.USER_ACCOUNT:
      return 'New User'
    case Account.Type.SERVICE_ACCOUNT:
      return 'New Service Account'
    default:
      return 'New Account';
  }
})
const accountType = useLocalProp(computed(() => {
  return props.accountTypes?.[0] ?? Account.Type.USER_ACCOUNT;
}));
const fullName = ref('');
const fullNameRules = computed(() => {
  return [
    (v) => !!v || 'Full Name is required',
    (v) => v.length >= 3 || 'Full Name must be at least 3 characters',
    (v) => v.length <= 100 || 'Full Name must be less than 100 characters',
  ];
})
const username = ref('');
const usernameRules = computed(() => {
  return [
    (v) => !!v || 'Username is required',
    (v) => v.length >= 3 || 'Username must be at least 3 characters',
    (v) => v.length <= 100 || 'Username must be less than 100 characters',
    (v) => /^[a-zA-Z0-9.-_@]+$/.test(v) || 'Username must use only alphanumerics with .-_@',
  ];
})
const password = ref('');
const showPassword = ref(false);
// guess a suitable username based on the entered full name
const nameToUsername = (name) => {
  if (!name) return '';
  return name.toLowerCase().replace(/[^a-zA-Z0-9.-_@]+/g, '.');
};
watch(fullName, (value, oldValue) => {
  if (username.value && username.value !== nameToUsername(oldValue)) {
    return; // manual username has been entered
  }
  username.value = nameToUsername(value);
});

const accountName = ref('');
const accountNameRules = computed(() => {
  return [
    (v) => !!v || 'Account Name is required',
    (v) => v.length >= 3 || 'Account Name must be at least 3 characters',
    (v) => v.length <= 100 || 'Account Name must be less than 100 characters',
  ];
});
const description = ref('');

const saveError = ref(null);
const errorStr = computed(() => {
  if (!saveError.value) return null;
  return saveError.value.message;
});

const formDisabled = ref(false);
const saveLoading = ref(false);
const reset = () => {
  const form = formRef.value;
  if (!form) return;
  form.reset();
  accountType.value = Account.Type.USER_ACCOUNT;
  saveError.value = null;
  showPassword.value = false;
}

const onSave = async () => {
  saveLoading.value = true;
  formDisabled.value = true;
  try {
    switch (accountType.value) {
      case Account.Type.USER_ACCOUNT: {
        const createAccountReq = {
          name: props.name,
          account: {
            type: Account.Type.USER_ACCOUNT,
            displayName: fullName.value,
            userDetails: {
              username: username.value,
            }
          },
          password: password.value,
        }
        const account = await createAccount(createAccountReq);
        emit('save', {account});
        break;
      }
      case Account.Type.SERVICE_ACCOUNT: {
        const createAccountReq = {
          name: props.name,
          account: {
            type: Account.Type.SERVICE_ACCOUNT,
            displayName: accountName.value,
            description: description.value,
          }
        };
        const account = await createAccount(createAccountReq);
        emit('save', {account});
        break;
      }
    }
    saveError.value = null;
  } catch (e) {
    saveError.value = e;
  } finally {
    saveLoading.value = false;
    formDisabled.value = false;
  }
};
const onCancel = () => {
  emit('cancel');
}

defineExpose({
  reset,
})
</script>

<style scoped>

</style>