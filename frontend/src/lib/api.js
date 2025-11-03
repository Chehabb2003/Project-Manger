// frontend/src/lib/api.js
const BASE = "/api";

async function req(path, { method = "GET", body, headers } = {}) {
  const init = {
    method,
    headers: { ...(headers || {}) },
    credentials: "include",
  };
  if (body !== undefined) {
    init.headers["Content-Type"] = "application/json";
    init.body = JSON.stringify(body);
  }
  const res = await fetch(`${BASE}${path}`, init);
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `${res.status} ${text?.trim() || res.statusText || "Request failed"}`
    );
  }
  if (res.status === 204) return {};
  const ct = res.headers.get("content-type") || "";
  return ct.includes("application/json") ? res.json() : {};
}

export function session() {
  return req("/session");
}
export function unlock(vault, master) {
  return req("/unlock", { method: "POST", body: { vault, master } });
}
export function lock() {
  return req("/lock", { method: "POST" });
}
export async function listItems(params = {}) {
  const qs = new URLSearchParams(params);
  const data = await req(`/items${qs.toString() ? `?${qs}` : ""}`);
  return Array.isArray(data) ? { items: data } : data || { items: [] };
}
export function getItem(id) {
  return req(`/items/${id}`);
}
export function createItem(item) {
  return req("/items", { method: "POST", body: item });
}
export function updateItem(id, patch) {
  return req(`/items/${id}`, { method: "PUT", body: patch });
}
export function deleteItem(id) {
  return req(`/items/${id}`, { method: "DELETE" });
}

const api = {
  session,
  unlock,
  lock,
  listItems,
  getItem,
  createItem,
  updateItem,
  deleteItem,
};
export default api;
