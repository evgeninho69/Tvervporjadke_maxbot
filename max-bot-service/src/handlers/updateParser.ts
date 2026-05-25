import type { MaxMessage, MaxUpdate, MaxUser } from '../types/max.js';

function asObject(value: unknown): Record<string, unknown> {
  return typeof value === 'object' && value !== null ? (value as Record<string, unknown>) : {};
}

function asNumber(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === 'string') {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }

  return undefined;
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined;
}

function extractCommandFromPayload(payload: unknown): string {
  if (typeof payload === 'string') {
    return payload.trim();
  }

  const data = asObject(payload);
  if (Object.keys(data).length === 0) {
    return '';
  }

  const direct =
    asString(data.command) ??
    asString(data.text) ??
    asString(data.action) ??
    asString(data.value) ??
    asString(data.payload);

  if (direct) {
    return direct.trim();
  }

  return '';
}

function normalizeSender(value: unknown): MaxUser | undefined {
  const data = asObject(value);
  if (Object.keys(data).length === 0) {
    return undefined;
  }

  return {
    user_id: asNumber(data.user_id),
    first_name: asString(data.first_name),
    last_name: asString(data.last_name),
    username: asString(data.username),
    name: asString(data.name)
  };
}

export function extractMessage(update: MaxUpdate): MaxMessage | null {
  const message = update.message ?? {};
  const callbackCommand = update.update_type === 'message_callback'
    ? extractCommandFromPayload(update.callback?.payload ?? update.payload)
    : '';

  const text = callbackCommand || message.body?.text || message.text || '';

  const chatId =
    message.recipient?.chat_id ??
    message.chat?.chat_id ??
    asNumber(update.chat_id);

  if (!chatId || chatId <= 0) {
    return null;
  }

  const sender = normalizeSender(message.sender ?? message.from ?? update.user);

  return {
    ...message,
    chat: { chat_id: chatId },
    recipient: {
      ...(message.recipient ?? {}),
      chat_id: chatId
    },
    text,
    from: sender ?? message.from,
    sender: sender ?? message.sender
  };
}

export function updateIdFor(update: MaxUpdate, message: MaxMessage): string {
  const id =
    update.update_id ||
    update.callback?.callback_id ||
    message.message_id ||
    message.body?.mid ||
    message.body?.seq;

  return id ? `max:${String(id)}` : '';
}

export function callbackIdFor(update: MaxUpdate): string {
  const callbackId = update.callback?.callback_id;
  return callbackId ? String(callbackId) : '';
}
