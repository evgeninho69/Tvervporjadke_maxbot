import { CoreApiClient, type CoreBotSession } from '../api/coreApiClient.js';
import type { BotFlow, BotSession, SessionDb } from '../bot/sessionTypes.js';

const coreApiClient = new CoreApiClient();

function nowTimestamp(): number {
  return Date.now();
}

function defaultSession(chatId: number): BotSession {
  return {
    chatId,
    flow: { step: 'idle' },
    updatedAt: nowTimestamp()
  };
}

function fromCore(session: CoreBotSession): BotSession {
  return {
    chatId: session.chat_id,
    maxUserId: session.max_user_id,
    maxLogin: session.max_login,
    consentAccepted: session.consent_accepted,
    consentAcceptedAt: session.consent_accepted_at,
    consentVersion: session.consent_version,
    flow: session.flow_json as BotFlow,
    updatedAt: session.updated_at
  };
}

function toCore(session: BotSession): CoreBotSession {
  return {
    chat_id: session.chatId,
    flow_json: session.flow as Record<string, unknown>,
    max_user_id: session.maxUserId,
    max_login: session.maxLogin,
    consent_accepted: session.consentAccepted,
    consent_accepted_at: session.consentAcceptedAt,
    consent_version: session.consentVersion,
    updated_at: session.updatedAt
  };
}

export class SessionStore {
  async get(chatId: number): Promise<BotSession> {
    const response = await coreApiClient.getBotSession(chatId);
    if (response.session) return fromCore(response.session);

    const session = defaultSession(chatId);
    await coreApiClient.saveBotSession(toCore(session));
    return session;
  }

  async save(session: BotSession): Promise<void> {
    session.updatedAt = nowTimestamp();
    await coreApiClient.saveBotSession(toCore(session));
  }

  async clear(chatId: number): Promise<BotSession> {
    const response = await coreApiClient.clearBotSession(chatId);
    return fromCore(response.session);
  }
}

export type { SessionDb };
