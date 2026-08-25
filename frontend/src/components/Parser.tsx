import React, { useState } from 'react';

export const ParseButton: React.FC = () => {
    const [loading, setLoading] = useState(false);
    const [result, setResult] = useState<string | null>(null);

    const handleParse = async () => {
        setLoading(true);
        setResult(null);

        try {
            const response = await fetch('http://127.0.0.1:8080/parse', {
                method: 'GET',
            });

            if (!response.ok) {
                throw new Error(`Ошибка сервера: ${response.status}`);
            }

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
        <div style={{
            position: 'fixed',
            top: '16px',
            right: '24px',
            zIndex: 1000,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'flex-end',
            gap: '8px',
            fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
        }}>
            {/* Основной контейнер кнопки и плашки */}
            <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: '10px',
                backgroundColor: '#1e293b',
                padding: '6px 6px 6px 14px',
                borderRadius: '20px',
                border: '1px solid #334155',
                boxShadow: '0 4px 16px rgba(0, 0, 0, 0.4)'
            }}>
                {/* Метка "Только для админов" */}
                <span style={{
                    fontSize: '11px',
                    fontWeight: '700',
                    color: '#f43f5e', // Неоново-красный/розовый цвет
                    backgroundColor: 'rgba(244, 63, 94, 0.12)',
                    padding: '3px 8px',
                    borderRadius: '12px',
                    border: '1px solid rgba(244, 63, 94, 0.3)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.5px'
                }}>
                    🔒 Admin Only
                </span>

                {/* Кнопка запуска */}
                <button
                    onClick={handleParse}
                    disabled={loading}
                    style={{
                        padding: '8px 16px',
                        fontSize: '13px',
                        fontWeight: '600',
                        borderRadius: '14px',
                        border: 'none',
                        background: loading
                            ? '#334155'
                            : 'linear-gradient(135deg, #e11d48 0%, #be123c 100%)', // Градиент для админских действий
                        color: loading ? '#94a3b8' : '#ffffff',
                        cursor: loading ? 'not-allowed' : 'pointer',
                        transition: 'all 0.2s ease',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '6px',
                        boxShadow: loading ? 'none' : '0 2px 8px rgba(225, 29, 72, 0.3)'
                    }}
                >
                    {loading ? (
                        <>
                            <span style={{ display: 'inline-block', animation: 'spin 1s linear infinite' }}>🔄</span>
                            Парсинг...
                        </>
                    ) : (
                        <>⚡ Запустить парсинг</>
                    )}
                </button>
            </div>

            {/* Всплывающее окно статуса парсинга со ссылкой на Qdrant */}
            {loading && (
                <div style={{
                    backgroundColor: '#0f172a',
                    border: '1px solid #334155',
                    borderRadius: '12px',
                    padding: '10px 14px',
                    fontSize: '12px',
                    color: '#94a3b8',
                    boxShadow: '0 4px 20px rgba(0, 0, 0, 0.5)',
                    animation: 'fadeIn 0.2s ease-in-out'
                }}>
                    Отслеживание (пачки по 100 ед.):{' '}
                    <a
                        href="http://localhost:6333/dashboard#/collections/news"
                        target="_blank"
                        rel="noreferrer"
                        style={{
                            color: '#60a5fa',
                            textDecoration: 'none',
                            fontWeight: '600'
                        }}
                    >
                        Дашборд Qdrant ↗
                    </a>
                </div>
            )}

            {/* Вывод результата парсинга */}
            {result && (
                <div style={{
                    maxWidth: '320px',
                    backgroundColor: '#0f172a',
                    border: '1px solid #334155',
                    borderRadius: '12px',
                    padding: '12px',
                    fontSize: '12px',
                    color: '#e2e8f0',
                    maxHeight: '200px',
                    overflowY: 'auto',
                    boxShadow: '0 4px 20px rgba(0, 0, 0, 0.5)'
                }}>
                    <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontFamily: 'monospace' }}>
                        {result}
                    </pre>
                </div>
            )}
        </div>
    );
};