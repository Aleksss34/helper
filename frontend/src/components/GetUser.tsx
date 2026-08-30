// contexts/UserContext.tsx
import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from 'react';
import { useAuth } from "./AuthModal.tsx";

type UserStatus = 'base' | 'VIP' | 'admin';

export interface UserData {
    id: number;
    username: string;
    email: string;
    status: UserStatus;
    count_questions: number;
}

interface UserContextType {
    user: UserData | null;
    loading: boolean;
    error: Error | null;
    refetch: () => void;
}

const UserContext = createContext<UserContextType | undefined>(undefined);

export function UserProvider({ children }: { children: ReactNode }) {
    const [user, setUser] = useState<UserData | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);
    const { accessToken, isLoading } = useAuth();

    // 1. Выносим fetchUser наружу с useCallback, чтобы передавать в refetch
    const fetchUser = useCallback(async () => {
        if (!accessToken) {
            setUser(null);
            setError(null);
            setLoading(false);
            return;
        }

        setLoading(true);
        try {
            const res = await fetch('http://localhost:8080/api/user/me', {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${accessToken}`,
                },
                credentials: 'include',
            });

            if (!res.ok) {
                throw new Error(`Ошибка запроса: status ${res.status}`);
            }

            const data: UserData = await res.json();
            setUser(data);
            setError(null);
        } catch (e) {
            setError(e as Error);
            setUser(null);
        } finally {
            setLoading(false);
        }
    }, [accessToken]);


    useEffect(() => {
        if (isLoading) return;

        if (accessToken) {
            fetchUser();
        } else {
            setUser(null);
            setError(null);
            setLoading(false);
        }
    }, [accessToken, isLoading, fetchUser]);

    return (
        <UserContext.Provider value={{ user, loading, error, refetch: fetchUser }}>
            {children}
        </UserContext.Provider>
    );
}

export function useUser() {
    const ctx = useContext(UserContext);
    if (!ctx) throw new Error('useUser must be used within UserProvider');
    return ctx;
}