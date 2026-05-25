import express from 'express';
import { env } from './config/env.js';
import { setupApiV2 } from './routes/apiV2.js';
import { getRepository } from './storage/repository.js';

const app = express();
app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true }));

app.get('/healthz', (_req, res) => {
  res.status(200).json({ ok: true, service: 'core-api', storage: env.CORE_API_STORAGE, build_id: env.BUILD_ID });
});

setupApiV2(app);

const repository = getRepository();

repository
  .initialize()
  .then(() => {
    app.listen(env.PORT, () => {
      console.log(JSON.stringify({ level: 'info', message: 'core-api started', port: env.PORT, storage: env.CORE_API_STORAGE }));
    });
  })
  .catch((error) => {
    console.error(
      JSON.stringify({
        level: 'error',
        message: 'core-api initialization failed',
        error: error instanceof Error ? error.message : 'unknown_error'
      })
    );
    process.exit(1);
  });
