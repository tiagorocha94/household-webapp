// Applies the person's appearance choices before the page paints, so a chosen theme or text size
// does not flash. It must give the same attributes as src/lib/appearance.ts (a test checks it).
(function () {
  try {
    var root = document.documentElement;
    var store = window.localStorage;
    [
      ['data-theme', 'household.theme', ['light', 'dark']],
      ['data-text-size', 'household.textSize', ['large', 'larger']],
      ['data-rows', 'household.rows', ['compact']],
      ['data-accent', 'household.accent', ['blue', 'purple', 'pink']],
    ].forEach(function (item) {
      var value = store.getItem(item[1]);
      if (item[2].indexOf(value) >= 0) root.setAttribute(item[0], value);
    });
  } catch (e) {
    // Storage can be blocked; the defaults then apply.
  }
})();
