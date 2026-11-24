import { describe, expect, it } from 'vitest';
import { readingText, updatedText } from './updated';

describe('updatedText', () => {
  it('shows the hour and minute', () => {
    const at = new Date(2026, 10, 21, 19, 1);
    expect(updatedText(at)).toBe('Updated 19:01');
  });

  it('pads the hour and the minute', () => {
    expect(updatedText(new Date(2026, 10, 21, 7, 5))).toBe('Updated 07:05');
  });
});

describe('readingText', () => {
  it('names what is being read', () => {
    expect(readingText('goals')).toBe('Reading your goals…');
  });
});
