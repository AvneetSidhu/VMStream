import React, {
  createContext,
  useState,
  useContext,
  useEffect,
  useRef,
} from "react";

interface AuthContextType {
  token: string | null;
  username: string | null;
  login: (token: string, username: string) => void;
  logout: () => void;
  refreshToken: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [token, setToken] = useState<string | null>(null);
  const [username, setUsername] = useState<string | null>(null);
  const refreshTimerRef = useRef<NodeJS.Timeout | null>(null);
  const backoffRef = useRef(0);

  const logout = () => {
    setToken(null);
    setUsername(null);
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
    refreshTimerRef.current = null;
    backoffRef.current = 0;
  };

  const login = (newToken: string, newUsername: string) => {
    setToken(newToken);
    setUsername(newUsername);
    scheduleRefresh(newToken);
  };

  const getTokenExpiry = (token: string) => {
    try {
      const payload = JSON.parse(atob(token.split(".")[1]));
      return payload.exp * 1000;
    } catch {
      return Date.now();
    }
  };

  const scheduleRefresh = (token: string) => {
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
    const expiry = getTokenExpiry(token);
    const timeout = expiry - Date.now() - 60_000;

    refreshTimerRef.current =
      timeout > 0
        ? setTimeout(() => refreshToken(), timeout)
        : setTimeout(() => refreshToken(), 0);
  };

  const refreshToken = async () => {
    try {
      const res = await fetch("/api/refresh", {
        method: "POST",
        credentials: "include",
      });
      if (!res.ok) throw new Error("Refresh failed");
      const data = await res.json();

      backoffRef.current = 0;

      setToken(data.token);
      setUsername(data.data);
      scheduleRefresh(data.token);
    } catch (err) {
      console.error("Token refresh failed", err);

      const delay = Math.min(2 ** backoffRef.current * 1000, 30_000);
      backoffRef.current += 1;

      if (delay < 30_000) {
        refreshTimerRef.current = setTimeout(() => refreshToken(), delay);
      } else {
        logout();
      }
    }
  };

  useEffect(() => {
    refreshToken();
  }, []);

  return (
    <AuthContext.Provider
      value={{ token, username, login, logout, refreshToken }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used within AuthProvider");
  return context;
};
