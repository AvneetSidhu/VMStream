// Login.tsx
import React from "react";
import { useAuth } from "../authContext";

const Login: React.FC = () => {
  const [isRegistering, setIsRegistering] = React.useState(false);
  const [username, setUsername] = React.useState("");
  const [password, setPassword] = React.useState("");
  const [confirmPassword, setConfirmPassword] = React.useState("");
  const { login } = useAuth();

  const handleLogin = async () => {
    try {
      const res = await fetch("/api/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          username: username,
          password: password,
          auth: "",
        }),
      });
      console.log("Response status:", res);

      if (!res.ok) {
        throw new Error("Login failed");
        return;
      }
      const data = await res.json();
      login(data.token, username);
    } catch (err) {
      alert("Login failed");
      console.error(err);
    }
  };

  const handleRegister = async () => {
    if (password !== confirmPassword) {
      alert("Passwords do not match");
      return;
    }
    try {
      const res = await fetch("/api/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });
      if (!res.ok) throw new Error("Registration failed");
      setIsRegistering(false);
    } catch (err) {
      alert("Registration failed");
      console.error(err);
    }
  };

  return (
    <div className="bg-gray-900 w-screen h-screen flex items-center justify-center">
      <div className="bg-amber-950 p-8 rounded shadow-md w-80">
        {isRegistering ? (
          <>
            <h2 className="text-2xl mb-4 font-bold text-white">Register</h2>
            <input
              type="text"
              placeholder="Username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="border p-2 mb-2 w-full rounded"
            />
            <input
              type="password"
              placeholder="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="border p-2 mb-2 w-full rounded"
            />
            <input
              type="password"
              placeholder="Confirm Password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="border p-2 mb-4 w-full rounded"
            />
            <button
              onClick={handleRegister}
              className="bg-green-500 text-white p-2 w-full rounded mb-2"
            >
              Register
            </button>
            <p className="text-sm text-white">
              Already have an account?{" "}
              <span
                className="text-blue-400 cursor-pointer"
                onClick={() => setIsRegistering(false)}
              >
                Login
              </span>
            </p>
          </>
        ) : (
          <>
            <h2 className="text-2xl mb-4 font-bold text-white">Login</h2>
            <input
              type="text"
              placeholder="Username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="border p-2 mb-2 w-full rounded"
            />
            <input
              type="password"
              placeholder="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="border p-2 mb-4 w-full rounded"
            />
            <button
              onClick={handleLogin}
              className="bg-blue-500 text-white p-2 w-full rounded mb-2"
            >
              Login
            </button>
            <p className="text-sm text-white">
              Don't have an account?{" "}
              <span
                className="text-blue-400 cursor-pointer"
                onClick={() => setIsRegistering(true)}
              >
                Register
              </span>
            </p>
          </>
        )}
      </div>
    </div>
  );
};

export default Login;
