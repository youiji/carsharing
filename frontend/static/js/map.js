// Карта Leaflet (подключается шаблоном "map-head" → window.L).
//
// Подложка векторная: тайлы OpenFreeMap, цветной стиль Liberty (схема OpenMapTiles, данные OpenStreetMap)
// рисует MapLibre GL через плагин-мост maplibre-gl-leaflet, подписи на русском.
// Маркеры, группы и подсказки остаются обычными объектами Leaflet.
// Если WebGL недоступен или стиль не загрузился, подключается растровая подложка
// Stadia Maps (Alidade Smooth).

const MOSCOW = [55.7522, 37.6156];
const STYLE_URL = 'https://tiles.openfreemap.org/styles/liberty';
const VECTOR_ATTRIBUTION = '<a href="https://openfreemap.org">OpenFreeMap</a>, &copy; <a href="https://www.openmaptiles.org/">OpenMapTiles</a>, данные &copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>';
const RASTER_ATTRIBUTION = '&copy; <a href="https://stadiamaps.com/">Stadia Maps</a>, &copy; <a href="https://openmaptiles.org/">OpenMapTiles</a>, &copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>';

export function createMap(container, { center = MOSCOW, zoom = 12 } = {}) {
  const L = window.L;
  // Пределы масштаба задаются у карты: векторный слой их не сообщает, а группировке маркеров они нужны.
  const map = L.map(container, { scrollWheelZoom: false, minZoom: 3, maxZoom: 19 }).setView(center, zoom);
  // Строка атрибуции: префикс Leaflet (ссылка и флаг) убран, лицензия библиотеки (BSD)
  // этого не требует, библиотека указана на странице «Об авторе». Указание источников
  // данных обязательно по условиям OpenFreeMap, OpenMapTiles и OpenStreetMap (ODbL).
  map.attributionControl.setPrefix(false);
  addBasemap(map);
  // Колесо мыши масштабирует карту только после клика по ней, чтобы не мешать прокрутке страницы.
  map.on('click', () => map.scrollWheelZoom.enable());
  map.on('mouseout', () => map.scrollWheelZoom.disable());
  return map;
}

async function addBasemap(map) {
  try {
    if (!window.maplibregl || !window.L.maplibreGL || !webglSupported()) throw new Error('vector map unavailable');
    const style = await russianStyle();
    // Плагин-мост берёт текст атрибуции из attributionControl.customAttribution.
    window.L.maplibreGL({ style, attributionControl: { customAttribution: VECTOR_ATTRIBUTION }, interactive: false }).addTo(map);
  } catch (err) {
    console.info('Векторная карта недоступна, используется растровая подложка:', err.message);
    window.L.tileLayer('https://tiles.stadiamaps.com/tiles/alidade_smooth/{z}/{x}/{y}{r}.png', {
      maxZoom: 20, attribution: RASTER_ATTRIBUTION,
    }).addTo(map);
  }
}

let stylePromise;

// Стиль загружается один раз на страницу; подписи переводятся на русский:
// name:ru, а если перевода нет, исходное название (name). Номера дорог (ref) не меняются.
function russianStyle() {
  if (!stylePromise) {
    // Воркер MapLibre берётся с нашего сервера: так работает строгая политика CSP (без blob:).
    window.maplibregl.setWorkerUrl('/static/vendor/maplibre/maplibre-gl-csp-worker.js');
    stylePromise = fetch(STYLE_URL)
      .then((r) => {
        if (!r.ok) throw new Error(`style ${r.status}`);
        return r.json();
      })
      .then((style) => {
        for (const layer of style.layers) {
          const field = layer.layout && layer.layout['text-field'];
          if (field && JSON.stringify(field).includes('name')) {
            layer.layout['text-field'] = ['coalesce', ['get', 'name:ru'], ['get', 'name']];
          }
        }
        return style;
      });
    stylePromise.catch(() => { stylePromise = null; });
  }
  return stylePromise;
}

function webglSupported() {
  try {
    const canvas = document.createElement('canvas');
    return Boolean(canvas.getContext('webgl2') || canvas.getContext('webgl'));
  } catch {
    return false;
  }
}

export function carIcon(status) {
  return window.L.divIcon({
    className: `car-marker car-marker-${status}`,
    html: '<span class="car-marker-pin"></span>',
    iconSize: [26, 34],
    iconAnchor: [13, 34],
    popupAnchor: [0, -30],
  });
}

/**
 * Группа маркеров с объединением близких машин (плагин Leaflet.markercluster).
 * При отдалении машины собираются в квадрат с числом, при приближении рассыпаются;
 * с 16-го масштаба (уровень улиц) каждая машина показывается отдельно.
 */
export function carClusterGroup() {
  const reduceMotion = matchMedia('(prefers-reduced-motion: reduce)').matches;
  return window.L.markerClusterGroup({
    maxClusterRadius: 48,
    disableClusteringAtZoom: 16,
    showCoverageOnHover: false,
    spiderfyOnMaxZoom: true,
    animate: !reduceMotion,
    iconCreateFunction(cluster) {
      const n = cluster.getChildCount();
      const size = n < 10 ? 36 : n < 100 ? 42 : 48;
      return window.L.divIcon({
        // Число собирается из счётчика плагина, пользовательские данные сюда не попадают.
        html: `<span class="car-cluster-count">${Number(n)}</span>`,
        className: 'car-cluster',
        iconSize: [size, size],
      });
    },
  });
}
