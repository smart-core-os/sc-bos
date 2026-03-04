<template>
  <div>
    <v-form @submit.prevent="login">
      <v-card-text>
        <p class="text-center">Sign in locally.</p>
      </v-card-text>
      <v-text-field
          autofocus
          label="Username"
          placeholder="Username"
          :rules="[rules.required]"
          variant="outlined"
          v-model="username"
          type="text"
          required/>
      <v-text-field
          label="Password"
          placeholder="Password"
          :rules="[rules.required]"
          variant="outlined"
          v-model="password"
          type="password"
          required/>
      <v-card-actions class="mx-2">
        <v-btn
            type="submit"
            color="primary"
            :disabled="disableSignIn"
            block
            size="large"
            class="text-body-1 font-weight-bold mb-4">
          Sign In
        </v-btn>
      </v-card-actions>
    </v-form>
  </div>
</template>

<script setup>
import {useAccountStore} from '@/stores/account.js';
import {computed, ref} from 'vue';

const accountStore = useAccountStore();
const password = ref('');
const username = ref('');
const rules = {
  required: (value) => !!value || 'Required.'
};
const login = async () => {
  // check if username and password are entered
  if (username.value && password.value) {
    const values = {username: username.value, password: password.value};
    // Forwards the login request to the account store with the values from the form
    await accountStore.loginWithLocalAuth(values);
  } else {
    console.error('username and password are required');
  }
};

const disableSignIn = computed(() => {
  const hasNone = !username.value && !password.value;
  const hasOne = !username.value && password.value || username.value && !password.value;
  return !!(hasNone || hasOne);
});
</script>

<style lang="scss" scoped>
.v-input {
  background-color: transparent;
}
</style>
