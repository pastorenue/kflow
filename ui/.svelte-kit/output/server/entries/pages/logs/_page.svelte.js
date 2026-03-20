import { b as attr, e as escape_html, c as ensure_array_like, d as attr_class, f as clsx } from "../../../chunks/index2.js";
import "@sveltejs/kit/internal";
import "../../../chunks/exports.js";
import "../../../chunks/utils.js";
import "@sveltejs/kit/internal/server";
import "../../../chunks/root.js";
import "../../../chunks/state.svelte.js";
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let logs = [];
    let total = 0;
    const limit = 50;
    let offset = 0;
    let filterExecId = "";
    let filterServiceName = "";
    let filterStateName = "";
    let filterLevel = "";
    let filterSince = "";
    let filterUntil = "";
    let filterQ = "";
    function levelBadge(level) {
      return `badge badge-${level.toLowerCase()}`;
    }
    $$renderer2.push(`<h1>Log Explorer</h1> <form class="filters"><input${attr("value", filterExecId)} placeholder="Execution ID"/> <input${attr("value", filterServiceName)} placeholder="Service name"/> <input${attr("value", filterStateName)} placeholder="State name"/> `);
    $$renderer2.select({ value: filterLevel }, ($$renderer3) => {
      $$renderer3.option({ value: "" }, ($$renderer4) => {
        $$renderer4.push(`All levels`);
      });
      $$renderer3.option({ value: "INFO" }, ($$renderer4) => {
        $$renderer4.push(`INFO`);
      });
      $$renderer3.option({ value: "WARN" }, ($$renderer4) => {
        $$renderer4.push(`WARN`);
      });
      $$renderer3.option({ value: "ERROR" }, ($$renderer4) => {
        $$renderer4.push(`ERROR`);
      });
      $$renderer3.option({ value: "DEBUG" }, ($$renderer4) => {
        $$renderer4.push(`DEBUG`);
      });
    });
    $$renderer2.push(` <input${attr("value", filterSince)} placeholder="Since (ISO 8601)"/> <input${attr("value", filterUntil)} placeholder="Until (ISO 8601)"/> <input${attr("value", filterQ)} placeholder="Search text" style="min-width:180px"/> <button type="submit" class="svelte-1lsf4ps">Search</button></form> `);
    if (logs.length === 0 && total === 0) {
      $$renderer2.push("<!--[2-->");
      $$renderer2.push(`<p class="empty">No logs found for the selected filters.</p>`);
    } else {
      $$renderer2.push("<!--[-1-->");
      $$renderer2.push(`<div class="pagination-info svelte-1lsf4ps">Showing ${escape_html(offset + 1)}–${escape_html(Math.min(offset + limit, total))} of ${escape_html(total)} results</div> <table><thead><tr><th>Time</th><th>Level</th><th>Source</th><th>State</th><th>Message</th></tr></thead><tbody><!--[-->`);
      const each_array = ensure_array_like(logs);
      for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
        let log = each_array[$$index];
        $$renderer2.push(`<tr><td style="font-size:0.8rem;white-space:nowrap">${escape_html(new Date(log.occurred_at).toLocaleString())}</td><td><span${attr_class(clsx(levelBadge(log.level)), "svelte-1lsf4ps")}>${escape_html(log.level)}</span></td><td style="font-size:0.8rem">`);
        if (log.execution_id) {
          $$renderer2.push("<!--[0-->");
          $$renderer2.push(`<code>${escape_html(log.execution_id.slice(0, 8))}…</code>`);
        } else if (log.service_name) {
          $$renderer2.push("<!--[1-->");
          $$renderer2.push(`${escape_html(log.service_name)}`);
        } else {
          $$renderer2.push("<!--[-1-->");
        }
        $$renderer2.push(`<!--]--></td><td style="font-size:0.8rem">${escape_html(log.state_name || "—")}</td><td style="font-size:0.85rem">${escape_html(log.message)}</td></tr>`);
      }
      $$renderer2.push(`<!--]--></tbody></table> <div class="pagination svelte-1lsf4ps"><button${attr("disabled", offset === 0, true)} class="svelte-1lsf4ps">← Prev</button> <button${attr("disabled", offset + limit >= total, true)} class="svelte-1lsf4ps">Next →</button></div>`);
    }
    $$renderer2.push(`<!--]-->`);
  });
}
export {
  _page as default
};
