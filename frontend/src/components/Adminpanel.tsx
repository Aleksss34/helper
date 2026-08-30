import React from 'react';
import { ParseButton } from './Parser';

interface AdminPanelModalProps {
    isOpen: boolean;
    onClose: () => void;
}

export const AdminPanelModal: React.FC<AdminPanelModalProps> = ({ isOpen, onClose }) => {
    if (!isOpen) return null;

    return (
        <div style={styles.overlay} onClick={onClose}>
            <div style={styles.modal} onClick={(e) => e.stopPropagation()}>
                <button style={styles.closeBtn} onClick={onClose} aria-label="Закрыть">
                    ✕
                </button>

                <h2 style={styles.title}>Админ панель</h2>
                <p style={styles.description}>
                    Это админская панель. Здесь доступны служебные действия, недоступные обычным пользователям.
                </p>

                <ParseButton />
            </div>
        </div>
    );
};

const styles: Record<string, React.CSSProperties> = {
    overlay: {
        position: 'fixed',
        inset: 0,
        background: 'rgba(5, 6, 15, 0.7)',
        backdropFilter: 'blur(4px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
    },
    modal: {
        position: 'fixed',
        top: 0,
        left: 0,
        bottom: 0,
        width: '33.33vw',             // 1/3 ширины экрана
        minWidth: '320px',             // Минимальная ширина, чтобы не сжималась на мобилках
        height: '100vh',               // Во всю высоту
        background: '#12141f',
        borderRight: '1px solid #262a3d', // Граница только справа
        borderRadius: '0px',           // Убираем скругления для края экрана
        padding: '32px',
        color: '#e5e7eb',
        boxShadow: '10px 0 40px rgba(0,0,0,0.5)',
        fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
        overflowY: 'auto',             // Чтобы появлялся скролл, если контент не влезает
        zIndex: 1000,                  // Чтобы была поверх остальных элементов
    },
    closeBtn: {
        position: 'absolute',
        top: '16px',
        right: '16px',
        background: 'transparent',
        border: 'none',
        color: '#8a8fa3',
        fontSize: '18px',
        cursor: 'pointer',
    },
    title: {
        textAlign: 'center',
        margin: '0 0 12px 0',
        fontSize: '20px',
        fontWeight: 600,
        color: '#ffffff',
    },
    description: {
        margin: '0 0 24px 0',
        fontSize: '13px',
        color: '#9ca3af',
        lineHeight: 1.5,
    },
};

export default AdminPanelModal;