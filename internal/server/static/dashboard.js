const API_BASE = '/api';

let windChart = null;
let hellmannChart = null;

async function loadLocations() {
    try {
        const res = await fetch(`${API_BASE}/locations`);
        if (!res.ok) {
            throw new Error(`HTTP error! status: ${res.status}`);
        }
        const locations = await res.json();

        const select = document.getElementById('locations');
        select.innerHTML = '';

        locations.forEach(loc => {
            const option = document.createElement('option');
            option.value = loc.name;
            option.textContent = loc.name;
            select.appendChild(option);
        });

        // Standardzeitraum setzen (letztes Jahr)
        const today = new Date();
        const oneYearAgo = new Date(today.getFullYear() - 1, today.getMonth(), today.getDate());

        document.getElementById('date-end').value = today.toISOString().split('T')[0];
        document.getElementById('date-start').value = oneYearAgo.toISOString().split('T')[0];
    } catch (error) {
        console.error('Fehler beim Laden der Standorte:', error);
        showError('Fehler beim Laden der Standorte: ' + error.message);
    }
}

async function loadTimeSeries(metric) {
    const locations = Array.from(document.getElementById('locations').selectedOptions)
        .map(opt => opt.value);

    if (locations.length === 0) {
        showError('Bitte mindestens einen Standort auswählen');
        return null;
    }

    const startDate = document.getElementById('date-start').value;
    const endDate = document.getElementById('date-end').value;

    if (!startDate || !endDate) {
        showError('Bitte Start- und Enddatum auswählen');
        return null;
    }

    try {
        const res = await fetch(`${API_BASE}/timeseries`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                locations,
                startDate,
                endDate,
                metric
            })
        });

        if (!res.ok) {
            throw new Error(`HTTP error! status: ${res.status}`);
        }

        return await res.json();
    } catch (error) {
        console.error(`Fehler beim Laden der ${metric}-Zeitreihe:`, error);
        showError(`Fehler beim Laden der ${metric}-Zeitreihe: ` + error.message);
        return null;
    }
}

function renderWindChart(data) {
    if (!data || Object.keys(data).length === 0) {
        showError('Keine Windgeschwindigkeitsdaten verfügbar');
        return;
    }

    if (!windChart) {
        windChart = echarts.init(document.getElementById('wind-chart'));
    }

    const locations = Object.keys(data);
    const series = locations.map(loc => ({
        name: loc,
        type: 'line',
        data: data[loc].map(p => [p.time, p.value]),
        smooth: true
    }));

    windChart.setOption({
        title: { text: 'Windgeschwindigkeit über Zeit' },
        tooltip: {
            trigger: 'axis',
            formatter: function(params) {
                let result = params[0].axisValue + '<br/>';
                params.forEach(param => {
                    result += `${param.marker} ${param.seriesName}: ${param.value[1].toFixed(2)} m/s<br/>`;
                });
                return result;
            }
        },
        xAxis: { type: 'time' },
        yAxis: { type: 'value', name: 'm/s' },
        series: series,
        legend: { data: locations },
        dataZoom: [
            { type: 'slider', bottom: 10 },
            { type: 'inside' }
        ],
        grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true }
    }, true);
}

function renderHellmannChart(data) {
    if (!data || Object.keys(data).length === 0) {
        showError('Keine Hellmann-Exponent-Daten verfügbar');
        return;
    }

    if (!hellmannChart) {
        hellmannChart = echarts.init(document.getElementById('hellmann-chart'));
    }

    const locations = Object.keys(data);
    const series = locations.map(loc => ({
        name: loc,
        type: 'line',
        data: data[loc].map(p => [p.time, p.value]),
        smooth: true
    }));

    hellmannChart.setOption({
        title: { text: 'Hellmann-Exponent über Zeit' },
        tooltip: {
            trigger: 'axis',
            formatter: function(params) {
                let result = params[0].axisValue + '<br/>';
                params.forEach(param => {
                    result += `${param.marker} ${param.seriesName}: ${param.value[1].toFixed(4)}<br/>`;
                });
                return result;
            }
        },
        xAxis: { type: 'time' },
        yAxis: { type: 'value', name: 'α' },
        series: series,
        legend: { data: locations },
        dataZoom: [
            { type: 'slider', bottom: 10 },
            { type: 'inside' }
        ],
        grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true }
    }, true);
}

function showError(message) {
    // Entferne existierende Fehlermeldungen
    const existingErrors = document.querySelectorAll('.error');
    existingErrors.forEach(el => el.remove());

    const errorDiv = document.createElement('div');
    errorDiv.className = 'error';
    errorDiv.textContent = message;

    const firstSection = document.querySelector('.filters');
    firstSection.parentNode.insertBefore(errorDiv, firstSection.nextSibling);

    // Auto-remove nach 5 Sekunden
    setTimeout(() => {
        errorDiv.remove();
    }, 5000);
}

async function loadValidation() {
    const locations = Array.from(document.getElementById('locations').selectedOptions)
        .map(opt => opt.value);

    if (locations.length === 0) {
        return;
    }

    try {
        const res = await fetch(`${API_BASE}/validation`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ locations })
        });

        if (!res.ok) {
            throw new Error(`HTTP error! status: ${res.status}`);
        }

        const data = await res.json();
        renderValidationTable(data);
    } catch (error) {
        console.error('Fehler beim Laden der Validierungsmetriken:', error);
    }
}

function renderValidationTable(data) {
    const container = document.getElementById('validation-table');

    if (!data || data.length === 0) {
        container.innerHTML = '<p class="hint">Keine Validierungsmetriken verfügbar.</p>';
        return;
    }

    let html = '<table><thead><tr><th>Standort</th><th>Höhe [m]</th><th>MAE</th><th>RMSE</th><th>Korrelation</th><th>Stichproben</th></tr></thead><tbody>';

    data.forEach(row => {
        html += `<tr>
            <td>${row.location}</td>
            <td>${row.heightM.toFixed(0)}</td>
            <td>${row.mae.toFixed(4)}</td>
            <td>${row.rmse.toFixed(4)}</td>
            <td>${row.correlation.toFixed(4)}</td>
            <td>${row.sampleCount}</td>
        </tr>`;
    });

    html += '</tbody></table>';
    container.innerHTML = html;
}

document.getElementById('load-btn').addEventListener('click', async () => {
    const btn = document.getElementById('load-btn');
    btn.disabled = true;
    btn.textContent = 'Laden...';

    try {
        const windData = await loadTimeSeries('wind');
        if (windData) {
            renderWindChart(windData);
        }

        const hellmannData = await loadTimeSeries('hellmann');
        if (hellmannData) {
            renderHellmannChart(hellmannData);
        }

        await loadValidation();
    } finally {
        btn.disabled = false;
        btn.textContent = 'Daten laden';
    }
});

// Fenster-Resize handling
window.addEventListener('resize', () => {
    if (windChart) windChart.resize();
    if (hellmannChart) hellmannChart.resize();
});

// Initialisierung
loadLocations();
