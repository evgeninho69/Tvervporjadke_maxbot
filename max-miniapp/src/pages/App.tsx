import { useEffect, useMemo, useState } from 'react';
import { ClaimForm } from '../components/ClaimForm';
import {
  getBootstrap,
  getClaimById,
  setConsent,
  setNotifications,
  readMaxIdentityFromUrl,
  type Announcement,
  type Category,
  type Claim,
  type ClaimStatus,
  type Profile
} from '../services/api';

type Tab = 'home' | 'history' | 'profile';

const STATUS_LABEL: Record<ClaimStatus, string> = {
  moderation: 'На модерации',
  in_work: 'В работе',
  resolved: 'Решено',
  rejected: 'Отклонено'
};

function statusClass(status: ClaimStatus): string {
  switch (status) {
    case 'moderation':
      return 'status-pill status-pill--moderation';
    case 'in_work':
      return 'status-pill status-pill--work';
    case 'resolved':
      return 'status-pill status-pill--resolved';
    case 'rejected':
      return 'status-pill status-pill--rejected';
    default:
      return 'status-pill';
  }
}

function formatDate(value?: string): string {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

function shortText(value: string, maxLength = 90): string {
  if (value.length <= maxLength) return value;
  return `${value.slice(0, maxLength)}...`;
}

function claimStatusSteps(status: ClaimStatus): string[] {
  if (status === 'rejected') {
    return ['Модерация', 'Рассмотрение', 'Отклонено'];
  }
  return ['Модерация', 'В работе', 'Решено'];
}

function claimStatusProgress(status: ClaimStatus): number {
  switch (status) {
    case 'moderation':
      return 0;
    case 'in_work':
      return 1;
    case 'resolved':
      return 2;
    case 'rejected':
      return 2;
    default:
      return 0;
  }
}

function buildSlaText(claim: Claim): string {
  if (!claim.created_at) return 'SLA: данные о времени создания отсутствуют.';

  const created = new Date(claim.created_at).getTime();
  if (!Number.isFinite(created)) return 'SLA: некорректная дата создания.';

  const deadline = created + 7 * 24 * 60 * 60 * 1000;
  const now = Date.now();
  const comparePoint = claim.status === 'resolved' || claim.status === 'rejected'
    ? new Date(claim.updated_at || claim.created_at).getTime()
    : now;

  if (!Number.isFinite(comparePoint)) {
    return 'SLA: ожидается обновление данных.';
  }

  const diffHours = Math.round(Math.abs(comparePoint - created) / (1000 * 60 * 60));

  if (claim.status === 'resolved' || claim.status === 'rejected') {
    return `Закрыто за ${diffHours} ч.`;
  }

  if (comparePoint > deadline) {
    const overdueHours = Math.round((comparePoint - deadline) / (1000 * 60 * 60));
    return `SLA превышен на ${overdueHours} ч.`;
  }

  const remainingHours = Math.round((deadline - comparePoint) / (1000 * 60 * 60));
  return `До SLA осталось ~${remainingHours} ч.`;
}

function claimMapUrl(claim: Claim): string | null {
  const { lat, lng } = claim.location;
  if (!Number.isFinite(lat) || !Number.isFinite(lng) || (lat === 0 && lng === 0)) {
    return null;
  }

  return `https://www.openstreetmap.org/?mlat=${lat}&mlon=${lng}#map=16/${lat}/${lng}`;
}

export function App() {
  const identity = useMemo(() => readMaxIdentityFromUrl(), []);

  const initialTab = ((): Tab => {
    if (identity.initialView === 'history') return 'history';
    if (identity.initialView === 'profile') return 'profile';
    return 'home';
  })();

  const initialFormOpen = identity.initialView === 'form';

  const [tab, setTab] = useState<Tab>(initialTab);
  const [isFormOpen, setIsFormOpen] = useState(initialFormOpen);

  const [profile, setProfile] = useState<Profile>({
    consent_accepted: false,
    notifications: { push_enabled: true }
  });
  const [categories, setCategories] = useState<Category[]>([]);
  const [claims, setClaims] = useState<Claim[]>([]);
  const [announcements, setAnnouncements] = useState<Announcement[]>([]);
  const [selectedClaim, setSelectedClaim] = useState<Claim | null>(null);

  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');

  const [historyFilter, setHistoryFilter] = useState<'all' | ClaimStatus>('all');

  const visibleClaims = useMemo(() => {
    if (historyFilter === 'all') return claims;
    return claims.filter((item) => item.status === historyFilter);
  }, [claims, historyFilter]);

  const latestClaims = useMemo(() => claims.slice(0, 3), [claims]);

  const claimStats = useMemo(() => {
    return claims.reduce(
      (acc, claim) => {
        acc.total += 1;
        acc[claim.status] += 1;
        return acc;
      },
      {
        total: 0,
        moderation: 0,
        in_work: 0,
        resolved: 0,
        rejected: 0
      }
    );
  }, [claims]);

  const loadBootstrap = async (): Promise<void> => {
    setLoading(true);
    setMessage('');

    try {
      const data = await getBootstrap();
      setProfile(data.profile);
      setCategories(data.categories || []);
      setClaims(data.claims || []);
      setAnnouncements(data.announcements || []);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Ошибка загрузки данных');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadBootstrap();
  }, []);

  const onCreated = async (claimId: number): Promise<void> => {
    setIsFormOpen(false);
    setTab('history');
    setMessage(`Спасибо. Ваше обращение №${claimId} отправлено.`);
    await loadBootstrap();
  };

  const openClaim = async (id: number): Promise<void> => {
    try {
      const response = await getClaimById(id);
      setSelectedClaim(response.claim);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Не удалось загрузить карточку обращения');
    }
  };

  const updateConsent = async (accepted: boolean): Promise<void> => {
    try {
      await setConsent(accepted);
      setProfile((prev) => ({ ...prev, consent_accepted: accepted }));
      setMessage(accepted ? 'Согласие сохранено.' : 'Согласие отклонено.');
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Не удалось обновить согласие');
    }
  };

  const updatePush = async (next: boolean): Promise<void> => {
    try {
      await setNotifications(next);
      setProfile((prev) => ({ ...prev, notifications: { push_enabled: next } }));
      setMessage('Настройки уведомлений обновлены.');
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Не удалось обновить уведомления');
    }
  };

  return (
    <div className='citizen-root'>
      <div className='role-note'>Citizen mini app</div>
      <section className='citizen-shell'>
        <header className='citizen-header'>
          <div>
            <p className='citizen-subtitle'>Тверь в порядке</p>
            <h1>
              Здравствуйте{identity.maxLogin ? `, ${identity.maxLogin}` : ''}
            </h1>
          </div>
          <button className='header-refresh' onClick={() => void loadBootstrap()} disabled={loading} type='button'>
            {loading ? '...' : '↻'}
          </button>
        </header>

        <main className='citizen-content'>
          {message ? <div className='inline-message'>{message}</div> : null}

          {tab === 'home' ? (
            <section className='panel-stack'>
              <button className='hero-action' onClick={() => setIsFormOpen(true)} type='button'>
                <span className='hero-action__icon'>＋</span>
                <span className='hero-action__title'>Сообщить о проблеме</span>
                <span className='hero-action__hint'>Целевой путь: меньше 1 минуты</span>
              </button>

              <section className='panel stats-panel'>
                <h2>Статистика обращений</h2>
                <div className='stats-grid'>
                  <div><strong>{claimStats.total}</strong><span>Всего</span></div>
                  <div><strong>{claimStats.in_work}</strong><span>В работе</span></div>
                  <div><strong>{claimStats.resolved}</strong><span>Решено</span></div>
                  <div><strong>{claimStats.moderation}</strong><span>Модерация</span></div>
                </div>
              </section>

              <section className='panel'>
                <div className='panel-title-row'>
                  <h2>Последние обращения</h2>
                  <button className='link-button' onClick={() => setTab('history')} type='button'>
                    Все
                  </button>
                </div>

                {latestClaims.length === 0 ? <p className='empty-state'>Пока нет обращений.</p> : null}

                {latestClaims.map((claim) => (
                  <button className='claim-card' key={claim.id} onClick={() => void openClaim(claim.id)} type='button'>
                    <div className='claim-card__row'>
                      <strong>{claim.category}</strong>
                      <span className={statusClass(claim.status)}>{STATUS_LABEL[claim.status]}</span>
                    </div>
                    <p>{shortText(claim.description)}</p>
                    <div className='claim-card__meta'>
                      <span>№{claim.id}</span>
                      <span>{formatDate(claim.created_at)}</span>
                    </div>
                  </button>
                ))}
              </section>

              <section className='panel'>
                <h2>Городские объявления</h2>
                {announcements.slice(0, 2).map((item) => (
                  <article key={item.id} className='announcement-item'>
                    <strong>{item.title}</strong>
                    <p>{shortText(item.description, 130)}</p>
                  </article>
                ))}
              </section>
            </section>
          ) : null}

          {tab === 'history' ? (
            <section className='panel-stack'>
              <section className='panel'>
                <div className='filter-row'>
                  <button className={historyFilter === 'all' ? 'chip chip--active' : 'chip'} onClick={() => setHistoryFilter('all')} type='button'>
                    Все
                  </button>
                  <button
                    className={historyFilter === 'moderation' ? 'chip chip--active' : 'chip'}
                    onClick={() => setHistoryFilter('moderation')}
                    type='button'
                  >
                    Модерация
                  </button>
                  <button className={historyFilter === 'in_work' ? 'chip chip--active' : 'chip'} onClick={() => setHistoryFilter('in_work')} type='button'>
                    В работе
                  </button>
                  <button className={historyFilter === 'resolved' ? 'chip chip--active' : 'chip'} onClick={() => setHistoryFilter('resolved')} type='button'>
                    Решено
                  </button>
                </div>

                {visibleClaims.length === 0 ? <p className='empty-state'>Ничего не найдено.</p> : null}

                {visibleClaims.map((claim) => (
                  <button className='claim-card' key={claim.id} onClick={() => void openClaim(claim.id)} type='button'>
                    <div className='claim-card__row'>
                      <strong>№{claim.id} · {claim.category}</strong>
                      <span className={statusClass(claim.status)}>{STATUS_LABEL[claim.status]}</span>
                    </div>
                    <p>{shortText(claim.description, 160)}</p>
                    <div className='claim-card__meta'>
                      <span>{claim.location.address_fallback || `${claim.location.lat.toFixed(5)}, ${claim.location.lng.toFixed(5)}`}</span>
                      <span>{formatDate(claim.updated_at)}</span>
                    </div>
                  </button>
                ))}
              </section>

              {selectedClaim ? (
                <section className='panel'>
                  <div className='panel-title-row'>
                    <h2>Карточка обращения №{selectedClaim.id}</h2>
                    <button className='link-button' onClick={() => setSelectedClaim(null)} type='button'>
                      Закрыть
                    </button>
                  </div>
                  <div className='claim-detail-status'>
                    <span className={statusClass(selectedClaim.status)}>{STATUS_LABEL[selectedClaim.status]}</span>
                  </div>

                  <div className='status-track'>
                    {claimStatusSteps(selectedClaim.status).map((step, index) => (
                      <div
                        className={index <= claimStatusProgress(selectedClaim.status) ? 'status-track__step status-track__step--done' : 'status-track__step'}
                        key={step}
                      >
                        <span>{step}</span>
                      </div>
                    ))}
                  </div>

                  <p className='muted-text'>{buildSlaText(selectedClaim)}</p>
                  <p>{selectedClaim.description}</p>
                  {selectedClaim.assignee ? <p className='muted-text'>Исполнитель: {selectedClaim.assignee}</p> : null}
                  <p className='muted-text'>Обновлено: {formatDate(selectedClaim.updated_at)}</p>

                  {selectedClaim.official_response ? (
                    <div className='official-response'>
                      <h3>Официальный ответ</h3>
                      <p>{selectedClaim.official_response}</p>
                    </div>
                  ) : null}

                  {claimMapUrl(selectedClaim) ? (
                    <a className='result-media-link' href={claimMapUrl(selectedClaim) || '#'} target='_blank' rel='noreferrer'>
                      Открыть точку на карте
                    </a>
                  ) : null}

                  {selectedClaim.result_media.length > 0 ? (
                    <div className='result-media-grid'>
                      {selectedClaim.result_media.map((item, index) => (
                        <a className='result-media-link' href={item.url} key={`${item.url}-${index}`} target='_blank' rel='noreferrer'>
                          Фото результата {index + 1}
                        </a>
                      ))}
                    </div>
                  ) : null}
                </section>
              ) : null}
            </section>
          ) : null}

          {tab === 'profile' ? (
            <section className='panel-stack'>
              <section className='panel'>
                <h2>Профиль</h2>
                <p className='muted-text'>MAX login: {profile.max_login || identity.maxLogin || 'не передан'}</p>
                <p className='muted-text'>Chat ID: {profile.chat_id || identity.chatId || 'не передан'}</p>
              </section>

              <section className='panel'>
                <h2>Согласие 152-ФЗ</h2>
                <p className='muted-text'>
                  {profile.consent_accepted
                    ? 'Согласие принято. Можно отправлять обращения.'
                    : 'Без согласия отправка обращений недоступна.'}
                </p>
                <div className='button-row'>
                  <button className='primary-button' onClick={() => void updateConsent(true)} type='button'>
                    Принять
                  </button>
                  <button className='ghost-button' onClick={() => void updateConsent(false)} type='button'>
                    Отклонить
                  </button>
                </div>
              </section>

              <section className='panel'>
                <h2>Уведомления</h2>
                <label className='switch-row'>
                  <span>Получать push о смене статуса</span>
                  <input
                    checked={profile.notifications.push_enabled}
                    onChange={(event) => void updatePush(event.target.checked)}
                    type='checkbox'
                  />
                </label>
              </section>
            </section>
          ) : null}
        </main>

        <nav className='citizen-tabbar'>
          <button className={tab === 'home' ? 'tabbar-button tabbar-button--active' : 'tabbar-button'} onClick={() => setTab('home')} type='button'>
            Главная
          </button>
          <button className={tab === 'history' ? 'tabbar-button tabbar-button--active' : 'tabbar-button'} onClick={() => setTab('history')} type='button'>
            История
          </button>
          <button className={tab === 'profile' ? 'tabbar-button tabbar-button--active' : 'tabbar-button'} onClick={() => setTab('profile')} type='button'>
            Профиль
          </button>
        </nav>
      </section>

      {isFormOpen ? <ClaimForm categories={categories} consentAccepted={profile.consent_accepted} onCreated={(id) => void onCreated(id)} onCancel={() => setIsFormOpen(false)} /> : null}
    </div>
  );
}
