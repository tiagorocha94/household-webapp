// A stand-in for the Go server, for working on the UI without it:
//   npm run dev:mock     (then npm run dev in another terminal)
// It serves the fixtures in fixtures.ts on :8080. Log in as alex / alex;
// the username "locked" answers 429, to see that message.
import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { randomUUID } from 'node:crypto';
import { users, routes } from './fixtures.ts';

const port = Number(process.env.PORT ?? 8080);
const sessions = new Map<string, number>();

function send(
  res: ServerResponse,
  status: number,
  body?: unknown,
  headers: Record<string, string> = {},
) {
  res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8', ...headers });
  res.end(body === undefined ? undefined : JSON.stringify(body));
}

function readBody(req: IncomingMessage): Promise<string> {
  return new Promise((resolve) => {
    let data = '';
    req.on('data', (chunk) => (data += chunk));
    req.on('end', () => resolve(data));
  });
}

function sessionUser(req: IncomingMessage) {
  const token = /(?:^|;\s*)session_token=([^;]+)/.exec(req.headers.cookie ?? '')?.[1];
  const id = token ? sessions.get(token) : undefined;
  return users.find((u) => u.id === id);
}

createServer(async (req, res) => {
  const url = new URL(req.url ?? '/', 'http://localhost');
  const path = url.pathname.replace(/^\/api\/v1/, '');
  console.log(req.method, url.pathname);

  if (req.method === 'GET' && path === '/healthz') return send(res, 200, { status: 'ok' });
  if (req.method === 'GET' && path === '/info') return send(res, 200, { version: 'mock' });

  if (req.method === 'POST' && path === '/auth/login') {
    const { username, password } = JSON.parse((await readBody(req)) || '{}');
    if (username === 'locked') {
      return send(res, 429, { error: 'Too Many Requests' }, { 'Retry-After': '120' });
    }
    const user = users.find((u) => u.username === username && u.username === password);
    if (!user) return send(res, 401, { error: 'Invalid credentials' });
    const token = randomUUID();
    sessions.set(token, user.id);
    return send(
      res,
      200,
      { id: user.id, username: user.username },
      {
        'Set-Cookie': `session_token=${token}; Path=/; HttpOnly; SameSite=Lax`,
      },
    );
  }
  if (req.method === 'POST' && path === '/auth/logout') {
    return send(res, 204, undefined, { 'Set-Cookie': 'session_token=; Path=/; Max-Age=0' });
  }

  const user = sessionUser(req);
  if (!user) return send(res, 401, { error: 'Unauthorized' });
  if (req.method === 'GET' && path === '/auth/me')
    return send(res, 200, { id: user.id, username: user.username });

  if (req.method === 'GET') {
    for (const route of routes) {
      const match = route.pattern.exec(path);
      if (match) {
        const result = route.handle(match.slice(1).map(Number));
        return result === null ? send(res, 404, { error: 'Not Found' }) : send(res, 200, result);
      }
    }
  }
  send(res, 404, { error: 'Not Found' });
}).listen(port, '127.0.0.1', () => console.log(`mock API on http://127.0.0.1:${port}`));
