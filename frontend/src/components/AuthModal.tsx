import React, { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from "react";

/**
 * DTO интерфейсы
 */
interface LoginRequest {
  username: string;
  password: string;
}

interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

interface AuthResponse {
  access_token: string;
}

interface ApiError {
  status?: number;
  message: string;
  code?: string;
}

const API_BASE = "http://localhost:8080/api";

const ERROR_MESSAGES: Record<string, string> = {
  USERNAME_ALREADY_EXISTS: "Пользователь с таким именем уже зарегистрирован",
  EMAIL_ALREADY_EXISTS: "Этот email уже используется",
  WEAK_PASSWORD: "Пароль слишком простой — не должен содержать имя пользователя",
  INVALID_CREDENTIALS: "Неверный логин или пароль",
  SESSION_EXPIRED: "Сессия истекла, войдите заново",
  LIMIT_REACHED: "Достигнут лимит запросов",
  INTERNAL_ERROR: "Что-то пошло не так, попробуйте позже",
};

function translateError(err: ApiError): string {
  if (err.code && ERROR_MESSAGES[err.code]) {
    return ERROR_MESSAGES[err.code];
  }
  return "Что-то пошло не так, попробуйте позже";
}

async function apiRequest<T>(path: string, body?: unknown): Promise<T> {
  const options: RequestInit = {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
  };

  if (body !== undefined) {
    options.body = JSON.stringify(body);
  }

  const res = await fetch(`${API_BASE}${path}`, options);
  const data = await res.json().catch(() => null);

  if (!res.ok) {
    const err: ApiError = data ?? { message: "Неизвестная ошибка" };
    throw new Error(translateError(err));
  }

  return data as T;
}

const login = (payload: LoginRequest) =>
    apiRequest<AuthResponse>("/login", payload);

const register = (payload: RegisterRequest) =>
    apiRequest<AuthResponse>("/register", payload);

const refresh = () =>
    apiRequest<AuthResponse>("/refresh");

const logoutApi = () =>
    apiRequest<{ message: string }>("/refresh/logout");

/**
 * AUTH CONTEXT
 */
interface AuthContextType {
  accessToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  doLogin: (payload: LoginRequest) => Promise<AuthResponse>;
  doRegister: (payload: RegisterRequest) => Promise<AuthResponse>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let isMounted = true;

    refresh()
        .then((data) => {
          if (isMounted && data?.access_token) {
            setAccessToken(data.access_token);
          }
        })
        .catch(() => {
          if (isMounted) setAccessToken(null);
        })
        .finally(() => {
          if (isMounted) setIsLoading(false);
        });

    return () => {
      isMounted = false;
    };
  }, []);

  const doLogin = useCallback(async (payload: LoginRequest) => {
    const data = await login(payload);
    setAccessToken(data.access_token);
    return data;
  }, []);

  const doRegister = useCallback(async (payload: RegisterRequest) => {
    const data = await register(payload);
    setAccessToken(data.access_token);
    return data;
  }, []);

  const logout = useCallback(async () => {
    try {
      await logoutApi();
    } catch (e) {
      console.error("Ошибка при выходе из системы:", e);
    } finally {
      setAccessToken(null);
    }
  }, []);

  return (
      <AuthContext.Provider
          value={{
            accessToken,
            isAuthenticated: !!accessToken,
            isLoading,
            doLogin,
            doRegister,
            logout,
          }}
      >
        {children}
      </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}

/**
 * Компонент модального окна
 */
type AuthMode = "login" | "register";

interface AuthModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: (token: string) => void;
}

export const AuthModal: React.FC<AuthModalProps> = ({ isOpen, onClose, onSuccess }) => {
  const { doLogin, doRegister } = useAuth();
  const [mode, setMode] = useState<AuthMode>("login");
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Сбрасываем режим на "login" и все поля каждый раз при открытии модалки,
  // чтобы кнопка "Войти" всегда открывала именно форму входа
  useEffect(() => {
    if (isOpen) {
      setMode("login");
      setUsername("");
      setEmail("");
      setPassword("");
      setConfirmPassword("");
      setShowPassword(false);
      setShowConfirmPassword(false);
      setError(null);
    }
  }, [isOpen]);

  if (!isOpen) return null;

  const resetFields = () => {
    setUsername("");
    setEmail("");
    setPassword("");
    setConfirmPassword("");
    setError(null);
  };

  const switchMode = (next: AuthMode) => {
    setMode(next);
    resetFields();
  };

  const handleClose = () => {
    onClose();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (mode === "register" && password !== confirmPassword) {
      setError("Пароли не совпадают");
      return;
    }

    setLoading(true);
    try {
      const data =
          mode === "login"
              ? await doLogin({ username, password })
              : await doRegister({ username, email, password });

      onSuccess?.(data.access_token);
      handleClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Что-то пошло не так");
    } finally {
      setLoading(false);
    }
  };

  return (
      <div style={styles.overlay} onClick={handleClose}>
        <div style={styles.modal} onClick={(e) => e.stopPropagation()}>
          <button style={styles.closeBtn} onClick={handleClose} aria-label="Закрыть">
            ✕
          </button>

          <h2 style={styles.title}>
            {mode === "login" ? "Вход в аккаунт" : "Регистрация"}
          </h2>

          <form onSubmit={handleSubmit} style={styles.form}>
            <label style={styles.label}>
              Имя пользователя
              <input
                  style={styles.input}
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  required
                  autoFocus
              />
            </label>

            {mode === "register" && (
                <label style={styles.label}>
                  Email
                  <input
                      style={styles.input}
                      type="email"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      required
                  />
                </label>
            )}

            <label style={styles.label}>
              Пароль
              <div style={styles.passwordWrapper}>
                <input
                    style={styles.passwordInput}
                    type={showPassword ? "text" : "password"}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    minLength={6}
                />
                <button
                    type="button"
                    style={styles.eyeBtn}
                    onClick={() => setShowPassword((v) => !v)}
                    aria-label={showPassword ? "Скрыть пароль" : "Показать пароль"}
                    tabIndex={-1}
                >
                  {showPassword ? "🙈" : "👁"}
                </button>
              </div>
            </label>

            {mode === "register" && (
                <label style={styles.label}>
                  Подтверждение пароля
                  <div style={styles.passwordWrapper}>
                    <input
                        style={styles.passwordInput}
                        type={showConfirmPassword ? "text" : "password"}
                        value={confirmPassword}
                        onChange={(e) => setConfirmPassword(e.target.value)}
                        required
                        minLength={6}
                    />
                    <button
                        type="button"
                        style={styles.eyeBtn}
                        onClick={() => setShowConfirmPassword((v) => !v)}
                        aria-label={showConfirmPassword ? "Скрыть пароль" : "Показать пароль"}
                        tabIndex={-1}
                    >
                      {showConfirmPassword ? "🙈" : "👁"}
                    </button>
                  </div>
                </label>
            )}

            {error && <div style={styles.error}>{error}</div>}

            <button type="submit" style={styles.submitBtn} disabled={loading}>
              {loading
                  ? "Подождите..."
                  : mode === "login"
                      ? "Войти"
                      : "Зарегистрироваться"}
            </button>
          </form>

          <div style={styles.switchRow}>
            {mode === "login" ? (
                <>
                  Ещё не зарегистрированы?{" "}
                  <button style={styles.switchLink} onClick={() => switchMode("register")}>
                    Зарегистрироваться
                  </button>
                </>
            ) : (
                <>
                  Уже есть аккаунт?{" "}
                  <button style={styles.switchLink} onClick={() => switchMode("login")}>
                    Войти
                  </button>
                </>
            )}
          </div>
        </div>
      </div>
  );
};

/**
 * Кнопка управления авторизацией для отображения в Header
 */
export const AuthHeaderButton: React.FC = () => {
  const [isOpen, setIsOpen] = useState(false);
  const { isAuthenticated, isLoading, logout } = useAuth();

  if (isLoading) {
    return (
        <button style={styles.headerBtn} disabled>
          ...
        </button>
    );
  }

  return (
      <>
        <button
            style={styles.headerBtn}
            onClick={() => (isAuthenticated ? logout() : setIsOpen(true))}
        >
          {isAuthenticated ? "Выйти" : "Войти"}
        </button>

        <AuthModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </>
  );
};

/**
 * Стили
 */
const styles: Record<string, React.CSSProperties> = {
  overlay: {
    position: "fixed",
    inset: 0,
    background: "rgba(5, 6, 15, 0.7)",
    backdropFilter: "blur(4px)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex: 1000,
  },
  modal: {
    position: "relative",
    width: "380px",
    maxWidth: "90vw",
    background: "#12141f",
    border: "1px solid #262a3d",
    borderRadius: "16px",
    padding: "32px",
    color: "#e5e7eb",
    boxShadow: "0 20px 60px rgba(0,0,0,0.5)",
    fontFamily: "system-ui, sans-serif",
  },
  closeBtn: {
    position: "absolute",
    top: "16px",
    right: "16px",
    background: "transparent",
    border: "none",
    color: "#8a8fa3",
    fontSize: "18px",
    cursor: "pointer",
  },
  title: {
    margin: "0 0 24px 0",
    fontSize: "20px",
    fontWeight: 600,
    color: "#ffffff",
  },
  form: {
    display: "flex",
    flexDirection: "column",
    gap: "16px",
  },
  label: {
    display: "flex",
    flexDirection: "column",
    gap: "6px",
    fontSize: "13px",
    color: "#9ca3af",
  },
  input: {
    background: "#1a1d2b",
    border: "1px solid #2c3044",
    borderRadius: "10px",
    padding: "10px 12px",
    color: "#f3f4f6",
    fontSize: "14px",
    outline: "none",
  },
  passwordWrapper: {
    position: "relative",
    display: "flex",
    alignItems: "center",
  },
  passwordInput: {
    background: "#1a1d2b",
    border: "1px solid #2c3044",
    borderRadius: "10px",
    padding: "10px 40px 10px 12px",
    color: "#f3f4f6",
    fontSize: "14px",
    outline: "none",
    width: "100%",
  },
  eyeBtn: {
    position: "absolute",
    right: "8px",
    background: "transparent",
    border: "none",
    cursor: "pointer",
    fontSize: "16px",
    padding: "4px",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    lineHeight: 1,
  },
  error: {
    color: "#f87171",
    fontSize: "13px",
    background: "rgba(248,113,113,0.1)",
    border: "1px solid rgba(248,113,113,0.3)",
    borderRadius: "8px",
    padding: "8px 10px",
  },
  submitBtn: {
    marginTop: "8px",
    background: "linear-gradient(90deg, #7c6cf6, #a855f7)",
    color: "#fff",
    border: "none",
    borderRadius: "10px",
    padding: "12px",
    fontSize: "14px",
    fontWeight: 600,
    cursor: "pointer",
  },
  switchRow: {
    marginTop: "20px",
    textAlign: "center",
    fontSize: "13px",
    color: "#9ca3af",
  },
  switchLink: {
    background: "none",
    border: "none",
    color: "#a855f7",
    fontWeight: 600,
    cursor: "pointer",
    padding: 0,
    fontSize: "13px",
  },
  headerBtn: {
    background: "linear-gradient(90deg, #7c6cf6, #a855f7)",
    color: "#fff",
    border: "none",
    borderRadius: "999px",
    padding: "8px 18px",
    fontSize: "13px",
    fontWeight: 600,
    cursor: "pointer",
  },
};

export default AuthHeaderButton;