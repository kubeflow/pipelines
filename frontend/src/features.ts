export interface Feature {
  name: FeatureKey;
  description: string;
  active: boolean;
}

export enum FeatureKey {
  V2 = 'v2', // Please start using V2_ALPHA instead of V2, because we have switched to V2_ALPHA as V2 feature is enabled by default.
  V2_ALPHA = 'v2_alpha',
  FUNCTIONAL_COMPONENT = 'functional_component',
  // We plan to refactor the class component to functional component.
  // To avoid breacking current behavior, enable this feature to do the bugbash / validation test for functional components.
}

const FEATURE_V2 = {
  name: FeatureKey.V2,
  description: 'Show v2 features.',
  active: false,
};

const FEATURE_V2_ALPHA = {
  name: FeatureKey.V2_ALPHA,
  description: 'Show v2 features, enabled by default starting at V2_ALPHA phase.',
  active: true,
};

const FEATURE_FUNCTIONAL_COMPONENT = {
  name: FeatureKey.FUNCTIONAL_COMPONENT,
  description: 'Use functional component',
  active: false,
};

const features: Feature[] = [FEATURE_V2, FEATURE_V2_ALPHA, FEATURE_FUNCTIONAL_COMPONENT];

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

function storageAvailable(type: 'localStorage' | 'sessionStorage'): boolean {
  let storage: Storage;
  try {
    storage = window[type];
    const x = '__storage_test__';
    storage.setItem(x, x);
    storage.removeItem(x);
    return true;
  } catch (e) {
    if (!(e instanceof DOMException)) {
      return false;
    }
    
    // Check for various storage-related error codes
    const isStorageError = 
      e.code === 22 || // Chrome
      e.code === 1014 || // Firefox
      // test name field too, because code might not be present
      e.name === 'QuotaExceededError' || // Chrome
      e.name === 'NS_ERROR_DOM_QUOTA_REACHED'; // Firefox
      
    // Return true only if there's a storage error AND there's something already stored
    return isStorageError && storage !== undefined && storage.length !== 0;
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
