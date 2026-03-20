import { s as ssr_context, a as store_get, u as unsubscribe_stores } from "../../chunks/index2.js";
import "clsx";
import { w as wsConnected } from "../../chunks/wsStore.js";
function onDestroy(fn) {
  /** @type {SSRContext} */
  ssr_context.r.on_destroy(fn);
}
function _layout($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    var $$store_subs;
    let { children } = $$props;
    onDestroy(() => {
    });
    $$renderer2.push(`<nav class="svelte-12qhfyh"><a href="/" class="svelte-12qhfyh">Executions</a> <a href="/services" class="svelte-12qhfyh">Services</a> <a href="/logs" class="svelte-12qhfyh">Logs</a> `);
    if (store_get($$store_subs ??= {}, "$wsConnected", wsConnected)) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<span class="ws-badge connected svelte-12qhfyh">WS live</span>`);
    } else {
      $$renderer2.push("<!--[-1-->");
      $$renderer2.push(`<span class="ws-badge disconnected svelte-12qhfyh">WS disconnected</span>`);
    }
    $$renderer2.push(`<!--]--></nav> <main class="svelte-12qhfyh">`);
    children($$renderer2);
    $$renderer2.push(`<!----></main>`);
    if ($$store_subs) unsubscribe_stores($$store_subs);
  });
}
export {
  _layout as default
};
