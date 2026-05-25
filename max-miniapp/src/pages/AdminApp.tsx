import { useEffect, useMemo, useState } from 'react';
import {
  activateInvite,
  assignClaim,
  clearToken,
  createInvite,
  getAdmins,
  getClaim,
  getDashboard,
  getInvites,
  getMe,
  getQueue,
  loginAdmin,
  saveToken,
  setAdminActive,
  updateAdminRole,
  updateClaimStatus,
  type AdminClaim,
  type AdminRole,
  type AdminUser,
  type ClaimStatus,
  type DashboardResponse,
  type Invite
} from '../services/adminApi';

type View = 'dash' | 'queue' | 'admins';

type QueueFilter = '' | ClaimStatus;

const STATUS_OPTIONS: Array<{ value: ClaimStatus; label: string }> = [
  { value: 'moderation', label: 'На модерации' },
  { value: 'in_work', label: 'В работе' },
  { value: 'resolved', label: 'Решено' },
  { value: 'rejected', label: 'Отклонено' }
];

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

function buildPolyline(values: number[], width: number, height: number, padding: number): string {
  const maxValue = Math.max(...values, 1);
  const innerWidth = width - padding * 2;
  const innerHeight = height - padding * 2;

  return values
    .map((value, index) => {
      const x = padding + (innerWidth * index) / Math.max(1, values.length - 1);
      const y = padding + innerHeight - (value / maxValue) * innerHeight;
      return `${x},${y}`;
    })
    .join(' ');
}

function TimelineChart({ timeline }: { timeline: DashboardResponse['timeline'] | undefined }) {
  const points = timeline || [];
  if (points.length === 0) {
    return <p className='empty-state'>Нет данных для построения графика.</p>;
  }

  const width = 760;
  const height = 220;
  const padding = 26;
  const createdPolyline = buildPolyline(points.map((item) => item.created), width, height, padding);
  const resolvedPolyline = buildPolyline(points.map((item) => item.resolved), width, height, padding);

  return (
    <div className='timeline-chart'>
      <svg viewBox={`0 0 ${width} ${height}`} role='img' aria-label='Динамика обращений'>
        <polyline className='timeline-chart__created' fill='none' points={createdPolyline} />
        <polyline className='timeline-chart__resolved' fill='none' points={resolvedPolyline} />
      </svg>
      <div className='timeline-chart__labels' style={{ gridTemplateColumns: `repeat(${points.length}, minmax(0, 1fr))` }}>
        {points.map((point) => (
          <span key={point.day}>{point.day}</span>
        ))}
      </div>
      <div className='timeline-chart__legend'>
        <span><i className='dot dot--created' />Новые</span>
        <span><i className='dot dot--resolved' />Решённые</span>
      </div>
    </div>
  );
}

export function AdminApp() {
  const [phone, setPhone] = useState('');
  const [inviteCode, setInviteCode] = useState('');

  const [authorized, setAuthorized] = useState(false);
  const [admin, setAdmin] = useState<AdminUser | null>(null);
  const [view, setView] = useState<View>('dash');

  const [dashboard, setDashboard] = useState<DashboardResponse | null>(null);
  const [queue, setQueue] = useState<AdminClaim[]>([]);
  const [selectedClaim, setSelectedClaim] = useState<AdminClaim | null>(null);

  const [admins, setAdmins] = useState<AdminUser[]>([]);
  const [invites, setInvites] = useState<Invite[]>([]);
  const [newInvitePhone, setNewInvitePhone] = useState('');
  const [newInviteRole, setNewInviteRole] = useState<AdminRole>('operator');
  const [queueFilter, setQueueFilter] = useState<QueueFilter>('');

  const [nextStatus, setNextStatus] = useState<ClaimStatus>('moderation');
  const [officialResponse, setOfficialResponse] = useState('');
  const [assignee, setAssignee] = useState('');

  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');

  const canManageAdmins = admin?.role === 'super_admin';

  const dashboardCards = useMemo(() => {
    const stats = dashboard?.stats || {};
    const sla = dashboard?.sla;
    const delivery = dashboard?.delivery;
    return [
      { label: 'Всего обращений', value: stats.total || 0 },
      { label: 'На модерации', value: stats.moderation || 0 },
      { label: 'В работе', value: stats.in_work || 0 },
      { label: 'Решено', value: stats.resolved || 0 },
      { label: 'Отклонено', value: stats.rejected || 0 },
      { label: 'Нарушение SLA > 7 дней', value: sla?.unresolved_over_7_days ?? dashboard?.unresolved_over_7_days ?? 0 },
      { label: 'SLA истекает сегодня', value: sla?.due_today ?? 0 },
      { label: 'P95 решения, ч', value: sla?.p95_resolution_hours ?? '—' },
      { label: 'Outbox в очереди', value: delivery?.pending ?? 0 },
      { label: 'DLQ уведомлений', value: delivery?.dead ?? 0 }
    ];
  }, [dashboard]);

  const refreshAll = async (): Promise<void> => {
    setLoading(true);
    setMessage('');

    try {
      const [me, dash, queueResp, adminsResp, invitesResp] = await Promise.all([
        getMe(),
        getDashboard(),
        getQueue(queueFilter || undefined),
        getAdmins(),
        getInvites()
      ]);

      setAdmin(me.admin);
      setDashboard(dash);
      setQueue(queueResp.claims);
      setAdmins(adminsResp.admins);
      setInvites(invitesResp.invites);

      if (!selectedClaim && queueResp.claims[0]) {
        const first = queueResp.claims[0];
        setSelectedClaim(first);
        setNextStatus(first.status);
        setOfficialResponse(first.official_response || '');
        setAssignee(first.assignee || '');
      }
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Ошибка загрузки');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void (async () => {
      try {
        const me = await getMe();
        setAuthorized(true);
        setAdmin(me.admin);
        await refreshAll();
      } catch {
        setAuthorized(false);
      }
    })();
  }, []);

  useEffect(() => {
    if (!authorized) return;
    void refreshAll();
  }, [queueFilter]);

  const onLogin = async (): Promise<void> => {
    try {
      const response = await loginAdmin(phone);
      saveToken(response.token);
      setAuthorized(true);
      setAdmin(response.admin);
      await refreshAll();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Ошибка входа');
    }
  };

  const onActivateInvite = async (): Promise<void> => {
    try {
      await activateInvite(inviteCode, phone);
      setMessage('Приглашение активировано. Выполните вход по телефону.');
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Ошибка активации приглашения');
    }
  };

  const logout = (): void => {
    clearToken();
    setAuthorized(false);
    setAdmin(null);
  };

  const openClaim = async (id: number): Promise<void> => {
    try {
      const data = await getClaim(id);
      setSelectedClaim(data.claim);
      setNextStatus(data.claim.status);
      setOfficialResponse(data.claim.official_response || '');
      setAssignee(data.claim.assignee || '');
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Ошибка загрузки обращения');
    }
  };

  const quickStatusUpdate = async (claim: AdminClaim, status: ClaimStatus): Promise<void> => {
    try {
      await updateClaimStatus({
        claimId: claim.id,
        status,
        officialResponse: claim.official_response || '',
        assignee: claim.assignee || ''
      });

      await refreshAll();
      if (selectedClaim?.id === claim.id) {
        await openClaim(claim.id);
      }
      setMessage(`Заявка №${claim.id} переведена в статус «${STATUS_OPTIONS.find((item) => item.value === status)?.label || status}».`);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Не удалось обновить статус');
    }
  };

  const saveClaimStatus = async (): Promise<void> => {
    if (!selectedClaim) return;

    try {
      await updateClaimStatus({
        claimId: selectedClaim.id,
        status: nextStatus,
        officialResponse,
        assignee
      });

      if (assignee.trim()) {
        await assignClaim(selectedClaim.id, assignee.trim());
      }

      await refreshAll();
      await openClaim(selectedClaim.id);
      setMessage('Карточка обращения обновлена.');
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Ошибка сохранения');
    }
  };

  const createInviteAction = async (): Promise<void> => {
    try {
      const result = await createInvite(newInvitePhone, newInviteRole);
      setInviteCode(result.invite.code);
      await refreshAll();
      setMessage(`Инвайт создан: ${result.invite.code}`);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Ошибка создания инвайта');
    }
  };

  if (!authorized) {
    return (
      <div className='admin-auth-root'>
        <section className='admin-auth-card'>
          <h1>Тверь Контроль</h1>
          <p>Вход в админку по номеру телефона.</p>
          <input className='field-input' value={phone} onChange={(event) => setPhone(event.target.value)} placeholder='+79101234567' />
          <button className='primary-button' onClick={() => void onLogin()} type='button'>
            Войти
          </button>

          <div className='auth-subcard'>
            <h2>Активация приглашения</h2>
            <input className='field-input' value={inviteCode} onChange={(event) => setInviteCode(event.target.value)} placeholder='Код приглашения' />
            <button className='ghost-button' onClick={() => void onActivateInvite()} type='button'>
              Активировать
            </button>
          </div>

          {message ? <div className='inline-message'>{message}</div> : null}
        </section>
      </div>
    );
  }

  return (
    <div className='admin-root'>
      <aside className='admin-sidebar'>
        <div>
          <div className='admin-brand'>
            <h1>Тверь<br />Контроль</h1>
            <p>{admin?.phone}</p>
          </div>
          <nav className='admin-nav'>
            <button className={view === 'dash' ? 'admin-nav__item admin-nav__item--active' : 'admin-nav__item'} onClick={() => setView('dash')} type='button'>
              Дашборд
            </button>
            <button className={view === 'queue' ? 'admin-nav__item admin-nav__item--active' : 'admin-nav__item'} onClick={() => setView('queue')} type='button'>
              Очередь
            </button>
            <button className={view === 'admins' ? 'admin-nav__item admin-nav__item--active' : 'admin-nav__item'} onClick={() => setView('admins')} type='button'>
              Права и инвайты
            </button>
          </nav>
        </div>

        <div className='admin-sidebar__footer'>
          <button className='ghost-button' onClick={() => void refreshAll()} disabled={loading} type='button'>
            {loading ? 'Обновление...' : 'Обновить'}
          </button>
          <button className='ghost-button' onClick={logout} type='button'>
            Выйти
          </button>
        </div>
      </aside>

      <main className='admin-main'>
        <header className='admin-main__header'>
          <h2>{view === 'dash' ? 'Аналитика' : view === 'queue' ? 'Модерация обращений' : 'Управление доступом'}</h2>
          <span className='status-pill status-pill--work'>Роль: {admin?.role}</span>
        </header>

        <section className='admin-main__content'>
          {message ? <div className='inline-message'>{message}</div> : null}

          {view === 'dash' ? (
            <div className='dashboard-grid'>
              {dashboardCards.map((card) => (
                <article className='dashboard-card' key={card.label}>
                  <p>{card.label}</p>
                  <strong>{card.value}</strong>
                </article>
              ))}

              <article className='dashboard-card dashboard-card--wide'>
                <h3>Динамика обращений</h3>
                <TimelineChart timeline={dashboard?.timeline} />
              </article>

              <article className='dashboard-card dashboard-card--wide'>
                <h3>Категории проблем</h3>
                <div className='simple-bars'>
                  {(dashboard?.category_stats || []).map((item) => (
                    <div key={item.category} className='simple-bars__row'>
                      <span>{item.category}</span>
                      <div className='simple-bars__track'>
                        <div
                          className='simple-bars__fill'
                          style={{
                            width: `${Math.min(100, Math.round((item.count / Math.max(1, dashboard?.stats?.total || 1)) * 100))}%`
                          }}
                        />
                      </div>
                      <strong>{item.count}</strong>
                    </div>
                  ))}
                </div>
              </article>

              <article className='dashboard-card dashboard-card--wide'>
                <h3>Лента событий</h3>
                <div className='feed-list'>
                  {(dashboard?.latest_feed || []).slice(0, 8).map((event) => (
                    <div className='feed-row' key={event.id}>
                      <p>{event.text}</p>
                      <span>{formatDate(event.created_at)}</span>
                    </div>
                  ))}
                </div>
              </article>
            </div>
          ) : null}

          {view === 'queue' ? (
            <div className='queue-layout'>
              <section className='queue-table-wrap'>
                <div className='queue-toolbar'>
                  <label className='field-label'>Фильтр очереди</label>
                  <select className='field-input' value={queueFilter} onChange={(event) => setQueueFilter(event.target.value as QueueFilter)}>
                    <option value=''>Все статусы</option>
                    {STATUS_OPTIONS.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </div>

                <table className='queue-table'>
                  <thead>
                    <tr>
                      <th>ID</th>
                      <th>Категория</th>
                      <th>Описание</th>
                      <th>Статус</th>
                      <th>Обновлено</th>
                      <th>Действия</th>
                    </tr>
                  </thead>
                  <tbody>
                    {queue.map((claim) => (
                      <tr key={claim.id} onClick={() => void openClaim(claim.id)}>
                        <td>#{claim.id}</td>
                        <td>{claim.category}</td>
                        <td>{claim.description.slice(0, 90)}</td>
                        <td>
                          <span className={statusClass(claim.status)}>{claim.status_label}</span>
                        </td>
                        <td>{formatDate(claim.updated_at)}</td>
                        <td>
                          <div className='queue-actions' onClick={(event) => event.stopPropagation()}>
                            {claim.status === 'moderation' ? (
                              <button className='ghost-button queue-actions__btn' onClick={() => void quickStatusUpdate(claim, 'in_work')} type='button'>
                                В работу
                              </button>
                            ) : null}
                            {claim.status === 'in_work' ? (
                              <button className='primary-button queue-actions__btn' onClick={() => void quickStatusUpdate(claim, 'resolved')} type='button'>
                                Решено
                              </button>
                            ) : null}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </section>

              <section className='queue-detail'>
                {!selectedClaim ? <p className='empty-state'>Выберите заявку из таблицы.</p> : null}
                {selectedClaim ? (
                  <>
                    <h3>Заявка №{selectedClaim.id}</h3>
                    <span className={statusClass(selectedClaim.status)}>{selectedClaim.status_label}</span>
                    <p>{selectedClaim.description}</p>
                    <p className='muted-text'>Локация: {selectedClaim.location.address_fallback || `${selectedClaim.location.lat}, ${selectedClaim.location.lng}`}</p>

                    <label className='field-label'>Назначить исполнителя</label>
                    <input className='field-input' value={assignee} onChange={(event) => setAssignee(event.target.value)} placeholder='ФИО или отдел' />

                    <label className='field-label'>Новый статус</label>
                    <select className='field-input' value={nextStatus} onChange={(event) => setNextStatus(event.target.value as ClaimStatus)}>
                      {STATUS_OPTIONS.map((option) => (
                        <option key={option.value} value={option.value}>
                          {option.label}
                        </option>
                      ))}
                    </select>

                    <label className='field-label'>Официальный ответ</label>
                    <textarea
                      className='field-textarea'
                      value={officialResponse}
                      onChange={(event) => setOfficialResponse(event.target.value)}
                      placeholder='Комментарий для жителя'
                    />

                    <button className='primary-button' onClick={() => void saveClaimStatus()} type='button'>
                      Сохранить изменения
                    </button>
                  </>
                ) : null}
              </section>
            </div>
          ) : null}

          {view === 'admins' ? (
            <div className='admins-layout'>
              {canManageAdmins ? (
                <section className='panel'>
                  <h3>Создать приглашение</h3>
                  <input className='field-input' value={newInvitePhone} onChange={(event) => setNewInvitePhone(event.target.value)} placeholder='+79101234567' />
                  <select className='field-input' value={newInviteRole} onChange={(event) => setNewInviteRole(event.target.value as AdminRole)}>
                    <option value='operator'>operator</option>
                    <option value='admin'>admin</option>
                    <option value='super_admin'>super_admin</option>
                  </select>
                  <button className='primary-button' onClick={() => void createInviteAction()} type='button'>
                    Создать инвайт
                  </button>
                  {inviteCode ? <div className='inline-message'>Последний код: {inviteCode}</div> : null}
                </section>
              ) : null}

              <section className='panel'>
                <h3>Администраторы</h3>
                <div className='admin-list'>
                  {admins.map((item) => (
                    <article className='admin-list__item' key={item.id}>
                      <div>
                        <strong>{item.phone}</strong>
                        <p className='muted-text'>{item.role} · {item.active ? 'active' : 'inactive'}</p>
                      </div>

                      {canManageAdmins ? (
                        <div className='admin-list__controls'>
                          <select className='field-input' defaultValue={item.role} onChange={(event) => void updateAdminRole(item.id, event.target.value as AdminRole).then(refreshAll)}>
                            <option value='operator'>operator</option>
                            <option value='admin'>admin</option>
                            <option value='super_admin'>super_admin</option>
                          </select>
                          <button className='ghost-button' onClick={() => void setAdminActive(item.id, !item.active).then(refreshAll)} type='button'>
                            {item.active ? 'Отключить' : 'Включить'}
                          </button>
                        </div>
                      ) : null}
                    </article>
                  ))}
                </div>
              </section>

              <section className='panel'>
                <h3>История инвайтов</h3>
                <div className='feed-list'>
                  {invites.map((invite) => (
                    <div className='feed-row' key={invite.code}>
                      <p>
                        {invite.phone} · {invite.role} · {invite.code}
                      </p>
                      <span>{invite.used_at ? `использован ${formatDate(invite.used_at)}` : 'не использован'}</span>
                    </div>
                  ))}
                </div>
              </section>
            </div>
          ) : null}
        </section>
      </main>
    </div>
  );
}
