import { ApiError } from '../api/client';

/** Data that is loaded from the API, with its loading and error state. */
export class Resource<T> {
  status = $state<'loading' | 'ready' | 'error'>('loading');
  data = $state<T | undefined>(undefined);
  error = $state<unknown>(undefined);
  /** When the data was last read successfully. */
  loadedAt = $state<Date | undefined>(undefined);

  readonly #fetch: () => Promise<T>;

  constructor(fetch: () => Promise<T>) {
    this.#fetch = fetch;
    void this.reload();
  }

  /** Loads the data again. */
  async reload(): Promise<void> {
    this.status = 'loading';
    try {
      this.data = await this.#fetch();
      this.loadedAt = new Date();
      this.status = 'ready';
    } catch (error) {
      if (!(error instanceof ApiError)) throw error;
      this.error = error;
      this.status = 'error';
    }
  }
}
