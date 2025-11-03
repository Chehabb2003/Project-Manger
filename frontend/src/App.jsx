// frontend/src/App.jsx
import { Routes, Route, Navigate } from "react-router-dom";
import Unlock from "./pages/Unlock.jsx";
import Vault from "./pages/Vault.jsx";
import ViewItem from "./pages/ViewItem.jsx";
import EditItem from "./pages/EditItem.jsx";
import NewItem from "./pages/NewItem.jsx";

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/unlock" replace />} />
      <Route path="/unlock" element={<Unlock />} />
      <Route path="/vault" element={<Vault />} />
      <Route path="/items/new" element={<NewItem />} />
      <Route path="/items/:id" element={<ViewItem />} />
      <Route path="/items/:id/edit" element={<EditItem />} />
      <Route path="*" element={<div style={{ padding: 24 }}>Not Found</div>} />
    </Routes>
  );
}
