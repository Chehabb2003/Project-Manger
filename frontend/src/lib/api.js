const BASE = "/api";

async function req(path, { method = "GET", body, headers } = {}) {
  const init = {
    method,
    headers: { ...(headers || {}) },
    credentials: "include",
  };
  const tok = (typeof localStorage !== 'undefined') ? localStorage.getItem('token') : null;
  if (tok) init.headers['Authorization'] = `Bearer ${tok}`;
  if (body !== undefined) {
    init.headers["Content-Type"] = "application/json";
    init.body = JSON.stringify(body);
  }

  const res = await fetch(`${BASE}${path}`, init);
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`${res.status} ${text?.trim() || res.statusText || "Request failed"}`);
  }
  if (res.status === 204) return {};
  const ct = res.headers.get("content-type") || "";
  return ct.includes("application/json") ? res.json() : {};
}

export function session() {
  return req("/session");
}

// ⬇️ backend expects only { master } now
export function unlock(master) {
  return req("/unlock", { method: "POST", body: { master } });
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

// ⬇️ use req()'s parsed return value; don't call .json() again
export async function login(identifier, password) {
  const data = await req('/login', { method: 'POST', body: { identifier, password } });
  if (data && data.token && typeof localStorage !== 'undefined') {
    localStorage.setItem('token', data.token);
  }
  return data;
}

export async function signup(username, email, password) {
  const data = await req('/signup', { method: 'POST', body: { username, email, password } });
  if (data && data.token && typeof localStorage !== 'undefined') {
    localStorage.setItem('token', data.token);
  }
  return data;
}

export async function changePassword(current, next) {
  const data = await req('/password', { method: 'PUT', body: { current, next } });
  if (data && data.token && typeof localStorage !== 'undefined') {
    localStorage.setItem('token', data.token);
  }
  return data;
}

export async function requestPasswordReset(email) {
  return req('/password/forgot', { method: 'POST', body: { email } });
}

export async function resetPassword(token, next) {
  return req('/password/reset', { method: 'POST', body: { token, next } });
}

const api = {
  login,
  signup,
  changePassword,
  requestPasswordReset,
  resetPassword,
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
