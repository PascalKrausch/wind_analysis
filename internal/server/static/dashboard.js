const API_BASE = '/api';

let windChart = null;
let hellmannChart = null;
let distributionCharts = [];
let hellmannDistributionCharts = [];

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
                let result = params[0].axisValueLabel + '<br/>';
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
                let result = params[0].axisValueLabel + '<br/>';
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

async function loadDistributions() {
    const locations = Array.from(document.getElementById('locations').selectedOptions)
        .map(opt => opt.value);
    const container = document.getElementById('distribution-results');

    if (locations.length === 0) {
        container.innerHTML = '<p class="hint">Bitte mindestens einen Standort auswählen.</p>';
        return;
    }

    try {
        const res = await fetch(`${API_BASE}/distributions`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                locations,
                startDate: document.getElementById('date-start').value,
                endDate: document.getElementById('date-end').value,
                heightM: Number(document.getElementById('distribution-height').value)
            })
        });

        if (!res.ok) {
            throw new Error((await res.text()).trim() || `HTTP error! status: ${res.status}`);
        }

        renderDistributions(await res.json());
    } catch (error) {
        console.error('Fehler beim Laden der Verteilungsanalysen:', error);
        container.textContent = 'Fehler beim Laden der Verteilungsanalysen: ' + error.message;
    }
}

async function loadHellmannDistributions() {
    const locations = Array.from(document.getElementById('locations').selectedOptions)
        .map(opt => opt.value);
    const container = document.getElementById('hellmann-distribution-results');

    if (locations.length === 0) {
        if (container) {
            container.innerHTML = '<p class="hint">Bitte mindestens einen Standort auswählen.</p>';
        }
        return;
    }

    try {
        const res = await fetch(`${API_BASE}/hellmann-distributions`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                locations,
                startDate: document.getElementById('date-start').value,
                endDate: document.getElementById('date-end').value,
                groupBy: 'location'
            })
        });

        if (!res.ok) {
            throw new Error((await res.text()).trim() || `HTTP error! status: ${res.status}`);
        }

        renderHellmannDistributions(await res.json());
    } catch (error) {
        console.error('Fehler beim Laden der Hellmann-Verteilungsanalysen:', error);
        if (container) {
            container.textContent = 'Fehler beim Laden der Hellmann-Verteilungsanalysen: ' + error.message;
        }
    }
}

function renderHellmannDistributions(data) {
    const container = document.getElementById('hellmann-distribution-results');
    if (!container) {
        console.error('Container für Hellmann-Verteilungen nicht gefunden');
        return;
    }

    // Bestehende Charts aufräumen
    hellmannDistributionCharts.forEach(chart => chart.dispose());
    hellmannDistributionCharts = [];

    container.replaceChildren();

    if (!data || data.length === 0) {
        container.innerHTML = '<p class="hint">Keine Hellmann-Verteilungsanalysen für die Auswahl verfügbar.</p>';
        return;
    }

    const table = document.createElement('table');
    table.innerHTML = '<thead><tr><th>Standort</th><th>Bestes Modell</th><th>Stichproben</th><th>AIC</th><th>BIC</th><th>KS</th><th>RMSE</th></tr></thead>';
    const tbody = document.createElement('tbody');

    data.forEach((result, index) => {
        const row = document.createElement('tr');
        [
            result.location,
            result.modelName,
            result.sampleCount,
            result.aic.toFixed(2),
            result.bic.toFixed(2),
            result.ksStatistic.toFixed(4),
            result.rmse.toFixed(4)
        ].forEach(value => {
            const cell = document.createElement('td');
            cell.textContent = value;
            row.appendChild(cell);
        });
        tbody.appendChild(row);

        const card = document.createElement('article');
        card.className = 'distribution-result';
        const heading = document.createElement('h3');
        heading.textContent = `${result.location} – ${result.modelName} (Hellmann-Exponent)`;
        card.appendChild(heading);

        const charts = document.createElement('div');
        charts.className = 'distribution-charts';
        const histogram = document.createElement('div');
        histogram.className = 'distribution-chart';
        histogram.id = `hellmann-histogram-${index}`;
        const cdf = document.createElement('div');
        cdf.className = 'distribution-chart';
        cdf.id = `hellmann-cdf-${index}`;
        charts.append(histogram, cdf);
        card.appendChild(charts);
        container.appendChild(card);

        const histogramChart = echarts.init(histogram);
        histogramChart.setOption({
            title: { text: 'Histogramm und angepasste Dichte' },
            tooltip: { trigger: 'axis' },
            legend: { data: ['Häufigkeit', `${result.modelName}-PDF`] },
            xAxis: { type: 'value', name: 'Hellmann-Exponent α' },
            yAxis: [
                { type: 'value', name: 'Häufigkeit' },
                { type: 'value', name: 'Dichte', min: 0 }
            ],
            series: [
                {
                    name: 'Häufigkeit',
                    type: 'bar',
                    data: result.histogram.map(bin => [(bin.lower + bin.upper) / 2, bin.count])
                },
                {
                    name: `${result.modelName}-PDF`,
                    type: 'line',
                    yAxisIndex: 1,
                    showSymbol: false,
                    data: result.fittedPDF.map(point => [point.x, point.y])
                }
            ],
            grid: { left: '12%', right: '12%', bottom: '15%' }
        });

        const cdfChart = echarts.init(cdf);
        cdfChart.setOption({
            title: { text: 'Empirische und angepasste CDF' },
            tooltip: { trigger: 'axis' },
            legend: { data: ['Empirisch', `${result.modelName}-Modell`] },
            xAxis: { type: 'value', name: 'Hellmann-Exponent α' },
            yAxis: { type: 'value', name: 'Kumulative Wahrscheinlichkeit', min: 0, max: 1 },
            series: [
                {
                    name: 'Empirisch',
                    type: 'line',
                    showSymbol: false,
                    data: result.empiricalCDF.map(point => [point.x, point.y])
                },
                {
                    name: `${result.modelName}-Modell`,
                    type: 'line',
                    showSymbol: false,
                    data: result.fittedCDF.map(point => [point.x, point.y])
                }
            ],
            grid: { left: '12%', right: '5%', bottom: '15%' }
        });
        hellmannDistributionCharts.push(histogramChart, cdfChart);
    });

    table.appendChild(tbody);
    container.prepend(table);
}

function renderDistributions(data) {
    const container = document.getElementById('distribution-results');
    distributionCharts.forEach(chart => chart.dispose());
    distributionCharts = [];
    container.replaceChildren();

    if (!data || data.length === 0) {
        container.innerHTML = '<p class="hint">Keine Verteilungsanalysen für die Auswahl verfügbar.</p>';
        return;
    }

    const table = document.createElement('table');
    table.innerHTML = '<thead><tr><th>Standort</th><th>Höhe [m]</th><th>Bestes Modell</th><th>Stichproben</th><th>AIC</th><th>BIC</th><th>KS</th><th>RMSE</th></tr></thead>';
    const tbody = document.createElement('tbody');

    data.forEach((result, index) => {
        const row = document.createElement('tr');
        [
            result.location,
            result.heightM,
            result.modelName,
            result.sampleCount,
            result.aic.toFixed(2),
            result.bic.toFixed(2),
            result.ksStatistic.toFixed(4),
            result.rmse.toFixed(4)
        ].forEach(value => {
            const cell = document.createElement('td');
            cell.textContent = value;
            row.appendChild(cell);
        });
        tbody.appendChild(row);

        const card = document.createElement('article');
        card.className = 'distribution-result';
        const heading = document.createElement('h3');
        heading.textContent = `${result.location} – ${result.modelName} @ ${result.heightM} m`;
        card.appendChild(heading);

        const charts = document.createElement('div');
        charts.className = 'distribution-charts';
        const histogram = document.createElement('div');
        histogram.className = 'distribution-chart';
        histogram.id = `distribution-histogram-${index}`;
        const cdf = document.createElement('div');
        cdf.className = 'distribution-chart';
        cdf.id = `distribution-cdf-${index}`;
        charts.append(histogram, cdf);
        card.appendChild(charts);
        container.appendChild(card);

        const histogramChart = echarts.init(histogram);
        histogramChart.setOption({
            title: { text: 'Histogramm und angepasste Dichte' },
            tooltip: { trigger: 'axis' },
            legend: { data: ['Häufigkeit', `${result.modelName}-PDF`] },
            xAxis: { type: 'value', name: 'Windgeschwindigkeit [m/s]' },
            yAxis: [
                { type: 'value', name: 'Häufigkeit' },
                { type: 'value', name: 'Dichte', min: 0 }
            ],
            series: [
                {
                    name: 'Häufigkeit',
                    type: 'bar',
                    data: result.histogram.map(bin => [(bin.lower + bin.upper) / 2, bin.count])
                },
                {
                    name: `${result.modelName}-PDF`,
                    type: 'line',
                    yAxisIndex: 1,
                    showSymbol: false,
                    data: result.fittedPDF.map(point => [point.x, point.y])
                }
            ],
            grid: { left: '12%', right: '12%', bottom: '15%' }
        });

        const cdfChart = echarts.init(cdf);
        cdfChart.setOption({
            title: { text: 'Empirische und angepasste CDF' },
            tooltip: { trigger: 'axis' },
            legend: { data: ['Empirisch', `${result.modelName}-Modell`] },
            xAxis: { type: 'value', name: 'Windgeschwindigkeit [m/s]' },
            yAxis: { type: 'value', name: 'Kumulative Wahrscheinlichkeit', min: 0, max: 1 },
            series: [
                {
                    name: 'Empirisch',
                    type: 'line',
                    showSymbol: false,
                    data: result.empiricalCDF.map(point => [point.x, point.y])
                },
                {
                    name: `${result.modelName}-Modell`,
                    type: 'line',
                    showSymbol: false,
                    data: result.fittedCDF.map(point => [point.x, point.y])
                }
            ],
            grid: { left: '12%', right: '5%', bottom: '15%' }
        });
        distributionCharts.push(histogramChart, cdfChart);
    });

    table.appendChild(tbody);
    container.prepend(table);
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

        await loadDistributions();
        await loadHellmannDistributions();
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
    distributionCharts.forEach(chart => chart.resize());
    hellmannDistributionCharts.forEach(chart => chart.resize());
});

// Initialisierung
loadLocations();
