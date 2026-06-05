export const normalizeMuiIds = (fragment: DocumentFragment) => {
  const idMap = new Map<string, string>();
  let nextId = 0;
  fragment.querySelectorAll('[id^="mui-"]').forEach((el) => {
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
    const updated = parts.map((part) => {
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

  fragment.querySelectorAll('[aria-labelledby]').forEach((el) => updateAttr(el, 'aria-labelledby'));
  fragment.querySelectorAll('[for]').forEach((el) => updateAttr(el, 'for'));
  fragment
    .querySelectorAll('[aria-describedby]')
    .forEach((el) => updateAttr(el, 'aria-describedby'));
};

/** Normalize React `useId()` values (e.g. `_r_abc_`, `:r1:`) for stable snapshots under StrictMode. */
export const normalizeReactUseIdAttrs = (fragment: DocumentFragment) => {
  const idMap = new Map<string, string>();
  let nextId = 0;

  const isReactUseId = (value: string): boolean =>
    /^_r_[A-Za-z0-9]+_$/.test(value) || /^:r[a-zA-Z0-9]*:$/.test(value);

  const mapToken = (token: string): string => {
    if (!isReactUseId(token)) {
      return token;
    }
    if (!idMap.has(token)) {
      idMap.set(token, `react-use-id-${nextId++}`);
    }
    return idMap.get(token)!;
  };

  const mapAttrValue = (value: string): string => value.split(/\s+/).map(mapToken).join(' ');

  (['id', 'aria-controls', 'aria-labelledby', 'for', 'aria-describedby', 'list'] as const).forEach(
    (attr) => {
      fragment.querySelectorAll(`[${attr}]`).forEach((el) => {
        const v = el.getAttribute(attr);
        if (!v) {
          return;
        }
        el.setAttribute(attr, mapAttrValue(v));
      });
    },
  );
};

export const normalizeRechartsIds = (fragment: DocumentFragment) => {
  const idMap = new Map<string, string>();
  let nextId = 0;
  fragment.querySelectorAll('[id^="recharts"]').forEach((el) => {
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
  normalizeReactUseIdAttrs(fragment);
  normalizeRechartsIds(fragment);
  return fragment;
};
