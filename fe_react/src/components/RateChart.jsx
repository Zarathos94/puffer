import React, { useEffect, useRef, useState } from 'react';
import { Line } from 'react-chartjs-2';
import {
    Chart as ChartJS,
    LineElement,
    PointElement,
    LinearScale,
    Title,
    CategoryScale,
    Tooltip,
    Legend
} from 'chart.js';
import axios from 'axios';
import styles from './RateChart.module.css';

ChartJS.register(LineElement, PointElement, LinearScale, Title, CategoryScale, Tooltip, Legend);

const logoUrl = 'https://docs.puffer.fi/img/Logo%20Mark.svg';
const apiBase = import.meta.env.VITE_API_URL || 'http://localhost:8080';

const baseChartOptions = {
    responsive: true,
    plugins: {
        legend: { display: false },
        title: { display: false },
        tooltip: { enabled: true },
    },
    scales: {
        x: {
            title: { display: true, text: 'Time (24h)', font: { size: 16 } },
            ticks: {
                maxTicksLimit: 24,
                font: { size: 14 },
                callback: function (val, idx, ticks) {
                    return this.getLabelForValue(val);
                }
            },
            grid: { color: '#e0e0e0' }
        },
        y: {
            title: { display: true, text: 'Rate (ETH)', font: { size: 16 } },
            beginAtZero: false,
            ticks: { font: { size: 14 } },
            grid: { color: '#e0e0e0' }
        }
    },
    hover: { mode: 'nearest', intersect: false }
};

function formatHourMinute(ts) {
    const d = new Date(ts * 1000);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false });
}

const RateChart = () => {
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [history, setHistory] = useState([]);
    const [liveData, setLiveData] = useState([]);
    const [viewMode, setViewMode] = useState('history'); // 'history' or 'live'
    const eventSourceRef = useRef(null);

    const latestRate = (() => {
        if (history.length === 0 && liveData.length === 0) return 'N/A';
        if (viewMode === 'live' && liveData.length > 0) {
            return liveData[liveData.length - 1].rate.toFixed(6);
        }
        if (history.length > 0) {
            return history[history.length - 1].rate.toFixed(6);
        }
        return 'N/A';
    })();

    const minRate = (() => {
        const arr = viewMode === 'live' ? liveData : history;
        if (!arr.length) return 'N/A';
        return Math.min(...arr.map(x => x.rate)).toFixed(6);
    })();
    const maxRate = (() => {
        const arr = viewMode === 'live' ? liveData : history;
        if (!arr.length) return 'N/A';
        return Math.max(...arr.map(x => x.rate)).toFixed(6);
    })();

    // Helper to get latest totalSupply
    const latestTotalSupply = (() => {
        if (viewMode === 'live' && liveData.length > 0) {
            const val = liveData[liveData.length - 1].total_supply;
            if (val !== undefined && val !== null) return val;
        }
        if (history.length > 0) {
            const val = history[history.length - 1].total_supply;
            if (val !== undefined && val !== null) return val;
        }
        return 'N/A';
    })();

    const chartDataHistory = {
        labels: (history.length > 24
            ? history.filter((_, i) => i % Math.floor(history.length / 24) === 0)
            : history
        ).map(item => formatHourMinute(item.timestamp)),
        datasets: [
            {
                label: 'pufETH Conversion Rate (24h)',
                data: (history.length > 24
                    ? history.filter((_, i) => i % Math.floor(history.length / 24) === 0)
                    : history
                ).map(item => item.rate),
                borderColor: '#1976d2',
                backgroundColor: 'rgba(25, 118, 210, 0.12)',
                fill: true,
                tension: 0.4,
                pointRadius: 0,
                borderWidth: 3,
            }
        ]
    };

    const chartOptionsHistory = (() => {
        const arr = history;
        if (!arr.length) return baseChartOptions;
        const min = Math.min(...arr.map(x => x.rate));
        const max = Math.max(...arr.map(x => x.rate));
        const padding = (max - min) * 0.1 || 0.001;
        return {
            ...baseChartOptions,
            scales: {
                x: {
                    ...baseChartOptions.scales.x,
                    display: true,
                    grid: {
                        display: false,
                        drawBorder: false,
                    },
                    ticks: {
                        color: '#aab0bb',
                        font: {
                            size: 11,
                        },
                        maxTicksLimit: 6,
                    },
                    title: {
                        display: false,
                    }
                },
                y: {
                    ...baseChartOptions.scales.y,
                    display: true,
                    position: 'right',
                    min: min - padding,
                    max: max + padding,
                    grid: {
                        color: 'rgba(70, 80, 100, 0.08)',
                        drawBorder: false,
                    },
                    ticks: {
                        color: '#aab0bb',
                        font: {
                            size: 11,
                        },
                        maxTicksLimit: 5,
                        callback: function (value) {
                            return value.toFixed(4);
                        }
                    },
                    title: {
                        display: false,
                    }
                }
            },
            plugins: {
                legend: { display: false },
                tooltip: {
                    enabled: true,
                    backgroundColor: 'rgba(15, 20, 30, 0.75)',
                    titleColor: '#fff',
                    bodyColor: '#fff',
                    padding: 10,
                    cornerRadius: 6,
                    displayColors: false,
                    callbacks: {
                        label: function (context) {
                            return `Rate: ${context.parsed.y.toFixed(6)} ETH`;
                        }
                    }
                },
            },
            elements: {
                line: {
                    borderJoinStyle: 'round',
                    borderCapStyle: 'round',
                }
            }
        };
    })();

    // Live chart data (not shown in UI, but could be used for live chart)
    // const chartDataLive = ...

    function refresh() {
        setError('');
        setLoading(true);
        if (viewMode === 'history') {
            axios.get(apiBase + '/rate/history').then(res => {
                setHistory(res.data);
            }).catch(e => {
                setError('Failed to fetch history.');
            }).finally(() => {
                setLoading(false);
            });
        } else {
            setLiveData([]);
            startSSE();
            setLoading(false);
        }
    }

    function startSSE() {
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
            eventSourceRef.current = null;
        }
        const es = new window.EventSource(apiBase + '/sse/rate');
        eventSourceRef.current = es;
        es.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                setLiveData(prev => {
                    const next = [...prev, data];
                    if (next.length > 100) next.shift();
                    return next;
                });
            } catch (e) { }
        };
        es.onerror = () => {
            setError('Live update connection lost.');
            es.close();
            eventSourceRef.current = null;
        };
    }

    useEffect(() => {
        refresh();
        // Cleanup on unmount
        return () => {
            if (eventSourceRef.current) eventSourceRef.current.close();
        };
        // eslint-disable-next-line
    }, []);

    useEffect(() => {
        setError('');
        setLoading(true);
        if (viewMode === 'history') {
            if (eventSourceRef.current) eventSourceRef.current.close();
            refresh();
        } else {
            setLiveData([]);
            startSSE();
            setLoading(false);
        }
        // eslint-disable-next-line
    }, [viewMode]);

    return (
        <div className={styles.card}>
            <header className={styles.header}>
                <div className={styles['logo-wrap']}>
                    <img src={logoUrl} alt="Puffer Logo" className={styles.logo} />
                </div>
                <div>
                    <h2>pufETH Conversion Rate</h2>
                    <p className={styles.subtitle}>Track pufETH/ETH over time</p>
                </div>
            </header>
            <div className={styles.toolbar}>
                <button className={viewMode === 'history' ? styles.active : ''} onClick={() => setViewMode('history')} title="Show 24h historical chart">24h History</button>
                <button className={viewMode === 'live' ? styles.active : ''} onClick={() => setViewMode('live')} title="Show live updates">Live</button>
                <button onClick={refresh} disabled={loading} title="Refresh data" style={{ float: 'right' }}>⟳</button>
            </div>
            <div>
                {error ? (
                    <div className={styles.error}>⚠️ {error}</div>
                ) : loading ? (
                    <div className={styles.loading}><span className={styles.spinner}></span> Loading data...</div>
                ) : (
                    <>
                        {viewMode === 'history' && (
                            <div className={styles['stats-box']}>
                                <div className={styles['stats-horizontal']}>
                                    <div className={`${styles['stat-card']} ${styles.latest}`}>
                                        <div className={styles['stat-label']}>Latest</div>
                                        <div className={styles['stat-value']}>{latestRate} <span className={styles.unit}>ETH</span></div>
                                    </div>
                                    <div className={`${styles['stat-card']} ${styles.min}`}>
                                        <div className={styles['stat-label']}>Min</div>
                                        <div className={styles['stat-value']}>{minRate} <span className={styles.unit}>ETH</span></div>
                                    </div>
                                    <div className={`${styles['stat-card']} ${styles.max}`}>
                                        <div className={styles['stat-label']}>Max</div>
                                        <div className={styles['stat-value']}>{maxRate} <span className={styles.unit}>ETH</span></div>
                                    </div>
                                    <div className={`${styles['stat-card']} ${styles.supply}`}>
                                        <div className={styles['stat-label']}>Total Supply</div>
                                        <div className={styles['stat-value']}>
                                            {latestTotalSupply !== 'N/A' ? (
                                                <>{latestTotalSupply} <span className={styles.unit}>pufETH</span></>
                                            ) : (
                                                <>N/A <span className={styles.unit}>pufETH</span></>
                                            )}
                                        </div>
                                    </div>
                                </div>
                            </div>
                        )}
                        {viewMode === 'history' ? (
                            <div className={`${styles['chart-container']} ${styles['chart-large']}`}>
                                <Line data={chartDataHistory} options={chartOptionsHistory} />
                            </div>
                        ) : (
                            <div className={styles['live-price']}>
                                {latestRate !== 'N/A' ? (
                                    <span className={styles['live-big']}>{latestRate} <span className={styles.unit}>ETH</span></span>
                                ) : (
                                    <span className={styles['live-big']}>--</span>
                                )}
                            </div>
                        )}
                    </>
                )}
            </div>
            <footer className={styles.footer}>
                <a href="https://docs.puffer.fi" target="_blank" rel="noopener noreferrer">Learn more at Puffer Docs</a>
            </footer>
        </div>
    );
};

export default RateChart; 