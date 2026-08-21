import React, { useState } from 'react';

// Список серверов со скриншота + LIME
const SERVERS = [
    'RED',
    'YELLOW',
    'GREEN',
    'AZURE',
    'SILVER',
    'ROSE',
    'BLACK',
    'SKY',
    'TITAN',
    'X',
    'FIRE',
    'LIME',
];

export const SearchStream: React.FC = () => {
    const [question, setQuestion] = useState('');
    const [selectedServer, setSelectedServer] = useState('');
    const [answer, setAnswer] = useState('');
    const [isLoading, setIsLoading] = useState(false);

    const handleSearch = async (e: React.FormEvent) => {
        e.preventDefault();

        // Блокируем отправку, если не введен вопрос или не выбран сервер
        if (!question.trim() || !selectedServer) return;

        setAnswer('');
        setIsLoading(true);

        try {
            const response = await fetch('http://localhost:8080/search', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    question,
                    server: selectedServer // Передаем сервер в ВЕРХНЕМ регистре (например: "RED")
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
                        setAnswer((prev) => prev + chunk);
                    }
                }
            }
        } catch (error) {
            console.error('Ошибка потока:', error);
            setAnswer((prev) => prev + '\n[Ошибка получения данных]');
        } finally {
            setIsLoading(false);
        }
    };

    // Кнопка заблокирована, если не заполнено поле вопроса ИЛИ не выбран сервер
    const isSubmitDisabled = isLoading || !question.trim() || !selectedServer;

    return (
        <div style={{ maxWidth: '600px', margin: '40px auto', fontFamily: 'sans-serif' }}>
            <h2>Поиск</h2>

            <form onSubmit={handleSearch} style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginBottom: '20px' }}>
                <div style={{ display: 'flex', gap: '8px' }}>
                    {/* Выпадающий список серверов */}
                    <select
                        value={selectedServer}
                        onChange={(e) => setSelectedServer(e.target.value)}
                        disabled={isLoading}
                        style={{ padding: '8px', fontSize: '16px', borderRadius: '4px', border: '1px solid #ccc' }}
                    >
                        <option value="">-- Выберите сервер --</option>
                        {SERVERS.map((server) => (
                            <option key={server} value={server}>
                                {server}
                            </option>
                        ))}
                    </select>

                    {/* Поле ввода вопроса */}
                    <input
                        type="text"
                        value={question}
                        onChange={(e) => setQuestion(e.target.value)}
                        placeholder="Введите ваш вопрос..."
                        disabled={isLoading}
                        style={{ flex: 1, padding: '8px', fontSize: '16px', borderRadius: '4px', border: '1px solid #ccc' }}
                    />
                </div>

                {/* Кнопка отправки */}
                <button
                    type="submit"
                    disabled={isSubmitDisabled}
                    style={{
                        padding: '10px 16px',
                        fontSize: '16px',
                        cursor: isSubmitDisabled ? 'not-allowed' : 'pointer'
                    }}
                >
                    {isLoading ? 'Идет поиск...' : 'Искать'}
                </button>
            </form>

            {/* Вывод ответа */}
            <div
                style={{
                    border: '1px solid #ccc',
                    padding: '16px',
                    borderRadius: '4px',
                    minHeight: '100px',
                    whiteSpace: 'pre-wrap',
                    background: '#f9f9f9',
                }}
            >
                {answer || (isLoading ? 'Загрузка...' : 'Здесь будет ответ...')}
            </div>
        </div>
    );
};