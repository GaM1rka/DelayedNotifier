const API_URL = window.API_URL || "http://localhost:8080";

const form = document.querySelector("#notify-form");
const formMessage = document.querySelector("#form-message");
const list = document.querySelector("#notifications");
const count = document.querySelector("#count");
const refresh = document.querySelector("#refresh");

form.send_at.value = defaultDateTime();

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  formMessage.textContent = "Создаём уведомление...";

  const data = Object.fromEntries(new FormData(form).entries());
  const payload = {
    email: data.email,
    subject: data.subject,
    message: data.message,
    send_at: new Date(data.send_at).toISOString(),
  };

  try {
    const response = await fetch(`${API_URL}/notify`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    const body = await response.json();
    if (!response.ok) {
      throw new Error(body.error || "Не удалось создать уведомление");
    }

    form.reset();
    form.send_at.value = defaultDateTime();
    formMessage.textContent = "Уведомление поставлено в очередь.";
    await loadNotifications();
  } catch (error) {
    formMessage.textContent = error.message;
  }
});

refresh.addEventListener("click", loadNotifications);

async function loadNotifications() {
  list.innerHTML = `<div class="notification"><p>Загружаем...</p></div>`;
  try {
    const response = await fetch(`${API_URL}/notify`);
    const items = await response.json();
    if (!response.ok) {
      throw new Error(items.error || "Не удалось загрузить список");
    }

    renderNotifications(items);
  } catch (error) {
    list.innerHTML = `<div class="notification"><p>${escapeHTML(error.message)}</p></div>`;
  }
}

function renderNotifications(items) {
  count.textContent = items.length;
  if (items.length === 0) {
    list.innerHTML = `<div class="notification"><p>Пока нет уведомлений.</p></div>`;
    return;
  }

  list.innerHTML = items
    .map((item) => {
      const canCancel = ["scheduled", "failed", "sending"].includes(item.status);
      return `
        <article class="notification">
          <div>
            <h3>${escapeHTML(item.subject)}</h3>
            <p>${escapeHTML(item.email)}</p>
            <p>${escapeHTML(item.message)}</p>
            <div class="meta">
              <span class="badge ${item.status}">${escapeHTML(item.status)}</span>
              <span class="badge">${formatDate(item.send_at)}</span>
              <span class="badge">attempts: ${item.attempts}</span>
            </div>
            ${item.last_error ? `<p>${escapeHTML(item.last_error)}</p>` : ""}
          </div>
          ${canCancel ? `<button class="cancel" title="Отменить" data-id="${item.id}">×</button>` : ""}
        </article>
      `;
    })
    .join("");

  document.querySelectorAll(".cancel").forEach((button) => {
    button.addEventListener("click", async () => {
      await cancelNotification(button.dataset.id);
    });
  });
}

async function cancelNotification(id) {
  const response = await fetch(`${API_URL}/notify/${id}`, { method: "DELETE" });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    formMessage.textContent = body.error || "Не удалось отменить уведомление";
    return;
  }
  formMessage.textContent = "Уведомление отменено.";
  await loadNotifications();
}

function defaultDateTime() {
  const date = new Date(Date.now() + 5 * 60 * 1000);
  date.setSeconds(0, 0);
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
}

function formatDate(value) {
  return new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

loadNotifications();
setInterval(loadNotifications, 10000);
