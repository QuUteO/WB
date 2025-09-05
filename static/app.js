const API_BASE = "http://localhost:8081";

// ========== HELPERS ==========
function renderOrder(order) {
    if (!order) return "<p>❌ Order not found</p>";

    let html = `<div class="card">
        <div class="section"><b>Order UID:</b> ${order.order_uid}</div>
        <div class="section"><b>Track:</b> ${order.track_number}</div>
        <div class="section"><b>Customer:</b> ${order.customer_id}</div>
        <div class="section"><b>Date:</b> ${order.date_created}</div>

        <h4>Delivery</h4>
        <div class="section">${order.delivery.name}, ${order.delivery.city}, ${order.delivery.address}, 
        ${order.delivery.region}, ${order.delivery.zip}, ${order.delivery.phone}, ${order.delivery.email}</div>

        <h4>Payment</h4>
        <div class="section"><b>Amount:</b> ${order.payment.amount} ${order.payment.currency}</div>
        <div class="section"><b>Bank:</b> ${order.payment.bank}</div>
        <div class="section"><b>Provider:</b> ${order.payment.provider}</div>

        <h4>Items</h4>
        <table>
            <tr><th>Name</th><th>Brand</th><th>Price</th><th>Qty</th><th>Total</th></tr>
            ${order.items.map(i => `
                <tr>
                    <td>${i.name}</td>
                    <td>${i.brand}</td>
                    <td>${i.price}</td>
                    <td>1</td>
                    <td>${i.total_price}</td>
                </tr>`).join("")}
        </table>
    </div>`;
    return html;
}

function renderOrdersList(orders) {
    if (!orders || orders.length === 0) return "<p>📭 No orders</p>";

    let rows = orders.map(o => `
        <tr>
            <td><a href="#" onclick="showOrder('${o.order_uid}'); return false;">${o.order_uid}</a></td>
            <td>${o.customer_id}</td>
            <td>${o.delivery ? o.delivery.city : "-"}</td>
            <td>${o.payment ? o.payment.amount + " " + o.payment.currency : "-"}</td>
            <td>${o.payment ? o.payment.bank : "-"}</td>
            <td>${o.payment ? o.payment.provider : "-"}</td>
            <td>${o.date_created}</td>
        </tr>`).join("");

    return `<table>
        <tr>
            <th>Order UID</th>
            <th>Customer</th>
            <th>City</th>
            <th>Amount</th>
            <th>Bank</th>
            <th>Provider</th>
            <th>Created</th>
        </tr>
        ${rows}
    </table>`;
}

// При клике на UID → открыть подробности
async function showOrder(uid) {
    document.getElementById("orderUidInput").value = uid;
    await getOrder();
    window.scrollTo({ top: 0, behavior: "smooth" });
}

// ========== API ==========
async function getOrder() {
    const orderUid = document.getElementById("orderUidInput").value.trim();
    const resultElem = document.getElementById("orderResult");
    resultElem.innerHTML = "";

    if (!orderUid) {
        resultElem.innerHTML = "<p>⚠️ Please enter order_uid</p>";
        return;
    }

    try {
        const res = await fetch(`${API_BASE}/order/${orderUid}`);
        if (!res.ok) {
            const text = await res.text();
            resultElem.innerHTML = `<p>❌ Error ${res.status}: ${text}</p>`;
            return;
        }
        const data = await res.json();
        resultElem.innerHTML = renderOrder(data);
    } catch (err) {
        resultElem.innerHTML = `<p>⚠️ Fetch error: ${err}</p>`;
    }
}

async function getOrders() {
    const resultElem = document.getElementById("ordersResult");
    resultElem.innerHTML = "";

    try {
        const res = await fetch(`${API_BASE}/orders`);
        if (!res.ok) {
            const text = await res.text();
            resultElem.innerHTML = `<p>❌ Error ${res.status}: ${text}</p>`;
            return;
        }
        const data = await res.json();
        resultElem.innerHTML = renderOrdersList(data);
    } catch (err) {
        resultElem.innerHTML = `<p>⚠️ Fetch error: ${err}</p>`;
    }
}

async function publishOrder() {
    const jsonInput = document.getElementById("orderJsonInput").value.trim();
    const resultElem = document.getElementById("publishResult");
    resultElem.textContent = "";

    if (!jsonInput) {
        resultElem.textContent = "⚠️ Please enter JSON";
        return;
    }

    let order;
    try {
        order = JSON.parse(jsonInput);
    } catch (err) {
        resultElem.textContent = "❌ Invalid JSON: " + err;
        return;
    }

    try {
        const res = await fetch(`${API_BASE}/publish-order`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(order)
        });
        if (!res.ok) {
            const text = await res.text();
            resultElem.textContent = `❌ Error ${res.status}: ${text}`;
            return;
        }
        const data = await res.json();
        resultElem.textContent = JSON.stringify(data, null, 2);
    } catch (err) {
        resultElem.textContent = "⚠️ Fetch error: " + err;
    }
}
