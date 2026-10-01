import {describe, it, expect} from 'vitest';
import {safeHttpUrl} from './externalLink.js';

const BASE = 'https://scos.example.uk/nabersdashboard/';

describe('safeHttpUrl', () => {
  it('accepts an absolute http(s) URL and returns it normalised', () => {
    expect(safeHttpUrl('https://scos.example.uk/ops/overview/nabers'))
      .toBe('https://scos.example.uk/ops/overview/nabers');
    expect(safeHttpUrl('http://scos.example.uk/ops')).toBe('http://scos.example.uk/ops');
    // A bare origin gains the path the URL parser considers canonical, so the
    // value handed to the anchor is the one the browser would navigate to.
    expect(safeHttpUrl('https://scos.example.uk')).toBe('https://scos.example.uk/');
  });

  it('tolerates surrounding whitespace, which hand-edited JSON collects', () => {
    expect(safeHttpUrl('  https://scos.example.uk/ops  ')).toBe('https://scos.example.uk/ops');
  });

  it('refuses a script-bearing scheme however it is cased or spaced', () => {
    expect(safeHttpUrl('javascript:alert(1)')).toBeNull();
    expect(safeHttpUrl('JavaScript:alert(1)')).toBeNull();
    expect(safeHttpUrl('  javascript:alert(1)')).toBeNull();
    expect(safeHttpUrl('data:text/html,<script>alert(1)</script>')).toBeNull();
    expect(safeHttpUrl('file:///etc/passwd')).toBeNull();
    // Still rejected with a base: resolution does not change the scheme.
    expect(safeHttpUrl('javascript:alert(1)', BASE)).toBeNull();
  });

  it('treats an absent, blank or non-string value as "no link configured"', () => {
    expect(safeHttpUrl(undefined)).toBeNull();
    expect(safeHttpUrl(null)).toBeNull();
    expect(safeHttpUrl('')).toBeNull();
    expect(safeHttpUrl('   ')).toBeNull();
    expect(safeHttpUrl(42)).toBeNull();
    expect(safeHttpUrl({url: 'https://scos.example.uk'})).toBeNull();
  });

  it('rejects a relative path with no base rather than guessing an origin', () => {
    expect(safeHttpUrl('/ops/overview/nabers')).toBeNull();
    expect(safeHttpUrl('ops/overview/nabers')).toBeNull();
    // Protocol-relative: no base means no scheme to inherit.
    expect(safeHttpUrl('//evil.example/ops')).toBeNull();
  });

  it('resolves a relative path against a base, for same-origin deployments', () => {
    expect(safeHttpUrl('/ops/overview/nabers', BASE))
      .toBe('https://scos.example.uk/ops/overview/nabers');
    // Relative to the dashboard's own directory, not the origin root.
    expect(safeHttpUrl('../ops/overview/nabers', BASE))
      .toBe('https://scos.example.uk/ops/overview/nabers');
  });

  it('does not confine a configured link to the base origin', () => {
    // The ops UI is often on another host, so cross-origin is the normal case,
    // not something to filter out. The guard is on the scheme alone.
    expect(safeHttpUrl('https://other.example.uk/ops', BASE))
      .toBe('https://other.example.uk/ops');
  });
});
