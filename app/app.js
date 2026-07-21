document.addEventListener('DOMContentLoaded', async () => {
    const token = getAccessToken();
    const refreshToken = getRefreshToken();

    await loadServicesTree();

    await loadEmployees();

    await loadAvailability(await getAvailabilityEmployeeId());
    //rendern nachdem der service tree from backend geladen wurde
    render(servicesTree.data);

    if (!token && refreshToken) {
        const refreshed = await refreshAccessToken();
        if (!refreshed) {
            logout();
            return;
        }
    }

    if (hasAuthSession()) {
        await showAuthenticatedState();
        return;
    }

    showLoggedOutState();
});
// Termine neu laden (api request)
document.getElementById('refresh-appointments').addEventListener('click', async () => {
    await loadUserAppointments();
});

document.getElementById('toggle-appointments').addEventListener('click', async () => {
    const nextVisible = !appointmentsVisible;
    setAppointmentsPanelVisible(nextVisible);
    if (nextVisible) {
        await loadUserAppointments();
    }
});

document.getElementById('cancle-all-appintments').addEventListener('click', async () => {
    await cancleAllAppointments();
});

document.getElementById('login-form').addEventListener('submit', async (event) => {
    event.preventDefault();
    await login();
});

document.getElementById('logout-button').addEventListener('click', () => {
    logout();
});

let toastTimeoutId = null;
let appointmentsVisible = false;
let refreshRequestPromise = null;

function getAccessToken() {
    return localStorage.getItem('token');
}

function getRefreshToken() {
    return localStorage.getItem('refresh_token');
}

// speichert beider tokens zentral im localStorage
function persistAuthTokens(accessToken, refreshToken) {
    if (accessToken) {
        localStorage.setItem('token', accessToken);
    }
    if (refreshToken) {
        localStorage.setItem('refresh_token', refreshToken);
    }
}
// local beide tokens löschen
function clearAuthTokens() {
    localStorage.removeItem('token');
    localStorage.removeItem('refresh_token');
}
// prüfen das beide tokens vorhanden sond
function hasAuthSession() {
    return Boolean(getAccessToken() || getRefreshToken());
}

let revokeInFlight = false;
// api request um den refresh token ungültig zu machen
async function revokeRefreshTokenSilently(refreshToken) {
    if (!refreshToken || revokeInFlight) {
        return;
    }

    revokeInFlight = true;
    try {
        await fetch('/api/revoke', {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${refreshToken}`,
            },
        });
    } catch (_) {
        // user wird ausgeloggt auch wenn das backend nicht erreichbar ist oder /api/revoke nicht klappt
    } finally {
        revokeInFlight = false;
    }
}

function isApiRequest(input) {
    const requestUrl = typeof input === 'string'
        ? input
        : (input && typeof input.url === 'string' ? input.url : '');

    // filtert auf backend api requests damit 401 handling nur dort greift.
    return requestUrl.startsWith('/api') || requestUrl.includes('/api/');
}
// Authorization header für die api
function makeAuthHeaders(existingHeaders, token) {
    const headers = new Headers(existingHeaders || {});
    if (token) {
        headers.set('Authorization', `Bearer ${token}`);
    }
    return headers;
}
// wenn der access token nach der angegebenen zeit ausläuft wird ein neuer angefordert mit dem refresh token solange der noch gültig ist
async function refreshAccessToken() {
    if (refreshRequestPromise) {
        return refreshRequestPromise;
    }

    refreshRequestPromise = (async () => {
        const refreshToken = getRefreshToken();
        if (!refreshToken) {
            return false;
        }

        const refreshRes = await fetch('/api/refresh', {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${refreshToken}`,
            },
        });

        if (!refreshRes.ok) {
            return false;
        }

        const data = await refreshRes.json();
        if (!data || !data.token || !data.refresh_token) {
            return false;
        }

        persistAuthTokens(data.token, data.refresh_token);
        return true;
    })();

    try {
        return await refreshRequestPromise;
    } finally {
        refreshRequestPromise = null;
    }
}

function isAuthPath(input) {
    const requestUrl = typeof input === 'string'
        ? input
        : (input && typeof input.url === 'string' ? input.url : '');
    // nur auth endpoints sollten hir sein um refresh/retry loops zu vermeiden
    return requestUrl.startsWith('/api/login') || requestUrl.startsWith('/api/users') || requestUrl.startsWith('/api/refresh');
}

// Bei 401 wird einmal versucht den Access Token zu erneuern und die Request zu wiederholen
async function apiFetch(input, init, options = {}) {
    const config = init || {};
    const shouldAttachAuth = options.attachAuth === true;
    const canAttemptRefresh = options.allowRefresh !== false;

    const firstHeaders = shouldAttachAuth
        ? makeAuthHeaders(config.headers, getAccessToken())
        : new Headers(config.headers || {});

    const firstRequestInit = {
        ...config,
        headers: firstHeaders,
    };

    const firstRes = await fetch(input, firstRequestInit);

    if (!isApiRequest(input) || firstRes.status !== 401 || !canAttemptRefresh || isAuthPath(input)) {
        if (isApiRequest(input) && firstRes.status === 401 && !isAuthPath(input)) {
            logout();
        }
        return firstRes;
    }

    const refreshed = await refreshAccessToken();
    if (!refreshed) {
        logout();
        return firstRes;
    }

    const retryHeaders = shouldAttachAuth
        ? makeAuthHeaders(config.headers, getAccessToken())
        : new Headers(config.headers || {});

    const retryRequestInit = {
        ...config,
        headers: retryHeaders,
    };

    return fetch(input, retryRequestInit);
}
// user anmeldung
async function login() {
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;

    try {
        const res = await apiFetch('/api/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ email, password }),
        });
        const data = await res.json();
        if (!res.ok) {
            throw new Error(`Failed to login: ${data.error}`);
        }

        if (data.token) {
            persistAuthTokens(data.token, data.refresh_token || null);
            await showAuthenticatedState();
        } else {
            alert('Login failed. Please check your credentials.');
        }
    } catch (error) {
        alert(`Error: ${error.message}`);
    }
}
// user account registrierung
async function signup() {
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;

    try {
        const res = await apiFetch('/api/users', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ email, password }),
        });
        if (!res.ok) {
            const data = await res.json();
            throw new Error(`Failed to create user: ${data.error}`);
        }
        console.log('User created!');
        await login();
    } catch (error) {
        alert(`Error: ${error.message}`);
    }
}

function logout() {
    const refreshToken = getRefreshToken();
    void revokeRefreshTokenSilently(refreshToken);
    clearAuthTokens();
    showLoggedOutState();
}
//  ansicht für nicht eingeloggte user
function showLoggedOutState() {
    document.getElementById('auth-section').style.display = 'block';
    document.getElementById('toggle-appointments').style.display = 'none';
    setAppointmentsPanelVisible(false);
    document.getElementById('step-services').style.display = 'none';
    setAppointmentsFeedback('');
    document.getElementById('appointments-list').innerHTML = '';
}
// ansicht für eingeloggte user
async function showAuthenticatedState() {
    document.getElementById('auth-section').style.display = 'none';
    document.getElementById('toggle-appointments').style.display = 'inline-block';
    setAppointmentsPanelVisible(false);
    document.getElementById('step-services').style.display = 'block';
}
// termine für den user anzeigen
function setAppointmentsPanelVisible(visible) {
    appointmentsVisible = visible;

    document.getElementById('appointments-section').style.display = visible ? 'block' : 'none';
    document.getElementById('toggle-appointments').textContent = visible
        ? 'Meine Termine ausblenden'
        : 'Meine Termine anzeigen';

    if (!visible) {
        document.getElementById('appointments-list').innerHTML = '';
    }
}
// feedback nachdem ein Termin gelöscht wurde
function setAppointmentsFeedback(message, isError = false) {
    const feedbackEl = document.getElementById('toast');

    if (toastTimeoutId) {
        clearTimeout(toastTimeoutId);
        toastTimeoutId = null;
    }

    if (!message) {
        feedbackEl.textContent = '';
        feedbackEl.classList.remove('visible', 'toast-success', 'toast-error');
        return;
    }

    feedbackEl.textContent = message;
    feedbackEl.classList.remove('toast-success', 'toast-error');
    feedbackEl.classList.add(isError ? 'toast-error' : 'toast-success', 'visible');

    toastTimeoutId = window.setTimeout(() => {
        feedbackEl.classList.remove('visible', 'toast-success', 'toast-error');
        feedbackEl.textContent = '';
        toastTimeoutId = null;
    }, 3200);
}

function formatAppointmentDate(dateString) {
    return new Date(dateString + 'T00:00:00').toLocaleDateString('de-DE', {
        weekday: 'long',
        day: '2-digit',
        month: '2-digit',
        year: 'numeric'
    });
}
// termine des nutzers laden
function renderUserAppointments(appointments) {
    const listEl = document.getElementById('appointments-list');
    listEl.innerHTML = '';

    if (!appointments.length) {
        const emptyState = document.createElement('div');
        emptyState.className = 'appointment-empty';
        emptyState.textContent = 'Noch keine Termine gebucht.';
        listEl.appendChild(emptyState);
        return;
    }

    appointments.forEach((appointment) => {
        const card = document.createElement('div');
        card.className = 'appointment-card';

        const services = Array.isArray(appointment.services) ? appointment.services.join(', ') : '';
        const durationText = appointment.total_duration_minutes
            ? `${appointment.total_duration_minutes} Min`
            : 'Dauer unbekannt';
        const priceText = Number(appointment.total_price || 0).toFixed(2) + ' EUR';

        card.innerHTML = `
            <div class="appointment-card-header">
                <div>
                    <div class="appointment-card-title">${formatAppointmentDate(appointment.date)}</div>
                    <div class="appointment-card-time">${appointment.start_time} - ${appointment.end_time}</div>
                </div>
                <div class="appointment-price">${priceText}</div>
            </div>
            <div class="appointment-card-meta">
                <div><strong>Mitarbeiter:</strong> ${appointment.employee_name || 'Unbekannt'}</div>
                <div><strong>Dienstleistungen:</strong> ${services || 'Keine Angabe'}</div>
                <div><strong>Gesamtdauer:</strong> ${durationText}</div>
            </div>
            <div class="appointment-card-actions">
                <button type="button" class="appointment-edit" data-appointment-id="${appointment.id}">Termin bearbeiten</button>
                <button type="button" class="appointment-cancel" data-appointment-id="${appointment.id}">Termin stornieren</button>
            </div>
        `;
        
        const editBtn = card.querySelector('.appointment-edit');
        editBtn.addEventListener('click', async () => {
            await toggleAppointmentEditor(card, appointment);
        });

        const cancelBtn = card.querySelector('.appointment-cancel');
        cancelBtn.addEventListener('click', async () => {
            await cancelAppointment(appointment.id);
        });

        listEl.appendChild(card);
    });
}

function getActiveEmployees() {
    return Array.isArray(employeesData.data)
        ? employeesData.data.filter((emp) => emp && emp.is_active)
        : [];
}

function getEmployeeNameByID(employeeID) {
    const employee = getActiveEmployees().find((emp) => emp.id === employeeID);
    return employee ? employee.name : employeeID;
}
//api request für die verfügbaren termine eines mitarbeiters
async function fetchAvailabilityForEmployee(employeeID) {
    if (!employeeID) {
        return { dates: [] };
    }

    const res = await apiFetch(`/api/availability?employee_id=${encodeURIComponent(employeeID)}`);
    if (!res.ok) {
        throw new Error('Verfuegbarkeit konnte nicht geladen werden.');
    }

    const data = await res.json();
    if (!data || !Array.isArray(data.dates)) {
        return { dates: [] };
    }

    return data;
}

function buildSlotOptionsForDate(availability, date) {
    const day = (availability.dates || []).find((entry) => entry.date === date);
    if (!day || !Array.isArray(day.slots)) {
        return [];
    }

    return day.slots.filter((slot) => slot && slot.is_available).map((slot) => ({
        start_time: slot.start_time,
        end_time: slot.end_time,
    }));
}

function renderEditSlotSelect(slotSelect, slots, preferredSlot) {
    slotSelect.innerHTML = '';

    if (!slots.length) {
        const option = document.createElement('option');
        option.value = '';
        option.textContent = 'Keine freien Zeitslots';
        slotSelect.appendChild(option);
        slotSelect.disabled = true;
        return;
    }

    slotSelect.disabled = false;

    slots.forEach((slot) => {
        const option = document.createElement('option');
        option.value = `${slot.start_time}|${slot.end_time}`;
        option.textContent = `${slot.start_time} - ${slot.end_time}`;
        slotSelect.appendChild(option);
    });

    if (preferredSlot && slots.some((slot) => `${slot.start_time}|${slot.end_time}` === preferredSlot)) {
        slotSelect.value = preferredSlot;
        return;
    }

    slotSelect.value = `${slots[0].start_time}|${slots[0].end_time}`;
}
//termin bearbeitung darstellen
async function toggleAppointmentEditor(card, appointment) {
    const existingEditor = card.querySelector('.appointment-edit-form');
    if (existingEditor) {
        existingEditor.remove();
        return;
    }

    card.querySelectorAll('.appointment-edit-form').forEach((node) => node.remove());

    const editor = document.createElement('div');
    editor.className = 'appointment-edit-form';
    editor.innerHTML = `
        <div class="appointment-edit-title">Termin bearbeiten</div>
            <div class="appointment-edit-fields">
                <label class="appointment-edit-field">
                    Mitarbeiter
                    <select class="appointment-edit-employee"></select>
                </label>
                <label class="appointment-edit-field">
                    Datum
                    <select class="appointment-edit-date"></select>
                </label>
                <label class="appointment-edit-field">
                    Uhrzeit
                    <select class="appointment-edit-slot"></select>
                </label>
            </div>
            <div class="appointment-edit-actions">
                <button type="button" class="appointment-edit-save">Aenderungen speichern</button>
                <button type="button" class="appointment-edit-cancel">Abbrechen</button>
            </div>
    `;

    card.appendChild(editor);

    const employeeSelect = editor.querySelector('.appointment-edit-employee');
    const dateSelect = editor.querySelector('.appointment-edit-date');
    const slotSelect = editor.querySelector('.appointment-edit-slot');
    const saveBtn = editor.querySelector('.appointment-edit-save');
    const cancelBtn = editor.querySelector('.appointment-edit-cancel');

    const employees = getActiveEmployees();
    employeeSelect.innerHTML = '';
    employees.forEach((emp) => {
        const option = document.createElement('option');
        option.value = emp.id;
        option.textContent = emp.name;
        employeeSelect.appendChild(option);
    });

    if (!employees.length) {
        dateSelect.innerHTML = '<option value="">Keine Mitarbeiter verfuegbar</option>';
        slotSelect.innerHTML = '<option value="">Keine Mitarbeiter verfuegbar</option>';
        employeeSelect.disabled = true;
        dateSelect.disabled = true;
        slotSelect.disabled = true;
        saveBtn.disabled = true;
        cancelBtn.addEventListener('click', () => editor.remove());
        return;
    }

    const defaultEmployeeID = employees.some((emp) => emp.id === appointment.employee_id)
        ? appointment.employee_id
        : employees[0].id;
    employeeSelect.value = defaultEmployeeID;

    let availability = { dates: [] };
    //datum und zur verfügung stehende termine überprüfen um keine bereits belegten termine anzuzeigen
    const syncDateAndSlotOptions = () => {
        const selectedEmployeeID = employeeSelect.value;
        const previousDateValue = dateSelect.value;

        const availableDates = (availability.dates || [])
            .filter((day) => day && Array.isArray(day.slots) && day.slots.some((slot) => slot.is_available))
            .map((day) => day.date);

        if (selectedEmployeeID === appointment.employee_id && !availableDates.includes(appointment.date)) {
            availableDates.push(appointment.date);
        }

        dateSelect.innerHTML = '';

        if (!availableDates.length) {
            dateSelect.innerHTML = '<option value="">Keine verfuegbaren Tage</option>';
            dateSelect.disabled = true;
            renderEditSlotSelect(slotSelect, [], null);
            return;
        }

        availableDates.sort();
        availableDates.forEach((date) => {
            const option = document.createElement('option');
            option.value = date;
            option.textContent = formatAppointmentDate(date);
            dateSelect.appendChild(option);
        });

        dateSelect.disabled = false;

        if (previousDateValue && availableDates.includes(previousDateValue)) {
            dateSelect.value = previousDateValue;
        } else if (selectedEmployeeID === appointment.employee_id && availableDates.includes(appointment.date)) {
            dateSelect.value = appointment.date;
        } else {
            dateSelect.value = availableDates[0];
        }

        let slots = buildSlotOptionsForDate(availability, dateSelect.value);
        if (
            selectedEmployeeID === appointment.employee_id &&
            dateSelect.value === appointment.date &&
            !slots.some((slot) => slot.start_time === appointment.start_time && slot.end_time === appointment.end_time)
        ) {
            slots = [...slots, { start_time: appointment.start_time, end_time: appointment.end_time }];
        }

        const preferredSlot =
            selectedEmployeeID === appointment.employee_id && dateSelect.value === appointment.date
                ? `${appointment.start_time}|${appointment.end_time}`
                : null;

        renderEditSlotSelect(slotSelect, slots, preferredSlot);
    };
    //verfügbarkeit von mitarbeitern laden und mit den termin slots synchronisieren
    const loadAvailabilityAndSync = async () => {
        saveBtn.disabled = true;
        try {
            availability = await fetchAvailabilityForEmployee(employeeSelect.value);
            syncDateAndSlotOptions();
        } catch (error) {
            dateSelect.innerHTML = '<option value="">Fehler beim Laden</option>';
            slotSelect.innerHTML = '<option value="">Fehler beim Laden</option>';
            dateSelect.disabled = true;
            slotSelect.disabled = true;
            setAppointmentsFeedback(`Fehler beim Laden der Verfuegbarkeit: ${error.message}`, true);
        } finally {
            saveBtn.disabled = false;
        }
    };

    employeeSelect.addEventListener('change', async () => {
        await loadAvailabilityAndSync();
    });

    dateSelect.addEventListener('change', () => {
        let slots = buildSlotOptionsForDate(availability, dateSelect.value);
        if (
            employeeSelect.value === appointment.employee_id &&
            dateSelect.value === appointment.date &&
            !slots.some((slot) => slot.start_time === appointment.start_time && slot.end_time === appointment.end_time)
        ) {
            slots = [...slots, { start_time: appointment.start_time, end_time: appointment.end_time }];
        }

        const preferredSlot =
            employeeSelect.value === appointment.employee_id && dateSelect.value === appointment.date
                ? `${appointment.start_time}|${appointment.end_time}`
                : null;

        renderEditSlotSelect(slotSelect, slots, preferredSlot);
    });

    cancelBtn.addEventListener('click', () => {
        editor.remove();
    });

    saveBtn.addEventListener('click', async () => {
        if (!employeeSelect.value || !dateSelect.value || !slotSelect.value) {
            setAppointmentsFeedback('Bitte Mitarbeiter, Datum und Uhrzeit waehlen.', true);
            return;
        }

        const slotParts = slotSelect.value.split('|');
        if (slotParts.length !== 2) {
            setAppointmentsFeedback('Ungueltiger Zeitslot.', true);
            return;
        }

        const payload = {
            date: dateSelect.value,
            start_time: slotParts[0],
            end_time: slotParts[1],
            employee_id: employeeSelect.value,
        };
        //api request um einen termin zu ändern
        try {
            const res = await apiFetch(`/api/appointments/${encodeURIComponent(appointment.id)}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(payload),
            }, {
                attachAuth: true,
            });

            const data = await res.json();
            if (!res.ok) {
                throw new Error(data.error || 'Termin konnte nicht aktualisiert werden.');
            }

            await loadUserAppointments();
            await loadAvailability(await getAvailabilityEmployeeId());
            setAppointmentsFeedback(`Termin aktualisiert: ${getEmployeeNameByID(payload.employee_id)} ${payload.start_time}-${payload.end_time}`);
        } catch (error) {
            setAppointmentsFeedback(`Fehler beim Aktualisieren: ${error.message}`, true);
        }
    });

    await loadAvailabilityAndSync();
}
// einen termin löschen 
async function cancelAppointment(appointmentId) {
    if (!hasAuthSession()) {
        showLoggedOutState();
        return;
    }

    const shouldCancel = window.confirm('Soll dieser Termin wirklich storniert werden?');
    if (!shouldCancel) {
        return;
    }

    try {
        const res = await apiFetch(`/api/appointments/${encodeURIComponent(appointmentId)}`, {
            method: 'DELETE',
        }, {
            attachAuth: true,
        });

        const data = await res.json();
        if (!res.ok) {
            throw new Error(data.error || 'Termin konnte nicht storniert werden.');
        }

        await loadUserAppointments();
        await loadAvailability(await getAvailabilityEmployeeId());
        setAppointmentsFeedback('Termin wurde storniert.');
    } catch (error) {
        setAppointmentsFeedback(`Fehler beim Stornieren: ${error.message}`, true);
    }
}
// alle termine löschen
async function cancleAllAppointments() {
    if (!hasAuthSession()) {
        showLoggedOutState();
        return;
    }

    const shouldCancel = window.confirm('Sollen alle Termin wirklich storniert werden?');
    if (!shouldCancel) {
        return;
    }

    try {
        const res = await apiFetch(`/api/appointments/delete`, {
            method: 'DELETE',
        }, {
            attachAuth: true,
        });

        const data = await res.json();
        if (!res.ok) {
            throw new Error(data.error || 'Termine konnten nicht storniert werden.');
        }

        await loadUserAppointments();
        await loadAvailability(await getAvailabilityEmployeeId());
        setAppointmentsFeedback('Alle Termine wurden storniert.');
    } catch (error) {
        setAppointmentsFeedback(`Fehler beim Stornieren: ${error.message}`, true);
    }
}
// termine des users durch api von der datenbank laden
async function loadUserAppointments() {
    if (!hasAuthSession()) {
        showLoggedOutState();
        return;
    }

    try {
        const res = await apiFetch('/api/appointments', {}, {
            attachAuth: true,
        });

        const data = await res.json();
        if (!res.ok) {
            throw new Error(data.error || 'Termine konnten nicht geladen werden.');
        }

        renderUserAppointments(Array.isArray(data.data) ? data.data : []);
        setAppointmentsFeedback('');
    } catch (error) {
        renderUserAppointments([]);
        setAppointmentsFeedback(`Fehler beim Laden der Termine: ${error.message}`, true);
    }
}

let servicesTree = { data: [] };

async function loadServicesTree() {
    try {
        const res = await apiFetch('/api/services/tree');
        if (!res.ok) {
            throw new Error('Could not load services');
        }

        const data = await res.json();
        if (data && Array.isArray(data.data)) {
            servicesTree = { data: data.data };
            return;
        }
    } catch (error) {
        console.error('Failed to load services tree', error);
    }

    servicesTree = { data: [] };
}

// --- Mitarbeiter daten in db speichern mit der api ---
let employeesData = { data: [] };
// --- Mitarbeiter daten von der db laden---
async function loadEmployees() {
    try {
        const res = await apiFetch('/api/employees');
        if (!res.ok) {
            throw new Error('Could not load employees');
        }
        const data = await res.json();
        employeesData = data;
    } catch (error) {
        console.error(error);
        employeesData = { data: [] };
    }
}
// --- Termine sind nun im backend ---
let availabilityData = {
    employee_id: null,
    dates: []
};

async function getAvailabilityEmployeeId() {
    if (selectedEmployee && selectedEmployee !== 'no_preference') {
        return selectedEmployee;
    }

    if (selectedEmployee === 'no_preference') {
        const serviceNames = Array.from(selectedServices.values()).map(service => service.name);
        try {
            const res = await apiFetch('/api/employees/resolve', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ services: serviceNames })
            });
            if (!res.ok) {
                throw new Error('Could not resolve employee');
            }
            const data = await res.json();
            if (data && data.employee_id) {
                return data.employee_id;
            }
        } catch (error) {
            console.error('Failed to resolve employee', error);
        }

        return null;
    }

    return null;
}

async function loadAvailability(employeeId) {
    if (employeeId === undefined) {
        employeeId = await getAvailabilityEmployeeId();
    }

    if (!employeeId) {
        availabilityData = {
            employee_id: null,
            dates: []
        };
        return;
    }

    try {
        const res = await apiFetch(`/api/availability?employee_id=${encodeURIComponent(employeeId)}`);
        if (!res.ok) {
            throw new Error('Could not load availability');
        }
        const data = await res.json();
        if (data && Array.isArray(data.dates)) {
            availabilityData = {
                employee_id: employeeId,
                dates: data.dates
            };
        } else {
            availabilityData = {
                employee_id: employeeId,
                dates: []
            };
        }
    } catch (error) {
        console.error('Failed to load availability', error);
        availabilityData = {
            employee_id: employeeId,
            dates: []
        };
    }
}

// --- State ---
let path = [];
let selectedServices = new Map();
let selectedEmployee = null;
let selectedTimeSlot = null;
let currentStep = "services"; // "services", "employee", "appointment"
let calendarView = "day"; // "day" or "month"
let currentDate = null; // Aktuell angezeigtes Datum

// --- Rendering ---
function render(nodes) {
    const listEl = document.getElementById("service-list");
    listEl.innerHTML = "";

    if (!Array.isArray(nodes) || nodes.length === 0) {
        const emptyState = document.createElement("div");
        emptyState.className = "appointment-empty";
        emptyState.textContent = "Keine Dienstleistungen verfügbar.";
        listEl.appendChild(emptyState);
        renderBreadcrumbs();
        return;
    }

    nodes.forEach(node => {
        if (node.is_active === false) return;

        const card = document.createElement("div");
        card.className = "flex-service-card";

        if (node.children) {
            card.innerHTML = `<div class="flex-service-name">${node.name}</div>`;
            card.addEventListener("click", () => {
                path.push(node);
                render(node.children);
                renderBreadcrumbs();
            });
        } else {
            const checked = selectedServices.has(node.id) ? "checked" : "";
            card.innerHTML = `
                <input type="checkbox" class="flex-checkbox" ${checked}>
                <div class="flex-service-content">
                    <div class="flex-service-name">${node.name}</div>
                    <div class="flex-service-description">${node.description}</div>
                    <div class="flex-service-meta">
                        Dauer: ${node.duration_minutes} Min<br>
                        Preis: ${node.price.toFixed(2)} ${node.currency}
                    </div>
                </div>
            `;

            const checkbox = card.querySelector(".flex-checkbox");

            // Klick auf Karte toggelt Haken
            card.addEventListener("click", (e) => {
                if (e.target !== checkbox) {
                    checkbox.checked = !checkbox.checked;
                }
                toggleSelection(node, checkbox.checked);
            });

            // Klick auf Checkbox direkt
            checkbox.addEventListener("change", (e) => {
                toggleSelection(node, e.target.checked);
            });
        }

        listEl.appendChild(card);
    });

    renderBreadcrumbs();
}

// --- Toggle Auswahl ---
function toggleSelection(node, checked) {
    if (checked) {
        selectedServices.set(node.id, node);
    } else {
        selectedServices.delete(node.id);
    }
    renderSelected();
}

// --- Breadcrumbs ---
function resetBookingFlow() {
    path = [];
    selectedServices.clear();
    selectedEmployee = null;
    selectedTimeSlot = null;
    currentStep = "services";
    calendarView = "day";
    currentDate = null;
    availabilityData = { employee_id: null, dates: [] };
    renderSelected();
    showStep("services");
    render(servicesTree.data);
    renderBreadcrumbs();
}

function renderBreadcrumbs() {
    const bcEl = document.getElementById("breadcrumbs");
    bcEl.innerHTML = "";

    const rootSpan = document.createElement("span");
    rootSpan.textContent = "Start";
    rootSpan.addEventListener("click", () => {
        resetBookingFlow();
    });
    bcEl.appendChild(rootSpan);

    path.forEach((node, idx) => {
        const span = document.createElement("span");
        span.textContent = node.name;
        span.addEventListener("click", () => {
            path = path.slice(0, idx + 1);
            render(node.children);
        });
        bcEl.appendChild(span);
    });
}

// --- Ausgewählte Dienstleistungen ---
function renderSelected() {
    const container = document.getElementById("selected-services");
    const listEl = document.getElementById("selected-list");
    listEl.innerHTML = "";

    if (selectedServices.size === 0) {
        container.style.display = "none";
        return;
    }

    container.style.display = "block";
    selectedServices.forEach(node => {
        const item = document.createElement("div");
        item.className = "flex-selected-item";

        const textSpan = document.createElement("span");
        textSpan.textContent = node.name + " (" + node.duration_minutes + " Min, " + node.price.toFixed(2) + " " + node.currency + ")";

        const removeBtn = document.createElement("button");
        removeBtn.className = "flex-remove-btn";
        removeBtn.textContent = "✕";
        removeBtn.addEventListener("click", () => {
            selectedServices.delete(node.id);
            renderSelected();
            render(path.length > 0 ? path[path.length - 1].children : servicesTree.data);
        });

        item.appendChild(textSpan);
        item.appendChild(removeBtn);
        listEl.appendChild(item);
    });
}

// --- Bestätigungsbutton ---
document.getElementById("confirm-selection").addEventListener("click", () => {
    if (selectedServices.size === 0) {
        alert("Bitte wählen Sie mindestens eine Dienstleistung aus.");
        return;
    }
    currentStep = "employee";
    showStep(currentStep);
    renderEmployees();
});

// --- Alle entfernen Button ---
document.getElementById("clear-selection").addEventListener("click", () => {
    selectedServices.clear();
    renderSelected();
    render(path.length > 0 ? path[path.length - 1].children : servicesTree.data);
});

// --- Mitarbeiter rendern ---
function renderEmployees() {
    const listEl = document.getElementById("employee-list");
    listEl.innerHTML = "";

    // Dienstleistungszusammenfassung anzeigen
    renderServicesSummary();

    // "Keine Präferenz" Option
    const noPreferenceCard = document.createElement("div");
    noPreferenceCard.className = "flex-employee-card";
    const noPreferenceChecked = selectedEmployee === "no_preference" ? "checked" : "";
    noPreferenceCard.innerHTML = `
        <input type="checkbox" class="flex-employee-checkbox" ${noPreferenceChecked}>
        <div class="flex-employee-info">
            <div class="flex-employee-avatar">?</div>
            <div class="flex-employee-details">
                <div class="flex-employee-name">Keine Präferenz</div>
                <div class="flex-employee-title">Erster Passender Mitarbeiter</div>
            </div>
        </div>
    `;

    const noPreferenceCheckbox = noPreferenceCard.querySelector(".flex-employee-checkbox");

    noPreferenceCard.addEventListener("click", (e) => {
        if (e.target !== noPreferenceCheckbox) {
            noPreferenceCheckbox.checked = !noPreferenceCheckbox.checked;
        }
        if (noPreferenceCheckbox.checked) {
            selectedEmployee = "no_preference";
            renderEmployees();
        }
    });

    noPreferenceCheckbox.addEventListener("change", (e) => {
        if (e.target.checked) {
            selectedEmployee = "no_preference";
            renderEmployees();
        }
    });

    listEl.appendChild(noPreferenceCard);

    // Mitarbeiter
    employeesData.data.forEach(emp => {
        if (!emp.is_active) return;

        const card = document.createElement("div");
        card.className = "flex-employee-card";
        const empChecked = selectedEmployee === emp.id ? "checked" : "";

        const initials = emp.name.split(" ").map(n => n[0]).join("");

        card.innerHTML = `
            <input type="checkbox" class="flex-employee-checkbox" ${empChecked}>
            <div class="flex-employee-info">
                <div class="flex-employee-avatar">${initials}</div>
                <div class="flex-employee-details">
                    <div class="flex-employee-name">${emp.name}</div>
                    <div class="flex-employee-title">${emp.title}</div>
                    <div class="flex-employee-specialties">Spezialisiert auf: ${emp.specialties.join(", ")}</div>
                </div>
            </div>
        `;

        const checkbox = card.querySelector(".flex-employee-checkbox");

        card.addEventListener("click", (e) => {
            if (e.target !== checkbox) {
                checkbox.checked = !checkbox.checked;
            }
            if (checkbox.checked) {
                selectedEmployee = emp.id;
                renderEmployees();
            }
        });

        checkbox.addEventListener("change", (e) => {
            if (e.target.checked) {
                selectedEmployee = emp.id;
                renderEmployees();
            }
        });

        listEl.appendChild(card);
    });
}

// --- Schritt wechseln ---
function showStep(step) {
    document.getElementById("step-services").classList.remove("active");
    document.getElementById("step-employee").classList.remove("active");
    document.getElementById("step-appointment").classList.remove("active");
    document.getElementById("step-" + step).classList.add("active");
}

// --- Dienstleistungszusammenfassung auf Mitarbeiterseite ---
function renderServicesSummary() {
    const summaryList = document.getElementById("services-summary-list");
    summaryList.innerHTML = "";

    let totalDuration = 0;
    let totalPrice = 0;

    selectedServices.forEach(node => {
        const item = document.createElement("div");
        item.className = "flex-selected-item";
        item.style.marginBottom = "5px";

        const textSpan = document.createElement("span");
        textSpan.textContent = node.name + " (" + node.duration_minutes + " Min, " + node.price.toFixed(2) + " " + node.currency + ")";

        item.appendChild(textSpan);
        summaryList.appendChild(item);

        totalDuration += node.duration_minutes;
        totalPrice += node.price;
    });

    // Gesamtsumme anzeigen
    const totalItem = document.createElement("div");
    totalItem.style.marginTop = "10px";
    totalItem.style.fontWeight = "bold";
    totalItem.style.paddingTop = "10px";
    totalItem.style.borderTop = "2px solid #ddd";
    totalItem.textContent = `Gesamt: ${totalDuration} Min, ${totalPrice.toFixed(2)} EUR`;
    summaryList.appendChild(totalItem);
}

// --- Zurück Button ---
document.getElementById("back-to-services").addEventListener("click", () => {
    currentStep = "services";
    showStep(currentStep);
});

// --- Mitarbeiter bestätigen ---
document.getElementById("confirm-employee").addEventListener("click", async () => {
    if (!selectedEmployee) {
        alert("Bitte wählen Sie einen Mitarbeiter aus oder wählen Sie 'Keine Präferenz'.");
        return;
    }

    currentStep = "appointment";
    showStep(currentStep);
    const resolvedEmployeeId = await getAvailabilityEmployeeId();
    await loadAvailability(resolvedEmployeeId);

    // Ersten verfügbaren Tag finden
    if (availabilityData.dates.length > 0) {
        currentDate = availabilityData.dates[0].date;
    }

    renderAppointmentSummary();
    renderDayView();
});

// --- Terminzusammenfassung ---
function renderAppointmentSummary() {
    const summaryEl = document.getElementById("appointment-summary");
    summaryEl.innerHTML = "";

    // Mitarbeiter
    const empName = selectedEmployee === "no_preference"
        ? `Keine Präferenz (${employeesData.data.find(e => e.id === availabilityData.employee_id)?.name || "Unbekannt"})`
        : employeesData.data.find(e => e.id === selectedEmployee)?.name || "Unbekannt";

    const empDiv = document.createElement("div");
    empDiv.style.marginBottom = "10px";
    empDiv.innerHTML = `<strong>Mitarbeiter:</strong> ${empName}`;
    summaryEl.appendChild(empDiv);

    // Dienstleistungen
    const servicesDiv = document.createElement("div");
    servicesDiv.innerHTML = "<strong>Dienstleistungen:</strong>";
    summaryEl.appendChild(servicesDiv);

    let totalDuration = 0;
    let totalPrice = 0;

    selectedServices.forEach(node => {
        const item = document.createElement("div");
        item.style.marginLeft = "20px";
        item.textContent = "• " + node.name + " (" + node.duration_minutes + " Min, " + node.price.toFixed(2) + " EUR)";
        summaryEl.appendChild(item);

        totalDuration += node.duration_minutes;
        totalPrice += node.price;
    });

    const totalDiv = document.createElement("div");
    totalDiv.style.marginTop = "10px";
    totalDiv.style.fontWeight = "bold";
    totalDiv.innerHTML = `<strong>Gesamt:</strong> ${totalDuration} Min, ${totalPrice.toFixed(2)} EUR`;
    summaryEl.appendChild(totalDiv);
}

// --- Tagesansicht rendern ---
function renderDayView() {
    const dateData = availabilityData.dates.find(d => d.date === currentDate);

    if (!dateData) {
        document.getElementById("timeslot-list").innerHTML = "<p>Keine Termine verfügbar für diesen Tag.</p>";
        return;
    }

    // Datum formatieren
    const date = new Date(currentDate + "T00:00:00");
    const options = { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' };
    document.getElementById("current-day-title").textContent = date.toLocaleDateString('de-DE', options);

    const listEl = document.getElementById("timeslot-list");
    listEl.innerHTML = "";

    dateData.slots.forEach(slot => {
        const slotDiv = document.createElement("div");
        slotDiv.className = "flex-timeslot " + (slot.is_available ? "available" : "blocked");

        if (slot.is_available) {
            const isChecked = selectedTimeSlot &&
                selectedTimeSlot.date === currentDate &&
                selectedTimeSlot.start_time === slot.start_time;

            slotDiv.innerHTML = `
                <input type="checkbox" class="flex-timeslot-checkbox" ${isChecked ? "checked" : ""}>
                <div class="flex-timeslot-time">${slot.start_time} - ${slot.end_time}</div>
            `;

            const checkbox = slotDiv.querySelector(".flex-timeslot-checkbox");

            slotDiv.addEventListener("click", (e) => {
                if (e.target !== checkbox) {
                    checkbox.checked = !checkbox.checked;
                }
                if (checkbox.checked) {
                    selectedTimeSlot = {
                        date: currentDate,
                        start_time: slot.start_time,
                        end_time: slot.end_time
                    };
                    renderDayView();
                } else {
                    selectedTimeSlot = null;
                }
            });

            checkbox.addEventListener("change", (e) => {
                if (e.target.checked) {
                    selectedTimeSlot = {
                        date: currentDate,
                        start_time: slot.start_time,
                        end_time: slot.end_time
                    };
                    renderDayView();
                } else {
                    selectedTimeSlot = null;
                }
            });
        } else {
            slotDiv.innerHTML = `
                <input type="checkbox" class="flex-timeslot-checkbox" disabled>
                <div class="flex-timeslot-time">${slot.start_time} - ${slot.end_time} (Besetzt)</div>
            `;
        }

        listEl.appendChild(slotDiv);
    });
}

// --- Monatsansicht rendern ---
function renderMonthView() {
    const date = new Date(currentDate + "T00:00:00");
    const year = date.getFullYear();
    const month = date.getMonth();

    const monthNames = ["Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"];
    document.getElementById("current-month-title").textContent = monthNames[month] + " " + year;

    const firstDay = new Date(year, month, 1);
    const lastDay = new Date(year, month + 1, 0);
    const startDay = firstDay.getDay() === 0 ? 6 : firstDay.getDay() - 1;

    const gridEl = document.getElementById("month-grid");
    gridEl.innerHTML = "";

    // Leere Zellen vor dem ersten Tag
    for (let i = 0; i < startDay; i++) {
        const emptyDiv = document.createElement("div");
        emptyDiv.className = "flex-month-day empty";
        gridEl.appendChild(emptyDiv);
    }

    // Tage des Monats
    for (let day = 1; day <= lastDay.getDate(); day++) {
        const dateStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
        const dayData = availabilityData.dates.find(d => d.date === dateStr);
        const hasAvailableSlots = dayData && dayData.slots.some(s => s.is_available);

        const dayDiv = document.createElement("div");
        dayDiv.className = "flex-month-day";
        dayDiv.textContent = day;

        if (hasAvailableSlots) {
            dayDiv.classList.add("has-slots");
            dayDiv.addEventListener("click", () => {
                currentDate = dateStr;
                calendarView = "day";
                document.getElementById("view-day").classList.add("active");
                document.getElementById("view-month").classList.remove("active");
                document.getElementById("day-view").style.display = "block";
                document.getElementById("month-view").style.display = "none";
                renderDayView();
            });
        } else {
            dayDiv.classList.add("no-slots");
        }

        gridEl.appendChild(dayDiv);
    }
}

// --- View Toggle ---
document.getElementById("view-day").addEventListener("click", () => {
    calendarView = "day";
    document.getElementById("view-day").classList.add("active");
    document.getElementById("view-month").classList.remove("active");
    document.getElementById("day-view").style.display = "block";
    document.getElementById("month-view").style.display = "none";
});

document.getElementById("view-month").addEventListener("click", () => {
    calendarView = "month";
    document.getElementById("view-month").classList.add("active");
    document.getElementById("view-day").classList.remove("active");
    document.getElementById("day-view").style.display = "none";
    document.getElementById("month-view").style.display = "block";
    renderMonthView();
});

// --- Tag Navigation ---
document.getElementById("prev-day").addEventListener("click", () => {
    const currentIndex = availabilityData.dates.findIndex(d => d.date === currentDate);
    if (currentIndex > 0) {
        currentDate = availabilityData.dates[currentIndex - 1].date;
        renderDayView();
    }
});

document.getElementById("next-day").addEventListener("click", () => {
    const currentIndex = availabilityData.dates.findIndex(d => d.date === currentDate);
    if (currentIndex < availabilityData.dates.length - 1) {
        currentDate = availabilityData.dates[currentIndex + 1].date;
        renderDayView();
    }
});

// --- Monat Navigation ---
document.getElementById("prev-month").addEventListener("click", () => {
    const date = new Date(currentDate + "T00:00:00");
    date.setMonth(date.getMonth() - 1);
    currentDate = date.toISOString().split('T')[0];
    renderMonthView();
});

document.getElementById("next-month").addEventListener("click", () => {
    const date = new Date(currentDate + "T00:00:00");
    date.setMonth(date.getMonth() + 1);
    currentDate = date.toISOString().split('T')[0];
    renderMonthView();
});

// --- Zurück zu Mitarbeiter ---
document.getElementById("back-to-employee").addEventListener("click", () => {
    currentStep = "employee";
    showStep(currentStep);
});

// --- Termin buchen ---
document.getElementById("confirm-appointment").addEventListener("click", async () => {
    if (!selectedTimeSlot) {
        alert("Bitte wählen Sie einen Zeitslot aus.");
        return;
    }

    const serviceIds = Array.from(selectedServices.keys());
    const payload = {
        date: selectedTimeSlot.date,
        start_time: selectedTimeSlot.start_time,
        end_time: selectedTimeSlot.end_time,
        employee_id: selectedEmployee === "no_preference" ? "" : selectedEmployee,
        no_preference: selectedEmployee === "no_preference",
        service_ids: serviceIds
    };
    // --- An api senden mit dem derzeitig eingeloggten user ---
    try {
        const res = await apiFetch('/api/appointments', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(payload),
        }, {
            attachAuth: true,
        });

        const data = await res.json();

        if (!res.ok) {
            throw new Error(data.error || 'Termin konnte nicht gesendet werden.');
        }

        // --- Nachdem ein Termin gebucht wurde wird er belegt und dann die seite neu geladen ---
        await loadAvailability(await getAvailabilityEmployeeId());
        await loadUserAppointments();
        resetBookingFlow();

        alert(`Termin gesendet!\n\n${JSON.stringify(data, null, 2)}`);
    } catch (error) {
        alert(`Fehler: ${error.message}`);
    }
});

// --- Initial render happens after data loading in DOMContentLoaded ---