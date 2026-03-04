<template>
  <div class="wrapper">
    <div class="svg">
      <div id="score">
        <em>{{ currentWaterUnitStr }}</em>m³
        <template v-if="props.showMaxValue">
          <span>({{ currentMaxValue }}m³)</span>
        </template>
      </div>
      <svg viewBox="0 0 285 285" xmlns="http://www.w3.org/2000/svg" ref="waterTankSVG">
        <g clip-path="url(#glass)">
          <g id="waterfill1" class="waterfill">
            <path
                d="M0.499786 11.0536C0.499786 11.0536 70.7894 -13.817 140.5 11.0536C210.21 35.9243 280.5 11.0536 280.5 11.0536L280.5 220H0.499542L0.499786 11.0536Z"
                class="darkblue">
              <animate
                  dur="3s"
                  attributeType="XML"
                  attributeName="d"
                  repeatCount="indefinite"
                  values="M0.499786 11.0536C0.499786 11.0536 70.7894 -13.817 140.5 11.0536C210.21 35.9243 280.5 11.0536 280.5 11.0536L280.5 220H0.499542L0.499786 11.0536Z;M0.499786 11.0536C0.499786 11.0536 70.9995 35.1072 140.5 11.0536C210 -13 280.5 11.0536 280.5 11.0536L280.5 220H0.499542L0.499786 11.0536Z;M0.499786 11.0536C0.499786 11.0536 70.7894 -13.817 140.5 11.0536C210.21 35.9243 280.5 11.0536 280.5 11.0536L280.5 220H0.499542L0.499786 11.0536Z"/>
            </path>
          </g>
          <rect id="tapwater" x="50" y="49" width="7" height="235" class="blue"/>
          <path
              id="tap"
              d="M60.5 45.3672C60.3828 47.3594 58.7422 49 56.75 49H49.25C47.1406 49 45.5 47.3594 45.5 45.3672C45.3828 43.2579 43.9766 41.5 41.8672 41.5H39.6406C37.0625 46.0704 32.2578 49 26.75 49C21.125 49 16.3203 46.0704 13.7422 41.5H2.375C1.32031 41.5 0.5 40.6797 0.5 39.625C0.5 38.6875 1.32031 37.75 2.375 37.75H16.0859C17.7266 42.2032 21.8281 45.25 26.75 45.25C31.5547 45.25 35.6562 42.2032 37.2969 37.75H41.75C45.8516 37.75 49.25 41.1485 49.25 45.25H56.75C56.75 37.0469 49.9531 30.25 41.75 30.25H37.2969C35.6562 25.9141 31.5547 22.75 26.75 22.75C21.8281 22.75 17.7266 25.9141 16.0859 30.25H2.375C1.32031 30.25 0.5 29.4297 0.5 28.375C0.5 27.4375 1.32031 26.5 2.375 26.5H13.7422C16.0859 22.6329 20.0703 19.8204 24.875 19.2344V13.375H15.6172C14.5625 13.375 13.7422 12.5547 13.7422 11.5C13.7422 10.5625 14.5625 9.74224 15.5 9.74224C28.7257 9.57691 24.6129 9.57281 37.625 9.74224C38.5625 9.62505 39.5 10.5625 39.5 11.5C39.5 12.5547 38.5625 13.375 37.625 13.375H28.625V19.2344C33.3125 19.8204 37.2969 22.6329 39.6406 26.5H41.75C52.0625 26.5 60.5 34.9375 60.5 45.3672Z"
              fill="black"/>
          <g id="waterfill2" class="waterfill">
            <path
                d="M0.499786 11.0536C0.499786 11.0536 70.9995 35.1072 140.5 11.0536C210 -13 280.5 11.0536 280.5 11.0536L280.5 220H0.499542L0.499786 11.0536Z"
                class="blue">
              <animate
                  dur="3s"
                  attributeType="XML"
                  attributeName="d"
                  repeatCount="indefinite"
                  values="M0.499786 11.0536C0.499786 11.0536 70.9995 35.1072 140.5 11.0536C210 -13 280.5 11.0536 280.5 11.0536L280.5 220H0.499542L0.499786 11.0536Z;M0.499786 11.0536C0.499786 11.0536 70.7894 -13.817 140.5 11.0536C210.21 35.9243 280.5 11.0536 280.5 11.0536L280.5 220H0.499542L0.499786 11.0536Z;M0.499786 11.0536C0.499786 11.0536 70.9995 35.1072 140.5 11.0536C210 -13 280.5 11.0536 280.5 11.0536L280.5 220H0.499542L0.499786 11.0536Z;"/>
            </path>
          </g>
        </g>
        <defs>
          <clipPath id="glass">
            <path
                d="M0.5 0H280.5V230C280.5 257.614 258.114 280 230.5 280H50.5C22.8858 280 0.5 257.614 0.5 230V0Z"/>
          </clipPath>
        </defs>
      </svg>
    </div>
    <h2>Water Usage</h2>
  </div>
</template>


<script setup>
import {format} from '@/util/number.js';
import {computed, onMounted, ref, watch} from 'vue';

const props = defineProps({
  waterUnit: {
    type: Number,
    required: true,
  },
  maxValue: {
    type: Number,
    default: 100
  },
  runDemo: {
    type: Boolean,
    default: false,
    required: false,
  },
  showMaxValue: {
    type: Boolean,
    default: false,
    required: false,
  },
});

let autoInterval;
const MAX_RANGE = 200; // Max height of the water tank fill
const OFFSET = 60; // Top offset of the maximum fill position

const waterTankSVG = ref(null);

const currentWaterUnit = ref(props.waterUnit);
const currentMaxValue = ref(props.maxValue);

const currentWaterUnitStr = computed(() => format(currentWaterUnit.value));

let waterfills;

// Update the water fill position based on the value
const setValue = (value) => {
  currentWaterUnit.value = value;
  currentMaxValue.value =
    currentWaterUnit.value > currentMaxValue.value
      ? currentWaterUnit.value
      : currentMaxValue.value;

  // MAX_RANGE + OFFSET are fixed. The max range is the pixel height the tank can fill between
  // and the OFFSET accounts for the top offset of the maximum fill position.
  const tankVal = MAX_RANGE + OFFSET - Math.floor(value / (currentMaxValue.value / MAX_RANGE));

  waterfills.forEach((fill) => {
    fill.style.transform = `translateY(${tankVal}px)`;
  });
};

// Demo: generate random values
const autoValues = () => {
  const randomValue = Math.floor(Math.random() * currentMaxValue.value) + 1;
  setValue(randomValue);
};

// Auto-update the display
const startAuto = () => {
  if (props.runDemo) {
    clearInterval(autoInterval);
    autoInterval = setInterval(autoValues, 5000);
  }
};

onMounted(() => {
  waterfills = waterTankSVG.value.querySelectorAll('.waterfill');

  // Start demo
  startAuto();
});

watch(() => props.waterUnit, (newVal) => {
  setValue(newVal);
});

watch(() => props.maxValue, (newVal) => {
  currentMaxValue.value = newVal;
  setValue(currentWaterUnit.value);
});

</script>

<style lang="scss" scoped>
$blue: #2196f3;
$darkblue: #1a78c2;

.wrapper {
  container-name: svgWrapper;
  //container-type: inline-size;
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: space-between;
  height: 100%;
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
  top: 0;
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

.blue {
  fill: $blue;
}

.darkblue {
  fill: $darkblue;
}

.black {
  fill: #000000;
}

.waterfill {
  transform: translate(0, 260px);
  transition: 2s ease-in-out;
}
</style>
