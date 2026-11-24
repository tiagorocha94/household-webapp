import { hashPath } from './router';

/** The current path, kept in step with the location hash. */
export const router = $state({ path: hashPath(window.location.hash) });

window.addEventListener('hashchange', () => {
  router.path = hashPath(window.location.hash);
});

/** Goes to path, for example "/goals/3". */
export function navigate(path: string): void {
  window.location.hash = path;
}

/** The href that leads to path. */
export function href(path: string): string {
  return `#${path}`;
}
