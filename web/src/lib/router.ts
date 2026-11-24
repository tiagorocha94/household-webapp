// Hash routing: the address is "#/goals/3". The matching is pure, so it is
// tested without a browser; router.svelte.ts holds the reactive state.
export interface Route {
  name: string;
  pattern: string;
}

export interface Match {
  name: string;
  params: Record<string, string>;
}

/** Turns a location hash such as "#/goals/3?x=1" into the path "/goals/3". */
export function hashPath(hash: string): string {
  const path = hash.replace(/^#/, '').split('?')[0];
  return path === '' ? '/' : path;
}

/** Finds the first route whose pattern matches path, with its ":param" values. */
export function matchRoute(path: string, routes: readonly Route[]): Match | null {
  const parts = segments(path);
  for (const route of routes) {
    const pattern = segments(route.pattern);
    if (pattern.length !== parts.length) continue;
    const params: Record<string, string> = {};
    const ok = pattern.every((part, i) => {
      if (part.startsWith(':')) {
        params[part.slice(1)] = decodeURIComponent(parts[i]);
        return true;
      }
      return part === parts[i];
    });
    if (ok) return { name: route.name, params };
  }
  return null;
}

function segments(path: string): string[] {
  return path.split('/').filter((part) => part !== '');
}
