
import bigSmokeImg from './assets/bigsmoouk-removebg-preview.png'; // Убедись, что тут PNG с прозрачным фоном
import { SearchStream } from './components/Search.tsx';
import { ParseButton } from './components/Parser.tsx';

function App() {
    return (
        <div style={{ position: 'relative', minHeight: '100vh', width: '100vw', overflow: 'hidden' }}>

            {/* Биг Смоук зафиксирован слева внизу и отображается ВСЕГДА */}
            <img
                src={bigSmokeImg}
                alt="Big Smoke"
                style={{
                    position: 'fixed',
                    bottom: '100px',          // Отступ от нижнего края
                    left: '0px',            // Отступ от левого края
                    width: '400px',          // Крупный размер
                    height: 'auto',
                    maxHeight: '500px',
                    objectFit: 'contain',
                    zIndex: 50,              // Поверх фона, но не перекрывает кнопки
                    pointerEvents: 'none',   // Чтобы не мешал кликам, если закроет элементы
                    filter: 'drop-shadow(0 8px 16px rgba(0,0,0,0.6))'
                }}
            />

            <SearchStream />
            <ParseButton />
        </div>
    );
}

export default App;