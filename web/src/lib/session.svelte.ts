import { ApiError, api, onUnauthorized, type User } from '../api/client';
import { navigate } from './router.svelte';

interface Session {
  status: 'loading' | 'anonymous' | 'authenticated';
  user: User | null;
}

/** Who is signed in. It starts as "loading" until restoreSession has answered. */
export const session: Session = $state({ status: 'loading', user: null });

function signedOut(): void {
  session.status = 'anonymous';
  session.user = null;
}

/** Asks the server whether the browser still has a valid session. */
export async function restoreSession(): Promise<void> {
  try {
    session.user = await api.me();
    session.status = 'authenticated';
  } catch (error) {
    // A 401 just means nobody is signed in. A failure to reach the server also
    // ends here, and the login page then reports it on submit.
    if (!(error instanceof ApiError)) throw error;
    signedOut();
  }
}

/** Signs in. It throws an ApiError when the server refuses. */
export async function logIn(username: string, password: string): Promise<void> {
  session.user = await api.login(username, password);
  session.status = 'authenticated';
}

/** Signs out, and shows the login page even if the server cannot be reached. */
export async function logOut(): Promise<void> {
  try {
    await api.logout();
  } finally {
    signedOut();
    navigate('/login');
  }
}

// A session that expires while the app is open ends on the login page.
onUnauthorized(() => {
  signedOut();
  navigate('/login');
});
