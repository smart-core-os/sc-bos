import {describe, expect, it} from 'vitest';
import {createSSRApp, h} from 'vue';
import {renderToString} from 'vue/server-renderer';
import {createVuetify} from 'vuetify';
import Gauge from './NabersRatingGauge.vue';

/**
 * Render the gauge to a string.
 *
 * SSR rather than a mounted DOM, because everything worth asserting here is
 * which branch of the state chain ran and which group each element landed in —
 * both plain in the markup, and neither needing a browser.
 *
 * HTML comments are stripped. Vue renders them, this template carries a lot of
 * them, and they discuss the very states being asserted about — so a naive
 * `not.toContain('Too early to rate')` matched the comment explaining why the
 * loading branch has to outrank it.
 *
 * @param {Object} props
 * @return {Promise<string>} the rendered markup, comments removed
 */
async function render(props) {
  const app = createSSRApp({render: () => h(Gauge, props)});
  app.use(createVuetify());
  return (await renderToString(app)).replace(/<!--[\s\S]*?-->/g, '');
}

/** A settled rating, matching 3CS: 5.55 stars on 55.3 kWhe/m². */
const RATED = {
  rating:               {stars: 5.5452, bandedStars: 5.5, intensity: 55.3, benchmarkingFactor: 38.55},
  hasMeters:            true,
  benchmark:            143.45,
  nextStarTarget:       {stars: 6, ceiling: 38.01},
  reductionNeeded:      17.29,
  progressToNextStar:   61,
  estimatedSharePct:    2.4,
  estimatedMeterLabels: ['Lifts: lift-1']
};

/**
 * The slice of markup belonging to one of the two layout groups.
 *
 * @param {string} html
 * @param {'rating-primary'|'rating-support'} cls
 * @return {string}
 */
function group(html, cls) {
  const start = html.indexOf(`class="${cls}"`);
  if (start < 0) return '';
  const end = cls === 'rating-primary' ? html.indexOf('class="rating-support"', start) : html.length;
  return html.slice(start, end < 0 ? html.length : end);
}

describe('which state the gauge shows', () => {
  it('says it is loading rather than that it is too early', async () => {
    // Reported from site six days into a fresh rating period. The period-to-date
    // read resolves two boundaries and the monthly table thirteen, so the section
    // renders on the first while the second is still in flight — and every state
    // below `loading` in the chain is a claim about data that has not arrived.
    // For four weeks after each anniversary that window reads "Too early to
    // rate", moments before twelve measured months land and produce one.
    const html = await render({hasMeters: true, rating: null, loading: true, tooEarly: true, elapsedDays: 6});
    expect(html).toContain('Loading');
    expect(html).not.toContain('Too early to rate');
    expect(html).not.toContain('day(s) into the rating period');
  });

  it('keeps showing a rating it already has while the next fetch runs', async () => {
    // The daily refresh must not pull a good figure off the screen.
    const html = await render({...RATED, loading: true});
    expect(html).toContain('5.55');
    expect(html).not.toContain('Loading');
  });

  it('lets a config fault outrank loading, since waiting will not fix it', async () => {
    const missing = await render({missingInputs: ['postcode'], hasMeters: true, loading: true});
    expect(missing).toContain('Rating inputs incomplete');
    expect(missing).not.toContain('Loading');

    const noMeters = await render({hasMeters: false, loading: true});
    expect(noMeters).toContain('No base building meters are configured');
    expect(noMeters).not.toContain('Loading');
  });

  it('still reports the settled states once nothing is in flight', async () => {
    expect(await render({hasMeters: true, rating: null, tooEarly: true, elapsedDays: 6}))
      .toContain('Too early to rate');
    expect(await render({hasMeters: true, rating: null, unreadableCount: 2, configuredCount: 7}))
      .toContain('unreadable');
    expect(await render({hasMeters: true, rating: null}))
      .toContain('Consumption history is still accumulating');
  });
});

describe('the layout split', () => {
  it('puts only the rating in primary, and every supporting element in support', async () => {
    const html = await render({...RATED, layout: 'row'});
    const primary = group(html, 'rating-primary');
    const support = group(html, 'rating-support');

    for (const cls of ['stars-row', 'decimal-stars', 'intensity-value']) {
      expect(primary, cls).toContain(cls);
      expect(support, cls).not.toContain(cls);
    }
    for (const cls of ['chip-row', 'basis-chip', 'benchmark-line', 'caveat',
      'rating-desc', 'next-star-text', 'progress-bar']) {
      expect(support, cls).toContain(cls);
      expect(primary, cls).not.toContain(cls);
    }
  });

  it('renders identical markup either way, so the tenancy view is untouched', async () => {
    const strip = s => s.replace(/rating-gauge--\w+/, '');
    expect(strip(await render({...RATED, layout: 'row'})))
      .toBe(strip(await render({...RATED, layout: 'column'})));
  });

  it('defaults to column, and applies the modifier asked for', async () => {
    expect(await render(RATED)).toContain('rating-gauge--column');
    expect(await render({...RATED, layout: 'row'})).toContain('rating-gauge--row');
  });

  it('shows the disclaimer but no supporting detail when there is no rating', async () => {
    // The support group is gated on `showRating`, not on the tail of the state
    // chain, which is what stops `rating.bandedStars` being read off a null.
    for (const props of [
      {missingInputs: ['postcode'], hasMeters: true},
      {hasMeters: false},
      {hasMeters: true, rating: null, loading: true},
      {hasMeters: true, rating: null, unreadableCount: 2, configuredCount: 7},
      {hasMeters: true, rating: null, tooEarly: true, elapsedDays: 4},
      {hasMeters: true, rating: null, benchmark: 143.45}
    ]) {
      const html = await render({...props, layout: 'row'});
      expect(html).toContain('rating-desc');
      expect(html).not.toContain('stars-row');
      expect(html).not.toContain('chip-row');
      expect(html).not.toContain('benchmark-line');
    }
  });
});
