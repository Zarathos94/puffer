import React from 'react';
import RateChart from './components/RateChart';

function App() {
    return (
        <div style={{
            minHeight: '100vh',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center'
        }}>
            <div style={{
                maxWidth: 800,
                width: '100%',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center'
            }}>
                <RateChart />
            </div>
        </div>
    );
}

export default App; 