import { a as store_get, b as attr, u as unsubscribe_stores } from "../../chunks/index2.js";
import "@sveltejs/kit/internal";
import "../../chunks/exports.js";
import "../../chunks/utils.js";
import "@sveltejs/kit/internal/server";
import "../../chunks/root.js";
import "../../chunks/state.svelte.js";
import { a as wsEvents } from "../../chunks/wsStore.js";
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    var $$store_subs;
    let executions = [];
    let filterWorkflow = "";
    let filterStatus = "";
    if (store_get($$store_subs ??= {}, "$wsEvents", wsEvents)?.type === "state_transition") {
      const p = store_get($$store_subs ??= {}, "$wsEvents", wsEvents).payload;
      const idx = executions.findIndex((ex) => ex.id === p.execution_id);
      if (idx >= 0) {
        executions[idx] = {
          ...executions[idx],
          status: p.to_status,
          updated_at: store_get($$store_subs ??= {}, "$wsEvents", wsEvents).timestamp
        };
        executions = [...executions];
      }
    }
    $$renderer2.push(`<h1>Executions</h1> <div class="filters"><input${attr("value", filterWorkflow)} placeholder="Workflow name"/> `);
    $$renderer2.select({ value: filterStatus }, ($$renderer3) => {
      $$renderer3.option({ value: "" }, ($$renderer4) => {
        $$renderer4.push(`All statuses`);
      });
      $$renderer3.option({ value: "Pending" }, ($$renderer4) => {
        $$renderer4.push(`Pending`);
      });
      $$renderer3.option({ value: "Running" }, ($$renderer4) => {
        $$renderer4.push(`Running`);
      });
      $$renderer3.option({ value: "Completed" }, ($$renderer4) => {
        $$renderer4.push(`Completed`);
      });
      $$renderer3.option({ value: "Failed" }, ($$renderer4) => {
        $$renderer4.push(`Failed`);
      });
    });
    $$renderer2.push(` <button>Refresh</button></div> `);
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
