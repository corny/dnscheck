"use strict";

// Show check times relative to now, keep the absolute time as tooltip.
function relativeTimes() {
  const format = new Intl.RelativeTimeFormat(document.documentElement.lang, { numeric: "auto" });
  const units = [["day", 86400], ["hour", 3600], ["minute", 60], ["second", 1]];

  for (const el of document.querySelectorAll("time[datetime]")) {
    const seconds = (Date.parse(el.getAttribute("datetime")) - Date.now()) / 1000;
    if (Number.isNaN(seconds)) {
      continue;
    }
    const [unit, size] = units.find(([, size]) => Math.abs(seconds) >= size) || units[units.length - 1];
    el.title = el.textContent;
    el.textContent = format.format(Math.round(seconds / size), unit);
  }
}

// Submit every line of the form as a nameserver, one after another.
function submitForm(form) {
  const results = document.getElementById("results");
  const progress = results.querySelector("progress");
  const list = results.querySelector("ol");
  const msg = form.dataset;

  form.addEventListener("submit", async (event) => {
    event.preventDefault();

    const addresses = form.elements.addresses.value
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line !== "");

    if (addresses.length === 0) {
      alert(msg.empty);
      return;
    }

    form.hidden = true;
    results.hidden = false;
    progress.max = addresses.length;

    const items = addresses.map((address) => {
      const item = document.createElement("li");
      const code = document.createElement("code");
      const status = document.createElement("span");
      code.textContent = address;
      status.textContent = " " + msg.pending;
      item.append(code, status);
      list.append(item);
      return { address, status };
    });

    for (const { address, status } of items) {
      status.textContent = " " + msg.checking;
      try {
        const res = await fetch(form.action, {
          method: "POST",
          body: new URLSearchParams({ address }),
        });
        if (res.ok) {
          status.textContent = " " + msg.added;
        } else if (res.status === 409) {
          status.textContent = " " + msg.duplicate;
        } else {
          status.textContent = " " + msg.failed + ": " + (await res.text()).trim();
        }
      } catch (err) {
        status.textContent = " " + msg.failed + ": " + err.message;
      }
      progress.value += 1;
    }
  });
}

relativeTimes();
const form = document.getElementById("submit");
if (form) {
  submitForm(form);
}
