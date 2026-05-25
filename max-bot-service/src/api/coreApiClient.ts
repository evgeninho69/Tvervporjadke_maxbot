import { env } from '../config/env.js';

type BotFeedEventType = 'incoming_message' | 'claim_created' | 'claim_updated' | 'status_notification';

type PendingDomainEvent = {
  id: number;
  type: string;
  payload: Record<string, unknown>;
  attempts?: number;
  status?: string;
  last_error?: string;
};

type ClaimStatusPayload = {
  claim: {
    id: number;
    status: string;
    status_label: string;
    category?: string;
    updated_at?: string;
    official_response?: string | null;
  };
};

export type CoreBotSession = {
  chat_id: number;
  flow_json: Record<string, unknown>;
  max_user_id?: number;
  max_login?: string;
  consent_accepted?: boolean;
  consent_accepted_at?: number;
  consent_version?: string;
  updated_at: number;
};

function trimSlash(value: string): string {
  return value.endsWith('/') ? value.slice(0, -1) : value;
}

export class CoreApiClient {
  private readonly baseUrl: string;
  private readonly internalToken: string;

  constructor() {
    this.baseUrl = trimSlash(env.CORE_API_BASE_URL);
    this.internalToken = env.CORE_API_INTERNAL_TOKEN;
  }

  private async request<T>(path: string, init?: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...init,
      headers: {
        'Content-Type': 'application/json',
        'x-internal-token': this.internalToken,
        ...(init?.headers || {})
      }
    });

    if (!response.ok) {
      const body = await response.text();
      throw new Error(`core-api ${path} failed: HTTP ${response.status} ${body}`);
    }

    if (response.status === 204) {
      return {} as T;
    }

    return (await response.json()) as T;
  }

  async bindAdminChat(phone: string, chatId: number, maxLogin?: string): Promise<{ admin: { phone: string; role: string } }> {
    return this.request<{ admin: { phone: string; role: string } }>('/internal/v2/bot/admin-bind', {
      method: 'POST',
      body: JSON.stringify({
        phone,
        chat_id: chatId,
        ...(maxLogin ? { max_login: maxLogin } : {})
      })
    });
  }

  async addFeedEvent(params: {
    type: BotFeedEventType;
    text: string;
    chatId?: number;
    claimId?: number;
    payload?: Record<string, unknown>;
  }): Promise<void> {
    await this.request('/internal/v2/bot/feed-event', {
      method: 'POST',
      body: JSON.stringify({
        type: params.type,
        text: params.text,
        chat_id: params.chatId,
        claim_id: params.claimId,
        payload: params.payload
      })
    });
  }

  async claimIncomingUpdate(params: { updateId: string; source: string; chatId?: number; messageId?: string }): Promise<{ claimed: boolean }> {
    return this.request<{ claimed: boolean }>('/internal/v2/bot/incoming-updates/claim', {
      method: 'POST',
      body: JSON.stringify({
        update_id: params.updateId,
        source: params.source,
        chat_id: params.chatId,
        message_id: params.messageId
      })
    });
  }

  async completeIncomingUpdate(params: { updateId: string; result: 'processed' | 'failed'; error?: string }): Promise<void> {
    await this.request('/internal/v2/bot/incoming-updates/complete', {
      method: 'POST',
      body: JSON.stringify({
        update_id: params.updateId,
        result: params.result,
        error: params.error
      })
    });
  }

  async getLastClaimStatus(chatId: number, claimId?: number): Promise<ClaimStatusPayload> {
    const query = new URLSearchParams({ chat_id: String(chatId) });
    if (claimId && Number.isFinite(claimId) && claimId > 0) {
      query.set('claim_id', String(claimId));
    }

    return this.request<ClaimStatusPayload>(`/internal/v2/bot/last-claim-status?${query.toString()}`, { method: 'GET' });
  }

  async getPendingEvents(limit = 20): Promise<{ events: PendingDomainEvent[] }> {
    const query = new URLSearchParams({ limit: String(limit) });
    return this.request<{ events: PendingDomainEvent[] }>(`/internal/v2/events/pending?${query.toString()}`, { method: 'GET' });
  }

  async ackEvent(eventId: number): Promise<void> {
    await this.request(`/internal/v2/events/${eventId}/ack`, { method: 'POST' });
  }

  async failEvent(eventId: number, error: string): Promise<void> {
    await this.request(`/internal/v2/events/${eventId}/fail`, {
      method: 'POST',
      body: JSON.stringify({ error })
    });
  }

  async getBotSession(chatId: number): Promise<{ session: CoreBotSession | null }> {
    return this.request<{ session: CoreBotSession | null }>(`/internal/v2/bot/sessions/${chatId}`, { method: 'GET' });
  }

  async saveBotSession(session: CoreBotSession): Promise<{ session: CoreBotSession }> {
    return this.request<{ session: CoreBotSession }>(`/internal/v2/bot/sessions/${session.chat_id}`, {
      method: 'PUT',
      body: JSON.stringify(session)
    });
  }

  async clearBotSession(chatId: number): Promise<{ session: CoreBotSession }> {
    return this.request<{ session: CoreBotSession }>(`/internal/v2/bot/sessions/${chatId}`, {
      method: 'DELETE'
    });
  }
}
