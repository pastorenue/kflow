import { a as store_get, u as unsubscribe_stores } from "../../../chunks/index2.js";
import "@sveltejs/kit/internal";
import "../../../chunks/exports.js";
import "../../../chunks/utils.js";
import "@sveltejs/kit/internal/server";
import "../../../chunks/root.js";
import "../../../chunks/state.svelte.js";
import { a as wsEvents } from "../../../chunks/wsStore.js";
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    var $$store_subs;
    let services = [];
    if (store_get($$store_subs ??= {}, "$wsEvents", wsEvents)?.type === "service_update") {
      const p = store_get($$store_subs ??= {}, "$wsEvents", wsEvents).payload;
      const idx = services.findIndex((s) => s.name === p.service_name);
      if (idx >= 0) {
        services[idx] = {
          ...services[idx],
          status: p.status,
          updated_at: store_get($$store_subs ??= {}, "$wsEvents", wsEvents).timestamp
        };
        services = [...services];
      }
    }
    $$renderer2.push(`<h1>Services</h1> <div class="filters"><button>Refresh</button></div> `);
    {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<p class="empty">Loading…</p>`);
    }
    $$renderer2.push(`<!--]-->`);
    if ($$store_subs) unsubscribe_stores($$store_subs);
  });
}
export {
  _page as default
};
