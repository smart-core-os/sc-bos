<template>
  <side-bar @close="onCloseClick">
    <template #actions>
      <v-btn v-if="!editMode" @click="onEditClick" icon="mdi-pencil" variant="plain" size="small"/>
    </template>
    <template v-if="!editMode">
      <div v-if="account" class="details pt-4">
        <p v-if="account.description" class="px-4">{{ account.description }}</p>
        <template v-if="account.userDetails?.username">
          <v-list-subheader>Username</v-list-subheader>
          <p class="px-4">{{ account.userDetails?.username }}</p>
        </template>
        <template v-if="account.type === Account.Type.SERVICE_ACCOUNT">
          <v-list-subheader>Client ID</v-list-subheader>
          <p class="px-4">
            <copy-div :text="account.id" icon-size="18" :btn-props="{size: 'small', class: 'ma-n2'}"/>
          </p>
          <p class="px-4">
            <v-btn block variant="outlined">
              <v-icon icon="mdi-key" start/>
              Generate Secret...
              <v-dialog activator="parent" width="440" v-model="rotateSecretDialogVisible">
                <v-card title="Generate a New Secret">
                  <v-card-text>
                    Replace the secret associated with {{ sidebar.title }} with a new one.
                  </v-card-text>
                  <v-card-text class="pt-0">
                    The old secret will remain valid for a short period of time to allow for a smooth transition
                    while systems using that secret are updated.
                  </v-card-text>
                  <v-card-actions class="px-6 mb-6">
                    Expire old secret:
                    <v-select :items="rotateSecretExpiryItems"
                              v-model="rotateSecretExpiry"
                              density="compact"
                              hide-details/>
                  </v-card-actions>
                  <v-card-actions>
                    <v-btn text="cancel" @click="onRotateCancel"/>
                    <v-btn type="submit" color="primary" variant="flat" @click="onRotateSave">
                      Generate New Secret
                    </v-btn>
                  </v-card-actions>
                  <v-expand-transition>
                    <div v-if="rotateSecretError">
                      <v-alert type="error" tile>
                        Error rotating secret:
                        {{ rotateSecretError.error?.message || rotateSecretError.message || rotateSecretError }}
                      </v-alert>
                    </div>
                  </v-expand-transition>
                </v-card>
              </v-dialog>
            </v-btn>
          </p>
          <template v-if="previousSecretExpireTime && previousSecretNotExpired">
            <v-list-subheader>Old Secret Expires</v-list-subheader>
            <p class="px-4">{{ previousSecretExpireTime.toLocaleString() }}</p>
          </template>
        </template>
        <v-list-subheader>Account Created</v-list-subheader>
        <p class="px-4">{{ timestampToDate(account.createTime).toLocaleString() }}</p>
      </div>
    </template>
    <template v-else>
      <v-form @submit.prevent="onSaveClick" v-model="formValid">
        <v-text-field
            class="ma-4"
            v-model="editDisplayNameModel"
            :rules="editDisplayNameRules"
            :counter="100"
            label="Name"
            autocomplete="off"/>
        <v-textarea
            class="ma-4"
            v-model="editDescriptionModel"
            label="Description"
            :counter="250"
            autocomplete="off"/>
        <div class="d-flex ga-2 ma-4">
          <v-btn @click="onSaveClick" variant="flat" color="primary" type="submit" :disabled="!formValid">Save</v-btn>
          <v-btn @click="onCancelClick" variant="flat">Cancel</v-btn>
        </div>
      </v-form>
      <v-expand-transition>
        <div v-if="saveError">
          <v-alert type="error" tile :text="saveErrorStr"/>
        </div>
      </v-expand-transition>
    </template>
    <v-list>
      <v-list-subheader>
        {{ roleAssignments.length }} Role{{ roleAssignments.length === 1 ? '' : 's' }}
      </v-list-subheader>
      <v-list-item v-for="roleAssignment in roleAssignments" :key="roleAssignment.id" min-height="0px">
        <v-list-item-title>
          <role-assignment-link :role-assignment="roleAssignment"/>
        </v-list-item-title>
        <template #append>
          <v-list-item-action>
            <v-btn
                icon="mdi-delete"
                size="small"
                variant="plain"
                v-tooltip:bottom="'Remove role'"
                class="my-n2"
                :loading="roleDeleteLoading[roleAssignment.id]"
                @click="onRoleRemoveClick(roleAssignment)"/>
          </v-list-item-action>
        </template>
      </v-list-item>
    </v-list>
  </side-bar>
</template>

<script setup>
import {timestampToDate} from '@/api/convpb.js';
import CopyDiv from '@/components/CopyDiv.vue';
import {DAY, HOUR, MINUTE} from '@/components/now.js';
import SideBar from '@/components/SideBar.vue';
import {useIsFutureDate} from '@/composables/time.js';
import RoleAssignmentLink from '@/routes/auth/accounts/RoleAssignmentLink.vue';
import {useSidebarStore} from '@/stores/sidebar.js';
import {Account} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/account/v1/account_pb';
import {computed, reactive, ref} from 'vue';
import {useRoute, useRouter} from 'vue-router';

const sidebar = useSidebarStore();
const account = computed(() => sidebar.data?.account);
const roleAssignmentsCollection = computed(() => sidebar.data?.roleAssignments);

const roleAssignments = computed(() => roleAssignmentsCollection.value?.items ?? []);

const previousSecretExpireTime = computed(() => timestampToDate(account.value?.serviceDetails?.previousSecretExpireTime));
const previousSecretNotExpired = useIsFutureDate(previousSecretExpireTime);

const editMode = ref(false);
const formValid = ref(false);
const saving = ref(false);
const saveError = ref(null);
const saveErrorStr = computed(() => {
  const err = saveError.value;
  if (!err) return null;
  const msg = `${err.error?.message ?? err.message ?? err}`;
  return 'Error saving account: ' + msg;
});

const rotateSecretExpiryItems = [
  {title: 'Immediately', value: null},
  {title: 'In 15 minutes', value: 15 * MINUTE},
  {title: 'In an hour', value: HOUR},
  {title: 'In a day', value: DAY},
  {title: 'In 7 days', value: 7 * DAY},
  {title: 'In 30 days', value: 30 * DAY},
];
const rotateSecretExpiry = ref(null);
const rotateSecretDialogVisible = ref(false);
const rotateSecretInProgress = ref(false);
const rotateSecretError = ref(null);
const onRotateCancel = () => {
  rotateSecretDialogVisible.value = false;
}
const onRotateSave = async () => {
  rotateSecretInProgress.value = true
  try {
    rotateSecretDialogVisible.value = false;
    const expireTime = (() => {
      if (!rotateSecretExpiry.value) return null;
      return new Date(Date.now() + rotateSecretExpiry.value);
    })();
    await sidebar.data.rotateServiceAccountSecret({
      account: account.value,
      expireTime
    });
    rotateSecretDialogVisible.value = false;
  } catch (e) {
    rotateSecretError.value = e;
  } finally {
    rotateSecretInProgress.value = false;
  }
}

const router = useRouter();
const route = useRoute();
const onCloseClick = () => {
  router.push({name: route.name}); // remove props from the route
}
const onEditClick = () => {
  editDisplayNameModel.value = account.value?.displayName ?? '';
  editDescriptionModel.value = account.value?.description ?? '';
  editMode.value = true;
};
const onCancelClick = () => {
  editMode.value = false;
};
const onSaveClick = async () => {
  try {
    saving.value = true;
    const newAccount = {
      ...account.value,
      displayName: editDisplayNameModel.value,
      description: editDescriptionModel.value,
    }
    await sidebar.data.updateAccount({account: newAccount});
    saveError.value = null;
    editMode.value = false;
  } catch (e) {
    saveError.value = e;
  } finally {
    saving.value = false;
  }
};

const editDisplayNameModel = ref(null);
const editDisplayNameRules = computed(() => {
  return [
    (v) => !!v || 'Name is required',
    (v) => v.length >= 3 || 'Name must be at least 3 characters',
    (v) => v.length <= 100 || 'Name must be less than 100 characters',
  ];
})
const editDescriptionModel = ref(null);

const roleDeleteLoading = reactive({});
const roleDeleteError = reactive({});
const onRoleRemoveClick = async (roleAssignment) => {
  roleDeleteLoading[roleAssignment.id] = true;
  try {
    delete roleDeleteError[roleAssignment.id];
    await sidebar.data.removeRole(roleAssignment);
  } catch (e) {
    roleDeleteError[roleAssignment.id] = e;
  } finally {
    delete roleDeleteLoading[roleAssignment.id];
  }
}
</script>

<style scoped>
.details .v-list-subheader {
  min-height: initial;
}
</style>