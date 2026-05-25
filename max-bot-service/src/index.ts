import express from 'express';
import { env } from './config/env.js';
import { handleWebhook } from './handlers/webhookHandler.js';
import { startPolling } from './handlers/pollingWorker.js';
import { log } from './utils/log.js';
import { startNotificationWorker } from './handlers/notificationWorker.js';
import { MaxClient } from './api/maxClient.js';

const app = express();
app.use(express.json({ limit: '1mb' }));
app.use(express.urlencoded({ extended: true }));

app.get('/healthz', (_req, res) => {
  res.status(200).json({ ok: true, service: 'max-bot-service', mode: env.MAX_MODE, build_id: env.BUILD_ID });
});

app.post('/integrations/max/webhook', async (req, res) => {
  try {
    await handleWebhook(req.headers, req.body);
    res.status(200).json({ ok: true });
  } catch (error) {
    log.error('Webhook handling failed', {
      error: error instanceof Error ? error.message : 'unknown_error'
    });
    res.status(401).json({ ok: false });
  }
});

app.listen(env.PORT, () => {
  log.info('MAX bot service started', { port: env.PORT });
});

if ((env.MAX_MODE === 'webhook' || env.MAX_MODE === 'hybrid') && env.MAX_WEBHOOK_URL) {
  new MaxClient().setWebhook(env.MAX_WEBHOOK_URL).then(() => {
    log.info('MAX webhook subscription registered', { url: env.MAX_WEBHOOK_URL });
  }).catch((error) => {
    log.error('MAX webhook registration failed', {
      error: error instanceof Error ? error.message : 'unknown_error'
    });
  });
}

if (env.MAX_MODE === 'polling' || env.MAX_MODE === 'hybrid') {
  startPolling().catch((error) => {
    log.error('Polling worker crashed', {
      error: error instanceof Error ? error.message : 'unknown_error'
    });
  });
}

startNotificationWorker().catch((error) => {
  log.error('Notification worker crashed', {
    error: error instanceof Error ? error.message : 'unknown_error'
  });
});
