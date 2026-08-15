import React, { useState } from 'react';

export const SearchStream: React.FC = () => {
    const [question, setQuestion] = useState('');
    const [answer, setAnswer] = useState('');
    const [isLoading, setIsLoading] = useState(false);

    const handleSearch = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!question.trim()) return;

        setAnswer('');
        setIsLoading(true);

        try {
            const response = await fetch('http://localhost:8080/search', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ question }),
            });

            if (!response.ok || !response.body) {
                throw new Error(' Ошибка запроса');
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder();

            while (true) {
                const { value, done } = await reader.read();
                if (done) break;

                const text = decoder.decode(value);
                // Разбиваем полученный буфер по линиям SSE
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

    return (
        <div style={{ maxWidth: '600px', margin: '40px auto', fontFamily: 'sans-serif' }}>
            <h2>Поиск</h2>
            <form onSubmit={handleSearch} style={{ display: 'flex', gap: '8px', marginBottom: '20px' }}>
                <input
                    type="text"
                    value={question}
                    onChange={(e) => setQuestion(e.target.value)}
                    placeholder="Введите ваш вопрос..."
                    disabled={isLoading}
                    style={{ flex: 1, padding: '8px', fontSize: '16px' }}
                />
                <button type="submit" disabled={isLoading} style={{ padding: '8px 16px', fontSize: '16px' }}>
                    {isLoading ? 'Идет поиск...' : 'Искать'}
                </button>
            </form>

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