(function (global) {
  function formatXAxisLabel(value, mode) {
    if (!value) return "-";

    if (mode === "time") {
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) {
        return String(value).slice(11, 16);
      }
      return date.toLocaleTimeString("zh-CN", {
        hour: "2-digit",
        minute: "2-digit",
        hour12: false,
      });
    }

    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return String(value).slice(5, 10);
    }
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    return `${month}-${day}`;
  }

  function buildXAxisTicks(data, maxTicks, mode) {
    if (!Array.isArray(data) || data.length === 0) {
      return [];
    }

    const effectiveMaxTicks = Math.max(2, maxTicks || 4);
    if (data.length <= effectiveMaxTicks) {
      return data.map((item, index) => ({
        index,
        label: formatXAxisLabel(item.time, mode),
      }));
    }

    const indexes = new Set();
    for (let i = 0; i < effectiveMaxTicks; i += 1) {
      const ratio = i / (effectiveMaxTicks - 1);
      indexes.add(Math.round((data.length - 1) * ratio));
    }

    return Array.from(indexes)
      .sort((a, b) => a - b)
      .map((index) => ({
        index,
        label: formatXAxisLabel(data[index].time, mode),
      }));
  }

  function getTradingSessionStatus(input) {
    const date = input instanceof Date ? input : new Date(input);
    if (Number.isNaN(date.getTime())) {
      return {
        key: "unknown",
        label: "交易状态未知",
        detail: "当前时间无法识别，暂时不能判断上金所交易状态。",
      };
    }

    const day = date.getDay();
    const minutes = date.getHours() * 60 + date.getMinutes();

    if (day === 6) {
      return {
        key: "weekend_closed",
        label: "周末休市",
        detail: "周六没有新的上金所连续交易数据，价格长时间不动属于正常现象。",
      };
    }

    if (day === 0) {
      return {
        key: "weekend_closed",
        label: "周末休市",
        detail: "周日白天没有新的上金所连续交易数据，需等待夜盘或下一个交易日。",
      };
    }

    if (minutes >= 9 * 60 && minutes <= 15 * 60 + 30) {
      return {
        key: "day_session",
        label: "日盘交易中",
        detail: "当前处于上金所日盘时段，价格更新更有参考意义。",
      };
    }

    if (minutes >= 20 * 60 || minutes <= 2 * 60 + 30) {
      return {
        key: "night_session",
        label: "夜盘交易中",
        detail: "当前处于上金所夜盘时段，价格仍会继续波动。",
      };
    }

    if (minutes > 15 * 60 + 30 && minutes < 20 * 60) {
      return {
        key: "waiting_night",
        label: "日盘已收盘，等待夜盘",
        detail: "15:30 到 20:00 之间常出现长时间不动，这通常不是程序故障。",
      };
    }

    return {
      key: "waiting_day",
      label: "夜盘已收盘，等待日盘",
      detail: "凌晨收盘到早盘开始前，价格不动通常是正常休市空档。",
    };
  }

  const api = {
    buildXAxisTicks,
    formatXAxisLabel,
    getTradingSessionStatus,
  };

  if (typeof module !== "undefined" && module.exports) {
    module.exports = api;
  }

  global.ChartUtils = api;
})(typeof window !== "undefined" ? window : globalThis);
