/**
 * Get trend icon based on change value
 *
 * @param {number|null} change - The percentage change value
 * @return {string} Material Design icon name
 */
export function getTrendIcon(change) {
  if (change === null || change === undefined || Math.abs(change) < 0.05) return 'mdi-minus';
  return change > 0 ? 'mdi-trending-up' : 'mdi-trending-down';
}

/**
 * Get trend color based on change value
 *
 * @param {number|null} change - The percentage change value
 * @param {boolean} successOnUp - If true, positive change is success (green), negative is error (red)
 * @return {string} Vuetify color name
 */
export function getTrendColor(change, successOnUp = false) {
  if (change === null || change === undefined || Math.abs(change) < 0.05) return 'grey-lighten-1';
  if (successOnUp) {
    return change > 0 ? 'success' : 'error';
  }
  return change > 0 ? 'error' : 'success';
}

/**
 * Format trend value for display as a percentage string (e.g., "+5.2%", "-12%", "0%")
 *
 * @param {number|null} change - The percentage change value
 * @return {string} Formatted percentage string
 */
export function formatTrend(change) {
  if (change === null || change === undefined || isNaN(change) || Math.abs(change) < 0.05) return '0%';
  const absChange = Math.abs(change);
  const sign = change > 0 ? '+' : '-';
  if (absChange < 9.95) {
    return `${sign}${absChange.toFixed(1)}%`;
  }
  return `${sign}${Math.round(absChange)}%`;
}
