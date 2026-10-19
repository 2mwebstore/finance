function formatMoney(amount, currency) {
  const parsed = typeof amount === "number" ? amount : Number(amount || 0);
  const value = Number.isFinite(parsed) ? parsed : 0;
  const digits = currency === "USD" ? 2 : 0;
  const formatted = new Intl.NumberFormat("en-US", {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits
  }).format(value);
  return `${currency === "USD" ? "$" : "\u17DB"}${formatted}`;
}

export { formatMoney as f };
//# sourceMappingURL=money-BqQfZu3a.mjs.map
