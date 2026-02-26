<template>
  <div class="wrapper">
    <div class="svg">
      <div id="score">
        <em>{{ currentNetConsumptionStr }}</em>kWh
        <template v-if="props.showEnergy">
          <span>{{ 'D ' + currentDemand + ' / G ' + currentGenerated }}</span>
        </template>
      </div>
      <svg
          id="energyMeterSVG"
          width="366"
          height="285"
          viewBox="0 0 366 285"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          ref="energyMeterSVG">
        <g id="pipsCrescent">
          <g id="generating">
            <path
                d="M308.05 241.42L327.39 250.13C328.64 247.39 329.81 244.61 330.9 241.79L310.62 235.03L308.05 241.42Z"
                class="fillGray pip"/>
            <path
                d="M314.16 224.7L334.52 231.41C335.41 228.54 336.21 225.63 336.93 222.69L316.03 218.08L314.16 224.7Z"
                class="fillGray pip"/>
            <path
                d="M318.58 206.29L339.44 210.41C339.92 207.46 340.33 204.48 340.64 201.48L319.59 199.5L318.57 206.3L318.58 206.29Z"
                class="fillGray pip"/>
            <path
                d="M341.45 181.21L320.63 181.87L320.5 188.74L341.4 190.19C341.46 188.47 341.49 186.74 341.49 185C341.49 183.73 341.47 182.47 341.44 181.21H341.45Z"
                class="fillGray pip"/>
            <path
                d="M320.07 170.56L340.71 169.18C340.41 166.19 340.02 163.23 339.56 160.3L319.25 163.75L320.08 170.57L320.07 170.56Z"
                class="fillGray pip"/>
            <path
                d="M317.22 153.14L337.36 149.04C336.68 146.12 335.9 143.24 335.06 140.39L315.47 146.5L317.23 153.15L317.22 153.14Z"
                class="fillGray pip"/>
            <path
                d="M311.91 136.02L331.3 129.25C330.24 126.45 329.1 123.68 327.88 120.96L309.25 129.67L311.91 136.02Z"
                class="fillGray pip"/>
            <path
                d="M304.33 119.8L322.75 110.54C321.33 107.9 319.83 105.3 318.26 102.75L300.85 113.85L304.33 119.8Z"
                class="fillGray pip"/>
            <path
                d="M294.5 104.44L311.64 92.8399C309.88 90.3999 308.04 88.0199 306.15 85.6899L290.27 98.9899L294.5 104.45V104.44Z"
                class="fillGray pip"/>
            <path
                d="M282.75 90.4299L298.32 76.7799C296.25 74.5899 294.11 72.4699 291.92 70.3999L277.85 85.5399L282.74 90.4299H282.75Z"
                class="fillGray pip"/>
            <path
                d="M269.3 77.9401L283 62.6001C280.66 60.7001 278.27 58.8701 275.82 57.1101L263.84 73.6701L269.3 77.9401Z"
                class="fillGray pip"/>
            <path
                d="M254.39 67.14L265.85 50.49C263.29 48.92 260.67 47.44 258.02 46.02L248.43 63.58L254.39 67.14Z"
                class="fillGray pip"/>
            <path
                d="M238.29 58.2699L247.36 40.8199C244.63 39.6099 241.86 38.4799 239.04 37.4299L231.93 55.4499L238.29 58.2599V58.2699Z"
                class="fillGray pip"/>
            <path
                d="M221.31 51.52L227.75 33.6899C224.9 32.8699 222.03 32.12 219.12 31.45L214.63 49.5499L221.31 51.52Z"
                class="fillGray pip"/>
            <path
                d="M203.82 47.0999L207.48 29.2499C204.58 28.8099 201.64 28.4599 198.69 28.1899L196.93 46.0899L203.83 47.0999H203.82Z"
                class="fillGray pip"/>
            <path
                d="M187.52 45.61L188.42 27.56C186.95 27.52 185.48 27.49 184 27.49C182.52 27.49 181.18 27.51 179.77 27.55L180.67 45.61H187.51H187.52Z"
                class="fillGray pip"/>
            <path
                d="M169.56 46.0901L167.81 28.3301C164.87 28.6301 161.95 29.0201 159.05 29.4901L162.66 47.1101L169.56 46.1001V46.0901Z"
                class="fillGray pip"/>
            <path
                d="M151.85 49.55L147.45 31.79C144.56 32.48 141.7 33.25 138.88 34.1L145.18 51.52L151.86 49.55H151.85Z"
                class="fillGray pip"/>
            <path
                d="M134.55 55.4599L127.64 37.9399C124.86 39.0099 122.11 40.1499 119.41 41.3699L128.2 58.2799L134.56 55.4699L134.55 55.4599Z"
                class="fillGray pip"/>
            <path
                d="M118.05 63.5799L108.8 46.6399C106.18 48.0699 103.61 49.5799 101.08 51.1499L112.09 67.1399L118.05 63.5799Z"
                class="fillGray pip"/>
            <path
                d="M102.64 73.6701L91.1901 57.8301C88.7801 59.5901 86.4201 61.4101 84.1201 63.3101L97.1801 77.9401L102.64 73.6701Z"
                class="fillGray pip"/>
            <path
                d="M88.63 85.5399L75.27 71.1599C73.12 73.2199 71.03 75.3399 69 77.5099L83.74 90.4299L88.63 85.5399Z"
                class="fillGray pip"/>
            <path
                d="M76.2201 98.9799L61.2401 86.4399C59.3801 88.7499 57.5801 91.1099 55.8501 93.5199L71.9801 104.43L76.2101 98.9699L76.2201 98.9799Z"
                class="fillGray pip"/>
            <path
                d="M65.6299 113.85L49.2999 103.45C47.7699 105.97 46.3099 108.54 44.9199 111.15L62.1399 119.8L65.6199 113.85H65.6299Z"
                class="fillGray pip"/>
            <path
                d="M57.23 129.67L39.86 121.55C38.67 124.24 37.56 126.96 36.52 129.72L54.57 136.02L57.23 129.67Z"
                class="fillGray pip"/>
            <path
                d="M51.0201 146.49L32.8101 140.82C31.9901 143.63 31.2301 146.46 30.5701 149.33L49.2601 153.14L51.0201 146.49Z"
                class="fillGray pip"/>
            <path
                d="M47.24 163.74L28.4 160.54C27.95 163.43 27.57 166.34 27.28 169.27L46.41 170.55L47.24 163.73V163.74Z"
                class="fillGray pip"/>
            <path
                d="M45.85 181.88L26.55 181.26C26.52 182.5 26.5 183.75 26.5 185C26.5 186.7 26.54 188.4 26.59 190.09L45.98 188.75L45.85 181.88Z"
                class="fillGray pip"/>
            <path
                d="M46.8901 199.49L27.3501 201.33C27.6601 204.28 28.0501 207.21 28.5201 210.12L47.9101 206.29L46.8901 199.49Z"
                class="fillGray pip"/>
            <path
                d="M50.45 218.09L30.99 222.38C31.7 225.27 32.48 228.13 33.34 230.96L52.32 224.71L50.45 218.09Z"
                class="fillGray pip"/>
            <path
                d="M55.8599 235.03L36.9199 241.34C37.9899 244.12 39.1199 246.86 40.3399 249.56L58.4199 241.42L55.8499 235.03H55.8599Z"
                class="fillGray pip"/>
          </g>
          <g id="consuming">
            <path
                d="M349.32 260L353.6 249.35L335.65 243.37C334.5 246.35 333.27 249.29 331.95 252.18L349.32 260Z"
                class="fillGray pip"/>
            <path
                d="M357.19 238.88L360.3 227.85L341.81 223.77C341.05 226.87 340.2 229.94 339.27 232.98L357.19 238.88Z"
                class="fillGray pip"/>
            <path
                d="M363.06 215.07L364.76 203.74L345.62 201.94C345.29 205.11 344.86 208.26 344.35 211.37L363.07 215.06L363.06 215.07Z"
                class="fillGray pip"/>
            <path
                d="M346.4 190.54L365.77 191.88L365.99 180.43L346.46 181.05C346.49 182.36 346.51 183.68 346.51 185C346.51 186.85 346.47 188.7 346.41 190.54H346.4Z"
                class="fillGray pip"/>
            <path
                d="M365.34 167.52L363.96 156.15L344.49 159.45C344.98 162.55 345.39 165.68 345.7 168.83L365.34 167.51V167.52Z"
                class="fillGray pip"/>
            <path
                d="M361.65 144.09L358.72 133.01L339.83 138.9C340.72 141.91 341.54 144.96 342.26 148.04L361.65 144.09Z"
                class="fillGray pip"/>
            <path
                d="M354.68 121.1L350.25 110.52L332.42 118.85C333.71 121.73 334.91 124.65 336.03 127.61L354.69 121.1H354.68Z"
                class="fillGray pip"/>
            <path
                d="M344.75 99.4901L338.95 89.5701L322.48 100.07C324.14 102.76 325.72 105.51 327.22 108.3L344.75 99.4901Z"
                class="fillGray pip"/>
            <path
                d="M331.9 79.13L324.84 70.03L309.98 82.47C311.99 84.93 313.92 87.45 315.78 90.03L331.9 79.12V79.13Z"
                class="fillGray pip"/>
            <path
                d="M302.08 73.4801L316.65 60.7101L308.49 52.5601L295.32 66.7401C297.64 68.9201 299.89 71.1701 302.08 73.4901V73.4801Z"
                class="fillGray pip"/>
            <path
                d="M299.27 44.37L290.17 37.26L278.75 53.06C281.34 54.92 283.87 56.86 286.34 58.87L299.28 44.37H299.27Z"
                class="fillGray pip"/>
            <path
                d="M279.86 30.1299L269.92 24.2L260.41 41.6199C263.22 43.1199 265.98 44.7 268.69 46.36L279.86 30.1299Z"
                class="fillGray pip"/>
            <path
                d="M258.98 18.48L248.38 13.79L240.88 32.79C243.85 33.9 246.78 35.1 249.67 36.38L258.98 18.48Z"
                class="fillGray pip"/>
            <path
                d="M236.53 9.40012L225.4 6.12012L220.32 26.6001C223.4 27.3101 226.44 28.1001 229.45 28.9801L236.53 9.40012Z"
                class="fillGray pip"/>
            <path
                d="M212.79 3.27008L201.3 1.58008L199.17 23.2101C202.3 23.5001 205.4 23.8701 208.48 24.3401L212.8 3.27008H212.79Z"
                class="fillGray pip"/>
            <path
                d="M188.67 22.57L189.8 0H178.4L179.53 22.56C181.02 22.52 182.51 22.5 184.01 22.5C185.51 22.5 187.13 22.53 188.68 22.57H188.67Z"
                class="fillGray pip"/>
            <path
                d="M165.18 1.59009L153.69 3.28009L158.05 24.5901C161.11 24.1001 164.2 23.6801 167.32 23.3601L165.18 1.60009V1.59009Z"
                class="fillGray pip"/>
            <path
                d="M141.08 6.11011L129.95 9.39011L137.17 29.3801C140.16 28.4801 143.18 27.6601 146.24 26.9301L141.08 6.11011Z"
                class="fillGray pip"/>
            <path
                d="M118.1 13.78L107.5 18.47L117.09 36.92C119.95 35.62 122.85 34.42 125.8 33.28L118.1 13.78Z"
                class="fillGray pip"/>
            <path
                d="M96.5601 24.1999L86.6201 30.1299L98.2501 47.0199C100.92 45.3499 103.64 43.7599 106.41 42.2399L96.5501 24.1899L96.5601 24.1999Z"
                class="fillGray pip"/>
            <path
                d="M76.32 37.25L67.22 44.36L80.8 59.57C83.23 57.57 85.73 55.64 88.28 53.77L76.33 37.24L76.32 37.25Z"
                class="fillGray pip"/>
            <path
                d="M71.8701 67.49L57.9901 52.55L49.8301 60.7001L65.2301 74.2C67.3801 71.9 69.5801 69.6501 71.8601 67.4801L71.8701 67.49Z"
                class="fillGray pip"/>
            <path
                d="M41.6401 70.04L34.5801 79.14L51.7201 90.73C53.5401 88.18 55.4501 85.68 57.4101 83.24L41.6401 70.04Z"
                class="fillGray pip"/>
            <path
                d="M27.53 89.5701L21.73 99.4901L40.46 108.9C41.93 106.14 43.47 103.42 45.09 100.76L27.53 89.5701Z"
                class="fillGray pip"/>
            <path
                d="M16.2401 110.51L11.8101 121.09L31.8101 128.07C32.9101 125.15 34.0801 122.27 35.3401 119.43L16.2501 110.51H16.2401Z"
                class="fillGray pip"/>
            <path
                d="M7.76008 133.01L4.83008 144.09L25.6701 148.33C26.3701 145.29 27.1701 142.29 28.0401 139.33L7.76008 133.01Z"
                class="fillGray pip"/>
            <path
                d="M2.5299 156.16L1.1499 167.53L22.2999 168.95C22.6099 165.85 22.9999 162.77 23.4799 159.72L2.5399 156.17L2.5299 156.16Z"
                class="fillGray pip"/>
            <path
                d="M21.55 181.1L0.5 180.43L0.72 191.88L21.6 190.44C21.54 188.64 21.5 186.83 21.5 185.01C21.5 183.71 21.52 182.41 21.55 181.11V181.1Z"
                class="fillGray pip"/>
            <path
                d="M1.71997 203.74L3.41997 215.07L23.61 211.09C23.11 208.02 22.69 204.93 22.37 201.8L1.72997 203.74H1.71997Z"
                class="fillGray pip"/>
            <path
                d="M6.17992 227.85L9.28992 238.88L28.5899 232.52C27.6699 229.53 26.8499 226.51 26.0999 223.45L6.16992 227.85H6.17992Z"
                class="fillGray pip"/>
            <path
                d="M12.8801 249.35L17.1601 260L35.7801 251.61C34.4901 248.76 33.2901 245.86 32.1701 242.92L12.8701 249.35H12.8801Z"
                class="fillGray pip"/>
          </g>
        </g>

        <g id="pylon">

          <path d="M169.881 242L130.452 185.403" class="strokeBlack" stroke-width="5"
                stroke-miterlimit="10"
                stroke-linecap="round"/>
          <path d="M157.54 126.698H137.044L113 133.532H181L157.54 126.698Z" class="strokeBlack"
                stroke-width="5" stroke-linecap="square" stroke-linejoin="round"/>
          <path d="M129.669 94.5982V92H163.886V94.5982" class="strokeBlack" stroke-width="5"
                stroke-linecap="round" stroke-linejoin="round"/>
          <path d="M161.472 163.583H132.882L113 170.417H181L161.472 163.583Z" class="strokeBlack"
                stroke-width="5" stroke-linecap="square" stroke-linejoin="round"/>
          <path d="M169.853 242L153.867 92H146.999H140.902L124.147 242" class="strokeBlack"
                stroke-width="5"
                stroke-miterlimit="10" stroke-linecap="round"/>
          <path d="M124.177 242L163.609 185.403" class="strokeBlack" stroke-width="5"
                stroke-miterlimit="10"
                stroke-linecap="round"/>
          <path d="M113 133.532V138.525" class="strokeBlack" stroke-width="5" stroke-linecap="round"
                stroke-linejoin="round"/>
          <path d="M181 133.532V138.525" class="strokeBlack" stroke-width="5" stroke-linecap="round"
                stroke-linejoin="round"/>
          <path d="M113 170.417V175.41" class="strokeBlack" stroke-width="5" stroke-linecap="round"
                stroke-linejoin="round"/>
          <path d="M181 170.417V175.41" class="strokeBlack" stroke-width="5" stroke-linecap="round"
                stroke-linejoin="round"/>

        </g>
        <g id="turbine">
          <path
              d="M208.523 143.418L208.535 143.905L210.996 240.367L210.997 240.38H210.497L210.997
    240.382V240.383C210.997 240.384 210.996 240.385 210.996 240.386V240.411C210.996 240.423 210.994
    240.437 210.992 240.453C210.989 240.484 210.983 240.523 210.973 240.566C210.952 240.653 210.914
    240.759 210.848 240.877C210.715 241.115 210.478 241.378 210.073 241.619C209.279 242.092 207.811
    242.5 205.059 242.5C202.307 242.5 200.84 242.092 200.045 241.619C199.64 241.378 199.403 241.115
    199.27 240.877C199.204 240.759 199.167 240.653 199.146 240.566C199.135 240.523 199.13 240.484
    199.126 240.453C199.124 240.437 199.123 240.423 199.122 240.411C199.122 240.405 199.121 240.399
    199.121 240.394V240.382C199.121 240.381 199.121 240.38 199.621 240.38H199.121V240.367L201.499
    143.905L201.511 143.418H208.523Z"
              class="strokeWhite fillBlack" stroke-miterlimit="10"/>
          <g id="blades" class="stop">
            <path
                stroke="red"
                stroke-width="0"
                d="M205.561 70.5C246.394 70.5 279.498 103.631 279.498 144.5C279.498 185.369 246.394 218.5 205.561 218.5C164.727 218.5 131.624 185.369 131.624 144.5C131.624 103.631 164.727 70.5001 205.561 70.5Z"/>
            <path
                d="M209.247 70.7871V141.06H201.412V140.133C201.306 140.012 201.184 139.869 201.048 139.705C200.593 139.154 199.982 138.364 199.349 137.398C198.101 135.494 196.69 132.786 196.338 129.818L196.253 129.023C195.458 120.63 197.061 106.611 198.845 94.707C199.772 88.5236 200.759 82.8463 201.513 78.7158C201.89 76.6505 202.21 74.9708 202.435 73.8066C202.548 73.2245 202.637 72.7711 202.698 72.4629C202.728 72.3088 202.752 72.1909 202.768 72.1113C202.776 72.0719 202.782 72.0418 202.786 72.0215C202.788 72.0114 202.789 72.0033 202.791 71.998C202.791 71.9956 202.791 71.9935 202.792 71.9922L202.792 71.9902L203.036 70.7871H209.247Z"
                class="strokeWhite fillBlack" stroke-width="3" stroke-miterlimit="10"/>
            <path
                d="M201.524 143.167L202.963 145.661L205.711 142.915L204.084 148.52L202.643 148.102C204.084 148.52 204.084 148.521 204.084 148.521V148.522L204.083 148.524C204.082 148.526 204.081 148.527 204.081 148.53C204.079 148.535 204.078 148.541 204.076 148.549C204.071 148.565 204.065 148.587 204.056 148.614C204.04 148.668 204.016 148.744 203.984 148.84C203.921 149.033 203.827 149.307 203.702 149.642C203.452 150.312 203.073 151.237 202.553 152.267C201.528 154.3 199.887 156.876 197.492 158.665C190.81 163.656 177.389 169.5 165.823 174.05C160.003 176.34 154.592 178.325 150.636 179.737C148.658 180.443 147.044 181.007 145.923 181.394C145.362 181.588 144.924 181.737 144.627 181.838C144.478 181.889 144.365 181.927 144.288 181.953C144.249 181.966 144.22 181.976 144.2 181.983C144.19 181.986 144.182 181.988 144.177 181.99C144.175 181.991 144.173 181.991 144.172 181.992L144.171 181.993C144.17 181.993 144.17 181.993 143.692 180.571L144.17 181.993L143.007 182.383L142.393 181.321L140.65 178.303L139.899 177.003L141.199 176.253L199.475 142.619L200.774 141.869L201.524 143.167Z"
                class="strokeWhite fillBlack" stroke-width="3" stroke-miterlimit="10"/>
            <path
                d="M213.816 142.458C216.09 142.329 219.141 142.461 221.887 143.638C229.552 146.928 241.326 155.626 251.05 163.365C255.943 167.259 260.368 170.951 263.569 173.67C265.169 175.029 266.465 176.145 267.361 176.923C267.809 177.311 268.157 177.615 268.393 177.822C268.511 177.925 268.602 178.005 268.663 178.058C268.693 178.085 268.716 178.105 268.732 178.119C268.739 178.126 268.746 178.131 268.75 178.135L268.755 178.14C268.752 178.145 268.692 178.213 267.764 179.265L268.756 178.14L269.676 178.953L269.063 180.015L267.32 183.034L266.57 184.332L265.271 183.583L206.995 149.947L205.696 149.198L206.446 147.898L208.862 143.712L209.295 142.962H210C210.039 142.953 210.083 142.943 210.133 142.932C210.332 142.891 210.615 142.835 210.968 142.775C211.673 142.657 212.663 142.523 213.816 142.458Z"
                class="strokeWhite fillBlack" stroke-width="3" stroke-miterlimit="10"/>
          </g>
          <path
              d="M205.371 136.291C209.583 136.291 213 139.705 213 143.917C213 148.129 209.583 151.544 205.371 151.544C201.158 151.544 197.743 148.129 197.743 143.917C197.743 139.705 201.159 136.291 205.371 136.291Z"
              class="strokeWhite fillBlack" stroke-width="3" stroke-miterlimit="10"/>
        </g>
      </svg>
    </div>
    <h2>Net Energy Consumption</h2>
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
import {format} from '@/util/number.js';
import {sentenceCase} from 'change-case';
import {computed, onMounted, ref, watch} from 'vue';

const props = defineProps({
  demand: {
    type: Number,
    default: 0,
  },
  generated: {
    type: Number,
    default: 0,
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
  showEnergy: {
    type: Boolean,
    default: false,
    required: false,
  },
  totalRange: {
    type: Number,
    default: 1000
  },
  runDemo: {
    type: Boolean,
    default: false,
    required: false,
  },

});

let autoInterval;

const energyMeterSVG = ref(null);

const currentDemand = ref(props.demand);
const currentGenerated = ref(props.generated);
const currentTotalRange = ref(props.totalRange);

const currentNetConsumptionStr = computed(() => {
  return format(currentDemand.value - currentGenerated.value);
});

const periodStr = computed(() => {
  return `per ${sentenceCase(props.period)}`;
});


let blades;
let generateObj;
let demandObj;
let generatePips;
let demandPips;

// Demo: generate random values
const autoValues = () =>
  setValue(
    Math.floor(Math.random() * (currentTotalRange.value) + 1),
    Math.floor(Math.random() * (currentTotalRange.value) + 1),
  );

// Sleep helper for async delays
const sleep = (ms) => {
  return new Promise((resolve) => setTimeout(resolve, ms));
};

// Adjust turbine blade speed
const turbineSpeed = (val) => {
  blades.classList.remove('stop', 'slow', 'medium', 'fast', 'fastest');
  if (val === 0) blades.classList.add('stop');
  if (val > 0) blades.classList.add('slow');
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
const setValue = async (generated, demanded, cancelled) => {
  clearInterval(autoInterval);

  currentDemand.value = demanded;
  currentGenerated.value = generated;

  currentTotalRange.value = Math.max(currentTotalRange.value, currentDemand.value, currentGenerated.value);

  // Demand
  let demandPipVal = Math.floor(Math.abs(demanded) / currentTotalRange.value * demandPips.length);
  let generatePipVal = Math.floor(Math.abs(generated) / currentTotalRange.value * generatePips.length);
  turbineSpeed(Math.abs(generatePipVal));

  for (let i = 0; i < demandPips.length; i++) {
    // Demand pips
    if (i < demandPipVal || i === 0) {
      demandPips[i].classList.add('onPylon');
    } else {
      demandPips[i].classList.remove('onPylon');
    }
    // Generation pips
    if (i < generatePipVal || i === 0) {
      generatePips[i].classList.add('onTurbine');
    } else {
      generatePips[i].classList.remove('onTurbine');
    }
    if (cancelled) return;
    await sleep(50);
  }

  startAuto();
};

onMounted(() => {
  blades = energyMeterSVG.value.getElementById('blades');

  generateObj = energyMeterSVG.value.getElementById('generating');
  demandObj = energyMeterSVG.value.getElementById('consuming');

  generatePips = [...generateObj.querySelectorAll('.pip')].reverse();
  demandPips = [...demandObj.querySelectorAll('.pip')].reverse();

  // Start demo
  startAuto();
  // setValue(props.generated, props.demand, false);
});

watch(() => props.generated, (newGenVal, _, onCleanup) => {
  let cancelled = false;
  onCleanup(() => {
    cancelled = true;
  });
  setValue(newGenVal, props.demand, cancelled);
});

watch(() => props.demand, (newDemandVal, _, onCleanup) => {
  let cancelled = false;
  onCleanup(() => {
    cancelled = true;
  });
  setValue(props.generated, newDemandVal, cancelled);
});
</script>

<style lang="scss" scoped>
$grey: #c6c3c1;
$red: #ff4848;
$green: #4caf50;
$lightGreen: #a5d6a7;

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

  &.fillGray {
    fill: #C6C3C1;
  }

  &.onPylon {
    fill: $red;

    &:not(.onPylon ~ .onPylon) {
      animation: flickerRed 0.5s linear infinite;
      animation-delay: 0.5s;
    }
  }

  &.onTurbine {
    fill: $green;

    &:not(.onTurbine ~ .onTurbine) {
      animation: flickerGreen 0.5s linear infinite;
      animation-delay: 0.5s;
    }
  }
}

.fillBlack {
  fill: #000;
}

.strokeBlack {
  stroke: #000;
}

.strokeWhite {
  stroke: #fff;
}

.text {
  fill: black;
  font-family: sans-serif;
}

.hide {
  opacity: 0;
  transition: 0.5s ease-in-out;
}

#blades {
  transform-box: fill-box;
  transform-origin: center;
  animation: spin 6s linear infinite;
  transition: 1s ease-in-out;

  &.stop {
    animation-play-state: paused;
  }

  &.slow {
    animation-duration: 6s;
  }

  &.medium {
    animation-duration: 4s;
  }

  &.fast {
    animation-duration: 2s;
  }

  &.fastest {
    animation-duration: 1s;
  }
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(359deg);
  }
}

@keyframes flickerRed {
  45% {
    fill: $red;
  }

  55% {
    fill: $grey;
  }
}

@keyframes flickerGreen {
  45% {
    fill: $green;
  }

  55% {
    fill: $grey;
  }
}
</style>
