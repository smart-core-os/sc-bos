<template>
  <div
      @wheel="handleWheel"
      v-on="forwardEvents"
      :class="{ moving }"
      @panzoomstart="moving = true"
      @panzoomend="moving = false">
    <slot/>
  </div>
</template>

<script>
// Locally installed panzoom-package (see the ui folder - panzoom-package)
import panzoom from '@smart-core-os/sc-bos-panzoom-package';

// names of events we support
const names = ['start', 'change', 'zoom', 'end', 'pan', 'reset'];

/**
 * Vue component wrapping the @panzoom/panzoom npm module (currently unreleased)
 *
 * @fires init
 * @fires start
 * @fires change
 * @fires zoom
 * @fires end
 * @fires pan
 * @fires reset
 */
export default {
  name: 'PanZoom',
  props: {
    disable: Boolean,
    options: {
      type: Object,
      default() {
        return {};
      }
    }
  },
  emits: ['init', ...names],
  data() {
    return {
      pz: null,
      moving: false
    };
  },
  computed: {
    forwardEvents() {
      const on = {};
      for (let i = 0; i < names.length; i++) {
        const name = names[i];
        const eName = 'on' + name[0].toUpperCase() + name.slice(1);
        if (Object.hasOwn(this.$attrs, eName)) {
          on[`panzoom${name}`] = (e) => this.$emit(name, e.detail);
        }
      }
      return on;
    }
  },
  watch: {
    disable(v) {
      if (v) {
        this.pz.destroy();
        this.pz = null;
      } else {
        this.pz = panzoom(this.$el, {...this.$attrs, ...this.options});
        this.$emit('init', this.pz);
      }
    }
  },
  mounted() {
    if (!this.disable) {
      this.pz = panzoom(this.$el, {...this.$attrs, ...this.options});
      this.$emit('init', this.pz);
    }
  },
  beforeUnmount() {
    if (this.pz) {
      this.pz.destroy();
      this.pz = null;
    }
  },
  methods: {
    handleWheel(e) {
      if (!this.disable) {
        this.$emit('start');
        this.pz.zoomWithWheel(e);
      }
    }
  }
};
</script>

<style scoped>
.moving {
  will-change: transform;
}
</style>
