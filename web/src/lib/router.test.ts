import { describe, expect, it } from 'vitest';
import { hashPath, matchRoute } from './router';

const routes = [
  { name: 'home', pattern: '/' },
  { name: 'goals', pattern: '/goals' },
  { name: 'account', pattern: '/goals/accounts/:id' },
  { name: 'goal', pattern: '/goals/:id' },
];

describe('hashPath', () => {
  it('strips the hash sign and the query', () => {
    expect(hashPath('#/goals/3?x=1')).toBe('/goals/3');
  });

  it('treats an empty hash as the home path', () => {
    expect(hashPath('')).toBe('/');
    expect(hashPath('#')).toBe('/');
  });
});

describe('matchRoute', () => {
  it('matches fixed paths', () => {
    expect(matchRoute('/', routes)).toEqual({ name: 'home', params: {} });
    expect(matchRoute('/goals', routes)).toEqual({ name: 'goals', params: {} });
  });

  it('extracts parameters', () => {
    expect(matchRoute('/goals/accounts/7', routes)).toEqual({
      name: 'account',
      params: { id: '7' },
    });
    expect(matchRoute('/goals/9', routes)).toEqual({ name: 'goal', params: { id: '9' } });
  });

  it('ignores a trailing slash', () => {
    expect(matchRoute('/goals/', routes)?.name).toBe('goals');
  });

  it('returns null when nothing matches', () => {
    expect(matchRoute('/nowhere', routes)).toBeNull();
    expect(matchRoute('/goals/accounts/7/extra', routes)).toBeNull();
  });
});
