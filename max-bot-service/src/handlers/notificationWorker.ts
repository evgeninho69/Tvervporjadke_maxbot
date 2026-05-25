import { CoreApiClient } from '../api/coreApiClient.js';
import { MaxClient } from '../api/maxClient.js';
import { env } from '../config/env.js';
import { log } from '../utils/log.js';

const coreApiClient = new CoreApiClient();
const maxClient = new MaxClient();

type ClaimStatusChangedPayload = {
  claim_id?: number;
  chat_id?: number;
  status?: string;
  status_label?: string;
  category?: string;
  description?: string;
  official_response?: string | null;
  result_media?: string[];
};

function textForClaimStatus(payload: ClaimStatusChangedPayload): string {
  const claimId = payload.claim_id || 0;
  const statusLabel = payload.status_label || payload.status || 'обновлен';
  const lines = [`Обращение №${claimId}: ${statusLabel}.`];

  if (payload.category) {
    lines.push(`Категория: ${payload.category}.`);
  }

  if (payload.official_response) {
    lines.push(`Ответ администрации: ${payload.official_response}`);
  }

  if (payload.status === 'resolved') {
    lines.push('Оцените результат по шкале от 1 до 5: ⭐⭐⭐⭐⭐');
  }

  lines.push('Открыть мини-приложение: ' + env.MAX_MINIAPP_URL);
  return lines.join('\n');
}

async function processEvent(event: { id: number; type: string; payload: Record<string, unknown> }): Promise<void> {
  if (event.type !== 'claim_status_changed') {
    await coreApiClient.ackEvent(event.id);
    return;
  }

  const payload = event.payload as ClaimStatusChangedPayload;
  const chatId = Number(payload.chat_id || 0);
  if (!chatId || !Number.isFinite(chatId)) {
    await coreApiClient.ackEvent(event.id);
    return;
  }

  const text = textForClaimStatus(payload);
  await maxClient.sendMessage({ chat_id: chatId, text });
  await coreApiClient.ackEvent(event.id);

  coreApiClient.addFeedEvent({
    type: 'status_notification',
    text: `Уведомление отправлено по заявке #${payload.claim_id || '?'}: ${payload.status || 'unknown'}`,
    chatId,
    claimId: Number(payload.claim_id || 0) || undefined,
    payload
  }).catch((error) => {
    log.warn('Failed to write status notification feed event', {
      eventId: event.id,
      error: error instanceof Error ? error.message : 'unknown_error'
    });
  });
}

export async function startNotificationWorker(): Promise<void> {
  log.info('Notification worker started', { pollIntervalMs: env.CORE_API_EVENTS_POLL_MS });

  while (true) {
    try {
      const pending = await coreApiClient.getPendingEvents(25);
      for (const event of pending.events || []) {
        try {
          await processEvent(event);
        } catch (error) {
          const message = error instanceof Error ? error.message : 'unknown_error';
          log.error('Failed to process domain event', {
            eventId: event.id,
            type: event.type,
            attempts: event.attempts,
            error: message
          });

          await coreApiClient.failEvent(event.id, message).catch((failError) => {
            log.error('Failed to mark domain event as failed', {
              eventId: event.id,
              error: failError instanceof Error ? failError.message : 'unknown_error'
            });
          });
        }
      }
    } catch (error) {
      log.error('Notification polling failed', {
        error: error instanceof Error ? error.message : 'unknown_error'
      });
    }

    await new Promise((resolve) => setTimeout(resolve, env.CORE_API_EVENTS_POLL_MS));
  }
}
