// widget.tsx
import { useUser } from "./GetUser.tsx";

const statusConfig = {
    base: (remainingQuestions: number) => {
        // Если осталось 5 или меньше запросов
        if (remainingQuestions <= 5 && remainingQuestions > 0) {
            return {
                icon: '⚠️',
                text: <>У вас осталось <b>{remainingQuestions}</b> {getQuestionDeclension(remainingQuestions)}. Приобретите VIP для безграничного доступа!</>,
                bg: 'rgba(239, 68, 68, 0.15)', // Красный акцент
                border: 'rgba(239, 68, 68, 0.4)',
                color: '#fca5a5',
            };
        } else if (remainingQuestions === 0) {
            return {
                icon: '🔒',
                text: (
                    <span>
                Лимит исчерпан.{' '}
                        <a
                            href="https://www.google.com/"
                            target="_blank"
                            rel="noreferrer"
                            style={{
                                color: '#ffffff',
                                background: 'linear-gradient(135deg, #f59e0b 0%, #d97706 100%)',
                                padding: '4px 10px',
                                borderRadius: '12px',
                                textDecoration: 'none',
                                fontWeight: 600,
                                marginLeft: '4px',
                                boxShadow: '0 2px 8px rgba(245, 158, 11, 0.3)',
                                display: 'inline-block',
                                transition: 'transform 0.15s ease'
                            }}
                        >
                    Купить VIP ➔
                </a>
            </span>
                ),
                bg: 'rgba(245, 158, 11, 0.1)',
                border: 'rgba(245, 158, 11, 0.4)',
                color: '#fcd34d',
            };
        }

        // Если осталось больше 5 запросов
        return {
            icon: '🔒',
            text: <>Ваши запросы ограничены. Вы можете приобрести тариф VIP</>,
            bg: 'rgba(51, 65, 85, 0.5)',
            border: '#475569',
            color: '#cbd5e1',
        };
    },
    VIP: () => ({
        icon: '⭐',
        text: <><b>Premium</b> Подписка активна. Наслаждайтесь!</>,
        bg: 'rgba(245, 158, 11, 0.1)',
        border: 'rgba(245, 158, 11, 0.4)',
        color: '#fcd34d',
    }),
    admin: () => ({
        icon: '🚀',
        text: <>Вы являетесь <b>Администратором</b>. Безграничные возможности!</>,
        bg: 'rgba(16, 185, 129, 0.1)',
        border: 'rgba(16, 185, 129, 0.4)',
        color: '#6ee7b7',
    }),
};

// Функция для правильного склонения слова "запрос"
function getQuestionDeclension(count: number): string {
    const lastTwo = count % 100;
    const lastOne = count % 10;
    if (lastTwo >= 11 && lastTwo <= 19) return 'запросов';
    if (lastOne === 1) return 'запрос';
    if (lastOne >= 2 && lastOne <= 4) return 'запроса';
    return 'запросов';
}

export function UserStatusWidget() {
    const { user, loading, error } = useUser();

    if (loading || error || !user) return null;

    const getConfig = statusConfig[user.status as keyof typeof statusConfig];
    if (!getConfig) return null;

    // Передаем напрямую оставшееся число вопросов из поля user
    const questionsLeft = user.count_questions ?? user.count_questions ?? 0;
    const config = getConfig(questionsLeft);

    return (
        <div style={{ display: 'flex', justifyContent: 'center', paddingBottom: '12px' }}>
            <div
                style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '8px',
                    padding: '8px 18px',
                    borderRadius: '999px',
                    border: `1px solid ${config.border}`,
                    backgroundColor: config.bg,
                    color: config.color,
                    fontSize: '13px',
                    fontWeight: 500,
                    transition: 'all 0.3s ease',
                }}
            >
                <span>{config.icon}</span>
                <span>{config.text}</span>
            </div>
        </div>
    );
}