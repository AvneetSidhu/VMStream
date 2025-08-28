import "./App.css";
import Display from "./components/Display";
import Login from "./components/Login";
import { AuthProvider, useAuth } from "./authContext";

function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
}

function AppContent() {
  const { token } = useAuth();
  return (
    <div className="h-screen">
      <div id="app" className="w-screen flex">
        {token ? <Display /> : <Login />}
      </div>
    </div>
  );
}

export default App;
