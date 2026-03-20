import { g as getContext, a as store_get, u as unsubscribe_stores } from "../../../../chunks/index2.js";
import "clsx";
import "@sveltejs/kit/internal";
import "../../../../chunks/exports.js";
import "../../../../chunks/utils.js";
import "@sveltejs/kit/internal/server";
import "../../../../chunks/root.js";
import "../../../../chunks/state.svelte.js";
import { a as wsEvents, w as wsConnected } from "../../../../chunks/wsStore.js";
const getStores = () => {
  const stores$1 = getContext("__svelte__");
  return {
    /** @type {typeof page} */
    page: {
      subscribe: stores$1.page.subscribe
    },
    /** @type {typeof navigating} */
    navigating: {
      subscribe: stores$1.navigating.subscribe
    },
    /** @type {typeof updated} */
    updated: stores$1.updated
  };
};
const page = {
  subscribe(fn) {
    const store = getStores().page;
    return store.subscribe(fn);
  }
};
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    var $$store_subs;
    let id;
    let states = [];
    let wasDisconnected = false;
    id = store_get($$store_subs ??= {}, "$page", page).params.id;
    if (store_get($$store_subs ??= {}, "$wsEvents", wsEvents)?.type === "state_transition") {
      const p = store_get($$store_subs ??= {}, "$wsEvents", wsEvents).payload;
      if (p.execution_id === id) {
        const idx = states.findIndex((s) => s.state_name === p.state_name);
        if (idx >= 0) {
          states[idx] = {
            ...states[idx],
            status: p.to_status,
            error: p.error ?? "",
            updated_at: store_get($$store_subs ??= {}, "$wsEvents", wsEvents).timestamp
          };
          states = [...states];
        }
      }
    }
    {
      if (!store_get($$store_subs ??= {}, "$wsConnected", wsConnected) && false) ;
      if (store_get($$store_subs ??= {}, "$wsConnected", wsConnected) && wasDisconnected) ;
    }
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
