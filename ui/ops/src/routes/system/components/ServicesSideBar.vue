<template>
  <side-bar>
    <template v-if="canEdit" #actions>
      <v-btn :to="editLink" icon="mdi-pencil" variant="plain" size="small"/>
    </template>
    <status-card/>
    <v-divider/>
    <edit-config-card/>
    <v-divider/>
    <logs-card/>
  </side-bar>
</template>

<script setup>
import SideBar from '@/components/SideBar.vue';
import {useSidebarServiceRouterLink} from '@/dynamic/route.js';
import EditConfigCard from '@/routes/system/components/service-cards/EditConfigCard.vue';
import LogsCard from '@/routes/system/components/service-cards/LogsCard.vue';
import StatusCard from '@/routes/system/components/service-cards/StatusCard.vue';

const {hasLink: canEdit, to: editLink} = useSidebarServiceRouterLink();
</script>
