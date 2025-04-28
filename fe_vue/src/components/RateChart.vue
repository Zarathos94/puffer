<template>
  <div class="card">
    <header class="header">
      <div class="logo-wrap">
        <img :src="logoUrl" alt="Puffer Logo" class="logo" />
      </div>
      <div>
        <h2>pufETH Conversion Rate</h2>
        <p class="subtitle">Track pufETH/ETH over time</p>
      </div>
    </header>
    <div class="toolbar">
      <button :class="{active: viewMode==='history'}" @click="viewMode='history'" title="Show 24h historical chart">24h History</button>
      <button :class="{active: viewMode==='live'}" @click="viewMode='live'" title="Show live updates">Live</button>
      <button @click="refresh" :disabled="loading" title="Refresh data" style="float:right">⟳</button>
    </div>
    <div>
      <template v-if="error">
        <transition name="fade"><div class="error">⚠️ {{ error }}</div></transition>
      </template>
      <template v-else-if="loading">
        <transition name="fade"><div class="loading"><span class="spinner"></span> Loading data...</div></transition>
      </template>
      <template v-else>
        <div v-if="viewMode==='history'" class="stats-box">
          <div class="stats-horizontal">
            <div class="stat-card latest">
              <div class="stat-label">Latest</div>
              <div class="stat-value">{{ latestRate }} <span class="unit">ETH</span></div>
            </div>
            <div class="stat-card min">
              <div class="stat-label">Min</div>
              <div class="stat-value">{{ minRate }} <span class="unit">ETH</span></div>
            </div>
            <div class="stat-card max">
              <div class="stat-label">Max</div>
              <div class="stat-value">{{ maxRate }} <span class="unit">ETH</span></div>
            </div>
            <div class="stat-card supply">
              <div class="stat-label">Total Supply</div>
              <div class="stat-value">N/A <span class="unit">pufETH</span></div>
            </div>
          </div>
        </div>
        <div v-if="viewMode==='history'" class="chart-container chart-large">
          <Line :data="chartDataHistory" :options="chartOptionsHistory" />
        </div>
        <div v-else class="live-price">
          <span v-if="latestRate !== 'N/A'" class="live-big">{{ latestRate }} <span class="unit">ETH</span></span>
          <span v-else class="live-big">--</span>
        </div>
      </template>
    </div>
    <footer class="footer">
      <a href="https://docs.puffer.fi" target="_blank" rel="noopener">Learn more at Puffer Docs</a>
    </footer>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, onUnmounted, watch } from 'vue';
import { Line } from 'vue-chartjs';
import {
  Chart,
  LineElement,
  PointElement,
  LinearScale,
  Title,
  CategoryScale,
  Tooltip,
  Legend
} from 'chart.js';
import axios from 'axios';

Chart.register(LineElement, PointElement, LinearScale, Title, CategoryScale, Tooltip, Legend);

const logoUrl = 'https://docs.puffer.fi/img/Logo%20Mark.svg';
const apiBase = import.meta.env.VITE_API_URL || '';

const loading = ref(true);
const error = ref('');
const history = ref([]);
const liveData = ref([]);
const viewMode = ref('history'); // 'history' or 'live'
let eventSource = null;

const latestRate = computed(() => {
  if (history.value.length === 0 && liveData.value.length === 0) return 'N/A';
  if (viewMode.value === 'live' && liveData.value.length > 0) {
    return liveData.value[liveData.value.length - 1].rate.toFixed(6);
  }
  if (history.value.length > 0) {
    return history.value[history.value.length - 1].rate.toFixed(6);
  }
  return 'N/A';
});

const minRate = computed(() => {
  const arr = viewMode.value === 'live' ? liveData.value : history.value;
  if (!arr.length) return 'N/A';
  return Math.min(...arr.map(x => x.rate)).toFixed(6);
});
const maxRate = computed(() => {
  const arr = viewMode.value === 'live' ? liveData.value : history.value;
  if (!arr.length) return 'N/A';
  return Math.max(...arr.map(x => x.rate)).toFixed(6);
});
const avgRate = computed(() => {
  const arr = viewMode.value === 'live' ? liveData.value : history.value;
  if (!arr.length) return 'N/A';
  const avg = arr.reduce((sum, x) => sum + x.rate, 0) / arr.length;
  return avg.toFixed(6);
});

function formatHourMinute(ts) {
  const d = new Date(ts * 1000);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false });
}

const chartDataHistory = computed(() => {
  const points = history.value.length > 24
    ? history.value.filter((_, i) => i % Math.floor(history.value.length / 24) === 0)
    : history.value;
  return {
    labels: points.map(item => formatHourMinute(item.timestamp)),
    datasets: [
      {
        label: 'pufETH Conversion Rate (24h)',
        data: points.map(item => item.rate),
        borderColor: '#1976d2',
        backgroundColor: 'rgba(25, 118, 210, 0.12)',
        fill: true,
        tension: 0.4,
        pointRadius: 0,
        borderWidth: 3,
      }
    ]
  };
});

const chartOptionsHistory = computed(() => {
  const arr = history.value;
  if (!arr.length) return baseChartOptions;
  const min = Math.min(...arr.map(x => x.rate));
  const max = Math.max(...arr.map(x => x.rate));
  const padding = (max - min) * 0.1 || 0.001;
  return {
    ...baseChartOptions,
    scales: {
      x: {
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
          callback: function(value) {
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
          label: function(context) {
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
});

const chartDataLive = computed(() => {
  const points = liveData.value.length > 24
    ? liveData.value.slice(-24)
    : liveData.value;
  return {
    labels: points.map(item => formatHourMinute(item.timestamp)),
    datasets: [
      {
        label: 'pufETH Conversion Rate (Live)',
        data: points.map(item => item.rate),
        borderColor: '#1976d2',
        backgroundColor: 'rgba(25, 118, 210, 0.12)',
        fill: true,
        tension: 0.4,
        pointRadius: 0,
        borderWidth: 3,
      }
    ]
  };
});

const chartOptionsLive = chartOptionsHistory;

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
        callback: function(val, idx, ticks) {
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

function refresh() {
  error.value = '';
  loading.value = true;
  if (viewMode.value === 'history') {
    axios.get(apiBase + '/rate/history').then(res => {
      history.value = res.data;
    }).catch(e => {
      error.value = 'Failed to fetch history.';
    }).finally(() => {
      loading.value = false;
    });
  } else {
    liveData.value = [];
    startSSE();
    loading.value = false;
  }
}

function startSSE() {
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }
  eventSource = new EventSource(apiBase + '/sse/rate');
  eventSource.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      liveData.value.push(data);
      if (liveData.value.length > 100) liveData.value.shift();
    } catch (e) {}
  };
  eventSource.onerror = () => {
    error.value = 'Live update connection lost.';
    eventSource.close();
    eventSource = null;
  };
}

onMounted(async () => {
  refresh();
});

onUnmounted(() => {
  if (eventSource) eventSource.close();
});

watch(viewMode, (mode) => {
  error.value = '';
  loading.value = true;
  if (mode === 'history') {
    if (eventSource) eventSource.close();
    refresh();
  } else {
    liveData.value = [];
    startSSE();
    loading.value = false;
  }
});
</script>

<style scoped>
.card {
  background: var(--card-bg, #fff);
  border-radius: 12px;
  box-shadow: 0 2px 16px 0 rgba(0,0,0,0.08);
  padding: 2rem 2.5rem 1.5rem 2.5rem;
  max-width: 800px;
  margin: 40px auto;
  transition: background 0.3s;
}
.header {
  display: flex;
  align-items: center;
  margin-bottom: 1.5rem;
}
.logo-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  border-radius: 50%;
  box-shadow: 0 1px 8px 0 rgba(0,0,0,0.07);
  width: 56px;
  height: 56px;
  margin-right: 1rem;
  padding: 4px;
}
.logo {
  width: 48px;
  height: 48px;
  display: block;
}
.subtitle {
  color: #888;
  font-size: 1rem;
  margin-top: 0.2rem;
}
.toolbar {
  margin-bottom: 1rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
button {
  margin-right: 0.5rem;
  padding: 0.4rem 1.2rem;
  border: 1.5px solid #232736;
  background: #232736;
  border-radius: 6px;
  cursor: pointer;
  font-size: 1rem;
  color: #fff;
  font-weight: 500;
  transition: background 0.2s, color 0.2s, border-color 0.2s, box-shadow 0.2s;
  box-shadow: 0 1px 4px 0 rgba(25, 118, 210, 0.04);
  outline: none;
}
button.active {
  background: #1976d2;
  color: #fff;
  border-color: #1976d2;
  box-shadow: 0 2px 8px 0 rgba(25, 118, 210, 0.10);
}
button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
button:hover:not(:disabled), button:focus:not(:disabled) {
  background: #29304a;
  color: #fff;
  border-color: #1976d2;
}
.error {
  color: #b71c1c;
  background: #ffeaea;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  margin-bottom: 1rem;
  font-size: 1.1rem;
}
.loading {
  color: #1976d2;
  font-size: 1.1rem;
  display: flex;
  align-items: center;
  margin-bottom: 1rem;
}
.spinner {
  width: 18px;
  height: 18px;
  border: 3px solid #1976d2;
  border-top: 3px solid #e0e0e0;
  border-radius: 50%;
  margin-right: 0.7rem;
  animation: spin 1s linear infinite;
}
@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
.stats-box {
  background: var(--card-bg, #fff);
  border: 1.5px solid #e0e0e0;
  border-radius: 10px;
  padding: 1.2rem 0.5rem;
  margin-bottom: 1.2rem;
  box-shadow: 0 1px 6px 0 rgba(25, 118, 210, 0.04);
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  box-sizing: border-box;
}
.stats-horizontal {
  display: flex;
  flex-direction: row;
  justify-content: center;
  align-items: stretch;
  gap: clamp(0.3rem, 2vw, 1.2rem);
  margin-bottom: 0;
  width: 100%;
  max-width: 100%;
  margin-left: auto;
  margin-right: auto;
  box-sizing: border-box;
}
.stat-card {
  background: #232736;
  color: #fff;
  border-radius: 14px;
  box-shadow: 0 2px 8px 0 rgba(25, 118, 210, 0.04);
  padding: clamp(0.4rem, 1.5vw, 0.8rem) clamp(0.7rem, 3vw, 1.2rem);
  min-width: 0;
  max-width: 24%;
  flex: 1 1 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  box-sizing: border-box;
  transition: padding 0.2s, min-width 0.2s, font-size 0.2s;
}
.stat-card .stat-label {
  font-size: clamp(0.7rem, 1.5vw, 0.98rem);
  color: #b0b6c3;
  margin-bottom: 0.18rem;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  font-weight: 500;
}
.stat-card .stat-value {
  font-size: clamp(1rem, 3vw, 1.5rem);
  font-weight: 700;
  color: #fff;
}
.stat-card.latest {
  border: 2px solid #1976d2;
}
.stat-card.min {
  border: 2px solid #43a047;
}
.stat-card.max {
  border: 2px solid #d32f2f;
}
.stat-card.supply {
  border: 2px solid #ffd600;
}
.live-price {
  display: flex;
  justify-content: center;
  align-items: center;
  margin: 2.5rem 0 2.5rem 0;
}
.live-big {
  font-size: 2.8rem;
  font-weight: 700;
  color: #1976d2;
  background: #e3f2fd;
  border-radius: 10px;
  padding: 0.5rem 2.5rem;
  box-shadow: 0 2px 8px 0 rgba(25, 118, 210, 0.07);
}
.footer {
  margin-top: 2.5rem;
  text-align: right;
  font-size: 0.98rem;
}
.footer a {
  color: #1976d2;
  text-decoration: none;
  transition: color 0.2s;
}
.footer a:hover {
  color: #0d47a1;
  text-decoration: underline;
}
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
.stats-bar .latest {
  color: #1976d2;
  font-weight: bold;
  background: #e3f2fd;
  border-radius: 4px;
  padding: 2px 8px;
}
.stats-bar .min {
  color: #388e3c;
  font-weight: bold;
  background: #e8f5e9;
  border-radius: 4px;
  padding: 2px 8px;
}
.stats-bar .max {
  color: #d32f2f;
  font-weight: bold;
  background: #ffebee;
  border-radius: 4px;
  padding: 2px 8px;
}
.chart-container,
.chart-container.chart-large {
  background: #181c24;
  border-radius: 18px;
  box-shadow: 0 2px 16px 0 rgba(25, 118, 210, 0.08);
  padding: 2.5rem 1.5rem 2rem 1.5rem;
  width: 100%;
  min-height: 320px;
  max-width: 1100px;
  margin: 0 auto 0.5rem auto;
  display: flex;
  align-items: center;
  justify-content: center;
}
@media (max-width: 900px) {
  .stats-horizontal {
    gap: 0.5vw;
    max-width: 100vw;
  }
  .stat-card {
    max-width: 48%;
    padding: 0.5rem 0.5rem;
    font-size: 0.98rem;
  }
}
@media (max-width: 600px) {
  .stats-horizontal {
    gap: 0.2vw;
    max-width: 100vw;
  }
  .stat-card {
    max-width: 100%;
    padding: 0.3rem 0.2rem;
    font-size: 0.88rem;
  }
}

@media (max-width: 400px) {
  .card {
    padding: 0.5rem 0.1rem 0.5rem 0.1rem;
  }
}

@media (prefers-color-scheme: dark) {
  :root {
    --card-bg: #181c24;
    --btn-bg: #232a36;
  }
  .card {
    background: var(--card-bg, #181c24);
    color: #f3f3f3;
  }
  .subtitle {
    color: #bbb;
  }
  .stats-bar {
    color: #e0e0e0;
  }
  .error {
    background: #3a1a1a;
    color: #ffbaba;
  }
  .loading {
    color: #90caf9;
  }
  .footer a {
    color: #90caf9;
  }
  .footer a:hover {
    color: #42a5f5;
  }
  button {
    background: var(--btn-bg, #232a36);
    color: #f3f3f3;
    border-color: #444;
  }
  button.active {
    background: #1976d2;
    color: #fff;
    border-color: #1976d2;
  }
  .stats-box {
    background: #232736;
    border-color: #333a4d;
  }
}
</style> 