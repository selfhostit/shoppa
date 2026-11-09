import { describe, expect, it } from 'vitest';
import { shareUrl, sharedToken, type Share } from './shares';

describe('shares', () => {
  it('make and read a guest address', () => {
    const url = shareUrl({ token: 'abc' } as Share, 'https://shoppa.example');
    expect(url).toBe('https://shoppa.example/shared/abc');
    expect(sharedToken('/shared/abc')).toBe('abc');
    expect(sharedToken('/')).toBe('');
  });
});
