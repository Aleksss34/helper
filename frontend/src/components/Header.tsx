import React, { useState, useRef, useEffect } from "react";
import { useAuth, AuthModal } from "./AuthModal";
import { useUser } from "./GetUser";
import { AdminPanelModal } from "./Adminpanel";

/**
 * Модалка подтверждения выхода
 */
interface ConfirmLogoutModalProps {
    isOpen: boolean;
    onConfirm: () => void;
    onCancel: () => void;
}

const ConfirmLogoutModal: React.FC<ConfirmLogoutModalProps> = ({ isOpen, onConfirm, onCancel }) => {
    if (!isOpen) return null;

    return (
        <div style={styles.confirmOverlay} onClick={onCancel}>
            <div style={styles.confirmModal} onClick={(e) => e.stopPropagation()}>
                <h3 style={styles.confirmTitle}>Выйти из аккаунта?</h3>
                <p style={styles.confirmText}>
                    Вам придётся войти заново, чтобы продолжить пользоваться сервисом.
                </p>
                <div style={styles.confirmActions}>
                    <button style={styles.cancelBtn} onClick={onCancel}>
                        Отмена
                    </button>
                    <button style={styles.confirmBtn} onClick={onConfirm}>
                        Выйти
                    </button>
                </div>
            </div>
        </div>
    );
};

/**
 * Виджет пользователя в шапке
 */
const UserMenu: React.FC = () => {
    const { isAuthenticated, isLoading, logout } = useAuth();
    const { user } = useUser();

    const [isAuthModalOpen, setIsAuthModalOpen] = useState(false);
    const [isMenuOpen, setIsMenuOpen] = useState(false);
    const [isConfirmOpen, setIsConfirmOpen] = useState(false);
    const menuRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (!isMenuOpen) return;

        const handleClickOutside = (e: MouseEvent) => {
            if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
                setIsMenuOpen(false);
            }
        };

        document.addEventListener("mousedown", handleClickOutside);
        return () => document.removeEventListener("mousedown", handleClickOutside);
    }, [isMenuOpen]);

    if (isLoading) {
        return (
            <button style={styles.headerBtn} disabled>
                ...
            </button>
        );
    }

    if (!isAuthenticated) {
        return (
            <>
                <button style={styles.headerBtn} onClick={() => setIsAuthModalOpen(true)}>
                    Войти
                </button>
                <AuthModal isOpen={isAuthModalOpen} onClose={() => setIsAuthModalOpen(false)} />
            </>
        );
    }

    const status = user?.status;
    const nameColor =
        status === "admin" ? "#f43f5e" :
            status === "VIP" ? "#fbbf24" :
                "#e5e7eb";

    const statusIcon =
        status === "admin" ? "👑" :
            status === "VIP" ? "⭐" :
                null;

    return (
        <div style={styles.menuWrapper} ref={menuRef}>
            <button
                style={{ ...styles.userBtn, color: nameColor }}
                onClick={() => setIsMenuOpen((v) => !v)}
            >
                {statusIcon && <span style={styles.statusIcon}>{statusIcon}</span>}
                <span>{user?.username}</span>
            </button>

            {isMenuOpen && (
                <div style={styles.dropdown}>
                    <button
                        style={styles.dropdownItem}
                        onClick={() => {
                            setIsMenuOpen(false);
                            setIsConfirmOpen(true);
                        }}
                    >
                        Настройки
                    </button>

                    <button
                        style={styles.dropdownItem}
                        onClick={() => {
                            setIsMenuOpen(false);
                            setIsConfirmOpen(true);
                        }}
                    >
                        Выйти
                    </button>

                </div>
            )}

            <ConfirmLogoutModal
                isOpen={isConfirmOpen}
                onCancel={() => setIsConfirmOpen(false)}
                onConfirm={() => {
                    setIsConfirmOpen(false);
                    logout();
                }}
            />
        </div>
    );
};

export const Header: React.FC = () => {
    const [isAdminModalOpen, setIsAdminModalOpen] = useState(false);

    const { user } = useUser();
    const isAdmin = user?.status === "admin";

    return (
        <header style={styles.header}>
            <div style={styles.leftSide}>
                {isAdmin && (
                    <button
                        style={styles.adminBtn}
                        onClick={() => setIsAdminModalOpen(true)}
                    >
                        🔒 ADMIN ONLY
                    </button>
                )}
            </div>

            <div style={styles.logo}>
                <span style={styles.logoAccent}>AMAZING</span> AI Helper
            </div>

            <div style={styles.rightSide}>
                <UserMenu />
            </div>

            <AdminPanelModal
                isOpen={isAdminModalOpen}
                onClose={() => setIsAdminModalOpen(false)}
            />
        </header>
    );
};

const styles: Record<string, React.CSSProperties> = {
    header: {
        display: "grid",
        gridTemplateColumns: "1fr auto 1fr",
        alignItems: "center",
        padding: "16px 24px",
        backgroundColor: "#0f172a",
        borderBottom: "1px solid #1f293d",
        width: "100%",
        boxSizing: "border-box",
    },
    leftSide: {
        gridColumn: "1",
        justifySelf: "start",
        display: "flex",
        alignItems: "center",
    },
    adminBtn: {
        fontSize: "11px",
        fontWeight: 700,
        color: "#f43f5e",
        backgroundColor: "rgba(244, 63, 94, 0.12)",
        border: "1px solid rgba(244, 63, 94, 0.3)",
        borderRadius: "12px",
        padding: "6px 12px",
        textTransform: "uppercase",
        letterSpacing: "0.5px",
        cursor: "pointer",
    },
    logo: {
        gridColumn: "2",
        fontSize: "16px",
        fontWeight: 700,
        color: "#e5e7eb",
        textAlign: "center",
        whiteSpace: "nowrap",
    },
    logoAccent: {
        color: "#7c6cf6",
        marginRight: "4px",
    },
    rightSide: {
        gridColumn: "3",
        justifySelf: "end",
        display: "flex",
        alignItems: "center",
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
    menuWrapper: {
        position: "relative",
    },
    userBtn: {
        display: "flex",
        alignItems: "center",
        gap: "8px",
        background: "rgba(255,255,255,0.06)",
        border: "1px solid #2c3044",
        borderRadius: "999px",
        padding: "10px 20px",
        fontSize: "16px",       // было 13px
        fontWeight: 700,        // было 600
        cursor: "pointer",
    },
    statusIcon: {
        fontSize: "16px",       // было 13px
        lineHeight: 1,
    },
    dropdown: {
        position: "absolute",
        top: "calc(100% + 8px)",
        right: 0,
        background: "#12141f",
        border: "1px solid #262a3d",
        borderRadius: "10px",
        overflow: "hidden",
        boxShadow: "0 12px 30px rgba(0,0,0,0.5)",
        minWidth: "140px",
        zIndex: 100,
    },
    dropdownItem: {
        display: "block",
        width: "100%",
        textAlign: "left",
        background: "transparent",
        border: "none",
        color: "#f87171",
        padding: "10px 14px",
        fontSize: "13px",
        fontWeight: 600,
        cursor: "pointer",
    },
    confirmOverlay: {
        position: "fixed",
        inset: 0,
        background: "rgba(5, 6, 15, 0.7)",
        backdropFilter: "blur(4px)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
    },
    confirmModal: {
        width: "320px",
        maxWidth: "90vw",
        background: "#12141f",
        border: "1px solid #262a3d",
        borderRadius: "16px",
        padding: "24px",
        color: "#e5e7eb",
        boxShadow: "0 20px 60px rgba(0,0,0,0.5)",
        fontFamily: "system-ui, sans-serif",
    },
    confirmTitle: {
        margin: "0 0 8px 0",
        fontSize: "17px",
        fontWeight: 700,
        color: "#ffffff",
    },
    confirmText: {
        margin: "0 0 20px 0",
        fontSize: "13px",
        color: "#9ca3af",
        lineHeight: 1.5,
    },
    confirmActions: {
        display: "flex",
        gap: "10px",
        justifyContent: "flex-end",
    },
    cancelBtn: {
        background: "transparent",
        border: "1px solid #2c3044",
        color: "#cbd5e1",
        borderRadius: "10px",
        padding: "8px 16px",
        fontSize: "13px",
        fontWeight: 600,
        cursor: "pointer",
    },
    confirmBtn: {
        background: "linear-gradient(90deg, #ef4444, #dc2626)",
        border: "none",
        color: "#fff",
        borderRadius: "10px",
        padding: "8px 16px",
        fontSize: "13px",
        fontWeight: 600,
        cursor: "pointer",
    },
};

export default Header;