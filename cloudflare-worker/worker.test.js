import { describe, it, expect, beforeEach, vi } from 'vitest';
import worker, { renderText, formatLocalTime, fireRoutine } from './worker.js';

const TEST_TOKEN = 'sk-ant-oat01-test-only-not-real';
const TEST_URL = 'https://api.anthropic.com/v1/claude_code/routines/trig_TEST/fire';

/**
 * Captures `console.log` output for assertion. Restores on cleanup.
 * @returns {{ logs: string[], restore: () => void }}
 */
function captureLogs() {
  const logs = [];
  const original = console.log;
  console.log = (msg) => { logs.push(typeof msg === 'string' ? msg : JSON.stringify(msg)); };
  return { logs, restore: () => { console.log = original; } };
}

/**
 * Builds a minimal Env for tests.
 * @param {Partial<import('./worker.js').Env>} overrides
 * @returns {import('./worker.js').Env}
 */
function buildEnv(overrides = {}) {
  return {
    ROUTINE_FIRE_URL: TEST_URL,
    ROUTINE_FIRE_TOKEN: TEST_TOKEN,
    ...overrides,
  };
}

const fixedTime = Date.UTC(2026, 4, 9, 3, 19, 0); // 2026-05-09T03:19:00Z (cron 19 3 * * *)
const baseController = { cron: '19 3 * * *', scheduledTime: fixedTime };

describe('renderText', () => {
  it('substitutes {LocalTime}, {Cron}, {ISO}', () => {
    const out = renderText('At {LocalTime} cron {Cron} iso {ISO}', {
      LocalTime: '2026-05-09 10:19 GMT+7',
      Cron: '19 3 * * *',
      ISO: '2026-05-09T03:19:00.000Z',
    });
    expect(out).toBe('At 2026-05-09 10:19 GMT+7 cron 19 3 * * * iso 2026-05-09T03:19:00.000Z');
  });

  it('leaves unknown {Foo} tokens intact', () => {
    const out = renderText('hello {Foo}', { Bar: 'baz' });
    expect(out).toBe('hello {Foo}');
  });
});

describe('formatLocalTime', () => {
  it('defaults to UTC formatting', () => {
    const s = formatLocalTime(new Date(fixedTime), 'UTC');
    expect(s).toMatch(/^2026-05-09 03:19 /);
  });

  it('respects IANA tz', () => {
    const s = formatLocalTime(new Date(fixedTime), 'Asia/Ho_Chi_Minh');
    expect(s).toMatch(/^2026-05-09 10:19 /);
  });
});

describe('fireRoutine', () => {
  /** @type {ReturnType<typeof captureLogs>} */
  let cap;
  /** @type {ReturnType<typeof vi.fn>} */
  let fetchSpy;

  beforeEach(() => {
    cap = captureLogs();
    fetchSpy = vi.fn();
    globalThis.fetch = fetchSpy;
  });

  function tearDown() {
    cap.restore();
    vi.restoreAllMocks();
  }

  it('logs session_url on 2xx', async () => {
    fetchSpy.mockResolvedValue(new Response(JSON.stringify({
      type: 'ok',
      claude_code_session_id: 'sess_1',
      claude_code_session_url: 'https://claude.ai/s/sess_1',
    }), { status: 200 }));

    await fireRoutine(buildEnv(), baseController);

    const joined = cap.logs.join('\n');
    expect(joined).toContain('"level":"info"');
    expect(joined).toContain('https://claude.ai/s/sess_1');
    expect(joined).not.toContain(TEST_TOKEN);
    tearDown();
  });

  it('logs level:error on 401 non-2xx', async () => {
    fetchSpy.mockResolvedValue(new Response('unauthorized', { status: 401 }));

    await fireRoutine(buildEnv(), baseController);

    const joined = cap.logs.join('\n');
    expect(joined).toContain('"level":"error"');
    expect(joined).toContain('"status":401');
    expect(joined).not.toContain(TEST_TOKEN);
    tearDown();
  });

  it('logs error and does not throw when fetch rejects', async () => {
    fetchSpy.mockRejectedValue(new Error('network down'));

    await expect(fireRoutine(buildEnv(), baseController)).resolves.toBeUndefined();
    const joined = cap.logs.join('\n');
    expect(joined).toContain('"level":"error"');
    expect(joined).toContain('network down');
    expect(joined).not.toContain(TEST_TOKEN);
    tearDown();
  });

  it('sends required headers and JSON body', async () => {
    fetchSpy.mockResolvedValue(new Response(JSON.stringify({
      type: 'ok', claude_code_session_id: 's', claude_code_session_url: 'u',
    }), { status: 200 }));

    await fireRoutine(buildEnv(), baseController);

    expect(fetchSpy).toHaveBeenCalledOnce();
    const [url, init] = fetchSpy.mock.calls[0];
    expect(url).toBe(TEST_URL);
    expect(init.method).toBe('POST');
    expect(init.headers['Authorization']).toBe(`Bearer ${TEST_TOKEN}`);
    expect(init.headers['anthropic-version']).toBe('2023-06-01');
    expect(init.headers['anthropic-beta']).toBe('experimental-cc-routine-2026-04-01');
    expect(init.headers['Content-Type']).toBe('application/json');

    const parsed = JSON.parse(init.body);
    expect(parsed).toHaveProperty('text');
    expect(typeof parsed.text).toBe('string');
    tearDown();
  });

  it('uses default template when TEXT_TEMPLATE unset', async () => {
    fetchSpy.mockResolvedValue(new Response(JSON.stringify({
      type: 'ok', claude_code_session_id: 's', claude_code_session_url: 'u',
    }), { status: 200 }));

    await fireRoutine(buildEnv(), baseController);

    const body = JSON.parse(fetchSpy.mock.calls[0][1].body);
    expect(body.text).toMatch(/^Scheduled trigger at /);
    tearDown();
  });

  it('defaults TZ to UTC when env.TZ unset', async () => {
    fetchSpy.mockResolvedValue(new Response(JSON.stringify({
      type: 'ok', claude_code_session_id: 's', claude_code_session_url: 'u',
    }), { status: 200 }));

    await fireRoutine(buildEnv(), baseController);

    const body = JSON.parse(fetchSpy.mock.calls[0][1].body);
    expect(body.text).toContain('03:19'); // UTC hour from fixedTime
    tearDown();
  });

  it('warn-logs unparseable 2xx body', async () => {
    fetchSpy.mockResolvedValue(new Response('not-json', { status: 200 }));

    await fireRoutine(buildEnv(), baseController);

    const joined = cap.logs.join('\n');
    expect(joined).toContain('"level":"warn"');
    expect(joined).toContain('unparseable');
    tearDown();
  });
});

describe('default export (scheduled handler)', () => {
  it('exposes async scheduled function', () => {
    expect(typeof worker.scheduled).toBe('function');
  });
});
