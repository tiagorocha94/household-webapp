import { ApiError, api } from '../api/client';

/** How many things need a look; the number on the bell. */
export const attention = $state({ count: 0 });

/** Asks the server again. A failure leaves the last number: the bell is only a hint. */
export async function refreshAttention(): Promise<void> {
  try {
    attention.count = (await api.overview()).attention.length;
  } catch (error) {
    if (!(error instanceof ApiError)) throw error;
  }
}
