<template>
  <div class="fill-height layout-main-side">
    <header>
      <h3 class="text-h3">{{ title }}</h3>
    </header>
    <div class="sections">
      <section v-if="showSectionMain" class="section-main">
        <component
            v-for="(widget, index) in main"
            :key="widget.id ?? index"
            :is="widget.component"
            v-bind="widget.props"
            :style="mainWidgetStyle"/>
      </section>
      <section v-if="showSectionAfter" class="section-after" :style="afterStyle">
        <component
            v-for="(widget, index) in after"
            :key="widget.id ?? index"
            :is="widget.component"
            v-bind="widget.props"/>
      </section>
    </div>
  </div>
</template>

<script setup>
import {computed} from 'vue';

const props = defineProps({
  title: {
    type: String,
    default: ''
  },
  main: {
    type: Array,
    default: null
  },
  after: {
    type: Array,
    default: null
  },
  sideWidth: {
    type: Number,
    default: 260
  },
  mainWidgetMinHeight: {
    type: Number,
    default: 415
  }
});

const showSectionMain = computed(() => Boolean(props.main?.length > 0));
const showSectionAfter = computed(() => Boolean(props.after?.length > 0));

const afterStyle = computed(() => {
  const s = {};
  if (props.sideWidth > 0) {
    s.width = `${props.sideWidth}px`;
  }
  return s;
});
const mainWidgetStyle = computed(() => {
  const s = {};
  if (props.mainWidgetMinHeight > 0) {
    s.minHeight = `${props.mainWidgetMinHeight}px`;
  }
  return s;
});
</script>

<style scoped>
.sections {
  display: flex;
  gap: 24px;
  width: 100%;
  align-items: stretch;
  margin-top: 24px;
}

.section-main {
  flex: 1;
}

.section-main, .section-after {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.section-main {
  min-width: 0;
}

.layout-main-side > header {
}
</style>
