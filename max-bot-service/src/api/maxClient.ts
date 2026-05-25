import { env } from '../config/env.js';
import type { MaxSendMessagePayload, MaxUpdatesResponse } from '../types/max.js';

export class MaxClient {
  private readonly baseUrl: string;
  private readonly token: string;

  constructor() {
    this.baseUrl = env.MAX_API_BASE_URL;
    this.token = env.MAX_BOT_TOKEN;
  }

  private async withRetry<T>(fn: () => Promise<T>, attempts = 3): Promise<T> {
    let lastError: unknown;
    for (let i = 0; i < attempts; i++) {
      try {
        return await fn();
      } catch (error) {
        lastError = error;
        if (i < attempts - 1) {
          await new Promise((resolve) => setTimeout(resolve, 250 * (i + 1)));
        }
      }
    }
    throw lastError;
  }

  private async request<T>(path: string, init?: RequestInit): Promise<T> {
    const response = await this.withRetry(() =>
      fetch(`${this.baseUrl}${path}`, {
        ...init,
        headers: {
          Authorization: this.token,
          'Content-Type': 'application/json',
          ...(init?.headers ?? {})
        }
      })
    );

    if (!response.ok) {
      const body = await response.text();
      throw new Error(`MAX API error ${response.status}: ${body}`);
    }

    if (response.status === 204) {
      return {} as T;
    }

    return (await response.json()) as T;
  }

  async sendMessage(payload: MaxSendMessagePayload): Promise<void> {
    await this.request(`/messages?chat_id=${encodeURIComponent(String(payload.chat_id))}`, {
      method: 'POST',
      body: JSON.stringify({
        text: payload.text,
        ...(payload.attachments ? { attachments: payload.attachments } : {})
      })
    });
  }

  async getUpdates(marker?: number): Promise<MaxUpdatesResponse> {
    const parts = ['types=message_created,message_callback,bot_started', 'timeout=25', 'limit=100'];
    if (marker !== undefined) {
      parts.push(`marker=${encodeURIComponent(String(marker))}`);
    }
    const query = `?${parts.join('&')}`;
    return this.request<MaxUpdatesResponse>(`/updates${query}`, {
      method: 'GET'
    });
  }

  async setWebhook(url: string): Promise<void> {
    await this.request('/subscriptions', {
      method: 'POST',
      body: JSON.stringify({
        type: 'Webhook',
        url,
        secret: env.MAX_WEBHOOK_SECRET,
        update_types: ['message_created', 'message_callback', 'bot_started']
      })
    });
  }

  async answerCallback(callbackId: string, notification = 'Принято'): Promise<void> {
    await this.request(`/answers?callback_id=${encodeURIComponent(callbackId)}`, {
      method: 'POST',
      body: JSON.stringify({
        notification
      })
    });
  }
}
