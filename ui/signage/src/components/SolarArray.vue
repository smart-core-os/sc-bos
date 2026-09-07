<template>
  <div class="wrapper">
    <div class="svg">
      <div id="score">
        <em>{{ generatedStr }}</em>{{ unit }}
      </div>
      <svg
          id="solarArraySVG"
          width="366"
          height="285"
          viewBox="0 0 366 285"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          ref="solarArraySVG">
        <g id="sun">
          <g id="sunRays">
            <path v-for="(ray, i) in sunRays" :key="i" :d="ray.d" class="pip"/>
          </g>
          <circle :cx="SUN.cx" :cy="SUN.cy" :r="SUN.r" class="sunDisc"/>
        </g>
        <g id="panel" class="off">
          <path :d="panelOutline" class="panelFace"/>
          <g class="panelGrid">
            <line v-for="(l, i) in panelGrid" :key="i" :x1="l.x1" :y1="l.y1" :x2="l.x2" :y2="l.y2"/>
          </g>
          <path :d="panelOutline" class="panelEdge"/>
        </g>
      </svg>
    </div>
    <h2>Energy Generated</h2>
    <template v-if="props.showPeriod">
      <div>
        <sub>
          {{ periodStr }}
        </sub>
      </div>
    </template>
  </div>
</template>

<script setup>
import {format, roundTo} from '@/util/number.js';
import {sentenceCase} from 'change-case';
import {computed, onMounted, onUnmounted, ref, watch} from 'vue';

const props = defineProps({
  generated: {
    type: Number,
    default: 0,
  },
  unit: {
    type: String,
    default: 'kWh',
    required: false,
  },
  period: {
    type: String,
    default: 'Day',
    required: false,
  },
  showPeriod: {
    type: Boolean,
    default: false,
    required: false,
  },
  totalRange: {
    type: Number,
    default: 300
  },
  runDemo: {
    type: Boolean,
    default: false,
    required: false,
  },
});

// The artwork is a sun above a solar panel, drawn in the same 366x285 viewBox the other widgets
// use so every tile in the board sizes alike. Everything sits above y=240, leaving the bottom
// band clear for the #score readout.
const SUN = {cx: 183, cy: 72, r: 30};
const RAY_COUNT = 36;
const RAY_INNER = 38; // radius the rays start at, just clear of the disc
const RAY_OUTER = 64;
const RAY_INNER_HALF = 2.4; // half-width at RAY_INNER; rays taper outwards
const RAY_OUTER_HALF = 1.1;

// A ray is a quad: two points at RAY_INNER either side of the ray's axis, two at RAY_OUTER.
const rayPath = (angle) => {
  const [dx, dy] = [Math.cos(angle), Math.sin(angle)];
  const [px, py] = [-dy, dx]; // perpendicular to the ray, so half-widths measure across it
  const corner = (r, half) =>
    `${roundTo(SUN.cx + dx * r + px * half, 2)} ${roundTo(SUN.cy + dy * r + py * half, 2)}`;
  return `M${corner(RAY_INNER, RAY_INNER_HALF)}L${corner(RAY_OUTER, RAY_OUTER_HALF)}` +
      `L${corner(RAY_OUTER, -RAY_OUTER_HALF)}L${corner(RAY_INNER, -RAY_INNER_HALF)}Z`;
};

// Generated rather than hand-exported, so the ray count stays a single constant. Emitted in
// reverse lighting order — onMounted reverses the node list, which leaves the most recently lit
// pip first among its siblings, where the flicker selector can find it.
const RAY_STEP = (2 * Math.PI) / RAY_COUNT;
const sunRays = Array.from({length: RAY_COUNT}, (_, i) => {
  // Lighting runs clockwise from 12 o'clock, so drawing runs anticlockwise back to it.
  return {d: rayPath(-Math.PI / 2 + (RAY_COUNT - 1 - i) * RAY_STEP)};
});

// A parallelogram, so the panel reads as tilted away from the viewer.
const PANEL = {tl: [120, 150], tr: [312, 150], br: [246, 236], bl: [54, 236]};
const PANEL_COLS = 6;
const PANEL_ROWS = 3;

const panelOutline = `M${PANEL.tl.join(' ')}L${PANEL.tr.join(' ')}L${PANEL.br.join(' ')}L${PANEL.bl.join(' ')}Z`;

const lerp = (a, b, t) => [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t];
const gridLine = (a, b) => ({x1: a[0], y1: a[1], x2: b[0], y2: b[1]});
const panelGrid = [
  ...Array.from({length: PANEL_COLS - 1}, (_, i) => {
    const t = (i + 1) / PANEL_COLS;
    return gridLine(lerp(PANEL.tl, PANEL.tr, t), lerp(PANEL.bl, PANEL.br, t));
  }),
  ...Array.from({length: PANEL_ROWS - 1}, (_, i) => {
    const t = (i + 1) / PANEL_ROWS;
    return gridLine(lerp(PANEL.tl, PANEL.bl, t), lerp(PANEL.tr, PANEL.br, t));
  })
];

let autoInterval;

const solarArraySVG = ref(null);

const currentGenerated = ref(props.generated);
const currentTotalRange = ref(props.totalRange);

const generatedStr = computed(() => format(currentGenerated.value));

const periodStr = computed(() => {
  return `per ${sentenceCase(props.period)}`;
});

let sunPips;
let panelObj;

// Demo: generate random values
const autoValues = () => setValue(Math.floor(Math.random() * currentTotalRange.value + 1));

// Sleep helper for async delays
const sleep = (ms) => {
  return new Promise((resolve) => setTimeout(resolve, ms));
};

// Brighten the panel as more of the sun lights up
const panelBrightness = (fraction) => {
  panelObj.classList.remove('off', 'low', 'mid', 'high');
  if (!(fraction > 0)) panelObj.classList.add('off');
  else if (fraction < 1 / 3) panelObj.classList.add('low');
  else if (fraction < 2 / 3) panelObj.classList.add('mid');
  else panelObj.classList.add('high');
};

// Auto-update the display
const startAuto = () => {
  if (props.runDemo) {
    clearInterval(autoInterval);
    autoInterval = setInterval(autoValues, 3000);
  }
};

// Shared cancellation — any new setValue call cancels the previous one
let cancelCurrent = () => {};

// Update display and animations based on value
const setValue = async (generated) => {
  clearInterval(autoInterval);
  cancelCurrent();
  let cancelled = false;
  cancelCurrent = () => { cancelled = true; };

  currentGenerated.value = generated;

  // Grow the range to fit, so a site with no configured range still scales
  currentTotalRange.value = Math.max(currentTotalRange.value, currentGenerated.value);

  const pipVal = Math.floor(Math.abs(generated) / currentTotalRange.value * sunPips.length);
  // Light at least one pip whenever anything at all is being generated, so a figure too
  // small to fill a pip still reads as generating.
  const litPips = Math.max(pipVal, generated > 0 ? 1 : 0);
  panelBrightness(litPips / sunPips.length);

  for (let i = 0; i < sunPips.length; i++) {
    if (cancelled) return;
    if (i < litPips) {
      sunPips[i].classList.add('onSun');
    } else {
      sunPips[i].classList.remove('onSun');
    }
    await sleep(50);
  }

  startAuto();
};

onMounted(() => {
  panelObj = solarArraySVG.value.getElementById('panel');
  sunPips = [...solarArraySVG.value.querySelectorAll('#sunRays .pip')].reverse();

  setValue(props.generated);
  // Start demo
  startAuto();
});

onUnmounted(() => {
  clearInterval(autoInterval);
  cancelCurrent();
});

watch(() => props.generated, (newGenVal) => {
  setValue(newGenVal);
});
</script>

<style lang="scss" scoped>
$grey: #c6c3c1;
$amber: #ffb300;

.wrapper {
  container-name: svgWrapper;
  //container-type: inline-size;
  display: flex;
  position: relative;
  flex-direction: column;
  align-items: center;
  justify-content: space-between;
  height: 100%;
  width: 100%;
  color: #000;

  .svg {
    display: flex;
    flex-direction: column;
    align-items: center;
    position: relative;

    svg {
      width: 100%;
      height: auto;
    }
  }
}

#score {
  position: absolute;
  left: 0;
  right: 0;
  top: 90%;
  font-family: sans-serif;
  font-weight: 400;
  font-size: 30px;
  text-align: center;
  line-height: 100%;

  em {
    font-size: 40px;
    font-style: normal;
    font-weight: 600;
  }

  span {
    display: block;
    font-size: 0.6em;
  }
}

@container svgWrapper (width < 400px) {
  #score {
    font-size: clamp(0.5rem, -0.0169rem + 8.2707cqi, 1.875rem);

    em {
      font-size: clamp(0.75rem, 0.0921rem + 10.5263cqi, 2.5rem);
    }
  }

  h2 {
    font-size: clamp(0.375rem, -0.0479rem + 6.7669cqi, 1.5rem);
  }
}

.pip {
  fill: $grey;
  transition: 0.3s ease-in-out;

  &.onSun {
    fill: $amber;

    &:not(.onSun ~ .onSun) {
      animation: flickerAmber 0.5s linear infinite;
      animation-delay: 0.5s;
    }
  }
}

.sunDisc {
  fill: $amber;
}

#panel {
  .panelFace {
    fill: #8d8a88;
    transition: 0.6s ease-in-out;
  }

  .panelGrid line {
    stroke: rgba(247, 244, 241, 0.6);
    stroke-width: 2;
  }

  .panelEdge {
    fill: none;
    stroke: #000;
    stroke-width: 3;
  }

  &.off .panelFace {
    fill: #8d8a88;
  }

  &.low .panelFace {
    fill: #5d5347;
  }

  &.mid .panelFace {
    fill: #8a6f24;
  }

  &.high .panelFace {
    fill: #c69211;
  }
}

@keyframes flickerAmber {
  45% {
    fill: $amber;
  }

  55% {
    fill: $grey;
  }
}
</style>
