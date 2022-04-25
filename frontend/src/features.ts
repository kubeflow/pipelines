export interface Feature {
  name: FeatureKey;
  description: string;
  active: boolean;
}

export enum FeatureKey {
  V2 = 'v2',
}

const FEATURE_V2 = {
  name: FeatureKey.V2,
  description: 'Show v2 features',
  active: false,
};

const features: Feature[] = [FEATURE_V2];

declare global {
  var __FEATURE_FLAGS__: string;
}

export function initFeatures() {
  let updatedFeatures = features;
  if (!storageAvailable('localStorage')) {
    window.__FEATURE_FLAGS__ = JSON.stringify(features);
    return;
  }
  if (localStorage.getItem('flags')) {
    const originalFlags = localStorage.getItem('flags');
    let originalFlagsJSON: Feature[] = [];
    try {
      originalFlagsJSON = JSON.parse(originalFlags!);
      let originalFlagsMap = new Map(originalFlagsJSON.map(features => [features.name, features]));
      for (let i = 0; i < updatedFeatures.length; i++) {
        const feature = originalFlagsMap.get(updatedFeatures[i].name);
        if (feature) {
          updatedFeatures[i].active = feature.active;
        }
      }
    } catch (e) {
      console.warn(
        'Original feature flags format is null or not recognizable, overwriting with default feature flags.',
      );
    }
  }
  localStorage.setItem('flags', JSON.stringify(updatedFeatures));
  const flags = localStorage.getItem('flags');
  if (flags) {
    window.__FEATURE_FLAGS__ = flags;
  }
}

export function saveFeatures(f: Feature[]) {
  if (!storageAvailable('localStorage')) {
    window.__FEATURE_FLAGS__ = JSON.stringify(f);
    return;
  }
  localStorage.setItem('flags', JSON.stringify(f));
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

export function isFeatureEnabled(key: FeatureKey): boolean {
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

export function getFeatureList(): Feature[] {
  const stringifyFlags = (window as any).__FEATURE_FLAGS__ as string;
  if (!stringifyFlags) {
    return [];
  }

  try {
    const parsedFlags: Feature[] = JSON.parse(stringifyFlags);
    return parsedFlags;
  } catch (e) {
    console.log('cannot read feature flags: ' + e);
  }
  return [];
}
