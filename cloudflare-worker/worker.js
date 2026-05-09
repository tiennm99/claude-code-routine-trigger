/**
 * @typedef {object} Env
 * @property {string} ROUTINE_FIRE_URL    - secret: Anthropic /fire endpoint URL
 * @property {string} ROUTINE_FIRE_TOKEN  - secret: per-routine bearer token (sk-ant-oat01-...)
 * @property {string} [TEXT_TEMPLATE]     - plain var: prompt template, supports {LocalTime} {Cron} {ISO}
 * @property {string} [TZ]                - plain var: IANA tz, default 'UTC'
 */

/**
 * @typedef {object} FireResponse
 * @property {string} type
 * @property {string} claude_code_session_id
 * @property {string} claude_code_session_url
 */

const HEADER_VERSION = '2023-06-01';
const HEADER_BETA = 'experimental-cc-routine-2026-04-01';
const DEFAULT_TEMPLATE = 'Scheduled trigger at {LocalTime}';
const DEFAULT_TZ = 'UTC';

/**
 * Substitutes `{Key}` tokens in template. Unknown tokens left intact and warn-logged.
 * @param {string} template
 * @param {Record<string, string>} vars
 * @returns {string}
 */
export function renderText(template, vars) {
  return template.replace(/\{(\w+)\}/g, (match, key) => {
    if (Object.prototype.hasOwnProperty.call(vars, key)) {
      return vars[key];
    }
    return match;
  });
}

/**
 * Formats a Date as `YYYY-MM-DD HH:mm ZZZ` in the given IANA timezone.
 * @param {Date} date
 * @param {string} tz
 * @returns {string}
 */
export function formatLocalTime(date, tz) {
  const fmt = new Intl.DateTimeFormat('en-CA', {
    timeZone: tz,
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', hour12: false,
    timeZoneName: 'short',
  });
  const parts = Object.fromEntries(fmt.formatToParts(date).map(p => [p.type, p.value]));
  return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute} ${parts.timeZoneName}`;
}

/**
 * POSTs to the Claude Code routine /fire endpoint. Logs structured JSON.
 * Never throws — network/HTTP failures are logged at error level so the
 * Worker doesn't crash and CF doesn't retry (each /fire = new session).
 *
 * @param {Env} env
 * @param {{ cron: string, scheduledTime: number }} controller
 * @returns {Promise<void>}
 */
export async function fireRoutine(env, controller) {
  const cron = controller.cron;
  const tz = env.TZ ?? DEFAULT_TZ;
  const template = env.TEXT_TEMPLATE ?? DEFAULT_TEMPLATE;
  const now = new Date(controller.scheduledTime);

  const text = renderText(template, {
    ISO: now.toISOString(),
    LocalTime: formatLocalTime(now, tz),
    Cron: cron,
  });

  let resp;
  try {
    resp = await fetch(env.ROUTINE_FIRE_URL, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${env.ROUTINE_FIRE_TOKEN}`,
        'anthropic-version': HEADER_VERSION,
        'anthropic-beta': HEADER_BETA,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ text }),
    });
  } catch (err) {
    console.log(JSON.stringify({
      level: 'error', msg: 'fire request failed', cron,
      err: err instanceof Error ? err.message : String(err),
    }));
    return;
  }

  const respBody = await resp.text();
  if (!resp.ok) {
    console.log(JSON.stringify({
      level: 'error', msg: 'fire non-2xx response',
      cron, status: resp.status, body: respBody,
    }));
    return;
  }

  /** @type {FireResponse | null} */
  let parsed = null;
  try {
    parsed = JSON.parse(respBody);
  } catch {
    console.log(JSON.stringify({
      level: 'warn', msg: 'fire 2xx but body unparseable',
      cron, status: resp.status, body: respBody,
    }));
    return;
  }

  console.log(JSON.stringify({
    level: 'info', msg: 'fire ok', cron,
    session_url: parsed?.claude_code_session_url,
    session_id: parsed?.claude_code_session_id,
  }));
}

/** @type {ExportedHandler<Env>} */
export default {
  async scheduled(controller, env, ctx) {
    ctx.waitUntil(fireRoutine(env, controller));
  },
};
