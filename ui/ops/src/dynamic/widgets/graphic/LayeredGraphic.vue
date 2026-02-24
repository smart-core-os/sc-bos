<template>
  <v-card :height="props.height" class="overflow-hidden pa-3 d-flex flex-column" :key="props.background">
    <v-card-title v-if="showTitle" class="text-h4 ma-0 pa-0 d-flex" style="z-index: 1">
      <span class="mr-auto">{{ props.title }}</span>
      <v-btn v-if="!props.fixed" icon="mdi-cog" variant="flat" @click="toggleSettings"/>
    </v-card-title>
    <overlay-stack v-if="props.fixed" :style="stackStyle">
      <img v-if="bgSrc" :src="bgSrc" alt="Background for other layers" ref="bgRef">
      <graphic-layer
          v-for="(layer, i) in visibleLayers"
          :key="layer.title ?? i"
          :layer="layer"
          @click:element="onElementClick(layer, $event.element, $event.event)"
          :selected="selectionsByLayer[i]"
          @update:selected="onLayerSelectUpdate(i, $event)"/>
    </overlay-stack>
    <pinch-zoom v-else class="fill-height" :hide-controls="props.hideControls" center>
      <template #default>
        <overlay-stack :style="stackStyle">
          <img v-if="bgSrc" :src="bgSrc" alt="Background for other layers" ref="bgRef">
          <graphic-layer
              v-for="(layer, i) in visibleLayers"
              :key="layer.title ?? i"
              :layer="layer"
              @click:element="onElementClick(layer, $event.element, $event.event)"
              :selected="selectionsByLayer[i]"
              @update:selected="onLayerSelectUpdate(i, $event)"/>
        </overlay-stack>
      </template>
    </pinch-zoom>
  </v-card>
</template>

<script setup>
import {getMetadata} from '@/api/sc/traits/metadata.js';
import OverlayStack from '@/components/zoom/OverlayStack.vue';
import PinchZoom from '@/components/zoom/PinchZoom.vue';
import GraphicLayer from '@/dynamic/widgets/graphic/GraphicLayer.vue';
import {useNaturalSize} from '@/dynamic/widgets/graphic/img.js';
import LayeredGraphicSettings from '@/dynamic/widgets/graphic/LayeredGraphicSettings.vue';
import {usePathUtils} from '@/dynamic/widgets/graphic/path.js';
import {nameFromRequest} from '@/dynamic/widgets/graphic/traits.js';
import DeviceSideBar from '@/routes/devices/components/DeviceSideBar.vue';
import {useSidebarStore} from '@/stores/sidebar.js';
import {computed, ref, watch} from 'vue';

const props = defineProps({
  title: {
    type: String,
    default: null
  },
  height: {
    type: [Number, String],
    default: undefined
  },
  hideControls: {
    type: Boolean,
    default: false
  },
  fixed: {
    type: Boolean,
    default: false
  },
  background: {
    type: Object,
    default: null
  },
  // How many pixels are per meter in the graphic
  pixelsPerMeter: {
    type: [Number, String],
    default: null // null means no scaling will be applied
  },
  // When rendering the graphic how many pixels should fit into a meter at scale 1
  targetPixelsPerMeter: {
    type: [Number, String],
    default: 50
  },
  layers: {
    type: Array,
    default: () => []
  }
});

const {toPath} = usePathUtils();
const bgSrc = computed(() => toPath(props.background?.svgPath));
const bgRef = ref(null);
const {naturalWidth} = useNaturalSize(bgRef);
const scaledWidth = computed(() => {
  if (!naturalWidth.value || !props.pixelsPerMeter || !props.targetPixelsPerMeter) return undefined;
  return naturalWidth.value * props.targetPixelsPerMeter / props.pixelsPerMeter;
});
const stackStyle = computed(() => {
  const res = {};
  if (scaledWidth.value) {
    res.width = `${scaledWidth.value}px`;
  }
  return res;
});

const showTitle = computed(() => Boolean(props.title || !props.fixed));

const sidebar = useSidebarStore();

// options/settings for adjusting what we show in the graphic
const optLayerList = computed(() => {
  return props.layers.map((l) => {
    return {title: l.title, value: l.title};
  });
});
const optVisibleLayers = ref(/** @type {string[]} */ []);
watch(() => props.layers, () => {
  // todo: remember which layers are selected when this changes
  optVisibleLayers.value = props.layers
      .filter((layer) => !layer.hidden)
      .map((layer) => layer.title);
}, {immediate: true});

const toggleSettings = () => {
  if (sidebar.component === LayeredGraphicSettings) {
    sidebar.closeSidebar();
    return;
  }
  sidebar.title = 'Graphic Settings';
  sidebar.data = {
    layerList: optLayerList,
    visibleLayers: optVisibleLayers
  };
  sidebar.component = LayeredGraphicSettings;
  sidebar.visible = true;
};

const visibleLayers = computed(() => {
  return props.layers.filter((l) => optVisibleLayers.value.includes(l.title));
});

const selectionCtx = ref(null);
const onElementClick = async (layer, element) => {
  selectionCtx.value = {layer, element};
  // Find the name of the device we should be showing in the sidebar.
  // First we check if it's configured explicitly via the sidebar property.
  // Then we try to find a source that mentions a device name in the request.
  let name = element.sidebar?.name;
  if (!name) {
    for (const [, source] of Object.entries(element.sources ?? {})) {
      name = nameFromRequest(source.request);
      if (name) {
        break;
      }
    }
  }
  if (!name) return; // no name, nothing to show.

  // this is annoying, because the sidebar currently doesn't support fetching its own device info, so get it instead.
  const md = await getMetadata({name: name});
  if (md) {
    // open the sidebar with the metadata
    sidebar.title = md.appearance?.title ?? name;
    sidebar.data = {
      metadata: md, name,
      // these aren't used by the sidebar, but are used to work out if our selection is still the active one
      layer, element
    };
    sidebar.component = DeviceSideBar; // this line must be after the .data one!
    sidebar.visible = true;
  }
};

// force a single selection between all layers, and make sure we clear selection when someone else uses the sidebar
const selectionsByLayer = ref([]);
watch(() => props.layers, () => {
  selectionsByLayer.value = props.layers.map(() => null);
});
watch(() => sidebar.data, (c) => {
  if (c.element !== selectionCtx.value?.element || c.layer !== selectionCtx.value?.layer) {
    // someone else selected something, or the sidebar was hidden
    selectionsByLayer.value = selectionsByLayer.value.map(() => null);
    selectionCtx.value = null;
  }
});
const onLayerSelectUpdate = (layerIdx, selected) => {
  for (let i = 0; i < props.layers.length; i++) {
    if (i === layerIdx) selectionsByLayer.value[i] = selected;
    else selectionsByLayer.value[i] = null;
  }
};
</script>

<style scoped>
img {
  min-width: 0;
}
</style>
