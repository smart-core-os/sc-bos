

<template>
  <div class="wrapper">
    <div class="svg">
      <div id="score">
        <em>{{ currentQualityStr }}</em>
        ppm
      </div>
      <svg
          id="airQualitySVG"
          width="285"
          height="285"
          viewBox="0 0 285 285"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          ref="airQualitySVG">
        <path
            d="M155.001 48.3216C155.001 69.4137 149.402 76.6369 142.5 76.6369C135.598 76.6369 129.999 69.4137 129.999 48.3216C129.999 27.2296 142.5 0.257324 142.5 0.257324C142.5 0.257324 155.001 27.2296 155.001 48.3216Z"
            class="petal"/>
        <path
            d="M155.001 236.678C155.001 215.586 149.402 208.363 142.5 208.363C135.598 208.363 129.999 215.586 129.999 236.678C129.999 257.77 142.5 284.743 142.5 284.743C142.5 284.743 155.001 257.77 155.001 236.678Z"
            class="petal"/>
        <path
            d="M236.679 155.001C215.587 155.001 208.363 149.402 208.363 142.5C208.363 135.598 215.587 129.999 236.679 129.999C257.771 129.999 284.743 142.5 284.743 142.5C284.743 142.5 257.771 155.001 236.679 155.001Z"
            class="petal"/>
        <path
            d="M48.3216 155.001C69.4137 155.001 76.6369 149.402 76.6369 142.5C76.6369 135.598 69.4137 129.999 48.3216 129.999C27.2296 129.999 0.257324 142.5 0.257324 142.5C0.257324 142.5 27.2296 155.001 48.3216 155.001Z"
            class="petal"/>
        <path
            d="M217.932 84.749C203.016 99.6645 193.951 100.815 189.068 95.9319C184.185 91.0489 185.336 81.9841 200.251 67.0686C215.167 52.1531 243.077 41.9231 243.077 41.9231C243.077 41.9231 232.842 69.8335 217.932 84.749Z"
            class="petal"/>
        <path
            d="M84.7487 217.932C99.6642 203.016 100.815 193.951 95.9317 189.068C91.0487 184.185 81.9839 185.336 67.0684 200.251C52.1529 215.167 41.9229 243.077 41.9229 243.077C41.9229 243.077 69.8332 232.842 84.7487 217.932Z"
            class="petal"/>
        <path
            d="M200.256 217.932C185.341 203.016 184.19 193.951 189.073 189.068C193.956 184.185 203.021 185.336 217.937 200.251C232.852 215.167 243.082 243.077 243.082 243.077C243.082 243.077 215.172 232.842 200.256 217.932Z"
            class="petal"/>
        <path
            d="M67.0684 84.749C81.9839 99.6645 91.0487 100.815 95.9317 95.9319C100.815 91.0489 99.6642 81.9841 84.7487 67.0686C69.8332 52.1531 41.9229 41.9231 41.9229 41.9231C41.9229 41.9231 52.1578 69.8335 67.0684 84.749Z"
            class="petal"/>
        <path
            d="M190.169 60.3194C182.077 79.7969 174.138 84.3244 167.764 81.6731C161.39 79.0217 158.99 70.2087 167.078 50.7262C175.17 31.2487 197.057 11.1343 197.057 11.1343C197.057 11.1343 198.257 40.8369 190.164 60.3194H190.169Z"
            class="petal"/>
        <path
            d="M117.922 234.269C126.014 214.792 123.615 205.974 117.236 203.322C110.862 200.676 102.923 205.199 94.8307 224.676C86.7385 244.154 87.9382 273.861 87.9382 273.861C87.9382 273.861 109.83 253.747 117.917 234.269H117.922Z"
            class="petal"/>
        <path
            d="M224.681 190.169C205.203 182.077 200.676 174.138 203.327 167.764C205.979 161.39 214.792 158.991 234.274 167.078C253.752 175.17 273.866 197.057 273.866 197.057C273.866 197.057 244.163 198.257 224.681 190.165V190.169Z"
            class="petal"/>
        <path
            d="M50.7309 117.922C70.2084 126.014 79.0264 123.615 81.6777 117.236C84.3241 110.862 79.8015 102.923 60.324 94.8307C40.8465 86.7385 11.1389 87.9382 11.1389 87.9382C11.1389 87.9382 31.2534 109.83 50.7309 117.917V117.922Z"
            class="petal"/>
        <path
            d="M234.318 118.095C214.826 126.148 206.013 123.734 203.377 117.355C200.74 110.976 205.282 103.041 224.775 94.9887C244.267 86.936 273.97 88.1901 273.97 88.1901C273.97 88.1901 253.811 110.042 234.318 118.095Z"
            class="petal"/>
        <path
            d="M60.2303 190.012C79.7227 181.959 84.2649 174.03 81.6284 167.646C78.9919 161.262 70.1789 158.852 50.6866 166.905C31.1943 174.958 11.0354 196.81 11.0354 196.81C11.0354 196.81 40.738 198.064 60.2303 190.012Z"
            class="petal"/>
        <path
            d="M166.905 234.319C158.852 214.826 161.267 206.013 167.646 203.377C174.025 200.74 181.959 205.283 190.011 224.775C198.064 244.267 196.81 273.97 196.81 273.97C196.81 273.97 174.958 253.811 166.905 234.319Z"
            class="petal"/>
        <path
            d="M94.9886 60.2303C103.041 79.7227 110.971 84.2649 117.354 81.6284C123.733 78.9919 126.148 70.1789 118.095 50.6866C110.042 31.1943 88.19 11.0354 88.19 11.0354C88.19 11.0354 86.9359 40.738 94.9886 60.2303Z"
            class="petal"/>
        <path
            d="M142.5 74.137C180.256 74.137 210.863 104.744 210.863 142.5C210.863 180.256 180.256 210.863 142.5 210.864C104.744 210.864 74.1369 180.256 74.1368 142.5C74.1368 104.744 104.744 74.137 142.5 74.137Z"
            class="blackStroke"
            stroke-width="5"
            stroke-miterlimit="10"/>
      </svg>
    </div>
    <h2>Air Quality</h2>
  </div>
</template>
<script setup>
import {format} from '@/util/number.js';
import {computed, onMounted, ref, watch} from 'vue';

const props = defineProps({
  airQuality: {
    type: Number,
    required: true,
  },
  minValue: {
    type: Number,
    default: 400,
    required: false,
  },
  maxValue: {
    type: Number,
    default: 600,
    required: true,
  },
  runDemo: {
    type: Boolean,
    default: false,
    required: false,
  },
});

let autoInterval;

const airQualitySVG = ref(null);
let randomPetals;
const currentQuality = ref(props.airQuality);
const currentQualityStr = computed(() => {
  return format(currentQuality.value);
});
const currentMaxValue = ref(props.maxValue);
const currentMinValue = ref(props.minValue);
let range = currentMaxValue.value - currentMinValue.value;
let petals;
// Shuffle array using Fisher-Yates algorithm
const shuffleArray = (arr) => {
  const array = [...arr];
  for (let i = array.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [array[i], array[j]] = [array[j], array[i]];
  }
  return array;
};

// Demo: generate random values
const autoValues = () => {
  const randomValue = Math.floor(Math.random() * range) + currentMinValue.value + 1;
  setValue(randomValue);
};
// Auto-update the display
const startAuto = () => {
  if (props.runDemo) {
    clearInterval(autoInterval);
    autoInterval = setInterval(autoValues, 3000);
  }
};
// Sleep helper for async delays
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

// Update the lights based on value
const setValue = async (value) => {
  clearInterval(autoInterval);

  currentQuality.value = value;
  currentMaxValue.value =
    currentQuality.value > currentMaxValue.value ? currentQuality.value : currentMaxValue.value;

  range = currentMaxValue.value - currentMinValue.value;

  const activeCount = Math.floor((value - currentMinValue.value) / range * petals.length);

  for (let i = 0; i < petals.length; i++) {
    randomPetals[i].classList.toggle('on', i < activeCount);
    await sleep(50); // staggered animation
  }
};

onMounted(() => {
  petals = airQualitySVG.value.querySelectorAll('.petal');
  randomPetals = shuffleArray(petals);
  startAuto();
});

watch(() => props.airQuality, (newValue) => {
  setValue(newValue);
});

watch(() => props.minValue, (newValue) => {
  currentMinValue.value = newValue;
  setValue(currentQuality.value);
});

watch(() => props.maxValue, (newValue) => {
  currentMaxValue.value = newValue;
  setValue(currentQuality.value);
});
</script>
<style lang="scss" scoped>
$green: #4caf50;
$grey: #c6c3c1;

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
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  font-family: sans-serif;
  font-weight: 400;
  font-size: 30px;
  text-align: center;
  line-height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;

  em {
    display: block;
    font-size: 40px;
    font-style: normal;
    font-weight: 600;
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

.petal {
  fill: $green;
  transition: 0.3s ease-in-out;
}

.on {
  fill: $grey;
}

.blackStroke {
  stroke: #000;
}
</style>
