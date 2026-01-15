<template>
  <v-btn>
    {{ btnText }}
    <v-menu v-model="menuVisible" activator="parent" :close-on-content-click="false">
      <new-account-card
          :name="props.name"
          :account-types="props.accountTypes"
          @save="onSave"
          @cancel="onCancel"
          ref="cardRef"/>
    </v-menu>
  </v-btn>
</template>

<script setup>
import NewAccountCard from '@/routes/auth/accounts/NewAccountCard.vue';
import {Account} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/account/v1/account_pb';
import {computed, ref, watch} from 'vue';

const props = defineProps({
  name: {
    type: String,
    default: undefined
  },
  accountTypes: {
    type: Array,
    default: () => undefined
  }
});
const emit = defineEmits(['save', 'cancel']);

const btnText = computed(() => {
  if (!props.accountTypes || props.accountTypes.length > 1) return 'New Account...';
  switch (props.accountTypes[0]) {
    case Account.Type.USER_ACCOUNT:
      return 'New User...'
    case Account.Type.SERVICE_ACCOUNT:
      return 'New Service Account...'
    default:
      return 'New Account...';
  }
})
const menuVisible = ref(false);

const onSave = (e) => {
  emit('save', e);
  menuVisible.value = false;
};
const onCancel = () => {
  emit('cancel');
  menuVisible.value = false;
}
// reset the new account form once the menu is hidden
const cardRef = ref(null);
watch(menuVisible, (value) => {
  if (value) {
    setTimeout(() => {
      const form = cardRef.value;
      if (!form) return;
      form.reset();
    }, 250);
  }
});
</script>

<style scoped>

</style>