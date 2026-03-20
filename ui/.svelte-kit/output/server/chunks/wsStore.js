import { w as writable } from "./index.js";
const wsEvents = writable(null);
const wsConnected = writable(false);
export {
  wsEvents as a,
  wsConnected as w
};
