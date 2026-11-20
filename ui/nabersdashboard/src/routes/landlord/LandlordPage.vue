<template>
  <img src="@/assets/powered-by-sc.svg" alt="Powered by Smart Core" class="powered-by-logo">
  <div class="landlord-page">
    <div v-if="ready" class="content-wrapper">
      <NabersBaseSection/>
    </div>
    <div v-else class="d-flex justify-center align-center" style="height: 100%;">
      <v-progress-circular indeterminate size="48"/>
    </div>
  </div>
</template>

<script setup>
import NabersBaseSection from '@/components/nabers-base/NabersBaseSection.vue';
import {useUiConfigStore} from '@/stores/uiConfig.js';
import {useAuthStore} from '@/stores/auth.js';
import {onMounted, ref} from 'vue';

const ready = ref(false);

onMounted(async () => {
  await useUiConfigStore().loadConfig();
  await useAuthStore().fetchToken();
  ready.value = true;
});
</script>

<style scoped>
.landlord-page {
  height:           100vh;
  display:          flex;
  flex-direction:   column;
  overflow:         hidden;
}

.content-wrapper {
  flex:       1;
  min-height: 0;
  overflow-y: auto;
}

.powered-by-logo {
  position:       fixed;
  bottom:         24px;
  right:          40px;
  width:          320px;
  opacity:        0.45;
  pointer-events: none;
  z-index:        10;
}
</style>
