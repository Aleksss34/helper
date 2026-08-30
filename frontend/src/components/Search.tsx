import React, { useState, useRef, useEffect } from 'react';
import { Header } from './Header';
import { UserStatusWidget } from "./widget.tsx";
import { useAuth } from "./AuthModal.tsx";
import { useUser } from "./GetUser.tsx";

const SERVERS = [
    'RED', 'YELLOW', 'GREEN', 'AZURE', 'SILVER', 'ROSE',
    'BLACK', 'SKY', 'TITAN', 'X', 'FIRE', 'LIME',
];

interface Message {
    id: string;
    sender: 'user' | 'ai';
    text: string;
    server?: string;
}

export const SearchStream: React.FC = () => {
    const [question, setQuestion] = useState('');
    const [selectedServer, setSelectedServer] = useState(SERVERS[0]);
    const [messages, setMessages] = useState<Message[]>([]);
    const [isSending, setIsSending] = useState(false);

    const { accessToken, isLoading: isAuthLoading } = useAuth();
    const { user, refetch: refetchUser } = useUser();

    // Не авторизован — либо ещё грузится сессия, либо точно нет токена
    const isUnauthenticated = !isAuthLoading && !accessToken;

    const questionsLeft = user?.count_questions ?? 0;
    const isLimitReached = user?.status === 'base' && questionsLeft <= 0;

    // Флаги блокировки элементов ввода
    const isInputDisabled = isSending || isLimitReached || isUnauthenticated || isAuthLoading;
    const isSubmitDisabled = isSending || isAuthLoading || !accessToken || !question.trim() || !selectedServer || isLimitReached;

    const placeholderText = isUnauthenticated
        ? "Авторизуйтесь, чтобы задать вопрос"
        : isLimitReached
            ? "Лимит запросов исчерпан. Оформите VIP подписку."
            : "Спросите что-нибудь...";

    const isBlocked = isLimitReached || isUnauthenticated;

    const messagesEndRef = useRef<HTMLDivElement>(null);

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    };

    useEffect(() => {
        scrollToBottom();
    }, [messages]);

    const handleSearch = async (e: React.FormEvent) => {
        e.preventDefault();

        if (isSubmitDisabled) return;

        if (!accessToken) {
            console.error('Нет access token — пользователь не авторизован');
            return;
        }

        const userText = question.trim();
        const currentServer = selectedServer;
        setQuestion('');
        setIsSending(true);

        const userMsgId = Date.now().toString();
        const aiMsgId = (Date.now() + 1).toString();

        setMessages((prev) => [
            ...prev,
            { id: userMsgId, sender: 'user', text: userText, server: currentServer },
            { id: aiMsgId, sender: 'ai', text: '' }
        ]);

        try {
            const response = await fetch('http://localhost:8080/api/search', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${accessToken}`,
                },
                credentials: 'include',
                body: JSON.stringify({
                    question: userText,
                    server: currentServer
                }),
            });

            if (!response.ok || !response.body) {
                throw new Error('Ошибка запроса');
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder();

            while (true) {
                const { value, done } = await reader.read();
                if (done) break;

                const text = decoder.decode(value);
                const lines = text.split('\n');

                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        const chunk = line.replace('data: ', '');
                        setMessages((prev) =>
                            prev.map((msg) =>
                                msg.id === aiMsgId
                                    ? { ...msg, text: msg.text + chunk }
                                    : msg
                            )
                        );
                    }
                }
            }

            refetchUser?.();

        } catch (error) {
            console.error('Ошибка потока:', error);
            setMessages((prev) =>
                prev.map((msg) =>
                    msg.id === aiMsgId
                        ? { ...msg, text: msg.text + '\n[Ошибка получения данных]' }
                        : msg
                )
            );
        } finally {
            setIsSending(false);
        }
    };

    return (
        <div style={{
            display: 'flex',
            flexDirection: 'column',
            height: '100vh',
            backgroundColor: '#090d16',
            color: '#f3f4f6',
            fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
        }}>
            <Header />

            <div style={{
                flex: 1,
                overflowY: 'auto',
                padding: '24px 16px',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center'
            }}>
                <div style={{ width: '100%', maxWidth: '768px', display: 'flex', flexDirection: 'column', gap: '20px' }}>
                    {messages.length === 0 && (
                        <div style={{
                            textAlign: 'center',
                            color: '#64748b',
                            marginTop: '35vh',
                            display: 'flex',
                            flexDirection: 'column',
                            alignItems: 'center',
                            gap: '8px'
                        }}>
                            <div style={{ fontSize: '32px' }}>✨</div>
                            <div style={{ fontSize: '18px', fontWeight: '500' }}>Готов, когда ты готов...</div>
                        </div>
                    )}

                    {messages.map((msg) => (
                        <div
                            key={msg.id}
                            style={{
                                display: 'flex',
                                flexDirection: 'column',
                                alignItems: msg.sender === 'user' ? 'flex-end' : 'flex-start',
                            }}
                        >
                            {msg.sender === 'user' && (
                                <span style={{
                                    fontSize: '11px',
                                    fontWeight: '600',
                                    color: '#818cf8',
                                    marginBottom: '6px',
                                    textTransform: 'uppercase',
                                    letterSpacing: '0.5px',
                                    paddingRight: '4px'
                                }}>
                                    Сервер: {msg.server}
                                </span>
                            )}

                            <div
                                style={{
                                    maxWidth: '85%',
                                    padding: '14px 18px',
                                    borderRadius: '20px',
                                    background: msg.sender === 'user'
                                        ? 'linear-gradient(135deg, #4f46e5 0%, #3b82f6 100%)'
                                        : '#1e293b',
                                    color: msg.sender === 'user' ? '#ffffff' : '#e2e8f0',
                                    boxShadow: msg.sender === 'user'
                                        ? '0 4px 14px rgba(79, 70, 229, 0.3)'
                                        : '0 4px 12px rgba(0,0,0,0.2)',
                                    whiteSpace: 'pre-wrap',
                                    lineHeight: '1.6',
                                    fontSize: '15px',
                                    border: msg.sender === 'ai' ? '1px solid #334155' : 'none',
                                    borderBottomRightRadius: msg.sender === 'user' ? '4px' : '20px',
                                    borderBottomLeftRadius: msg.sender === 'ai' ? '4px' : '20px',
                                }}
                            >
                                {msg.text || (isSending && msg.sender === 'ai' ? (
                                    <span style={{ color: '#94a3b8', fontStyle: 'italic' }}>Идет генерация ответа...</span>
                                ) : '')}
                            </div>
                        </div>
                    ))}
                    <div ref={messagesEndRef} />
                </div>
            </div>

            <UserStatusWidget />

            <div style={{
                padding: '20px',
                backgroundColor: '#090d16',
                display: 'flex',
                justifyContent: 'center'
            }}>
                <form
                    onSubmit={handleSearch}
                    style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '10px',
                        width: '100%',
                        maxWidth: '768px',
                        backgroundColor: isBlocked ? '#111827' : '#1e293b',
                        padding: '8px 8px 8px 16px',
                        borderRadius: '28px',
                        border: isBlocked ? '1px solid #7f1d1d' : '1px solid #334155',
                        boxShadow: '0 8px 24px rgba(0, 0, 0, 0.4)',
                        transition: 'all 0.2s ease-in-out',
                        opacity: isBlocked ? 0.75 : 1,
                    }}
                >
                    <select
                        value={selectedServer}
                        onChange={(e) => setSelectedServer(e.target.value)}
                        disabled={isInputDisabled}
                        style={{
                            padding: '8px 12px',
                            fontSize: '13px',
                            fontWeight: '600',
                            borderRadius: '18px',
                            border: '1px solid #475569',
                            backgroundColor: '#0f172a',
                            color: isBlocked ? '#64748b' : '#cbd5e1',
                            cursor: isInputDisabled ? 'not-allowed' : 'pointer',
                            outline: 'none',
                        }}
                    >
                        {SERVERS.map((server) => (
                            <option key={server} value={server}>
                                {server}
                            </option>
                        ))}
                    </select>

                    <input
                        type="text"
                        value={question}
                        onChange={(e) => setQuestion(e.target.value)}
                        placeholder={placeholderText}
                        disabled={isInputDisabled}
                        style={{
                            flex: 1,
                            padding: '8px 4px',
                            fontSize: '15px',
                            border: 'none',
                            backgroundColor: 'transparent',
                            color: isBlocked ? '#94a3b8' : '#f8fafc',
                            cursor: isInputDisabled ? 'not-allowed' : 'text',
                            outline: 'none'
                        }}
                    />

                    <button
                        type="submit"
                        disabled={isSubmitDisabled}
                        style={{
                            width: '40px',
                            height: '40px',
                            borderRadius: '50%',
                            border: 'none',
                            background: isSubmitDisabled
                                ? '#334155'
                                : 'linear-gradient(135deg, #6366f1 0%, #3b82f6 100%)',
                            color: isSubmitDisabled ? '#64748b' : '#ffffff',
                            cursor: isSubmitDisabled ? 'not-allowed' : 'pointer',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontSize: '16px',
                            transition: 'all 0.2s ease',
                            boxShadow: isSubmitDisabled ? 'none' : '0 2px 8px rgba(99, 102, 241, 0.4)'
                        }}
                    >
                        ➔
                    </button>
                </form>
            </div>
        </div>
    );
};