import bigSmokeImg from "../assets/bigsmoouk-removebg-preview.png";

export function BigSmouk() {
    return (
        <img
            src={bigSmokeImg}
            alt="Big Smoke"
            style={{
                position: 'fixed',
                bottom: '100px',
                left: '0px',
                width: '400px',
                height: 'auto',
                maxHeight: '500px',
                objectFit: 'contain',
                zIndex: 50,
                pointerEvents: 'none',
                filter: 'drop-shadow(0 8px 16px rgba(0,0,0,0.6))'
            }}
        />
    );
}