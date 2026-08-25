import React, { useState } from 'react';

export const ParseLegistationButton: React.FC = () => {
    const [loading, setLoading] = useState(false);
    const [result, setResult] = useState<string | null>(null);

    const handleParse = async () => {
        setLoading(true);
        setResult(null);

        try {
            const response = await fetch('http://127.0.0.1:8080/parse-legislation', {
                method: 'GET',
            });

            if (!response.ok) {
                throw new Error(`Ошибка сервера: ${response.status}`);
            }

            // Если ручка возвращает JSON — можно использовать response.json()
            const data = await response.text();
            setResult(data);
        } catch (error) {
            console.error('Ошибка при запросе /parse:', error);
            setResult(error instanceof Error ? error.message : 'Произошла ошибка');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div style={{ margin: '20px 0', fontFamily: 'sans-serif' }}>
            <button
                onClick={handleParse}
                disabled={loading}
                style={{
                    padding: '10px 20px',
                    fontSize: '16px',
                    cursor: loading ? 'not-allowed' : 'pointer',
                }}
            >
                {loading ? 'Парсинг...' : 'Запустить парсинг законов'}
            </button>
            {loading && (
                <div>
                    Глядеть (каждые 100 ебашит туда): <a href="http://localhost:6333/dashboard#/collections/news" target="_blank" rel="noreferrer">Дашборд Qdrant</a>
                </div>
            )}
            {result && (
                <div style={{ marginTop: '12px', padding: '10px', background: '#f0f0f0', borderRadius: '4px' }}>
                    <pre style={{ margin: 0 }}>{result}</pre>
                </div>
            )}
        </div>
    );
};