import { useState } from "react";

function App() {
  const [backendResponse, setBackendResponse] = useState(null);
  const [authResponse, setAuthResponse] = useState(null);
  const [items, setItems] = useState(null);
  const [users, setUsers] = useState(null);
  const [newItemName, setNewItemName] = useState("");
  const [newUser, setNewUser] = useState({ username: "", email: "" });

  const BACKEND_URL = "";
  const AUTH_URL = "";
  const callBackend = async () => {
    try {
      const res = await fetch(`${BACKEND_URL}/api`);
      const data = await res.json();
      setBackendResponse(data);
    } catch (err) {
      setBackendResponse({ error: err.message });
    }
  };

  const callAuth = async () => {
    try {
      const res = await fetch(`${AUTH_URL}/auth`);
      const data = await res.json();
      setAuthResponse(data);
    } catch (err) {
      setAuthResponse({ error: err.message });
    }
  };

  // Backend MongoDB - Get Items
  const getItems = async () => {
    try {
      const res = await fetch(`${BACKEND_URL}/api/items`);
      const data = await res.json();
      setItems(data);
    } catch (err) {
      setItems({ error: err.message });
    }
  };

  // Backend MongoDB - Create Item
  const createItem = async () => {
    try {
      const res = await fetch(`${BACKEND_URL}/api/items/create`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: newItemName }),
      });
      const data = await res.json();
      setItems(data);
      setNewItemName("");
      getItems(); // Refresh list
    } catch (err) {
      setItems({ error: err.message });
    }
  };

  // Auth MongoDB - Get Users
  const getUsers = async () => {
    try {
      const res = await fetch(`${AUTH_URL}/auth/users`);
      const data = await res.json();
      setUsers(data);
    } catch (err) {
      setUsers({ error: err.message });
    }
  };

  // Auth MongoDB - Create User
  const createUser = async () => {
    try {
      const res = await fetch(`${AUTH_URL}/auth/users/create`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(newUser),
      });
      const data = await res.json();
      setUsers(data);
      setNewUser({ username: "", email: "" });
      getUsers(); // Refresh list
    } catch (err) {
      setUsers({ error: err.message });
    }
  };

  return (
    <div className="container">
      <h1>🚀 Kubernetes Practice Project</h1>
      <p>
        Frontend service running on <code>/</code>
      </p>

      {/* Backend Service Card */}
      <div className="card">
        <h2>Backend Service (Go)</h2>
        <p>
          Base URL: <code>{BACKEND_URL}</code>
        </p>

        <div className="button-group">
          <button onClick={callBackend}>GET /api</button>
          <button onClick={getItems}>GET /api/items</button>
        </div>

        <div className="input-group">
          <input
            type="text"
            placeholder="Item name"
            value={newItemName}
            onChange={(e) => setNewItemName(e.target.value)}
          />
          <button onClick={createItem}>POST /api/items/create</button>
        </div>

        {backendResponse && (
          <pre>{JSON.stringify(backendResponse, null, 2)}</pre>
        )}
        {items && <pre>{JSON.stringify(items, null, 2)}</pre>}
      </div>

      {/* Auth Service Card */}
      <div className="card">
        <h2>Auth Service (Go)</h2>
        <p>
          Base URL: <code>{AUTH_URL}</code>
        </p>

        <div className="button-group">
          <button onClick={callAuth}>GET /auth</button>
          <button onClick={getUsers}>GET /auth/users</button>
        </div>

        <div className="input-group">
          <input
            type="text"
            placeholder="Username"
            value={newUser.username}
            onChange={(e) =>
              setNewUser({ ...newUser, username: e.target.value })
            }
          />
          <input
            type="email"
            placeholder="Email"
            value={newUser.email}
            onChange={(e) => setNewUser({ ...newUser, email: e.target.value })}
          />
          <button onClick={createUser}>POST /auth/users/create</button>
        </div>

        {authResponse && <pre>{JSON.stringify(authResponse, null, 2)}</pre>}
        {users && <pre>{JSON.stringify(users, null, 2)}</pre>}
      </div>

      {/* Architecture Info */}
      <div className="info">
        <h3>📋 Services & Endpoints</h3>
        <table>
          <thead>
            <tr>
              <th>Service</th>
              <th>Port</th>
              <th>Endpoints</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>Frontend</td>
              <td>3000</td>
              <td>
                <code>/</code>
              </td>
            </tr>
            <tr>
              <td>Backend</td>
              <td>8080</td>
              <td>
                <code>/api</code>, <code>/api/items</code>,{" "}
                <code>/api/items/create</code>
              </td>
            </tr>
            <tr>
              <td>Auth</td>
              <td>8081</td>
              <td>
                <code>/auth</code>, <code>/auth/users</code>,{" "}
                <code>/auth/users/create</code>
              </td>
            </tr>
            <tr>
              <td>MongoDB</td>
              <td>27017</td>
              <td>-</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  );
}

export default App;
