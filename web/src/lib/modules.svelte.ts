// The modules the server runs, learned from GET /info. Until it answers, nothing is hidden.

export const serverModules = $state<{ enabled: readonly string[] | null }>({ enabled: null });

export function setServerModules(enabled: readonly string[]): void {
  serverModules.enabled = enabled;
}
