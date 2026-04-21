import Color from 'colorjs.io';
import {computed, ref} from 'vue';
import * as colors from 'vuetify/util/colors';

/**
 * Helper to give type assistance to chart.js plugins.
 *
 * @template {import('chart.js').Plugin} T
 * @param {T} plugin
 * @return {T}
 */
export function definePlugin(plugin) {
  return plugin;
}

/**
 * Captures legend items using a Chart.js plugin.
 *
 * @return {{
 *   legendItems: Ref<{
 *     text: string,
 *     hidden: boolean,
 *     bgColor: string,
 *     onClick: (e: MouseEvent) => void
 *   }[]>,
 *   vueLegendPlugin: import('chart.js').Plugin
 * }}
 */
export function useVueLegendPlugin() {
  const legendItems = ref([]);
  return {
    legendItems,
    vueLegendPlugin: definePlugin({
      id: 'vueLegend',
      afterUpdate(chart) {
        const items = chart.options.plugins.legend.labels.generateLabels(chart);
        legendItems.value = items.map((item) => {
          // Dashed line datasets (e.g. peak overlays) have no meaningful fill —
          // Chart.js parses 'transparent' to 'rgba(0,0,0,0)' which becomes opaque
          // black after forcing alpha to 1. Use strokeStyle for dashed lines instead.
          const isDashed = (item.lineDash?.length ?? 0) > 0;
          const colorSource = isDashed ? item.strokeStyle : item.fillStyle;
          let bgColor;
          try {
            bgColor = new Color(colorSource ?? '#fff');
            bgColor.alpha = 1;
            bgColor = bgColor.toString();
          } catch {
            bgColor = colorSource;
          }
          return {
            text: item.text,
            hidden: item.hidden,
            bgColor,
            datasetIndex: item.datasetIndex,
            isDashed,
            onClick: (e) => {
              const {type} = chart.config;
              if (type === 'pie' || type === 'doughnut') {
                // Pie and doughnut charts only have a single dataset and visibility is per item
                const currentVisibility = chart.getDataVisibility(item.index);
                if (currentVisibility !== e) {
                  chart.toggleDataVisibility(item.index);
                }
              } else {
                chart.setDatasetVisibility(item.datasetIndex, e);
              }
              chart.update();
            }
          };
        });
      }
    })
  }
}

/**
 * Colours dataset based on the current theme.
 *
 * @return {{themeColorPlugin: import('chart.js').Plugin}}
 */
export function useThemeColorPlugin() {
  const datasetColors = computed(() => {
    return [
      colors.blue.base,
      colors.green.base,
      colors.orange.base,
      colors.yellow.base,
      colors.red.base,
    ].filter(Boolean);
  });
  return {
    themeColorPlugin: definePlugin({
      id: 'themeColor',
      beforeLayout(chart) {
        const colors = datasetColors.value;
        let i = 0;
        chart.data.datasets.forEach((dataset) => {
          if (dataset.backgroundColor && !dataset._pluginColor) return;
          const color = colors[i % colors.length];
          i++;
          dataset.backgroundColor = color + '80'; // 80 is 50% opacity
          dataset.borderColor = color;
          dataset._pluginColor = true;
        });
      }
    })
  }
}


/**
 * @typedef {Object} TooltipData
 * @property {number} x
 * @property {number} y
 * @property {number} opacity
 * @property {import('chart.js').TooltipItem[]} dataPoints
 * @property {Record<string,string>} displayFormats
 */

/**
 * @typedef {function} ExternalFunc
 * @this {import('chart.js').TooltipModel}
 * @property {{
 *   chart: import('chart.js').Chart,
 *   tooltip: import('chart.js').TooltipModel
 * }} args
 * @return {void}
 */

/**
 * Returns a function that can be used by chart.js tooltip.external and a ref containing the data.
 *
 * @return {{
 *   data: import('vue').Ref<TooltipData | null>,
 *   external: ExternalFunc
 * }}
 */
export function useExternalTooltip() {
  const data = ref(null);
  return {
    data,
    external: (ctx) => {
      if (!ctx.tooltip) {
        data.value = null;
        return;
      }
      const canvasBounds = ctx.chart.canvas.getBoundingClientRect();
      data.value = {
        x: ctx.tooltip.caretX + canvasBounds.left,
        y: ctx.tooltip.caretY + canvasBounds.top,
        opacity: ctx.tooltip.opacity,
        dataPoints: ctx.tooltip.dataPoints,
        displayFormats: ctx.chart.options.scales.x.time?.displayFormats,
      };
    }
  }
}
