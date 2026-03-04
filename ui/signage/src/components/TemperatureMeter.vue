<template>
  <div class="wrapper">
    <div class="svg">
      <div id="score">
        <em>{{ temperatureStr }}</em>°C
        <template v-if="props.showEnergy">
          <span>{{ currentEnergyStr }}</span>
        </template>
      </div>
      <svg
          id="temperatureMeterSVG"
          width="366"
          height="294"
          viewBox="0 0 366 294"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          ref="temperatureMeterSVG">
        <g id="pipsCrescent">
          <path
              d="M353.597 249.355L349.316 260L308.049 241.418L310.618 235.031L353.597 249.355Z"
              class="pips"/>
          <path
              d="M360.303 227.85L357.189 238.881L314.159 224.704L316.028 218.085L360.303 227.85Z"
              class="pips"/>
          <path
              d="M364.759 203.741L363.064 215.069L318.577 206.288L319.594 199.492L364.759 203.741Z"
              class="pips"/>
          <path
              d="M365.982 180.428L365.763 191.876L320.504 188.745L320.635 181.876L365.982 180.428Z"
              class="pips"/>
          <path
              d="M363.954 156.155L365.336 167.524L320.071 170.561L319.242 163.74L363.954 156.155Z"
              class="pips"/>
          <path
              d="M358.717 133.007L361.648 144.086L317.217 153.137L315.458 146.489L358.717 133.007Z"
              class="pips"/>
          <path
              d="M350.245 110.513L354.679 121.096L311.911 136.022L309.251 129.672L350.245 110.513Z"
              class="pips"/>
          <path
              d="M338.95 89.5678L344.75 99.489L304.33 119.798L300.85 113.846L338.95 89.5678Z"
              class="pips"/>
          <path
              d="M324.844 70.0375L331.902 79.1328L294.498 104.441L290.263 98.9834L324.844 70.0375Z"
              class="pips"/>
          <path
              d="M308.489 52.5567L316.645 60.7109L282.746 90.4294L277.852 85.5369L308.489 52.5567Z"
              class="pips"/>
          <path
              d="M290.164 37.2549L299.268 44.3682L269.299 77.9387L263.836 73.6708L290.164 37.2549Z"
              class="pips"/>
          <path
              d="M269.925 24.1997L279.861 30.1345L254.388 67.1364L248.427 63.5755L269.925 24.1997Z"
              class="pips"/>
          <path
              d="M248.382 13.7839L258.98 18.4753L238.295 58.2722L231.936 55.4573L248.382 13.7839Z"
              class="pips"/>
          <path
              d="M225.402 6.11371L236.532 9.39819L221.309 51.5202L214.631 49.5495L225.402 6.11371Z"
              class="pips"/>
          <path
              d="M201.301 1.58552L212.793 3.27319L203.816 47.1035L196.921 46.0909L201.301 1.58552Z"
              class="pips"/>
          <path d="M189.798 0L178.394 0L180.675 45.614L187.517 45.614L189.798 0Z" class="pips"/>
          <path
              d="M165.181 1.58552L153.689 3.27319L162.666 47.1035L169.561 46.0909L165.181 1.58552Z"
              class="pips"/>
          <path
              d="M141.079 6.11371L129.949 9.39819L145.172 51.5202L151.85 49.5495L141.079 6.11371Z"
              class="pips"/>
          <path
              d="M118.101 13.7839L107.502 18.4753L128.187 58.2722L134.546 55.4573L118.101 13.7839Z"
              class="pips"/>
          <path
              d="M96.5563 24.1997L86.6206 30.1345L112.093 67.1364L118.055 63.5755L96.5563 24.1997Z"
              class="pips"/>
          <path
              d="M76.3174 37.2549L67.2131 44.3682L97.1825 77.9387L102.645 73.6708L76.3174 37.2549Z"
              class="pips"/>
          <path
              d="M57.9922 52.5567L49.8359 60.7109L83.7353 90.4294L88.629 85.5369L57.9922 52.5567Z"
              class="pips"/>
          <path
              d="M41.637 70.0375L34.5793 79.1328L71.9831 104.441L76.2177 98.9834L41.637 70.0375Z"
              class="pips"/>
          <path
              d="M27.5311 89.5678L21.731 99.489L62.1512 119.798L65.6313 113.846L27.5311 89.5678Z"
              class="pips"/>
          <path
              d="M16.2358 110.513L11.8018 121.096L54.5696 136.022L57.23 129.672L16.2358 110.513Z"
              class="pips"/>
          <path
              d="M7.76488 133.007L4.83374 144.086L49.2649 153.137L51.0236 146.489L7.76488 133.007Z"
              class="pips"/>
          <path
              d="M2.52731 156.155L1.14551 167.524L46.4109 170.561L47.24 163.74L2.52731 156.155Z"
              class="pips"/>
          <path
              d="M0.500094 180.428L0.718994 191.876L45.9778 188.745L45.8465 181.876L0.500094 180.428Z"
              class="pips"/>
          <path
              d="M1.72119 203.742L3.41626 215.069L47.9031 206.289L46.886 199.492L1.72119 203.742Z"
              class="pips"/>
          <path
              d="M6.1781 227.85L9.29248 238.881L52.322 224.704L50.4533 218.086L6.1781 227.85Z"
              class="pips"/>
          <path
              d="M12.8841 249.355L17.1648 260L58.4316 241.418L55.8632 235.031L12.8841 249.355Z"
              class="pips"/>
        </g>
        <g id="ice" class="hide">
          <g id="grill">
            <path
                d="M184.762 238.524C219.977 238.524 248.524 209.977 248.524 174.762C248.524 139.547 219.977 111 184.762 111C149.547 111 121 139.547 121 174.762C121 209.977 149.547 238.524 184.762 238.524Z"
                class="strokeBlack"
                stroke-miterlimit="10"/>
            <path
                d="M185.149 213.299C206.219 213.299 223.299 196.219 223.299 175.149C223.299 154.08 206.219 137 185.149 137C164.08 137 147 154.08 147 175.149C147 196.219 164.08 213.299 185.149 213.299Z"
                class="strokeBlack"
                stroke-miterlimit="10"/>
            <path
                d="M184.5 226C212.943 226 236 202.943 236 174.5C236 146.057 212.943 123 184.5 123C156.057 123 133 146.057 133 174.5C133 202.943 156.057 226 184.5 226Z"
                class="strokeBlack"
                stroke-miterlimit="10"/>
            <path d="M140 130L230.175 220.175" class="strokeBlack" stroke-miterlimit="10"/>
            <path d="M230.175 130L140 220.175" class="strokeBlack" stroke-miterlimit="10"/>
          </g>
          <g id="blades">
            <path
                d="M193.987 174L182.436 174C175.568 174 170 167.64 170 159.795L170 144.98C170 130.08 178.018 118 187.716 118C194.584 118 193.987 124.36 193.987 132.205L193.987 174Z"
                fill="white"
                class="strokeBlack"
                stroke-width="5"
                stroke-miterlimit="10"/>
            <path
                d="M174.013 174L185.564 174C192.432 174 198 180.36 198 188.205L198 203.02C198 217.92 189.982 230 180.284 230C173.416 230 174.013 223.64 174.013 215.795L174.013 174Z"
                fill="white"
                class="strokeBlack"
                stroke-width="5"
                stroke-miterlimit="10"/>
            <path
                d="M184 183.987L184 172.436C184 165.568 190.36 160 198.205 160L213.02 160C227.92 160 240 168.018 240 177.716C240 184.584 233.64 183.987 225.795 183.987L184 183.987Z"
                fill="white"
                class="strokeBlack"
                stroke-width="5"
                stroke-miterlimit="10"/>
            <path
                d="M184 164.013L184 175.564C184 182.432 177.64 188 169.795 188L154.98 188C140.08 188 128 179.982 128 170.284C128 163.416 134.36 164.013 142.205 164.013L184 164.013Z"
                fill="white"
                class="strokeBlack"
                stroke-width="5"
                stroke-miterlimit="10"/>
          </g>
          <circle cx="184" cy="174" r="16.5" fill="white" class="strokeBlack" stroke-width="5"/>
        </g>

        <g id="flame" class="hide">
          <path
              d="M183.591 242.031C156.113 241.414 145.772 206.232 147.055 186.134C149.5 146.782 174.951 122.789 195.096 115.971C195.5 115.971 187.237 139.161 194.72 152.093C200.963 162.88 220.05 172.797 220.05 195.792C220.05 216.531 207.839 241.934 183.634 242.031H183.589H183.591Z"
              class="strokeBlack"
              stroke-width="5"
              stroke-miterlimit="10">
            <animate
                dur="1.5s"
                attributeType="XML"
                attributeName="d"
                repeatCount="indefinite"
                values="
        M183.591 242.031C156.113 241.414 145.772 206.232 147.055 186.134C149.5 146.782 174.951 122.789 195.096 115.971C195.5 115.971 187.237 139.161 194.72 152.093C200.963 162.88 220.05 172.797 220.05 195.792C220.05 216.531 207.839 241.934 183.634 242.031H183.589H183.591Z;
        M182.589 241.537C155.496 240.926 145.301 206.266 146.563 186.463C146.455 156.118 182.48 134.768 175.056 102.463C174.968 102.079 197.21 118.089 193.68 146.266C190.939 168.167 218.525 173.32 218.525 195.98C218.525 216.414 206.487 241.438 182.618 241.537H182.579H182.589Z;
        M183.591 242.031C156.113 241.414 145.772 206.232 147.055 186.134C149.5 146.782 174.951 122.789 195.096 115.971C195.5 115.971 187.237 139.161 194.72 152.093C200.963 162.88 220.05 172.797 220.05 195.792C220.05 216.531 207.839 241.934 183.634 242.031H183.589H183.591Z"/>
          </path>
          <path
              d="M183.932 241.021C176.4 241.07 170.516 235.806 166.158 229.743C161.009 222.447 157.646 213.113 157.182 204.07C156.8 187.722 161.754 169.775 172.967 155.468C174.257 153.96 176.987 150.13 178.646 151.049C180.032 152.016 180.518 154.808 181.563 157.154C182.392 159.231 183.333 161.305 184.404 163.276C185.712 165.69 187.198 167.947 188.929 170.009C192.665 174.302 196.201 177.086 199.824 181.584C205.463 188.234 210.052 197.021 209.793 206.103C209.311 221.045 200.409 240.621 183.967 241.019H183.932V241.021Z"
              class="strokeBlack"
              stroke-width="5"
              stroke-miterlimit="10">
            <animate
                dur="1.5s"
                attributeType="XML"
                attributeName="d"
                repeatCount="indefinite"
                values="
        M183.932 241.021C176.4 241.07 170.516 235.806 166.158 229.743C161.009 222.447 157.646 213.113 157.182 204.07C156.8 187.722 161.754 169.775 172.967 155.468C174.257 153.96 176.987 150.13 178.646 151.049C180.032 152.016 180.518 154.808 181.563 157.154C182.392 159.231 183.333 161.305 184.404 163.276C185.712 165.69 187.198 167.947 188.929 170.009C192.665 174.302 196.201 177.086 199.824 181.584C205.463 188.234 210.052 197.021 209.793 206.103C209.311 221.045 200.409 240.621 183.967 241.019H183.932V241.021Z;
        M183.953 241C176.37 241.043 170.445 236.37 166.058 230.988C160.874 224.51 157.488 216.224
    157.021 208.196C156.636 193.682 161.624 177.749 172.913 165.048C174.212 163.709 176.96 160.309
    178.631 161.125C180.027 161.983 180.515 164.462 181.568 166.544C182.402 168.389 183.35 170.23
    184.428 171.98C185.745 174.123 187.241 176.127 188.984 177.957C192.745 181.768 196.306 184.24
    199.953 188.233C205.63 194.137 210.251 201.937 209.989 210C209.505 223.266 200.542 240.645
    183.988 240.998H183.953V241Z;
        M183.932 241.021C176.4 241.07 170.516 235.806 166.158 229.743C161.009 222.447 157.646 213.113 157.182 204.07C156.8 187.722 161.754 169.775 172.967 155.468C174.257 153.96 176.987 150.13 178.646 151.049C180.032 152.016 180.518 154.808 181.563 157.154C182.392 159.231 183.333 161.305 184.404 163.276C185.712 165.69 187.198 167.947 188.929 170.009C192.665 174.302 196.201 177.086 199.824 181.584C205.463 188.234 210.052 197.021 209.793 206.103C209.311 221.045 200.409 240.621 183.967 241.019H183.932V241.021Z"/>
          </path>
        </g>
      </svg>
    </div>
    <h2>Current Temperature</h2>
  </div>
</template>

<script setup>
import {format} from '@/util/number.js';
import {computed, onMounted, ref, watch} from 'vue';

const props = defineProps({
  temperature: {
    type: Number,
    required: true,
  },
  targetTemperature: {
    type: Number,
    required: false,
    default: 22,
  },
  energy: {
    type: Number,
    required: true,
  },
  totalRange: {
    type: Number,
    default: 3
  },
  runDemo: {
    type: Boolean,
    default: false,
    required: false,
  },
  showEnergy: {
    type: Boolean,
    default: false,
    required: false,
  },
});

let autoInterval;

const temperatureMeterSVG = ref(null);

const currentEnergy = ref(props.energy);
const currentTotalRange = ref(props.totalRange);

const temperatureStr = computed(() => {
  return format(props.temperature);
});

const currentEnergyStr = computed(() => {
  return format(currentEnergy.value, 'kWh');
});

let blades;
let ice;
let flame;
let pips;
let pipsReverse;
let wasHeating;

// Demo: generate random values
const autoValues = () =>
  setValue(
    Math.floor(Math.random() * (currentTotalRange.value * 2) + 1 - currentTotalRange.value),
    Math.floor(Math.random() * (currentTotalRange.value * 2) + 1 - currentTotalRange.value)
  );

wasHeating = false;

// Sleep helper for async delays
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

// Remove all pip classes
const removePipClasses = () => {
  pips.forEach((pip) => pip.classList.remove('onFlame', 'onIce'));
};

// Adjust turbine blade speed
const fanSpeed = (val) => {
  blades.classList.remove('medium', 'fast', 'fastest');
  if (val > 8) blades.classList.add('medium');
  if (val > 16) blades.classList.add('fast');
  if (val > 24) blades.classList.add('fastest');
};

// Auto-update the display
const startAuto = () => {
  if (props.runDemo) {
    clearInterval(autoInterval);
    autoInterval = setInterval(autoValues, 3000);
  }
};

// Update display and animations based on value
const setValue = async (temp, setPoint, cancelled) => {
  clearInterval(autoInterval);

  const isHeating = temp < setPoint;

  let pipVal = (Math.abs(temp - setPoint) / (currentTotalRange.value) * pips.length);

  if (isHeating) {
    // Using energy for heating
    flame.classList.remove('hide');
    ice.classList.add('hide');

    if (wasHeating === false) removePipClasses();

    for (let i = 0; i < pipsReverse.length; i++) {
      if (i < pipVal || i === 0) {
        pipsReverse[i].classList.add('onFlame');
      } else {
        pipsReverse[i].classList.remove('onFlame');
      }
      if (cancelled) return;
      await sleep(50);
    }
    if (cancelled) return;
    await sleep(50);

  } else {
    // Using energy for cooling
    flame.classList.add('hide');
    ice.classList.remove('hide');

    if (wasHeating === true) removePipClasses();

    fanSpeed(pipVal);

    for (let i = 0; i < pips.length; i++) {
      if (i < pipVal || i === 0) {
        pips[i].classList.add('onIce');
      } else {
        pips[i].classList.remove('onIce');
      }
      if (cancelled) return;
      await sleep(50);
    }
    if (cancelled) return;
    await sleep(50);

  }

  wasHeating = isHeating;
  startAuto();
};

onMounted(() => {
  blades = temperatureMeterSVG.value.getElementById('blades');
  ice = temperatureMeterSVG.value.getElementById('ice');
  flame = temperatureMeterSVG.value.getElementById('flame');

  pips = temperatureMeterSVG.value.querySelectorAll('.pips');
  pipsReverse = [...pips].reverse();

  // Start demo
  startAuto();
});

watch(() => props, (p, _, onCleanup) => {
  let cancelled = false;
  onCleanup(() => {
    cancelled = true;
  });
  setValue(p.temperature, p.targetTemperature, cancelled);
  currentEnergy.value = p.energy;
}, {deep: true});
</script>

<style lang="scss" scoped>
$grey: #c6c3c1;
$orange: #ff9800;
$blue: #00bed6;

.wrapper {
  container-name: svgWrapper;
  //container-type: inline-size;
  display: flex;
  position: relative;
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

.pips {
  fill: $grey;
  transition: 0.3s ease-in-out;

  &.onFlame {
    fill: $orange;
    &:not(.onFlame ~ .onFlame) {
      animation: flickerOrange 0.5s linear infinite;
      animation-delay: 0.5s;
    }
  }

  &.onIce {
    fill: $blue;

    &:has(+ :not(.onIce)) {
      animation: flickerBlue 0.5s linear infinite;
      animation-delay: 0.5s;
    }
  }
}

.strokeBlack {
  stroke: #000;
}

.hide {
  opacity: 0;
  transition: 0.5s ease-in-out;
}

#blades {
  transform-box: fill-box;
  transform-origin: center;
  animation: spin 2s linear infinite;
  transition: 1s ease-in-out;
}

#blades.med {
  animation-duration: 1s;
}

#blades.fast {
  animation-duration: 0.75s;
}

#blades.fastest {
  animation-duration: 0.5s;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(359deg);
  }
}

@keyframes flickerOrange {
  45% {
    fill: $orange;
  }

  55% {
    fill: $grey;
  }
}

@keyframes flickerBlue {
  45% {
    fill: $blue;
  }

  55% {
    fill: $grey;
  }
}
</style>
