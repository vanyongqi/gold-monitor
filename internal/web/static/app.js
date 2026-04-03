const state = {
  dashboard: null,
  historyPeriod: 30,
  secondaryMode: "live",
  liveSeries: [],
  refreshTimer: null,
  authMode: "login",
  profile: loadProfile(),
};

const $ = (id) => document.getElementById(id);
const { buildXAxisTicks, formatXAxisLabel, getTradingSessionStatus } = window.ChartUtils;
const tooltip = $("chart-tooltip");

function number(value, digits = 2) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  return Number(value).toFixed(digits);
}

function money(value) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  const sign = value > 0 ? "+" : "";
  return `${sign}${Number(value).toFixed(2)}`;
}

function localKey(instrument) {
  return `gold-monitor-live-${instrument}`;
}

function loadProfile() {
  const raw = localStorage.getItem("gold-monitor-profile");
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

function saveProfile(profile) {
  localStorage.setItem("gold-monitor-profile", JSON.stringify(profile));
  state.profile = profile;
}

function saveLiveSeries(instrument) {
  localStorage.setItem(localKey(instrument), JSON.stringify(state.liveSeries.slice(-240)));
}

function loadLiveSeries(instrument) {
  const raw = localStorage.getItem(localKey(instrument));
  if (!raw) return [];
  try {
    return JSON.parse(raw);
  } catch {
    return [];
  }
}

function addLiveSample(quote) {
  const instrument = quote.instrument || state.dashboard.instrument;
  if (!state.liveSeries.length) {
    state.liveSeries = loadLiveSeries(instrument);
  }
  const sample = {
    time: quote.fetched_at || new Date().toISOString(),
    value: quote.price,
  };
  const last = state.liveSeries[state.liveSeries.length - 1];
  if (!last || last.time !== sample.time) {
    state.liveSeries.push(sample);
    saveLiveSeries(instrument);
  }
}

function getQueryParams() {
  return new URLSearchParams({
    instrument: $("instrument").value || "Au99.99",
    cost: $("cost").value,
    grams: $("grams").value,
    sell_fee: $("sell-fee").value || "0.004",
  });
}

async function fetchDashboard() {
  const params = getQueryParams();
  const url = new URL("./api/dashboard", window.location.href);
  url.search = params.toString();
  const response = await fetch(url.toString());
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ error: "请求失败" }));
    throw new Error(payload.error || "请求失败");
  }
  return response.json();
}

function renderShell() {
  $("auth-screen").classList.toggle("hidden", !!state.profile);
  $("app-shell").classList.toggle("hidden", !state.profile);

  if (!state.profile) {
    toggleAuthMode(state.authMode);
    return;
  }

  $("user-badge").textContent = state.profile.name || state.profile.identity || "演示用户";
  if (state.profile.cost) $("cost").value = state.profile.cost;
  if (state.profile.grams) $("grams").value = state.profile.grams;
}

function toggleAuthMode(mode) {
  state.authMode = mode;
  $("show-login").classList.toggle("active", mode === "login");
  $("show-register").classList.toggle("active", mode === "register");
  $("login-form").classList.toggle("hidden", mode !== "login");
  $("register-form").classList.toggle("hidden", mode !== "register");
}

function updateSummary(data) {
  state.dashboard = data;
  addLiveSample(data.quote);
  updateSessionStatus(data.quote.fetched_at);

  $("refresh-interval").textContent = `${data.refresh_seconds} 秒`;
  $("advice-summary").textContent = data.advice.summary;
  $("advice-interpretation").textContent = interpretAdvice(data);
  $("quote-price").textContent = `${number(data.quote.price)} 元/克`;
  $("updated-at").textContent = new Date(data.quote.fetched_at).toLocaleTimeString("zh-CN", { hour12: false });
  $("hero-session-status").textContent = getTradingSessionStatus(data.quote.fetched_at).label;

  $("multi-cost").textContent = data.key_levels.break_even ? `${number(data.key_levels.break_even)} 元` : "--";
  $("multi-profit").textContent = data.has_position ? `${money(data.metrics.profit_amount)} 元` : "--";
  $("multi-profit-rate").textContent = data.has_position ? `${number(data.metrics.profit_rate * 100)}%` : "--";
  $("multi-action").textContent = shortActionText(data.advice.level);

  const range = data.key_levels.recent_resistance - data.key_levels.recent_support;
  const rank = range > 0 ? ((data.quote.price - data.key_levels.recent_support) / range) * 100 : 0;
  $("multi-range-rank").textContent = `${number(rank)}%`;
  $("multi-support-price").textContent = `${number(data.key_levels.recent_support)} 元`;
  $("multi-resistance-price").textContent = `${number(data.key_levels.recent_resistance)} 元`;
  $("multi-range-width").textContent = `${number(range)} 元`;
  $("entry-action").textContent = shortActionText(data.advice.level);
  $("multi-focus").textContent =
    data.advice.level === "take_profit" ? "看总仓止盈与兑现节奏" :
    data.advice.level === "hold_profit" ? "看趋势延续，避免浮盈回吐" :
    data.advice.level === "watch_buy" ? "观察是否出现下一次试仓窗口" :
    "等待更清晰的趋势或关键位信号";
  $("primary-chart-subtitle").textContent = data.has_position
    ? "主图叠加回本线、目标线和区间位置参考，作为主要决策视角。"
    : "未录入仓位时，主图主要用来看价格位置、区间和趋势节奏。";

  const reasons = $("advice-reasons");
  reasons.innerHTML = "";
  data.advice.reasons.forEach((reason) => {
    const li = document.createElement("li");
    li.textContent = reason;
    reasons.appendChild(li);
  });

  renderHistoryChart();
  renderSecondaryChart();
}

function interpretAdvice(data) {
  if (!data.has_position) {
    return data.advice.level === "avoid_chasing"
      ? "当前仍偏高位，第一次建仓不建议追。先看主趋势图和区间位置。"
      : "你还没录入仓位参数，当前页会按统一监控模式展示趋势和区间。";
  }
  switch (data.advice.level) {
    case "take_profit":
      return "当前已经进入更适合兑现的区间，重点是分批止盈而不是继续加风险。";
    case "hold_profit":
      return "已经明显跨过回本门槛，现阶段重点看趋势是否继续延续。";
    case "watch_buy":
      return "价格已经靠近更合理的观察区，可以准备下一笔计划，但不要一次冲进去。";
    default:
      return "当前更适合继续观察，不建议在没有更清晰信号时贸然动作。";
  }
}

function shortActionText(level) {
  switch (level) {
    case "take_profit":
      return "分批止盈";
    case "hold_profit":
      return "继续持有";
    case "avoid_chasing":
      return "不要追高";
    case "watch_buy":
      return "可试仓关注";
    default:
      return "先观察";
  }
}

function updateSessionStatus(timestamp) {
  const status = getTradingSessionStatus(timestamp);
  const pill = $("session-status");
  pill.textContent = status.label;
  pill.className = "session-pill";
  if (status.key === "day_session" || status.key === "night_session") {
    pill.classList.add("session-pill-live");
  } else if (status.key === "waiting_night" || status.key === "waiting_day") {
    pill.classList.add("session-pill-waiting");
  } else {
    pill.classList.add("session-pill-neutral");
  }
  $("session-detail").textContent = status.detail;
}

function filterHistory(period) {
  if (!state.dashboard) return [];
  return (state.dashboard.history || []).slice(-period);
}

function renderHistoryChart() {
  const svg = $("history-chart");
  const data = filterHistory(state.historyPeriod);
  const breakEven = state.dashboard.key_levels.break_even || 0;
  const target = state.dashboard.key_levels.target_one || 0;
  setChartDataset(svg, data, 900, 320);
  svg.dataset.axisMode = "date";
  svg.dataset.valueLabel = "价格";
  renderWithTooltip(svg, buildLineChart(data, {
    width: 900,
    height: 320,
    xAxisMode: "date",
    valueLabel: "价格",
    lineColor: "#b7791f",
    areaColor: "rgba(183,121,31,0.18)",
    dashedLines: [
      breakEven ? { value: breakEven, color: "#b45309", label: "回本线" } : null,
      target ? { value: target, color: "#2f855a", label: "目标线" } : null,
    ].filter(Boolean),
  }));
}

function renderSecondaryChart() {
  const svg = $("secondary-chart");
  const caption = $("secondary-caption");
  const title = $("secondary-title");
  let data = [];

  if (state.secondaryMode === "profit" && state.dashboard.profit_trend?.length) {
    data = state.dashboard.profit_trend.slice(-90);
    setChartDataset(svg, data, 900, 260);
    svg.dataset.axisMode = "date";
    svg.dataset.valueLabel = "收益";
    title.textContent = "收益曲线";
    caption.textContent = "这条线展示的是按你的成本和卖出手续费折算后的净收益变化。";
    renderWithTooltip(svg, buildLineChart(data, {
      width: 900,
      height: 260,
      xAxisMode: "date",
      valueLabel: "收益",
      lineColor: "#2f855a",
      areaColor: "rgba(47,133,90,0.16)",
      dashedLines: [{ value: 0, color: "#b45309", label: "收益为 0" }],
    }));
    return;
  }

  title.textContent = "收益与实时监控";
  data = (state.dashboard.live_trend?.length ? state.dashboard.live_trend : state.liveSeries)
    .slice(-120)
    .map((item) => ({ time: item.time, value: item.value }));
  setChartDataset(svg, data, 900, 260);
  svg.dataset.axisMode = "time";
  svg.dataset.valueLabel = "价格";
  caption.textContent = "这条线来自后端统一抓价后的快照，适合判断短线节奏。";
  renderWithTooltip(svg, buildLineChart(data, {
    width: 900,
    height: 260,
    xAxisMode: "time",
    valueLabel: "价格",
    lineColor: "#1f6f59",
    areaColor: "rgba(31,111,89,0.14)",
    dashedLines: [],
    emptyText: "实时样本还不够，页面多挂一会儿曲线会更完整。",
  }));
}

function setChartDataset(svg, data, width, height) {
  const padding = { top: 20, right: 20, bottom: 46, left: 46 };
  if (!data.length || data.length === 1) {
    svg.dataset.points = "[]";
    return;
  }
  const values = data.map((item) => item.value);
  const min = Math.min(...values);
  const max = Math.max(...values);
  const safeMin = min === max ? min - 1 : min;
  const safeMax = min === max ? max + 1 : max;
  const range = safeMax - safeMin;
  const plotWidth = width - padding.left - padding.right;
  const plotHeight = height - padding.top - padding.bottom;
  const points = data.map((item, index) => ({
    time: item.time,
    value: item.value,
    x: padding.left + (plotWidth * index) / (data.length - 1),
    y: padding.top + ((safeMax - item.value) / range) * plotHeight,
  }));
  svg.dataset.points = JSON.stringify(points);
}

function renderWithTooltip(svg, markup) {
  svg.innerHTML = markup;
  const data = JSON.parse(svg.dataset.points || "[]");
  const xAxisMode = svg.dataset.axisMode || "date";
  const valueLabel = svg.dataset.valueLabel || "价格";
  const crosshair = svg.querySelector("#chart-crosshair");
  const highlight = svg.querySelector("#chart-highlight");

  const move = (event) => {
    if (!data.length) return;
    const rect = svg.getBoundingClientRect();
    const relativeX = event.clientX - rect.left;
    const plotWidth = rect.width - 66;
    const ratio = Math.max(0, Math.min(1, (relativeX - 46) / Math.max(plotWidth, 1)));
    const index = Math.round((data.length - 1) * ratio);
    const point = data[index];
    if (!point) return;

    if (crosshair) {
      crosshair.setAttribute("x1", point.x);
      crosshair.setAttribute("x2", point.x);
      crosshair.setAttribute("opacity", "1");
    }
    if (highlight) {
      highlight.setAttribute("cx", point.x);
      highlight.setAttribute("cy", point.y);
      highlight.setAttribute("opacity", "1");
    }

    const prefix = xAxisMode === "time" ? "时间" : "日期";
    tooltip.textContent = `${prefix}: ${formatXAxisLabel(point.time, xAxisMode)}\n${valueLabel}: ${number(point.value)}`;
    tooltip.classList.remove("hidden");
    tooltip.style.left = `${event.clientX}px`;
    tooltip.style.top = `${event.clientY}px`;
  };

  const leave = () => {
    tooltip.classList.add("hidden");
    if (crosshair) crosshair.setAttribute("opacity", "0");
    if (highlight) highlight.setAttribute("opacity", "0");
  };

  svg.onmousemove = move;
  svg.onmouseleave = leave;
}

function buildLineChart(data, options) {
  const width = options.width;
  const height = options.height;
  const padding = { top: 20, right: 20, bottom: 46, left: 46 };
  if (!data.length || data.length === 1) {
    const message = options.emptyText || "样本不足";
    return `
      <rect x="0" y="0" width="${width}" height="${height}" rx="18" fill="rgba(255,255,255,0.1)"></rect>
      <text x="${width / 2}" y="${height / 2}" text-anchor="middle" fill="#7a6a55" font-size="18">${message}</text>
    `;
  }

  const values = data.map((item) => item.value);
  const extraValues = (options.dashedLines || []).map((item) => item.value);
  const min = Math.min(...values, ...extraValues);
  const max = Math.max(...values, ...extraValues);
  const safeMin = min === max ? min - 1 : min;
  const safeMax = min === max ? max + 1 : max;
  const range = safeMax - safeMin;
  const plotWidth = width - padding.left - padding.right;
  const plotHeight = height - padding.top - padding.bottom;

  const toX = (index) => padding.left + (plotWidth * index) / (data.length - 1);
  const toY = (value) => padding.top + ((safeMax - value) / range) * plotHeight;
  const points = data.map((item, index) => `${toX(index)},${toY(item.value)}`).join(" ");
  const area = `${padding.left},${height - padding.bottom} ${points} ${padding.left + plotWidth},${height - padding.bottom}`;
  const xAxisY = height - padding.bottom;
  const ticks = buildXAxisTicks(data, options.xAxisMode === "time" ? 5 : 4, options.xAxisMode || "date");

  const gridLines = Array.from({ length: 4 }).map((_, index) => {
    const y = padding.top + (plotHeight * index) / 3;
    return `<line x1="${padding.left}" y1="${y}" x2="${width - padding.right}" y2="${y}" stroke="#eadfc9" stroke-width="1"></line>`;
  }).join("");

  const labels = [safeMax, safeMin + range / 2, safeMin].map((value, index) => {
    const y = index === 0 ? padding.top + 4 : index === 1 ? padding.top + plotHeight / 2 + 4 : padding.top + plotHeight + 4;
    return `<text x="0" y="${y}" fill="#7a6a55" font-size="12">${number(value)}</text>`;
  }).join("");

  const dashedLines = (options.dashedLines || []).map((line) => {
    const y = toY(line.value);
    return `
      <line x1="${padding.left}" y1="${y}" x2="${width - padding.right}" y2="${y}" stroke="${line.color}" stroke-width="2" stroke-dasharray="8 8"></line>
      <text x="${padding.left + 4}" y="${y - 6}" fill="${line.color}" font-size="12">${line.label}</text>
    `;
  }).join("");

  const xAxis = `
    <line x1="${padding.left}" y1="${xAxisY}" x2="${width - padding.right}" y2="${xAxisY}" stroke="#d8ccb7" stroke-width="1.5"></line>
    ${ticks.map((tick) => {
      const x = toX(tick.index);
      return `
        <line x1="${x}" y1="${xAxisY}" x2="${x}" y2="${xAxisY + 6}" stroke="#c3b39a" stroke-width="1"></line>
        <text x="${x}" y="${height - 14}" text-anchor="middle" fill="#7a6a55" font-size="12">${tick.label}</text>
      `;
    }).join("")}
  `;

  return `
    <defs>
      <linearGradient id="chart-area-${options.lineColor.replace(/[^a-z0-9]/gi, "")}" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="${options.areaColor}" />
        <stop offset="100%" stop-color="rgba(255,255,255,0.02)" />
      </linearGradient>
    </defs>
    ${gridLines}
    ${dashedLines}
    ${xAxis}
    <polygon points="${area}" fill="url(#chart-area-${options.lineColor.replace(/[^a-z0-9]/gi, "")})"></polygon>
    <polyline points="${points}" fill="none" stroke="${options.lineColor}" stroke-width="4" stroke-linecap="round" stroke-linejoin="round"></polyline>
    <line id="chart-crosshair" x1="${padding.left}" y1="${padding.top}" x2="${padding.left}" y2="${height - padding.bottom}" stroke="${options.lineColor}" stroke-width="1.5" stroke-dasharray="6 6" opacity="0"></line>
    <circle id="chart-highlight" cx="${padding.left}" cy="${padding.top}" r="6" fill="${options.lineColor}" opacity="0"></circle>
    ${labels}
    <text x="${width - padding.right - 4}" y="${padding.top + 14}" text-anchor="end" fill="#7a6a55" font-size="12">最新: ${formatXAxisLabel(data[data.length - 1].time, options.xAxisMode || "date")}</text>
  `;
}

function handleAuthSuccess(identity, name) {
  saveProfile({
    identity,
    name: name || identity,
    cost: "",
    grams: "",
  });
  renderShell();
}

function bindEvents() {
  $("show-login").addEventListener("click", () => toggleAuthMode("login"));
  $("show-register").addEventListener("click", () => toggleAuthMode("register"));

  $("login-btn").addEventListener("click", () => {
    const identity = $("login-identity").value.trim() || "演示用户";
    handleAuthSuccess(identity, identity);
    loadAndRender();
  });

  $("register-btn").addEventListener("click", () => {
    const name = $("register-name").value.trim() || "新用户";
    const phone = $("register-phone").value.trim() || "演示账号";
    handleAuthSuccess(phone, name);
    loadAndRender();
  });

  $("refresh-btn").addEventListener("click", () => {
    if (state.profile) {
      state.profile.cost = $("cost").value;
      state.profile.grams = $("grams").value;
      saveProfile(state.profile);
    }
    loadAndRender();
  });

  document.querySelectorAll("#history-periods button").forEach((button) => {
    button.addEventListener("click", () => {
      document.querySelectorAll("#history-periods button").forEach((item) => item.classList.remove("active"));
      button.classList.add("active");
      state.historyPeriod = Number(button.dataset.period);
      renderHistoryChart();
    });
  });

  document.querySelectorAll("#monitor-periods button").forEach((button) => {
    button.addEventListener("click", () => {
      document.querySelectorAll("#monitor-periods button").forEach((item) => item.classList.remove("active"));
      button.classList.add("active");
      state.secondaryMode = button.dataset.period;
      renderSecondaryChart();
    });
  });
}

async function loadAndRender() {
  if (!state.profile) return;
  try {
    const data = await fetchDashboard();
    updateSummary(data);
    if (state.refreshTimer) clearTimeout(state.refreshTimer);
    state.refreshTimer = setTimeout(loadAndRender, (data.refresh_seconds || 60) * 1000);
  } catch (error) {
    $("advice-summary").textContent = "获取失败";
    $("advice-interpretation").textContent = error.message;
    if (state.refreshTimer) clearTimeout(state.refreshTimer);
    state.refreshTimer = setTimeout(loadAndRender, 60000);
  }
}

bindEvents();
renderShell();
if (state.profile) {
  loadAndRender();
}
