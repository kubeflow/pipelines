export interface Feature {
  name: string;
  description: string;
  active: boolean;
}

const features: Feature[] = [
  {
    name: 'v2',
    description: 'Show v2 features',
    active: false,
  },
];

declare global {
  var __FEATURE_FLAGS__: string;
}

export function initFeatures() {
  if (!storageAvailable('localStorage')) {
    window.__FEATURE_FLAGS__ = JSON.stringify(features);
    return;
  }
  if (!localStorage.getItem('flags')) {
    localStorage.setItem('flags', JSON.stringify(features));
  }
  const flags = localStorage.getItem('flags');
  if (flags) {
    window.__FEATURE_FLAGS__ = flags;
  }
}

function storageAvailable(type: string) {
  var storage;
  try {
    storage = window[type];
    var x = '__storage_test__';
    storage.setItem(x, x);
    storage.removeItem(x);
    return true;
  } catch (e) {
    return (
      e instanceof DOMException &&
      // everything except Firefox
      (e.code === 22 ||
        // Firefox
        e.code === 1014 ||
        // test name field too, because code might not be present
        // everything except Firefox
        e.name === 'QuotaExceededError' ||
        // Firefox
        e.name === 'NS_ERROR_DOM_QUOTA_REACHED') &&
      // acknowledge QuotaExceededError only if there's something already stored
      storage &&
      storage.length !== 0
    );
  }
}

export function isFeatureEnabled(key: string): boolean {
  const stringifyFlags = (window as any).__FEATURE_FLAGS__ as string;
  if (!stringifyFlags) {
    return false;
  }
  try {
    const parsedFlags: Feature[] = JSON.parse(stringifyFlags);
    const feature = parsedFlags.find(feature => feature.name === key);
    return feature ? feature.active : false;
  } catch (e) {
    console.log('cannot read feature flags: ' + e);
  }
  return false;
}
