<script setup>
import {onMounted, ref, watch} from 'vue';

const props = defineProps({
  occupants: {
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
});

let autoInterval;

const buildingLightsSVG = ref(null);

const currentOccupants = ref(props.occupants);
const currentMaxValue = ref(props.maxValue);
// Shuffle array using Fisher-Yates algorithm
const shuffleArray = (arr) => {
  const array = [...arr];
  for (let i = array.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[array[i], array[j]] = [array[j], array[i]];
  }
  return array;
};

// Sleep helper for async delays
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

let windows = [];
let randomWindows = [];
// Update the lights based on value
const setValue = async (value) => {
  clearInterval(autoInterval);

  currentOccupants.value = value;
  currentMaxValue.value =
    currentOccupants.value > currentMaxValue.value
      ? currentOccupants.value
      : currentMaxValue.value;

  const activeCount = Math.ceil((value / currentMaxValue.value) * windows.length);

  if (activeCount > 0) {
    for (let i = 0; i < randomWindows.length; i++) {
      randomWindows[i].classList.toggle('on', (i < activeCount || i === 0));
      await sleep(50);
    }
  }

  startAuto();
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
    autoInterval = setInterval(autoValues, 3000);
  }
};

onMounted(() => {
  windows = buildingLightsSVG.value.querySelectorAll('.light');
  randomWindows = shuffleArray(windows);
  // Initialize
  setValue(props.occupants);

  // Start demo
  startAuto();
});

watch(() => props.occupants, (newVal) => {
  setValue(newVal);
});

watch(() => props.maxValue, (newVal) => {
  currentMaxValue.value = newVal;
  setValue(currentOccupants.value);
});
</script>

<template>
  <div class="wrapper">
    <div class="svg">
      <div id="score">{{ currentOccupants }}</div>
      <svg
          id="buildingLightsSVG"
          width="376"
          height="285"
          viewBox="0 0 376 285"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          ref="buildingLightsSVG">
        <line x1="62.5" y1="2.5" x2="312.5" y2="2.5" class="blackStroke" stroke-width="5"/>
        <line x1="76" y1="5" x2="76" y2="50" class="blackStroke" stroke-width="5"/>
        <line x1="88.5" y1="5" x2="88.5" y2="50" class="greyStroke"/>
        <line x1="98.5" y1="5" x2="98.5" y2="50" class="greyStroke"/>
        <line x1="108.5" y1="5" x2="108.5" y2="50" class="greyStroke"/>
        <line x1="118.5" y1="5" x2="118.5" y2="50" class="greyStroke"/>
        <line x1="128.5" y1="5" x2="128.5" y2="50" class="greyStroke"/>
        <line x1="246.5" y1="5" x2="246.5" y2="50" class="greyStroke"/>
        <line x1="256.5" y1="5" x2="256.5" y2="50" class="greyStroke"/>
        <line x1="266.5" y1="5" x2="266.5" y2="50" class="greyStroke"/>
        <line x1="276.5" y1="5" x2="276.5" y2="50" class="greyStroke"/>
        <line x1="286.5" y1="5" x2="286.5" y2="50" class="greyStroke"/>
        <line x1="298.5" y1="5" x2="298.5" y2="50" class="blackStroke" stroke-width="5"/>
        <g class="windows">
          <path d="M12.5 60H32.5V80H12.5V60Z" class="light"/>
          <path d="M42.5 60H62.5V80H42.5V60Z" class="light"/>
          <path d="M72.5 60H92.5V80H72.5V60Z" class="light"/>
          <path d="M102.5 60H122.5V80H102.5V60Z" class="light"/>
          <path d="M132.5 60H152.5V80H132.5V60Z" class="light"/>
          <path d="M162.5 60H182.5V80H162.5V60Z" class="light"/>
          <path d="M192.5 60H212.5V80H192.5V60Z" class="light"/>
          <path d="M222.5 60H242.5V80H222.5V60Z" class="light"/>
          <path d="M252.5 60H272.5V80H252.5V60Z" class="light"/>
          <path d="M282.5 60H302.5V80H282.5V60Z" class="light"/>
          <path d="M312.5 60H332.5V80H312.5V60Z" class="light"/>
          <path d="M342.5 60H362.5V80H342.5V60Z" class="light"/>
          <path d="M12.5 95H32.5V115H12.5V95Z" class="light"/>
          <path d="M42.5 95H62.5V115H42.5V95Z" class="light"/>
          <path d="M72.5 95H92.5V115H72.5V95Z" class="light"/>
          <path d="M102.5 95H122.5V115H102.5V95Z" class="light"/>
          <path d="M132.5 95H152.5V115H132.5V95Z" class="light"/>
          <path d="M162.5 95H182.5V115H162.5V95Z" class="light"/>
          <path d="M192.5 95H212.5V115H192.5V95Z" class="light"/>
          <path d="M222.5 95H242.5V115H222.5V95Z" class="light"/>
          <path d="M252.5 95H272.5V115H252.5V95Z" class="light"/>
          <path d="M282.5 95H302.5V115H282.5V95Z" class="light"/>
          <path d="M312.5 95H332.5V115H312.5V95Z" class="light"/>
          <path d="M342.5 95H362.5V115H342.5V95Z" class="light"/>
          <path d="M12.5 130H32.5V150H12.5V130Z" class="light"/>
          <path d="M42.5 130H62.5V150H42.5V130Z" class="light"/>
          <path d="M72.5 130H92.5V150H72.5V130Z" class="light"/>
          <path d="M102.5 130H122.5V150H102.5V130Z" class="light"/>
          <path d="M132.5 130H152.5V150H132.5V130Z" class="light"/>
          <path d="M162.5 130H182.5V150H162.5V130Z" class="light"/>
          <path d="M192.5 130H212.5V150H192.5V130Z" class="light"/>
          <path d="M222.5 130H242.5V150H222.5V130Z" class="light"/>
          <path d="M252.5 130H272.5V150H252.5V130Z" class="light"/>
          <path d="M282.5 130H302.5V150H282.5V130Z" class="light"/>
          <path d="M312.5 130H332.5V150H312.5V130Z" class="light"/>
          <path d="M342.5 130H362.5V150H342.5V130Z" class="light"/>
          <path d="M12.5 165H32.5V185H12.5V165Z" class="light"/>
          <path d="M42.5 165H62.5V185H42.5V165Z" class="light"/>
          <path d="M72.5 165H92.5V185H72.5V165Z" class="light"/>
          <path d="M102.5 165H122.5V185H102.5V165Z" class="light"/>
          <path d="M132.5 165H152.5V185H132.5V165Z" class="light"/>
          <path d="M162.5 165H182.5V185H162.5V165Z" class="light"/>
          <path d="M192.5 165H212.5V185H192.5V165Z" class="light"/>
          <path d="M222.5 165H242.5V185H222.5V165Z" class="light"/>
          <path d="M252.5 165H272.5V185H252.5V165Z" class="light"/>
          <path d="M282.5 165H302.5V185H282.5V165Z" class="light"/>
          <path d="M312.5 165H332.5V185H312.5V165Z" class="light"/>
          <path d="M342.5 165H362.5V185H342.5V165Z" class="light"/>
          <path d="M12.5 200H32.5V220H12.5V200Z" class="light"/>
          <path d="M42.5 200H62.5V220H42.5V200Z" class="light"/>
          <path d="M72.5 200H92.5V220H72.5V200Z" class="light"/>
          <path d="M102.5 200H122.5V220H102.5V200Z" class="light"/>
          <path d="M132.5 200H152.5V220H132.5V200Z" class="light"/>
          <path d="M162.5 200H182.5V220H162.5V200Z" class="light"/>
          <path d="M192.5 200H212.5V220H192.5V200Z" class="light"/>
          <path d="M222.5 200H242.5V220H222.5V200Z" class="light"/>
          <path d="M252.5 200H272.5V220H252.5V200Z" class="light"/>
          <path d="M282.5 200H302.5V220H282.5V200Z" class="light"/>
          <path d="M312.5 200H332.5V220H312.5V200Z" class="light"/>
          <path d="M342.5 200H362.5V220H342.5V200Z" class="light"/>
        </g>
        <path
            class="tree black"
            d="M57.9199 253.75L61.6699 257.891C62.4512 258.672 62.6074 259.922 62.1387 260.938C61.6699 261.875 60.6543 262.5 59.4824 262.5H48.7012V268.75C48.7012 269.453 48.0762 270 47.4512 270C46.8262 270 46.2012 269.453 46.2012 268.75V262.5H35.4199C34.248 262.5 33.2324 261.875 32.7637 260.938C32.2949 259.922 32.4512 258.75 33.2324 257.891L37.0605 253.75C35.9668 253.672 35.1074 253.047 34.6387 252.031C34.1699 250.938 34.3262 249.766 35.0293 248.828L39.7168 243.75C38.8574 243.516 38.0762 242.891 37.6855 242.031C37.2949 241.094 37.4512 239.844 38.2324 239.062L45.7324 230.781C46.6699 229.844 48.3887 229.844 49.248 230.781L56.748 239.062C57.5293 239.844 57.6855 241.094 57.2168 242.109C56.9043 242.969 56.123 243.516 55.2637 243.75L59.7949 248.828C60.5762 249.766 60.7324 250.938 60.2637 252.031C59.873 253.047 58.9355 253.672 57.9199 253.75ZM59.5605 260C59.7949 260 60.1074 259.766 59.9512 259.609L52.2168 251.25H57.6855C58.0762 251.25 58.2324 250.781 57.998 250.391L49.6387 241.25H54.6387C54.9512 241.25 55.1074 240.938 54.873 240.703L47.373 232.5L40.1074 240.703C39.873 240.938 40.0293 241.25 40.3418 241.25H45.3418L36.9043 250.469C36.748 250.703 36.9043 251.25 37.2949 251.25H42.7637L35.0293 259.609C34.873 259.766 35.1074 260 35.4199 260H46.2793V254.219L42.7637 249.531C42.2949 248.984 42.4512 248.203 42.998 247.812C43.5449 247.344 44.3262 247.5 44.7168 248.047L46.2793 250.078V243.75C46.2793 243.125 46.8262 242.5 47.5293 242.5C48.1543 242.5 48.7793 243.125 48.7793 243.75V254.531L50.3418 252.891C50.8105 252.422 51.6699 252.422 52.1387 252.891C52.6074 253.359 52.6074 254.219 52.1387 254.688L48.7793 258.047V260H59.5605ZM81.748 257.891C82.5293 258.672 82.6855 259.922 82.2168 260.938C81.748 261.875 80.7324 262.5 79.5605 262.5H68.7793V268.75C68.7793 269.453 68.0762 270 67.5293 270C66.9043 270 66.2793 269.453 66.2793 268.75V254.219L62.8418 249.531C62.373 248.984 62.5293 248.203 63.0762 247.812C63.623 247.344 64.4043 247.5 64.7949 248.047L66.2793 250.078V243.75C66.2793 243.125 66.8262 242.5 67.5293 242.5C68.1543 242.5 68.7793 243.125 68.7793 243.75V254.531L70.3418 252.891C70.8105 252.422 71.6699 252.422 72.1387 252.891C72.6074 253.359 72.6074 254.219 72.1387 254.688L68.7793 258.047V260H79.5605C79.7949 260 80.1074 259.766 79.9512 259.609L72.2168 251.25H77.6855C78.0762 251.25 78.2324 250.781 77.998 250.391L69.6387 241.25H74.6387C74.9512 241.25 75.1074 240.938 74.873 240.781L67.373 232.5L62.2168 238.359C61.748 238.906 60.9668 238.906 60.4199 238.438C59.9512 237.969 59.873 237.188 60.3418 236.719L65.7324 230.781C66.6699 229.844 68.3105 229.844 69.248 230.781L76.748 239.062C77.5293 239.844 77.6855 241.094 77.2168 242.109C76.9043 242.969 76.123 243.516 75.2637 243.75L79.873 248.828C80.6543 249.766 80.8105 250.938 80.3418 252.031C79.873 253.047 79.0137 253.672 77.998 253.75L81.748 257.891Z"/>
        <path
            class="bench black"
            d="M97.168 255.938H110.293V254.062H97.168V255.938ZM94.3555 253.125C94.3555 252.129 95.1758 251.25 96.2305 251.25H111.23C112.227 251.25 113.105 252.129 113.105 253.125V256.875C113.105 257.754 112.461 258.516 111.699 258.691V260.625H113.574C114.336 260.625 114.98 261.27 114.98 262.031C114.98 262.852 114.336 263.438 113.574 263.438H113.105V268.594C113.105 269.414 112.461 270 111.699 270C110.879 270 110.293 269.414 110.293 268.594V263.438H97.168V268.594C97.168 269.414 96.5234 270 95.7617 270C94.9414 270 94.3555 269.414 94.3555 268.594V263.438H93.8867C93.0664 263.438 92.4805 262.852 92.4805 262.031C92.4805 261.27 93.0664 260.625 93.8867 260.625H95.7617V258.691C94.9414 258.516 94.3555 257.754 94.3555 256.875V253.125ZM98.5742 258.75V260.625H108.887V258.75H98.5742Z"/>
        <path class="door darkGrey" d="M162.5 230H187V270H162.5V230Z"/>
        <path class="door darkGrey" d="M188 230H212.5V270H188V230Z"/>
        <path
            class="bench black"
            d="M264.707 255.938H277.832V254.062H264.707V255.938ZM261.895 253.125C261.895 252.129 262.715 251.25 263.77 251.25H278.77C279.766 251.25 280.645 252.129 280.645 253.125V256.875C280.645 257.754 280 258.516 279.238 258.691V260.625H281.113C281.875 260.625 282.52 261.27 282.52 262.031C282.52 262.852 281.875 263.438 281.113 263.438H280.645V268.594C280.645 269.414 280 270 279.238 270C278.418 270 277.832 269.414 277.832 268.594V263.438H264.707V268.594C264.707 269.414 264.062 270 263.301 270C262.48 270 261.895 269.414 261.895 268.594V263.438H261.426C260.605 263.438 260.02 262.852 260.02 262.031C260.02 261.27 260.605 260.625 261.426 260.625H263.301V258.691C262.48 258.516 261.895 257.754 261.895 256.875V253.125ZM266.113 258.75V260.625H276.426V258.75H266.113Z"
            fill="black"/>
        <path
            class="tree black"
            d="M317.08 253.75L313.33 257.891C312.549 258.672 312.393 259.922 312.861 260.938C313.33 261.875 314.346 262.5 315.518 262.5H326.299V268.75C326.299 269.453 326.924 270 327.549 270C328.174 270 328.799 269.453 328.799 268.75V262.5H339.58C340.752 262.5 341.768 261.875 342.236 260.938C342.705 259.922 342.549 258.75 341.768 257.891L337.939 253.75C339.033 253.672 339.893 253.047 340.361 252.031C340.83 250.938 340.674 249.766 339.971 248.828L335.283 243.75C336.143 243.516 336.924 242.891 337.314 242.031C337.705 241.094 337.549 239.844 336.768 239.062L329.268 230.781C328.33 229.844 326.611 229.844 325.752 230.781L318.252 239.062C317.471 239.844 317.314 241.094 317.783 242.109C318.096 242.969 318.877 243.516 319.736 243.75L315.205 248.828C314.424 249.766 314.268 250.938 314.736 252.031C315.127 253.047 316.064 253.672 317.08 253.75ZM315.439 260C315.205 260 314.893 259.766 315.049 259.609L322.783 251.25H317.314C316.924 251.25 316.768 250.781 317.002 250.391L325.361 241.25H320.361C320.049 241.25 319.893 240.938 320.127 240.703L327.627 232.5L334.893 240.703C335.127 240.938 334.971 241.25 334.658 241.25H329.658L338.096 250.469C338.252 250.703 338.096 251.25 337.705 251.25H332.236L339.971 259.609C340.127 259.766 339.893 260 339.58 260H328.721V254.219L332.236 249.531C332.705 248.984 332.549 248.203 332.002 247.812C331.455 247.344 330.674 247.5 330.283 248.047L328.721 250.078V243.75C328.721 243.125 328.174 242.5 327.471 242.5C326.846 242.5 326.221 243.125 326.221 243.75V254.531L324.658 252.891C324.189 252.422 323.33 252.422 322.861 252.891C322.393 253.359 322.393 254.219 322.861 254.688L326.221 258.047V260H315.439ZM293.252 257.891C292.471 258.672 292.314 259.922 292.783 260.938C293.252 261.875 294.268 262.5 295.439 262.5H306.221V268.75C306.221 269.453 306.924 270 307.471 270C308.096 270 308.721 269.453 308.721 268.75V254.219L312.158 249.531C312.627 248.984 312.471 248.203 311.924 247.812C311.377 247.344 310.596 247.5 310.205 248.047L308.721 250.078V243.75C308.721 243.125 308.174 242.5 307.471 242.5C306.846 242.5 306.221 243.125 306.221 243.75V254.531L304.658 252.891C304.189 252.422 303.33 252.422 302.861 252.891C302.393 253.359 302.393 254.219 302.861 254.688L306.221 258.047V260H295.439C295.205 260 294.893 259.766 295.049 259.609L302.783 251.25H297.314C296.924 251.25 296.768 250.781 297.002 250.391L305.361 241.25H300.361C300.049 241.25 299.893 240.938 300.127 240.781L307.627 232.5L312.783 238.359C313.252 238.906 314.033 238.906 314.58 238.438C315.049 237.969 315.127 237.188 314.658 236.719L309.268 230.781C308.33 229.844 306.689 229.844 305.752 230.781L298.252 239.062C297.471 239.844 297.314 241.094 297.783 242.109C298.096 242.969 298.877 243.516 299.736 243.75L295.127 248.828C294.346 249.766 294.189 250.938 294.658 252.031C295.127 253.047 295.986 253.672 297.002 253.75L293.252 257.891Z"
            fill="black"/>
        <path class="building blackStroke" d="M373 270V50H3V270" stroke-width="5"/>
      </svg>
    </div>
    <h2>Occupancy</h2>
  </div>
</template>

<style lang="scss" scoped>
$grey: #c6c3c1;
$yellow: #ff9800;
$darkGrey: #4a4948;

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
    aspect-ratio: 376 / 285;

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
  top: 2%;
  font-family: sans-serif;
  font-weight: 600;
  font-size: 40px;
  text-align: center;
  line-height: 100%;
}

@container svgWrapper (width < 400px) {
  #score {
    font-size: clamp(0.75rem, 0.0921rem + 10.5263cqi, 2.5rem);
  }

  h2 {
    font-size: clamp(0.375rem, -0.0479rem + 6.7669cqi, 1.5rem);
  }
}

.light,
.grey {
  fill: $grey;
}

.greyStroke {
  stroke: $grey;
}

.yellow,
.on {
  fill: $yellow;
}

.darkGrey {
  fill: $darkGrey;
}

.black {
  fill: #000;
}

.blackStroke {
  stroke: #000;
}

.light {
  transition: 0.1s ease-in-out;
}
</style>
