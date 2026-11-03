describe('jsdom Range geometry', () => {
  it('supports editor text measurement without replacing native range behavior', () => {
    const text = document.createTextNode('pipeline');
    const range = document.createRange();
    range.setStart(text, 0);
    range.setEnd(text, 4);

    expect(range.toString()).toBe('pipe');
    expect(range.cloneRange().toString()).toBe('pipe');
    const rect = range.getBoundingClientRect();
    expect(rect).toBeInstanceOf(DOMRect);
    expect(rect.toJSON()).toEqual({
      x: 0,
      y: 0,
      width: 0,
      height: 0,
      top: 0,
      right: 0,
      bottom: 0,
      left: 0,
    });
    expect(range.getBoundingClientRect()).not.toBe(rect);
  });
});
