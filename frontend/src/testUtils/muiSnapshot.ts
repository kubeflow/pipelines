export const normalizeMuiIds = (fragment: DocumentFragment) => {
  const idMap = new Map<string, string>();
  let nextId = 0;
  fragment.querySelectorAll('[id^="mui-"]').forEach(el => {
    const oldId = el.getAttribute('id');
    if (!oldId) {
      return;
    }
    if (!idMap.has(oldId)) {
      idMap.set(oldId, `mui-id-${nextId++}`);
    }
    el.setAttribute('id', idMap.get(oldId)!);
  });

  const updateAttr = (el: Element, attr: string) => {
    const value = el.getAttribute(attr);
    if (!value) {
      return;
    }
    const parts = value.split(' ');
    let changed = false;
    const updated = parts.map(part => {
      const mapped = idMap.get(part);
      if (mapped) {
        changed = true;
        return mapped;
      }
      return part;
    });
    if (changed) {
      el.setAttribute(attr, updated.join(' '));
    }
  };

  fragment.querySelectorAll('[aria-labelledby]').forEach(el => updateAttr(el, 'aria-labelledby'));
  fragment.querySelectorAll('[for]').forEach(el => updateAttr(el, 'for'));
  fragment.querySelectorAll('[aria-describedby]').forEach(el => updateAttr(el, 'aria-describedby'));
};

export const normalizeRechartsIds = (fragment: DocumentFragment) => {
  const idMap = new Map<string, string>();
  let nextId = 0;
  fragment.querySelectorAll('[id^="recharts"]').forEach(el => {
    const oldId = el.getAttribute('id');
    if (!oldId) {
      return;
    }
    if (!idMap.has(oldId)) {
      idMap.set(oldId, `recharts-id-${nextId++}`);
    }
    el.setAttribute('id', idMap.get(oldId)!);
  });
};

export const stableMuiSnapshotFragment = (fragment: DocumentFragment) => {
  normalizeMuiIds(fragment);
  normalizeRechartsIds(fragment);
  return fragment;
};
