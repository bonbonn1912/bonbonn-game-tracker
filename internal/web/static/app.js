/**
 * BonBonn Game Tracker - Admin Dashboard Application
 * Vanilla ES6+ JavaScript. Zero external dependencies.
 * All dynamic content is safely rendered using textContent / DOM APIs to prevent XSS.
 */

(function () {
  'use strict';

  // --- API Routes & Configuration ---
  const API_BASE = '/admin/api';
  const ENDPOINTS = {
    status: `${API_BASE}/status`,
    streamers: `${API_BASE}/streamers`,
    events: `${API_BASE}/events`,
    webhookRequests: `${API_BASE}/webhook-requests`,
    players: `${API_BASE}/players`,
    playerMatches: (id, limit = 50, offset = 0) =>
      `${API_BASE}/players/${encodeURIComponent(id)}/matches?limit=${encodeURIComponent(limit)}&offset=${encodeURIComponent(offset)}`,
    matches: `${API_BASE}/matches`
  };

  // Strict regex for lowercase hexadecimal UUIDv4
  const UUID_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;

  // State
  let state = {
    activeTab: 'players', // 'players' | 'matches' | 'streamers' | 'events' | 'webhooks'
    streamers: [],
    events: [],
    webhookRequests: [],
    players: [],
    selectedPlayer: null,
    playerMatches: [],
    playerMatchesTotal: 0,
    playerMatchesOffset: 0,
    matches: {
      live: [],
      past: []
    },
    status: { database: 'unknown', activeMatches: 0 },
    lastEventsByStreamerId: new Map(),
    streamerToDelete: null
  };

  let playerHistoryRequestSeq = 0;

  // --- DOM Elements ---
  const elDbBadge = document.getElementById('db-status-badge');
  const elActiveMatches = document.getElementById('active-matches-count');
  const elBtnRefreshAll = document.getElementById('btn-refresh-all');
  const elNotificationArea = document.getElementById('notification-area');

  // Tab Navigation Elements
  const elTabBtnPlayers = document.getElementById('tab-btn-players');
  const elTabBtnMatches = document.getElementById('tab-btn-matches');
  const elTabBtnStreamers = document.getElementById('tab-btn-streamers');
  const elTabBtnEvents = document.getElementById('tab-btn-events');
  const elTabBtnWebhooks = document.getElementById('tab-btn-webhooks');

  const elTabPanelPlayers = document.getElementById('tab-panel-players');
  const elTabPanelMatches = document.getElementById('tab-panel-matches');
  const elTabPanelStreamers = document.getElementById('tab-panel-streamers');
  const elTabPanelEvents = document.getElementById('tab-panel-events');
  const elTabPanelWebhooks = document.getElementById('tab-panel-webhooks');

  // Players Elements
  const elPlayersListContainer = document.getElementById('players-list-container');
  const elPlayersDetailContainer = document.getElementById('players-detail-container');
  const elPlayersLoading = document.getElementById('players-loading');
  const elPlayersEmpty = document.getElementById('players-empty');
  const elPlayersTableWrapper = document.getElementById('players-table-wrapper');
  const elPlayersTbody = document.getElementById('players-tbody');
  const elBtnRefreshPlayers = document.getElementById('btn-refresh-players');

  // Player Detail Elements
  const elBtnPlayerDetailBack = document.getElementById('btn-player-detail-back');
  const elPlayerDetailTitle = document.getElementById('player-detail-title');
  const elPlayerDetailId = document.getElementById('player-detail-id');
  const elPlayerDetailGamecount = document.getElementById('player-detail-gamecount');
  const elPlayerDetailLastWebhook = document.getElementById('player-detail-lastwebhook');
  const elBtnRefreshPlayerMatches = document.getElementById('btn-refresh-player-matches');
  const elPlayerMatchesLoading = document.getElementById('player-matches-loading');
  const elPlayerMatchesError = document.getElementById('player-matches-error');
  const elPlayerMatchesEmpty = document.getElementById('player-matches-empty');
  const elPlayerMatchesTableWrapper = document.getElementById('player-matches-table-wrapper');
  const elPlayerMatchesTbody = document.getElementById('player-matches-tbody');
  const elPlayerMatchesPaginationInfo = document.getElementById('player-matches-pagination-info');
  const elBtnPlayerLoadMore = document.getElementById('btn-player-load-more');
  const elPlayerLoadMoreSpinner = document.getElementById('player-load-more-spinner');

  // Matches Elements
  const elMatchesLoading = document.getElementById('matches-loading');
  const elMatchesError = document.getElementById('matches-error');
  const elBtnRefreshMatches = document.getElementById('btn-refresh-matches');
  const elMatchesLiveEmpty = document.getElementById('matches-live-empty');
  const elMatchesLiveTableWrapper = document.getElementById('matches-live-table-wrapper');
  const elMatchesLiveTbody = document.getElementById('matches-live-tbody');
  const elMatchesPastEmpty = document.getElementById('matches-past-empty');
  const elMatchesPastTableWrapper = document.getElementById('matches-past-table-wrapper');
  const elMatchesPastTbody = document.getElementById('matches-past-tbody');

  // Streamers Elements
  const elStreamersLoading = document.getElementById('streamers-loading');
  const elStreamersEmpty = document.getElementById('streamers-empty');
  const elStreamersTableWrapper = document.getElementById('streamers-table-wrapper');
  const elStreamersTbody = document.getElementById('streamers-tbody');
  const elBtnOpenCreateModal = document.getElementById('btn-open-create-modal');
  const elBtnEmptyAddStreamer = document.getElementById('btn-empty-add-streamer');

  // Events Elements
  const elEventLimitSelect = document.getElementById('event-limit-select');
  const elBtnRefreshEvents = document.getElementById('btn-refresh-events');
  const elEventsLoading = document.getElementById('events-loading');
  const elEventsEmpty = document.getElementById('events-empty');
  const elEventsTableWrapper = document.getElementById('events-table-wrapper');
  const elEventsTbody = document.getElementById('events-tbody');
  const elWebhookLimitSelect = document.getElementById('webhook-limit-select');
  const elBtnRefreshWebhooks = document.getElementById('btn-refresh-webhooks');
  const elWebhooksLoading = document.getElementById('webhooks-loading');
  const elWebhooksEmpty = document.getElementById('webhooks-empty');
  const elWebhooksTableWrapper = document.getElementById('webhooks-table-wrapper');
  const elWebhooksTbody = document.getElementById('webhooks-tbody');

  // Streamer Form Modal Elements
  const elStreamerModal = document.getElementById('streamer-modal');
  const elStreamerForm = document.getElementById('streamer-form');
  const elModalTitle = document.getElementById('modal-title');
  const elModalError = document.getElementById('modal-error-message');
  const elFormMode = document.getElementById('form-streamer-mode');
  const elInputStreamerId = document.getElementById('input-streamer-id');
  const elInputStreamerName = document.getElementById('input-streamer-name');
  const elInputStreamerUrl = document.getElementById('input-streamer-url');
  const elInputStreamerEnabled = document.getElementById('input-streamer-enabled');
  const elBtnModalCancel = document.getElementById('btn-modal-cancel');
  const elBtnModalClose = document.getElementById('btn-modal-close');
  const elBtnModalSave = document.getElementById('btn-modal-save');
  const elBtnModalSpinner = document.getElementById('btn-modal-spinner');

  const elErrorId = document.getElementById('error-streamer-id');
  const elErrorName = document.getElementById('error-streamer-name');
  const elErrorUrl = document.getElementById('error-streamer-url');

  // Delete Modal Elements
  const elDeleteModal = document.getElementById('delete-modal');
  const elDeleteModalMessage = document.getElementById('delete-modal-message');
  const elBtnDeleteModalCancel = document.getElementById('btn-delete-modal-cancel');
  const elBtnDeleteModalClose = document.getElementById('btn-delete-modal-close');
  const elBtnDeleteModalConfirm = document.getElementById('btn-delete-modal-confirm');
  const elBtnDeleteSpinner = document.getElementById('btn-delete-spinner');

  // --- Utility Helpers ---

  /**
   * Safe DOM node factory to strictly prevent innerHTML injection.
   */
  function createEl(tagName, attributes = {}, ...children) {
    const node = document.createElement(tagName);
    for (const [key, val] of Object.entries(attributes)) {
      if (key === 'className') {
        node.className = val;
      } else if (key === 'textContent') {
        node.textContent = val;
      } else if (key.startsWith('on') && typeof val === 'function') {
        node.addEventListener(key.slice(2).toLowerCase(), val);
      } else if (val !== null && val !== undefined) {
        node.setAttribute(key, String(val));
      }
    }
    for (const child of children) {
      if (typeof child === 'string' || typeof child === 'number') {
        node.appendChild(document.createTextNode(String(child)));
      } else if (child instanceof Node) {
        node.appendChild(child);
      }
    }
    return node;
  }

  /**
   * Validate standard HTTP/HTTPS URLs.
   */
  function isValidHttpUrl(urlString) {
    if (!urlString || typeof urlString !== 'string') return false;
    try {
      const parsed = new URL(urlString.trim());
      return parsed.protocol === 'http:' || parsed.protocol === 'https:';
    } catch (_) {
      return false;
    }
  }

  /**
   * Format ISO date string into German locale representation.
   */
  function formatGermanDate(isoString) {
    if (!isoString) return '-';
    const date = new Date(isoString);
    if (isNaN(date.getTime())) return String(isoString);
    return new Intl.DateTimeFormat('de-DE', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    }).format(date);
  }

  /**
   * Human-friendly German outcome labels for access events.
   */
  function formatOutcomeBadge(outcome) {
    const raw = String(outcome || '').trim();
    let badgeClass = 'badge badge-outcome-warn';
    let text = raw || 'Unbekannt';

    if (raw === 'redirect_success' || raw === 'success' || raw === 'redirected') {
      badgeClass = 'badge badge-outcome-ok';
      text = 'Weitergeleitet';
    } else if (raw === 'no_active_match' || raw === 'no_game') {
      badgeClass = 'badge badge-outcome-warn';
      text = 'Kein aktives Match';
    } else if (raw === 'disabled' || raw === 'streamer_disabled') {
      badgeClass = 'badge badge-outcome-fail';
      text = 'Deaktiviert';
    } else if (raw === 'not_found') {
      badgeClass = 'badge badge-outcome-fail';
      text = 'Nicht gefunden';
    }

    return createEl('span', { className: badgeClass, textContent: text });
  }

  /**
   * Match status badge (Live / Beendet / Custom).
   */
  function formatMatchStatusBadge(status) {
    const raw = String(status || '').trim().toLowerCase();
    let badgeClass = 'badge badge-inactive';
    let text = raw || 'Unbekannt';

    if (raw === 'live' || raw === 'active' || raw === 'running') {
      badgeClass = 'badge badge-active';
      text = 'Live';
    } else if (raw === 'closed' || raw === 'finished' || raw === 'beendet') {
      badgeClass = 'badge badge-inactive';
      text = 'Beendet';
    } else if (status) {
      text = String(status);
    }

    return createEl('span', { className: badgeClass, textContent: text });
  }

  /**
   * Format Matchup information (teams and average Elo).
   */
  function formatMatchupText(match) {
    if (!match) return '-';
    const mu = match.matchup;
    if (typeof mu === 'string' && mu.trim()) {
      return mu;
    }
    if (mu && typeof mu === 'object') {
      if (mu.team1 && mu.team2) {
        const t1Name = mu.team1.name || 'Team 1';
        const t1Elo = (mu.team1.avgElo !== undefined && mu.team1.avgElo !== null)
          ? mu.team1.avgElo
          : (mu.team1.stats && mu.team1.stats.avgElo);
        const t2Name = mu.team2.name || 'Team 2';
        const t2Elo = (mu.team2.avgElo !== undefined && mu.team2.avgElo !== null)
          ? mu.team2.avgElo
          : (mu.team2.stats && mu.team2.stats.avgElo);

        const elo1Str = (t1Elo !== undefined && t1Elo !== null) ? ` (${t1Elo})` : '';
        const elo2Str = (t2Elo !== undefined && t2Elo !== null) ? ` (${t2Elo})` : '';
        return `${t1Name}${elo1Str} vs. ${t2Name}${elo2Str}`;
      }
      if (Array.isArray(mu) && mu.length >= 2) {
        const t1 = mu[0] || {};
        const t2 = mu[1] || {};
        const t1Name = t1.name || 'Team 1';
        const t1Elo = t1.avgElo ?? (t1.stats && t1.stats.avgElo);
        const t2Name = t2.name || 'Team 2';
        const t2Elo = t2.avgElo ?? (t2.stats && t2.stats.avgElo);
        const elo1Str = (t1Elo !== undefined && t1Elo !== null) ? ` (${t1Elo})` : '';
        const elo2Str = (t2Elo !== undefined && t2Elo !== null) ? ` (${t2Elo})` : '';
        return `${t1Name}${elo1Str} vs. ${t2Name}${elo2Str}`;
      }
    }
    // Fallback: check payload
    if (match.payload && match.payload.teams && Array.isArray(match.payload.teams) && match.payload.teams.length >= 2) {
      const t1 = match.payload.teams[0] || {};
      const t2 = match.payload.teams[1] || {};
      const t1Name = t1.name || 'Team 1';
      const t1Elo = t1.stats && t1.stats.avgElo;
      const t2Name = t2.name || 'Team 2';
      const t2Elo = t2.stats && t2.stats.avgElo;
      const elo1Str = (t1Elo !== undefined && t1Elo !== null) ? ` (${t1Elo})` : '';
      const elo2Str = (t2Elo !== undefined && t2Elo !== null) ? ` (${t2Elo})` : '';
      return `${t1Name}${elo1Str} vs. ${t2Name}${elo2Str}`;
    }
    return '-';
  }

  /**
   * Format Map information.
   */
  function formatMapText(match) {
    if (!match) return '-';
    if (match.map && typeof match.map === 'string' && match.map.trim()) {
      return match.map;
    }
    if (match.payload) {
      if (match.payload.map && typeof match.payload.map === 'string' && match.payload.map.trim()) {
        return match.payload.map;
      }
      if (
        match.payload.voting &&
        match.payload.voting.map &&
        Array.isArray(match.payload.voting.map.pick) &&
        match.payload.voting.map.pick.length > 0
      ) {
        return match.payload.voting.map.pick[0];
      }
    }
    return '-';
  }

  /**
   * Display accessible notification banner.
   */
  function showNotification(message, type = 'error', durationMs = 6000) {
    const banner = createEl('div', {
      className: `alert alert-${type}`,
      role: 'alert'
    });

    const msgSpan = createEl('span', { textContent: message });
    const btnDismiss = createEl('button', {
      type: 'button',
      className: 'btn-close',
      'aria-label': 'Hinweis schließen',
      textContent: '×',
      onClick: () => banner.remove()
    });

    banner.appendChild(msgSpan);
    banner.appendChild(btnDismiss);
    elNotificationArea.appendChild(banner);

    if (durationMs > 0) {
      setTimeout(() => {
        if (banner.parentElement) {
          banner.remove();
        }
      }, durationMs);
    }
  }

  // --- API Fetch Service ---

  async function apiFetch(url, options = {}) {
    const defaultHeaders = {
      Accept: 'application/json'
    };
    if (options.body && typeof options.body === 'string') {
      defaultHeaders['Content-Type'] = 'application/json';
    }

    const config = {
      ...options,
      headers: {
        ...defaultHeaders,
        ...(options.headers || {})
      }
    };

    try {
      const response = await fetch(new URL(url, window.location.origin), config);

      if (response.status === 401) {
        showNotification('Zugriff verweigert (401 Unauthorized). Bitte HTTP Basic Auth Zugangsdaten prüfen.', 'error');
        throw new Error('Authentifizierung fehlgeschlagen');
      }

      // Read response body text exactly once
      const rawText = await response.text();

      if (!response.ok) {
        let errDetail = `${response.status} ${response.statusText}`;
        if (rawText) {
          try {
            const errJson = JSON.parse(rawText);
            if (errJson && (errJson.error || errJson.message)) {
              errDetail = errJson.error || errJson.message;
            } else {
              errDetail = rawText;
            }
          } catch (_) {
            errDetail = rawText;
          }
        }
        throw new Error(errDetail);
      }

      // Handle empty response (e.g. 204 No Content)
      if (response.status === 204 || !rawText || !rawText.trim()) {
        return null;
      }

      return JSON.parse(rawText);
    } catch (err) {
      console.error(`API Error on ${url}:`, err);
      throw err;
    }
  }

  // --- Tab Navigation Handlers ---

  const TABS = ['players', 'matches', 'streamers', 'events', 'webhooks'];

  function switchTab(tabName) {
    if (!TABS.includes(tabName)) tabName = 'players';
    state.activeTab = tabName;

    // Update tab buttons
    elTabBtnPlayers.classList.toggle('tab-active', tabName === 'players');
    elTabBtnPlayers.setAttribute('aria-selected', tabName === 'players' ? 'true' : 'false');
    elTabBtnPlayers.setAttribute('tabindex', tabName === 'players' ? '0' : '-1');

    elTabBtnMatches.classList.toggle('tab-active', tabName === 'matches');
    elTabBtnMatches.setAttribute('aria-selected', tabName === 'matches' ? 'true' : 'false');
    elTabBtnMatches.setAttribute('tabindex', tabName === 'matches' ? '0' : '-1');

    elTabBtnStreamers.classList.toggle('tab-active', tabName === 'streamers');
    elTabBtnStreamers.setAttribute('aria-selected', tabName === 'streamers' ? 'true' : 'false');
    elTabBtnStreamers.setAttribute('tabindex', tabName === 'streamers' ? '0' : '-1');

    elTabBtnEvents.classList.toggle('tab-active', tabName === 'events');
    elTabBtnEvents.setAttribute('aria-selected', tabName === 'events' ? 'true' : 'false');
    elTabBtnEvents.setAttribute('tabindex', tabName === 'events' ? '0' : '-1');
    elTabBtnWebhooks.classList.toggle('tab-active', tabName === 'webhooks');
    elTabBtnWebhooks.setAttribute('aria-selected', tabName === 'webhooks' ? 'true' : 'false');
    elTabBtnWebhooks.setAttribute('tabindex', tabName === 'webhooks' ? '0' : '-1');

    // Update tab panels
    elTabPanelPlayers.hidden = tabName !== 'players';
    elTabPanelMatches.hidden = tabName !== 'matches';
    elTabPanelStreamers.hidden = tabName !== 'streamers';
    elTabPanelEvents.hidden = tabName !== 'events';
    elTabPanelWebhooks.hidden = tabName !== 'webhooks';

    // Lazy load data if currently empty
    if (tabName === 'players' && state.players.length === 0) {
      loadPlayers();
    } else if (tabName === 'matches' && state.matches.live.length === 0 && state.matches.past.length === 0) {
      loadMatches();
    } else if (tabName === 'streamers' && state.streamers.length === 0) {
      loadStreamers();
    } else if (tabName === 'events' && state.events.length === 0) {
      loadEvents();
    } else if (tabName === 'webhooks' && state.webhookRequests.length === 0) {
      loadWebhookRequests();
    }
  }

  function handleTabKeydown(e) {
    const tabBtns = [elTabBtnPlayers, elTabBtnMatches, elTabBtnStreamers, elTabBtnEvents, elTabBtnWebhooks];
    const currentIndex = tabBtns.indexOf(document.activeElement);
    if (currentIndex === -1) return;

    let targetIndex = -1;
    if (e.key === 'ArrowRight') {
      targetIndex = (currentIndex + 1) % tabBtns.length;
    } else if (e.key === 'ArrowLeft') {
      targetIndex = (currentIndex - 1 + tabBtns.length) % tabBtns.length;
    } else if (e.key === 'Home') {
      targetIndex = 0;
    } else if (e.key === 'End') {
      targetIndex = tabBtns.length - 1;
    }

    if (targetIndex !== -1) {
      e.preventDefault();
      tabBtns[targetIndex].focus();
      tabBtns[targetIndex].click();
    }
  }

  // --- Data Loading Handlers ---

  async function loadStatus() {
    try {
      const data = await apiFetch(ENDPOINTS.status);
      state.status = data || {};

      if (state.status.database === 'ok' || state.status.database === true) {
        elDbBadge.textContent = 'Verbunden';
        elDbBadge.className = 'status-pill status-ok';
      } else {
        elDbBadge.textContent = String(state.status.database || 'Fehler');
        elDbBadge.className = 'status-pill status-error';
      }

      const activeCount = typeof state.status.activeMatches === 'number'
        ? state.status.activeMatches
        : 0;
      elActiveMatches.textContent = String(activeCount);
    } catch (err) {
      elDbBadge.textContent = 'Getrennt';
      elDbBadge.className = 'status-pill status-error';
      elActiveMatches.textContent = '-';
    }
  }

  async function loadPlayers() {
    elPlayersLoading.hidden = false;
    elPlayersEmpty.hidden = true;
    elPlayersTableWrapper.hidden = true;

    try {
      const players = await apiFetch(ENDPOINTS.players);
      state.players = Array.isArray(players) ? players : [];
      if (state.selectedPlayer) {
        const updated = state.players.find(p => p.id === state.selectedPlayer.id);
        if (updated) {
          state.selectedPlayer = updated;
          updatePlayerDetailHeader(updated);
        }
      }
      renderPlayersTable();
    } catch (err) {
      showNotification(`Fehler beim Laden der Spieler: ${err.message}`, 'error');
      renderPlayersTable();
    } finally {
      elPlayersLoading.hidden = true;
    }
  }

  async function loadPlayerMatches(isInitial = false) {
    if (!state.selectedPlayer || !state.selectedPlayer.id) return;
    const playerId = state.selectedPlayer.id;
    const requestSeq = ++playerHistoryRequestSeq;

    if (isInitial) {
      elPlayerMatchesLoading.hidden = false;
      elPlayerMatchesError.hidden = true;
      elPlayerMatchesEmpty.hidden = true;
      elPlayerMatchesTableWrapper.hidden = true;
      elBtnPlayerLoadMore.hidden = true;
      state.playerMatches = [];
      state.playerMatchesOffset = 0;
      state.playerMatchesTotal = 0;
    } else {
      elBtnPlayerLoadMore.disabled = true;
      elPlayerLoadMoreSpinner.hidden = false;
    }

    try {
      const offset = isInitial ? 0 : state.playerMatchesOffset;
      const data = await apiFetch(ENDPOINTS.playerMatches(playerId, 50, offset));

      if (requestSeq !== playerHistoryRequestSeq || !state.selectedPlayer || state.selectedPlayer.id !== playerId) return;
      const incomingMatches = data && Array.isArray(data.matches) ? data.matches : [];
      const total = data && typeof data.total === 'number'
        ? data.total
        : (isInitial ? incomingMatches.length : state.playerMatchesTotal);

      state.playerMatches = isInitial ? incomingMatches : state.playerMatches.concat(incomingMatches);
      state.playerMatchesTotal = total;
      state.playerMatchesOffset = state.playerMatches.length;

      renderPlayerMatchesTable();
    } catch (err) {
      if (requestSeq !== playerHistoryRequestSeq || !state.selectedPlayer || state.selectedPlayer.id !== playerId) return;
      if (isInitial) {
        elPlayerMatchesError.textContent = `Fehler beim Laden der Match-Historie: ${err.message}`;
        elPlayerMatchesError.hidden = false;
        elPlayerMatchesEmpty.hidden = true;
        elPlayerMatchesTableWrapper.hidden = true;
      } else {
        showNotification(`Fehler beim Nachladen der Matches: ${err.message}`, 'error');
        elBtnPlayerLoadMore.disabled = false;
      }
    } finally {
      if (requestSeq === playerHistoryRequestSeq) {
        elPlayerMatchesLoading.hidden = true;
        elPlayerLoadMoreSpinner.hidden = true;
      }
    }
  }

  async function loadMatches() {
    elMatchesLoading.hidden = false;
    elMatchesError.hidden = true;
    elMatchesLiveEmpty.hidden = true;
    elMatchesLiveTableWrapper.hidden = true;
    elMatchesPastEmpty.hidden = true;
    elMatchesPastTableWrapper.hidden = true;

    try {
      const data = await apiFetch(ENDPOINTS.matches);
      state.matches.live = data && Array.isArray(data.live) ? data.live : [];
      state.matches.past = data && Array.isArray(data.past) ? data.past : [];

      renderMatchesTables();
    } catch (err) {
      elMatchesError.textContent = `Fehler beim Laden der Matches: ${err.message}`;
      elMatchesError.hidden = false;
      renderMatchesTables();
    } finally {
      elMatchesLoading.hidden = true;
    }
  }

  async function loadEvents() {
    elEventsLoading.hidden = false;
    elEventsEmpty.hidden = true;
    elEventsTableWrapper.hidden = true;

    const limit = elEventLimitSelect.value || '25';
    try {
      const events = await apiFetch(`${ENDPOINTS.events}?limit=${encodeURIComponent(limit)}`);
      state.events = Array.isArray(events) ? events : [];

      state.lastEventsByStreamerId.clear();
      for (const evt of state.events) {
        if (evt.streamerId && !state.lastEventsByStreamerId.has(evt.streamerId)) {
          state.lastEventsByStreamerId.set(evt.streamerId, evt);
        }
      }

      renderEventsTable();
      renderStreamersTable();
    } catch (err) {
      showNotification(`Fehler beim Laden der Events: ${err.message}`, 'error');
    } finally {
      elEventsLoading.hidden = true;
    }
  }

  async function loadWebhookRequests() {
    elWebhooksLoading.hidden = false;
    elWebhooksEmpty.hidden = true;
    elWebhooksTableWrapper.hidden = true;
    try {
      const limit = elWebhookLimitSelect.value || '50';
      const requests = await apiFetch(`${ENDPOINTS.webhookRequests}?limit=${encodeURIComponent(limit)}`);
      state.webhookRequests = Array.isArray(requests) ? requests : [];
      renderWebhookRequests();
    } catch (err) {
      showNotification(`Fehler beim Laden der Webhooks: ${err.message}`, 'error');
    } finally {
      elWebhooksLoading.hidden = true;
    }
  }

  function renderWebhookRequests() {
    elWebhooksTbody.replaceChildren();
    if (state.webhookRequests.length === 0) {
      elWebhooksEmpty.hidden = false;
      elWebhooksTableWrapper.hidden = true;
      return;
    }
    elWebhooksEmpty.hidden = true;
    elWebhooksTableWrapper.hidden = false;
    for (const request of state.webhookRequests) {
      const row = createEl('tr');
      row.appendChild(createEl('td', { textContent: formatGermanDate(request.timestamp) }));
      row.appendChild(createEl('td', { textContent: request.method || 'POST' }));
      row.appendChild(createEl('td', { textContent: request.path || '/match' }));
      const detailsCell = createEl('td');
      const details = createEl('details');
      details.appendChild(createEl('summary', { textContent: 'Header und Body anzeigen' }));
      details.appendChild(createEl('h3', { textContent: 'Header' }));
      details.appendChild(createEl('pre', { className: 'webhook-payload', textContent: JSON.stringify(request.headers || {}, null, 2) }));
      details.appendChild(createEl('h3', { textContent: 'Body' }));
      let bodyText = request.body || '';
      try { bodyText = JSON.stringify(JSON.parse(bodyText), null, 2); } catch (_) { /* keep raw form/text body */ }
      details.appendChild(createEl('pre', { className: 'webhook-payload', textContent: bodyText || '(leer)' }));
      detailsCell.appendChild(details);
      row.appendChild(detailsCell);
      elWebhooksTbody.appendChild(row);
    }
  }

  async function loadStreamers() {
    elStreamersLoading.hidden = false;
    elStreamersEmpty.hidden = true;
    elStreamersTableWrapper.hidden = true;

    try {
      const streamers = await apiFetch(ENDPOINTS.streamers);
      state.streamers = Array.isArray(streamers) ? streamers : [];
      renderStreamersTable();
    } catch (err) {
      showNotification(`Fehler beim Laden der Streamer: ${err.message}`, 'error');
    } finally {
      elStreamersLoading.hidden = true;
    }
  }

  async function refreshAll() {
    elBtnRefreshAll.disabled = true;
    try {
      const promises = [
        loadStatus(),
        loadPlayers(),
        loadMatches(),
        loadStreamers(),
        loadEvents(),
        loadWebhookRequests()
      ];
      if (state.selectedPlayer && !elPlayersDetailContainer.hidden) {
        promises.push(loadPlayerMatches(true));
      }
      await Promise.allSettled(promises);
    } finally {
      elBtnRefreshAll.disabled = false;
    }
  }

  // --- Player Selection Handlers ---

  function selectPlayerDetail(player) {
    state.selectedPlayer = player;
    state.playerMatches = [];
    state.playerMatchesTotal = 0;
    state.playerMatchesOffset = 0;

    updatePlayerDetailHeader(player);

    // Switch view
    elPlayersListContainer.hidden = true;
    elPlayersDetailContainer.hidden = false;

    // Load complete history (first 50)
    loadPlayerMatches(true);
  }

  function updatePlayerDetailHeader(player) {
    // Set header details
    elPlayerDetailTitle.textContent = player.username
      ? `Match-Historie: ${player.username}`
      : 'Match-Historie';
    elPlayerDetailId.textContent = player.id || '-';
    elPlayerDetailGamecount.textContent = typeof player.gameCount === 'number'
      ? `${player.gameCount} Spiele`
      : (player.gameCount ? `${player.gameCount} Spiele` : '0 Spiele');
    elPlayerDetailLastWebhook.textContent = player.lastWebhookAt
      ? formatGermanDate(player.lastWebhookAt)
      : 'Nie';

  }

  function closePlayerDetail() {
    ++playerHistoryRequestSeq;
    state.selectedPlayer = null;
    state.playerMatches = [];
    state.playerMatchesTotal = 0;
    state.playerMatchesOffset = 0;

    elPlayersDetailContainer.hidden = true;
    elPlayersListContainer.hidden = false;
  }

  // --- Rendering Functions ---

  function renderPlayersTable() {
    while (elPlayersTbody.firstChild) {
      elPlayersTbody.removeChild(elPlayersTbody.firstChild);
    }

    if (state.players.length === 0) {
      elPlayersEmpty.hidden = false;
      elPlayersTableWrapper.hidden = true;
      return;
    }

    elPlayersEmpty.hidden = true;
    elPlayersTableWrapper.hidden = false;

    state.players.forEach(player => {
      const tr = createEl('tr');

      // 1. Username (Accessible button to open detail)
      const tdUsername = createEl('td');
      const btnOpenDetail = createEl('button', {
        type: 'button',
        className: 'btn-link fw-bold',
        'aria-label': `Match-Historie von ${player.username || player.id} öffnen`,
        textContent: player.username || 'Unbekannt',
        onClick: () => selectPlayerDetail(player)
      });
      tdUsername.appendChild(btnOpenDetail);

      // 2. Faceit Player ID
      const tdId = createEl('td');
      const codeId = createEl('code', {
        className: 'mono-uuid',
        title: 'Faceit Player UUID',
        textContent: player.id || '-'
      });
      tdId.appendChild(codeId);

      // 3. Game count
      const tdCount = createEl('td', {
        textContent: typeof player.gameCount === 'number'
          ? String(player.gameCount)
          : (player.gameCount || '0')
      });

      // 4. Last Webhook
      const tdLastWebhook = createEl('td', {
        textContent: player.lastWebhookAt ? formatGermanDate(player.lastWebhookAt) : 'Nie'
      });

      // 5. Actions
      const tdActions = createEl('td', { className: 'text-right' });
      const btnView = createEl('button', {
        type: 'button',
        className: 'btn btn-secondary btn-sm',
        textContent: 'Historie',
        'aria-label': `Match-Historie von ${player.username || player.id} anzeigen`,
        onClick: () => selectPlayerDetail(player)
      });
      tdActions.appendChild(btnView);

      tr.appendChild(tdUsername);
      tr.appendChild(tdId);
      tr.appendChild(tdCount);
      tr.appendChild(tdLastWebhook);
      tr.appendChild(tdActions);

      elPlayersTbody.appendChild(tr);
    });
  }

  function createMatchRow(match, options = {}) {
    const tr = createEl('tr');

    // 1. Zeitpunkt (chronological time)
    const timeVal = match.lastSeenAt || match.firstSeenAt || (match.payload && match.payload.created_at);
    const tdTime = createEl('td', {
      textContent: formatGermanDate(timeVal)
    });
    tr.appendChild(tdTime);

    // Optional: Player (username + id) for general Matches tab
    if (options.showPlayer) {
      const tdPlayer = createEl('td');
      if (match.username) {
        tdPlayer.appendChild(createEl('div', { className: 'fw-bold', textContent: match.username }));
      }
      if (match.playerId) {
        tdPlayer.appendChild(createEl('code', { className: 'mono-uuid', textContent: match.playerId }));
      } else if (!match.username) {
        tdPlayer.textContent = '-';
      }
      tr.appendChild(tdPlayer);
    }

    // 2. Status
    const tdStatus = createEl('td');
    tdStatus.appendChild(formatMatchStatusBadge(match.status));
    tr.appendChild(tdStatus);

    // 3. Match-ID (with Faceit Link)
    const tdMatchId = createEl('td');
    const rawMatchId = match.matchId || (match.payload && match.payload.id);
    if (rawMatchId) {
      const link = createEl('a', {
        href: `https://www.faceit.com/en/cs2/room/${encodeURIComponent(rawMatchId)}`,
        target: '_blank',
        rel: 'noopener noreferrer',
        className: 'link-target mono-uuid',
        title: 'FACEIT Matchroom öffnen',
        textContent: rawMatchId
      });
      tdMatchId.appendChild(link);
    } else {
      tdMatchId.textContent = '-';
    }
    tr.appendChild(tdMatchId);

    // 4. Teams & Avg Elo
    const tdMatchup = createEl('td', {
      textContent: formatMatchupText(match)
    });
    tr.appendChild(tdMatchup);

    // 5. Map
    const tdMap = createEl('td', {
      textContent: formatMapText(match)
    });
    tr.appendChild(tdMap);

    // Optional: Event for player match history
    if (options.showEvent) {
      const tdEvent = createEl('td', {
        textContent: match.event || (match.payload && match.payload.event) || '-'
      });
      tr.appendChild(tdEvent);
    }

    return tr;
  }

  function renderPlayerMatchesTable() {
    while (elPlayerMatchesTbody.firstChild) {
      elPlayerMatchesTbody.removeChild(elPlayerMatchesTbody.firstChild);
    }

    if (state.playerMatches.length === 0) {
      elPlayerMatchesEmpty.hidden = false;
      elPlayerMatchesTableWrapper.hidden = true;
      return;
    }

    elPlayerMatchesEmpty.hidden = true;
    elPlayerMatchesTableWrapper.hidden = false;

    state.playerMatches.forEach(match => {
      const tr = createMatchRow(match, { showEvent: true });
      elPlayerMatchesTbody.appendChild(tr);
    });

    // Update pagination controls
    const loadedCount = state.playerMatches.length;
    const totalCount = state.playerMatchesTotal;
    elPlayerMatchesPaginationInfo.textContent = `${loadedCount} von ${totalCount} Matches angezeigt`;

    if (loadedCount < totalCount) {
      elBtnPlayerLoadMore.hidden = false;
      elBtnPlayerLoadMore.disabled = false;
    } else {
      elBtnPlayerLoadMore.hidden = true;
    }
  }

  function renderMatchesTables() {
    // 1. Live Matches
    while (elMatchesLiveTbody.firstChild) {
      elMatchesLiveTbody.removeChild(elMatchesLiveTbody.firstChild);
    }

    if (state.matches.live.length === 0) {
      elMatchesLiveEmpty.hidden = false;
      elMatchesLiveTableWrapper.hidden = true;
    } else {
      elMatchesLiveEmpty.hidden = true;
      elMatchesLiveTableWrapper.hidden = false;
      state.matches.live.forEach(match => {
        const tr = createMatchRow(match, { showPlayer: true });
        elMatchesLiveTbody.appendChild(tr);
      });
    }

    // 2. Past Matches (last 20)
    while (elMatchesPastTbody.firstChild) {
      elMatchesPastTbody.removeChild(elMatchesPastTbody.firstChild);
    }

    if (state.matches.past.length === 0) {
      elMatchesPastEmpty.hidden = false;
      elMatchesPastTableWrapper.hidden = true;
    } else {
      elMatchesPastEmpty.hidden = true;
      elMatchesPastTableWrapper.hidden = false;
      state.matches.past.forEach(match => {
        const tr = createMatchRow(match, { showPlayer: true });
        elMatchesPastTbody.appendChild(tr);
      });
    }
  }

  function renderStreamersTable() {
    while (elStreamersTbody.firstChild) {
      elStreamersTbody.removeChild(elStreamersTbody.firstChild);
    }

    if (state.streamers.length === 0) {
      elStreamersEmpty.hidden = false;
      elStreamersTableWrapper.hidden = true;
      return;
    }

    elStreamersEmpty.hidden = true;
    elStreamersTableWrapper.hidden = false;

    state.streamers.forEach(streamer => {
      const tr = createEl('tr');

      // 1. Status Badge
      const tdStatus = createEl('td');
      const isEnabled = streamer.enabled !== false;
      const statusBadge = createEl('span', {
        className: isEnabled ? 'badge badge-active' : 'badge badge-inactive',
        textContent: isEnabled ? 'Aktiv' : 'Inaktiv'
      });
      tdStatus.appendChild(statusBadge);

      // 2. Name
      const tdName = createEl('td', {
        textContent: streamer.name || 'Ohne Name'
      });

      // 3. Faceit Player ID
      const tdId = createEl('td');
      const codeId = createEl('code', {
        className: 'mono-uuid',
        title: 'Klicken zum Markieren',
        textContent: streamer.id || '-'
      });
      tdId.appendChild(codeId);

      // 4. Lobby-URL (Match-Link)
      const tdUrl = createEl('td');
      if (isValidHttpUrl(streamer.url)) {
        const link = createEl('a', {
          href: streamer.url,
          target: '_blank',
          rel: 'noopener noreferrer',
          className: 'link-target',
          textContent: streamer.url
        });
        tdUrl.appendChild(link);
      } else {
        tdUrl.appendChild(document.createTextNode(streamer.url || '-'));
      }

      // 5. Last Event in current slice
      const tdLastAccess = createEl('td');
      const lastEvt = state.lastEventsByStreamerId.get(streamer.id);
      if (lastEvt && lastEvt.createdAt) {
        const container = createEl('div');
        const timeText = createEl('div', { textContent: formatGermanDate(lastEvt.createdAt) });
        const outcomeBadge = formatOutcomeBadge(lastEvt.outcome);
        container.appendChild(timeText);
        container.appendChild(outcomeBadge);
        tdLastAccess.appendChild(container);
      } else {
        tdLastAccess.appendChild(createEl('span', {
          className: 'text-muted',
          textContent: 'Kein Aufruf in aktueller Liste'
        }));
      }

      // 6. Actions: Edit and Delete
      const tdActions = createEl('td', { className: 'text-right' });
      const actionsWrapper = createEl('div', { className: 'btn-actions' });

      const btnEdit = createEl('button', {
        type: 'button',
        className: 'btn btn-secondary btn-sm',
        textContent: 'Bearbeiten',
        onClick: () => openEditStreamerModal(streamer)
      });

      const btnDelete = createEl('button', {
        type: 'button',
        className: 'btn btn-danger btn-sm',
        textContent: 'Löschen',
        onClick: () => openDeleteStreamerModal(streamer)
      });

      actionsWrapper.appendChild(btnEdit);
      actionsWrapper.appendChild(btnDelete);
      tdActions.appendChild(actionsWrapper);

      tr.appendChild(tdStatus);
      tr.appendChild(tdName);
      tr.appendChild(tdId);
      tr.appendChild(tdUrl);
      tr.appendChild(tdLastAccess);
      tr.appendChild(tdActions);

      elStreamersTbody.appendChild(tr);
    });
  }

  function renderEventsTable() {
    while (elEventsTbody.firstChild) {
      elEventsTbody.removeChild(elEventsTbody.firstChild);
    }

    if (state.events.length === 0) {
      elEventsEmpty.hidden = false;
      elEventsTableWrapper.hidden = true;
      return;
    }

    elEventsEmpty.hidden = true;
    elEventsTableWrapper.hidden = false;

    state.events.forEach(evt => {
      const tr = createEl('tr');

      // 1. Timestamp
      const tdDate = createEl('td', {
        textContent: formatGermanDate(evt.createdAt)
      });

      // 2. Streamer Name / ID
      const tdStreamer = createEl('td');
      if (evt.streamerName) {
        tdStreamer.appendChild(createEl('div', {
          className: 'fw-bold',
          textContent: evt.streamerName
        }));
      }
      if (evt.streamerId) {
        tdStreamer.appendChild(createEl('code', {
          className: 'mono-uuid',
          textContent: evt.streamerId
        }));
      } else if (!evt.streamerName) {
        tdStreamer.textContent = '-';
      }

      // 3. Match ID
      const tdMatch = createEl('td', {
        textContent: evt.matchId || '-'
      });

      // 4. Target
      const tdTarget = createEl('td');
      if (isValidHttpUrl(evt.target)) {
        tdTarget.appendChild(createEl('a', {
          href: evt.target,
          target: '_blank',
          rel: 'noopener noreferrer',
          className: 'link-target',
          textContent: evt.target
        }));
      } else {
        tdTarget.textContent = evt.target || '-';
      }

      // 5. Outcome Badge
      const tdOutcome = createEl('td');
      tdOutcome.appendChild(formatOutcomeBadge(evt.outcome));

      // 6. Request ID
      const tdRequestId = createEl('td');
      if (evt.requestId) {
        tdRequestId.appendChild(createEl('code', {
          className: 'mono-uuid',
          textContent: evt.requestId
        }));
      } else {
        tdRequestId.textContent = '-';
      }

      tr.appendChild(tdDate);
      tr.appendChild(tdStreamer);
      tr.appendChild(tdMatch);
      tr.appendChild(tdTarget);
      tr.appendChild(tdOutcome);
      tr.appendChild(tdRequestId);

      elEventsTbody.appendChild(tr);
    });
  }

  // --- Modal Operations ---

  function resetFormValidation() {
    elModalError.hidden = true;
    elModalError.textContent = '';
    elErrorId.hidden = true;
    elErrorId.textContent = '';
    elErrorName.hidden = true;
    elErrorName.textContent = '';
    elErrorUrl.hidden = true;
    elErrorUrl.textContent = '';
  }

  function openCreateStreamerModal() {
    resetFormValidation();
    elFormMode.value = 'create';
    elModalTitle.textContent = 'Neuen Streamer registrieren';

    elInputStreamerId.value = '';
    elInputStreamerId.readOnly = false;
    elInputStreamerId.disabled = false;

    elInputStreamerName.value = '';
    elInputStreamerUrl.value = '';
    elInputStreamerEnabled.checked = true;

    elBtnModalSave.disabled = false;
    elBtnModalSpinner.hidden = true;

    elStreamerModal.showModal();
    elInputStreamerId.focus();
  }

  function openEditStreamerModal(streamer) {
    resetFormValidation();
    elFormMode.value = 'edit';
    elModalTitle.textContent = `Streamer bearbeiten: ${streamer.name || streamer.id}`;

    elInputStreamerId.value = streamer.id || '';
    elInputStreamerId.readOnly = true;

    elInputStreamerName.value = streamer.name || '';
    elInputStreamerUrl.value = streamer.url || '';
    elInputStreamerEnabled.checked = streamer.enabled !== false;

    elBtnModalSave.disabled = false;
    elBtnModalSpinner.hidden = true;

    elStreamerModal.showModal();
    elInputStreamerName.focus();
  }

  function closeStreamerModal() {
    elStreamerModal.close();
  }

  function openDeleteStreamerModal(streamer) {
    state.streamerToDelete = streamer;
    elDeleteModalMessage.textContent =
      `Möchten Sie den Streamer "${streamer.name || 'Ohne Name'}" (ID: ${streamer.id}) wirklich löschen?`;
    elBtnDeleteModalConfirm.disabled = false;
    elBtnDeleteSpinner.hidden = true;
    elDeleteModal.showModal();
  }

  function closeDeleteModal() {
    state.streamerToDelete = null;
    elDeleteModal.close();
  }

  // --- Form Validation & Submission ---

  function validateStreamerForm() {
    resetFormValidation();
    let isValid = true;

    const rawId = elInputStreamerId.value.trim().toLowerCase();
    const rawName = elInputStreamerName.value.trim();
    const rawUrl = elInputStreamerUrl.value.trim();

    // Validate Faceit Player UUID
    if (!rawId) {
      elErrorId.textContent = 'Die Faceit Player ID (UUID) ist erforderlich.';
      elErrorId.hidden = false;
      isValid = false;
    } else if (!UUID_REGEX.test(rawId)) {
      elErrorId.textContent =
        'Ungültiges Format: Die UUID muss 36 Zeichen lang und kleingeschrieben sein (z. B. f335cfd1-3a92-4365-9b25-2f0e82a6052f).';
      elErrorId.hidden = false;
      isValid = false;
    }

    // Validate Name
    if (!rawName) {
      elErrorName.textContent = 'Der Name des Streamers ist erforderlich.';
      elErrorName.hidden = false;
      isValid = false;
    } else if (rawName.length > 64) {
      elErrorName.textContent = 'Der Name darf maximal 64 Zeichen umfassen.';
      elErrorName.hidden = false;
      isValid = false;
    }

    // Validate Lobby URL
    if (!rawUrl) {
      elErrorUrl.textContent = 'Die Lobby-URL ist erforderlich.';
      elErrorUrl.hidden = false;
      isValid = false;
    } else if (!isValidHttpUrl(rawUrl)) {
      elErrorUrl.textContent =
        'Ungültige URL: Muss mit http:// oder https:// beginnen (z. B. https://www.faceit.com/...).';
      elErrorUrl.hidden = false;
      isValid = false;
    }

    return isValid;
  }

  async function handleStreamerFormSubmit(evt) {
    evt.preventDefault();
    if (!validateStreamerForm()) return;

    const mode = elFormMode.value;
    const streamerData = {
      id: elInputStreamerId.value.trim().toLowerCase(),
      name: elInputStreamerName.value.trim(),
      url: elInputStreamerUrl.value.trim(),
      enabled: elInputStreamerEnabled.checked
    };

    elBtnModalSave.disabled = true;
    elBtnModalSpinner.hidden = false;

    try {
      if (mode === 'create') {
        await apiFetch(ENDPOINTS.streamers, {
          method: 'POST',
          body: JSON.stringify(streamerData)
        });
        showNotification(`Streamer "${streamerData.name}" erfolgreich hinzugefügt.`, 'success');
      } else {
        await apiFetch(`${ENDPOINTS.streamers}/${encodeURIComponent(streamerData.id)}`, {
          method: 'PUT',
          body: JSON.stringify(streamerData)
        });
        showNotification(`Streamer "${streamerData.name}" erfolgreich aktualisiert.`, 'success');
      }

      closeStreamerModal();
      await loadStreamers();
    } catch (err) {
      elModalError.textContent = `Fehler beim Speichern: ${err.message}`;
      elModalError.hidden = false;
    } finally {
      elBtnModalSave.disabled = false;
      elBtnModalSpinner.hidden = true;
    }
  }

  async function handleDeleteConfirm() {
    if (!state.streamerToDelete) return;
    const { id, name } = state.streamerToDelete;

    elBtnDeleteModalConfirm.disabled = true;
    elBtnDeleteSpinner.hidden = false;

    try {
      await apiFetch(`${ENDPOINTS.streamers}/${encodeURIComponent(id)}`, {
        method: 'DELETE'
      });
      showNotification(`Streamer "${name || id}" wurde gelöscht.`, 'success');
      closeDeleteModal();
      await loadStreamers();
    } catch (err) {
      showNotification(`Fehler beim Löschen des Streamers: ${err.message}`, 'error');
    } finally {
      elBtnDeleteModalConfirm.disabled = false;
      elBtnDeleteSpinner.hidden = true;
    }
  }

  // --- Event Listeners Initialization ---

  function initEventListeners() {
    // Top Bar Refresh
    elBtnRefreshAll.addEventListener('click', refreshAll);

    // Tab Navigation
    elTabBtnPlayers.addEventListener('click', () => switchTab('players'));
    elTabBtnMatches.addEventListener('click', () => switchTab('matches'));
    elTabBtnStreamers.addEventListener('click', () => switchTab('streamers'));
    elTabBtnEvents.addEventListener('click', () => switchTab('events'));
    elTabBtnWebhooks.addEventListener('click', () => switchTab('webhooks'));

    const tabNav = document.querySelector('.tab-nav');
    if (tabNav) {
      tabNav.addEventListener('keydown', handleTabKeydown);
    }

    // Players listeners
    elBtnRefreshPlayers.addEventListener('click', loadPlayers);
    elBtnPlayerDetailBack.addEventListener('click', closePlayerDetail);
    elBtnRefreshPlayerMatches.addEventListener('click', () => loadPlayerMatches(true));
    elBtnPlayerLoadMore.addEventListener('click', () => loadPlayerMatches(false));

    // Matches listeners
    elBtnRefreshMatches.addEventListener('click', loadMatches);

    // Events listeners
    elBtnRefreshEvents.addEventListener('click', loadEvents);
    elEventLimitSelect.addEventListener('change', loadEvents);
    elBtnRefreshWebhooks.addEventListener('click', loadWebhookRequests);
    elWebhookLimitSelect.addEventListener('change', loadWebhookRequests);

    // Streamer CRUD listeners
    elBtnOpenCreateModal.addEventListener('click', openCreateStreamerModal);
    elBtnEmptyAddStreamer.addEventListener('click', openCreateStreamerModal);
    elBtnModalClose.addEventListener('click', closeStreamerModal);
    elBtnModalCancel.addEventListener('click', closeStreamerModal);

    elStreamerForm.addEventListener('submit', handleStreamerFormSubmit);

    elBtnDeleteModalClose.addEventListener('click', closeDeleteModal);
    elBtnDeleteModalCancel.addEventListener('click', closeDeleteModal);
    elBtnDeleteModalConfirm.addEventListener('click', handleDeleteConfirm);

    elStreamerModal.addEventListener('click', e => {
      const rect = elStreamerModal.getBoundingClientRect();
      if (
        e.clientX < rect.left || e.clientX > rect.right ||
        e.clientY < rect.top || e.clientY > rect.bottom
      ) {
        closeStreamerModal();
      }
    });

    elDeleteModal.addEventListener('click', e => {
      const rect = elDeleteModal.getBoundingClientRect();
      if (
        e.clientX < rect.left || e.clientX > rect.right ||
        e.clientY < rect.top || e.clientY > rect.bottom
      ) {
        closeDeleteModal();
      }
    });
  }

  // --- Application Bootstrap ---

  document.addEventListener('DOMContentLoaded', () => {
    initEventListeners();
    refreshAll();
  });

})();
