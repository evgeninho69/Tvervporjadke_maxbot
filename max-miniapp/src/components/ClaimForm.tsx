import { useEffect, useMemo, useState } from 'react';
import L from 'leaflet';
import { MapContainer, Marker, TileLayer, useMap, useMapEvents } from 'react-leaflet';
import { createClaim, type Category } from '../services/api';

type Props = {
  categories: Category[];
  consentAccepted: boolean;
  onCreated: (id: number) => void;
  onCancel: () => void;
};

const MAX_DESCRIPTION = 1000;
const MAX_MEDIA = 3;

const QUICK_TEMPLATES = [
  {
    label: 'Яма на дороге',
    category: 'Дороги',
    description: 'Обнаружена глубокая яма на проезжей части. Требуется срочный ремонт покрытия.'
  },
  {
    label: 'Не работает свет',
    category: 'Транспорт',
    description: 'Не работает уличное освещение. В тёмное время участок плохо освещён и небезопасен.'
  },
  {
    label: 'Мусор во дворе',
    category: 'ЖКХ',
    description: 'Переполнена контейнерная площадка. Просьба организовать вывоз мусора и уборку территории.'
  }
] as const;

L.Icon.Default.mergeOptions({
  iconRetinaUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon-2x.png',
  iconUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon.png',
  shadowUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-shadow.png'
});

function MapClickHandler({ onPick }: { onPick: (lat: number, lng: number) => void }) {
  useMapEvents({
    click: (event) => {
      onPick(event.latlng.lat, event.latlng.lng);
    }
  });

  return null;
}

function MapViewport({ lat, lng }: { lat: number; lng: number }) {
  const map = useMap();

  useEffect(() => {
    map.setView([lat, lng], map.getZoom(), { animate: false });
  }, [lat, lng, map]);

  return null;
}

function toDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ''));
    reader.onerror = () => reject(new Error('Не удалось прочитать файл'));
    reader.readAsDataURL(file);
  });
}

export function ClaimForm({ categories, consentAccepted, onCreated, onCancel }: Props) {
  const [category, setCategory] = useState('');
  const [description, setDescription] = useState('');
  const [latitude, setLatitude] = useState('56.85836');
  const [longitude, setLongitude] = useState('35.90057');
  const [addressFallback, setAddressFallback] = useState('');
  const [files, setFiles] = useState<File[]>([]);
  const [showMap, setShowMap] = useState(false);
  const [previewMode, setPreviewMode] = useState(false);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');

  const categoryOptions = useMemo(() => categories.map((item) => item.label), [categories]);
  const mapCenter = useMemo<[number, number]>(() => {
    const lat = Number(latitude);
    const lng = Number(longitude);

    if (Number.isFinite(lat) && Number.isFinite(lng) && lat !== 0 && lng !== 0) {
      return [lat, lng];
    }

    return [56.85836, 35.90057];
  }, [latitude, longitude]);

  const canSubmit = useMemo(() => {
    const hasCoordinates = Number(latitude) !== 0 && Number(longitude) !== 0;
    const hasAddress = addressFallback.trim().length > 2;
    return consentAccepted && !!category && description.trim().length >= 5 && description.trim().length <= MAX_DESCRIPTION && (hasCoordinates || hasAddress);
  }, [addressFallback, category, consentAccepted, description, latitude, longitude]);

  const detectLocation = (): void => {
    if (!navigator.geolocation) {
      setMessage('Геолокация не поддерживается на этом устройстве.');
      return;
    }

    navigator.geolocation.getCurrentPosition(
      (position) => {
        setLatitude(String(position.coords.latitude));
        setLongitude(String(position.coords.longitude));
        setShowMap(true);
        setMessage('Точка определена по GPS.');
      },
      () => {
        setMessage('GPS недоступен. Укажите точку вручную или введите адрес.');
      },
      { enableHighAccuracy: true, timeout: 10000 }
    );
  };

  const applyTemplate = (index: number): void => {
    const template = QUICK_TEMPLATES[index];
    if (!template) return;

    setDescription(template.description);

    if (categoryOptions.includes(template.category)) {
      setCategory(template.category);
    }
  };

  const onSubmit = async (): Promise<void> => {
    if (!canSubmit || loading) return;
    setLoading(true);
    setMessage('');

    try {
      const media = await Promise.all(files.slice(0, MAX_MEDIA).map((file) => toDataUrl(file)));
      const result = await createClaim({
        category,
        description: description.trim(),
        media,
        location: {
          lat: Number(latitude),
          lng: Number(longitude),
          address_fallback: addressFallback.trim() || undefined
        }
      });

      onCreated(result.claim_id);
      setDescription('');
      setFiles([]);
      setAddressFallback('');
      setPreviewMode(false);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Ошибка отправки обращения');
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className='form-sheet'>
      <header className='form-sheet__header'>
        <button className='icon-button' onClick={onCancel} type='button'>
          ✕
        </button>
        <h2>Новое обращение</h2>
        <span className='form-sheet__spacer' />
      </header>

      <div className='form-sheet__body'>
        {!previewMode ? (
          <>
            <label className='field-label'>Быстрые шаблоны</label>
            <div className='template-row'>
              {QUICK_TEMPLATES.map((template, index) => (
                <button className='template-chip' key={template.label} onClick={() => applyTemplate(index)} type='button'>
                  {template.label}
                </button>
              ))}
            </div>
          </>
        ) : null}

        <label className='field-label'>Категория</label>
        <select className='field-input' disabled={previewMode} value={category} onChange={(event) => setCategory(event.target.value)}>
          <option value=''>Выберите категорию</option>
          {categoryOptions.map((item) => (
            <option key={item} value={item}>
              {item}
            </option>
          ))}
        </select>

        <label className='field-label'>Описание (до {MAX_DESCRIPTION} символов)</label>
        <textarea
          className='field-textarea'
          disabled={previewMode}
          value={description}
          onChange={(event) => setDescription(event.target.value.slice(0, MAX_DESCRIPTION))}
          placeholder='Опишите проблему или предложение'
        />
        <div className='field-hint'>{description.trim().length} / {MAX_DESCRIPTION}</div>

        <label className='field-label'>Фото (до {MAX_MEDIA})</label>
        <input
          className='field-input'
          disabled={previewMode}
          type='file'
          accept='image/*'
          multiple
          onChange={(event) => setFiles(Array.from(event.target.files || []).slice(0, MAX_MEDIA))}
        />
        {files.length > 0 ? (
          <div className='media-chips'>
            {files.map((file) => (
              <span className='media-chip' key={file.name}>
                {file.name}
              </span>
            ))}
          </div>
        ) : null}

        <label className='field-label'>Геопозиция</label>
        <div className='coordinates-grid'>
          <input className='field-input' disabled={previewMode} value={latitude} onChange={(event) => setLatitude(event.target.value)} placeholder='Широта' />
          <input className='field-input' disabled={previewMode} value={longitude} onChange={(event) => setLongitude(event.target.value)} placeholder='Долгота' />
        </div>
        <button className='ghost-button' disabled={previewMode} onClick={detectLocation} type='button'>
          Определить по GPS
        </button>
        <button className='ghost-button' disabled={previewMode} onClick={() => setShowMap((value) => !value)} type='button'>
          {showMap ? 'Скрыть мини-карту' : 'Поставить точку на карте'}
        </button>

        {showMap ? (
          <div className='mini-map-wrap'>
            <MapContainer center={mapCenter} className='mini-map' scrollWheelZoom={false} zoom={14}>
              <TileLayer
                attribution='&copy; OpenStreetMap contributors'
                url='https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png'
              />
              <Marker position={mapCenter} />
              <MapViewport lat={mapCenter[0]} lng={mapCenter[1]} />
              <MapClickHandler
                onPick={(lat, lng) => {
                  setLatitude(lat.toFixed(6));
                  setLongitude(lng.toFixed(6));
                  setMessage('Точка на карте обновлена.');
                }}
              />
            </MapContainer>
            <div className='field-hint field-hint--left'>Нажмите на карту, чтобы поставить пин.</div>
          </div>
        ) : null}

        <label className='field-label'>Резервный адрес (если GPS не сработал)</label>
        <input
          className='field-input'
          disabled={previewMode}
          value={addressFallback}
          onChange={(event) => setAddressFallback(event.target.value)}
          placeholder='Например: ул. Советская, дом 12'
        />

        {previewMode ? (
          <section className='preview-card'>
            <h3>Предпросмотр обращения</h3>
            <p><strong>Категория:</strong> {category || '—'}</p>
            <p><strong>Описание:</strong> {description.trim() || '—'}</p>
            <p><strong>Координаты:</strong> {latitude}, {longitude}</p>
            <p><strong>Адрес:</strong> {addressFallback.trim() || 'не указан'}</p>
            <p><strong>Медиа:</strong> {files.length > 0 ? files.map((file) => file.name).join(', ') : 'не прикреплено'}</p>
          </section>
        ) : null}

        {!consentAccepted ? <div className='inline-warning'>Для отправки примите согласие в профиле.</div> : null}
        {message ? <div className='inline-message'>{message}</div> : null}
      </div>

      <footer className='form-sheet__footer'>
        {!previewMode ? (
          <button className='primary-button' disabled={!canSubmit || loading} onClick={() => setPreviewMode(true)} type='button'>
            Предпросмотр
          </button>
        ) : (
          <div className='preview-actions'>
            <button className='ghost-button' disabled={loading} onClick={() => setPreviewMode(false)} type='button'>
              Назад к редактированию
            </button>
            <button className='primary-button' disabled={!canSubmit || loading} onClick={() => void onSubmit()} type='button'>
              {loading ? 'Отправка...' : 'Подтвердить и отправить'}
            </button>
          </div>
        )}
      </footer>
    </section>
  );
}
