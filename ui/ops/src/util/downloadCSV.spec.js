import {escapeCSVField} from '@/util/downloadCSV.js';
import {describe, expect, it} from 'vitest';

describe('escapeCSVField', () => {
  describe('pass-through', () => {
    it('leaves a bare value unquoted', () => {
      expect(escapeCSVField('plain')).toBe('plain');
    });

    it('leaves an empty string empty', () => {
      expect(escapeCSVField('')).toBe('');
    });

    it('stringifies non-string values', () => {
      expect(escapeCSVField(42)).toBe('42');
      expect(escapeCSVField(false)).toBe('false');
    });

    it('keeps falsy-but-present values rather than blanking them', () => {
      expect(escapeCSVField(0)).toBe('0');
    });
  });

  describe('RFC 4180 quoting', () => {
    it('quotes a value containing a comma', () => {
      expect(escapeCSVField('hello, world')).toBe('"hello, world"');
    });

    it('quotes a value containing a double-quote and doubles the quote', () => {
      expect(escapeCSVField('say "hi"')).toBe('"say ""hi"""');
    });

    it('quotes a value containing LF', () => {
      expect(escapeCSVField('line1\nline2')).toBe('"line1\nline2"');
    });

    it('quotes a value containing CR', () => {
      expect(escapeCSVField('line1\rline2')).toBe('"line1\rline2"');
    });

    it('quotes a value containing CRLF', () => {
      expect(escapeCSVField('line1\r\nline2')).toBe('"line1\r\nline2"');
    });
  });

  describe('nullish handling', () => {
    it('renders null as an empty field', () => {
      expect(escapeCSVField(null)).toBe('');
    });

    it('renders undefined as an empty field', () => {
      expect(escapeCSVField(undefined)).toBe('');
    });
  });

  describe('formula-injection guard', () => {
    it.each([
      ['=', '=1+1'],
      ['+', '+1+1'],
      ['-', '-cmd'],
      ['@', '@SUM(A1)'],
      ['tab', '\t=cmd'],
      ['CR', '\rSUM(A1)']
    ])('prefixes a non-numeric value leading with %s', (_, value) => {
      expect(escapeCSVField(value)).toContain(`'${value}`);
    });

    it('neutralises a formula without quoting when it holds no special characters', () => {
      expect(escapeCSVField('=1+1')).toBe(`'=1+1`);
    });

    it('both neutralises and quotes a formula containing special characters', () => {
      expect(escapeCSVField('=HYPERLINK("http://evil","click")'))
          .toBe(`"'=HYPERLINK(""http://evil"",""click"")"`);
    });

    it.each([
      ['negative integer', '-5'],
      ['positive integer', '+5'],
      ['negative decimal', '-5.5'],
      ['negative exponent', '-5.5e3']
    ])('leaves a %s intact', (_, value) => {
      expect(escapeCSVField(value)).toBe(value);
    });

    it('leaves a negative number passed as a number intact', () => {
      expect(escapeCSVField(-5)).toBe('-5');
    });
  });
});
