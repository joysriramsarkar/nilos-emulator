// public/device.js — নীল ওএস (NilOS) স্টুডিও ডিভাইস এমুলেটর কন্ট্রোলার

// ─── গ্লোবাল ওএস স্টেট ───────────────────────────────────────────────
const osState = {
    isLocked: false,
    activeApp: 'notes', // 'notes', 'calc', 'pos', 'music', 'shell', 'store', 'files', 'settings', 'home'
    recentApps: ['notes', 'pos', 'calc', 'music'],
    currentProfile: 'phone', // 'phone', 'tablet', 'pos', 'desktop'
    isRotated: false,
    theme: 'theme-dark',
    focusedInput: null,

    // Notes Data
    notes: [],
    activeNoteId: null,
    activeCategory: 'all',
    searchQuery: '',

    // Calculator State
    calcExpr: '',
    calcCurrent: '0',
    calcHistory: [],

    // POS State
    posCatalog: [],
    posCart: [],

    // Music State
    musicTracks: [],
    currentTrackIndex: 0,
    isPlaying: false,
    audioTimer: null,

    // NilBus Log
    busLogs: [],

    // Hardware State
    batteryLevel: 98,
    isCharging: true,
    networkSpeed: '5g',
    volume: 75,
    brightness: 85
};

// ─── অ্যাপ্লিকেশন ইনিশিয়ালাইজেশন ──────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
    initClock();
    initDeviceProfiles();
    initHardwareButtons();
    initTouchEvents();
    initAppManager();
    initDevTools();
    initFpsChart();
    initAudioVisualizer();
    initQEMUFramebuffer();
    initNDBDebugger();
    loadExtensionsList();

    // ডেটা লোডিং
    loadNotesData();
    loadPOSCatalog();
    loadMusicTracks();
    loadStorePackages();
    loadFilesList();

    // প্রাথমিক নীলবাস ইভেন্ট লগ
    logNilBus('org.onuron.Kernel', 'BootCompleted', { kernel: '0.1.0-alpha', profile: osState.currentProfile });
    logNilBus('org.nilang.runtime', 'VMReady', { jit: true, arch: 'x86_64', gc: 'compacting' });
});

// ─── ১. ঘড়ি ও লাইভ টাইম ──────────────────────────────────────────────
function initClock() {
    function update() {
        const now = new Date();
        let hours = now.getHours();
        let minutes = now.getMinutes();
        const hStr = hours < 10 ? '0' + hours : hours;
        const mStr = minutes < 10 ? '0' + minutes : minutes;
        const timeStr = `${hStr}:${mStr}`;

        const clockEl = document.getElementById('clockText');
        const lockClockEl = document.getElementById('lockClock');
        if (clockEl) clockEl.textContent = timeStr;
        if (lockClockEl) lockClockEl.textContent = timeStr;
    }
    update();
    setInterval(update, 1000);
}

// ─── ২. ডিভাইস প্রোফাইল হ্যান্ডলার ────────────────────────────────────
function initDeviceProfiles() {
    const tabs = document.querySelectorAll('.profile-tab');
    const wrapper = document.getElementById('deviceWrapper');

    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            const profile = tab.dataset.profile;
            osState.currentProfile = profile;
            wrapper.className = `device-frame-wrapper profile-${profile}`;

            triggerIslandNotice(`📱 প্রোফাইল পরিবর্তন: ${tab.textContent.trim()}`);
            logNilBus('org.onuron.DisplayManager', 'ProfileChanged', { profile, width: wrapper.offsetWidth, height: wrapper.offsetHeight });
        });
    });
}

// ─── ৩. হার্ডওয়্যার বাটন ও সেন্সর কন্ট্রোলস ─────────────────────────────
function initHardwareButtons() {
    // পাওয়ার বাটন
    document.getElementById('hwPower').addEventListener('click', toggleLockScreen);
    document.getElementById('btnLockToggle').addEventListener('click', toggleLockScreen);

    // রোটেট বাটন
    document.getElementById('btnRotate').addEventListener('click', () => {
        osState.isRotated = !osState.isRotated;
        const body = document.getElementById('phoneBody');
        if (osState.isRotated) {
            body.style.transform = 'rotate(90deg) scale(0.85)';
            triggerIslandNotice("🔄 ল্যান্ডস্কেপ মোড (Landscape)");
        } else {
            body.style.transform = 'none';
            triggerIslandNotice("📱 পোর্ট্রেট মোড (Portrait)");
        }
    });

    // থিম সুইচ
    document.getElementById('btnThemeToggle').addEventListener('click', cycleTheme);
    const themeSelect = document.getElementById('settingsThemeSelect');
    if (themeSelect) {
        themeSelect.addEventListener('change', (e) => applyTheme(e.target.value));
    }

    // কীবোর্ড টগল
    document.getElementById('btnKeyboardToggle').addEventListener('click', () => {
        document.getElementById('virtualKeyboard').classList.toggle('hidden');
    });

    document.getElementById('hideKeyboardBtn').addEventListener('click', () => {
        document.getElementById('virtualKeyboard').classList.add('hidden');
    });

    // কন্ট্রোল সেন্টার টগল
    document.getElementById('btnControlCenter').addEventListener('click', () => {
        document.getElementById('controlCenter').classList.toggle('hidden');
    });
    document.getElementById('btnCloseCC').addEventListener('click', () => {
        document.getElementById('controlCenter').classList.add('hidden');
    });

    // ওএস রিস্টার্ট
    document.getElementById('btnRestartOS').addEventListener('click', () => {
        const screen = document.getElementById('touchScreen');
        screen.style.opacity = '0';
        triggerIslandNotice("⚡ নীল ওএস রিস্টার্ট হচ্ছে...");
        logNilBus('org.onuron.Kernel', 'RebootRequest', {});
        setTimeout(() => {
            screen.style.opacity = '1';
            showLockScreen();
            logNilBus('org.onuron.Kernel', 'RebootCompleted', {});
        }, 600);
    });

    // স্ক্রিনশট
    document.getElementById('btnScreenshot').addEventListener('click', () => {
        triggerIslandNotice("📸 স্ক্রিনশট গ্যালারিতে সংরক্ষিত");
        logNilBus('org.onuron.MediaService', 'ScreenshotSaved', { path: '/data/screenshots/snap-01.png' });
    });

    // টাস্ক সুইচার
    document.getElementById('btnAppSwitcher').addEventListener('click', toggleAppSwitcher);

    // ভলিউম ও অ্যাকশন বাটন
    document.getElementById('hwVolUp').addEventListener('click', () => {
        osState.volume = Math.min(100, osState.volume + 5);
        triggerIslandNotice(`🔊 ভলিউম: ${osState.volume}%`);
    });
    document.getElementById('hwVolDown').addEventListener('click', () => {
        osState.volume = Math.max(0, osState.volume - 5);
        triggerIslandNotice(`🔉 ভলিউম: ${osState.volume}%`);
    });
    document.getElementById('hwActionBtn').addEventListener('click', () => {
        triggerIslandNotice("⚡ অ্যাকশন বাটন: কুইক নোট সক্রিয়");
        launchApp('notes');
        createNewNoteMobile();
    });

    // সেন্সর স্লাইডারস
    document.getElementById('batterySlider').addEventListener('input', (e) => {
        const val = e.target.value;
        osState.batteryLevel = parseInt(val, 10);
        document.getElementById('batteryValText').textContent = val + '%';
        document.getElementById('statusBatteryPct').textContent = val + '%';
        const batIcon = document.getElementById('statusBatteryIcon');
        batIcon.textContent = val < 20 ? '🪫' : '🔋';
    });

    document.getElementById('networkSelect').addEventListener('change', (e) => {
        const net = e.target.value;
        osState.networkSpeed = net;
        const carrierEl = document.getElementById('statusCarrier');
        if (net === 'offline') {
            carrierEl.textContent = 'অফলাইন';
            document.getElementById('statusWifi').textContent = '❌';
        } else {
            carrierEl.textContent = `Onuron ${net.toUpperCase()}`;
            document.getElementById('statusWifi').textContent = '📶';
        }
        triggerIslandNotice(`📶 নেটওয়ার্ক: ${carrierEl.textContent}`);
        logNilBus('org.onuron.NetworkService', 'QoSChanged', { network: net });
    });

    // বারকোড ও পেপার টেস্ট
    document.getElementById('btnSimulateScan').addEventListener('click', () => {
        triggerBarcodeScanAnimation("8901030012345");
    });
    document.getElementById('btnSimulatePrint').addEventListener('click', () => {
        simulateThermalPrintReceipt();
    });

    // QEMU বুট লঞ্চার
    document.getElementById('btnLaunchQEMU').addEventListener('click', async () => {
        triggerIslandNotice("🚀 QEMU x86_64 ভার্চুয়াল মেশিন শুরু হচ্ছে...");
        try {
            const res = await fetch('/api/qemu/launch');
            const data = await res.json();
            if (data.success) {
                alert(`🎉 QEMU ভার্চুয়াল মেশিন সফলভাবে বুট হয়েছে!\nPID: ${data.pid}\n\nউইন্ডোজে QEMU GUI উইন্ডো প্রদর্শিত হবে।`);
                logNilBus('org.onuron.QEMUBridge', 'VMStarted', { pid: data.pid });
            } else {
                alert(`⚠️ QEMU চালানো যায়নি: ${data.error}`);
            }
        } catch (e) {
            alert("⚠️ QEMU এমুলেটর বুট কমান্ড পাঠানো হয়েছে!");
        }
    });

    // ডেভটুলস টগল
    document.getElementById('btnDevToolsToggle').addEventListener('click', () => {
        const dt = document.getElementById('emulatorDevtools');
        dt.classList.toggle('hidden');
    });

    // রিসিট ক্লোজ
    document.getElementById('btnCloseReceipt').addEventListener('click', () => {
        document.getElementById('thermalReceiptPaper').classList.add('hidden');
    });
}

// ─── ৪. টাচ ইভেন্টস ও স্ক্রিন নেভিগেশন ────────────────────────────────
function initTouchEvents() {
    // লকস্ক্রিন আনলক
    document.getElementById('lockSwipeArea').addEventListener('click', unlockScreen);
    document.getElementById('lockScreen').addEventListener('click', unlockScreen);

    // নোট অ্যাপ ইভেন্টস
    document.getElementById('mCategoryPills').addEventListener('click', (e) => {
        const pill = e.target.closest('.cat-pill');
        if (!pill) return;
        document.querySelectorAll('.cat-pill').forEach(p => p.classList.remove('active'));
        pill.classList.add('active');
        osState.activeCategory = pill.dataset.cat;
        renderNotesListMobile();
    });

    document.getElementById('mSearchBtn').addEventListener('click', () => {
        const bar = document.getElementById('mSearchBar');
        bar.classList.toggle('hidden');
        if (!bar.classList.contains('hidden')) {
            document.getElementById('mSearchInput').focus();
        }
    });

    document.getElementById('mCloseSearchBtn').addEventListener('click', () => {
        document.getElementById('mSearchBar').classList.add('hidden');
        document.getElementById('mSearchInput').value = '';
        osState.searchQuery = '';
        renderNotesListMobile();
    });

    document.getElementById('mSearchInput').addEventListener('input', (e) => {
        osState.searchQuery = e.target.value.trim().toLowerCase();
        renderNotesListMobile();
    });

    document.getElementById('mNewNoteBtn').addEventListener('click', createNewNoteMobile);
    document.getElementById('mFabNewBtn').addEventListener('click', createNewNoteMobile);

    document.getElementById('mBackToListBtn').addEventListener('click', () => {
        document.getElementById('mEditorView').classList.add('hidden');
        document.getElementById('mNotesListView').classList.remove('hidden');
        document.getElementById('mFabNewBtn').classList.remove('hidden');
        renderNotesListMobile();
    });

    // এডিটর ইনপুট হ্যান্ডলার্স
    const titleInput = document.getElementById('mTitleInput');
    const contentInput = document.getElementById('mContentInput');
    const tagsInput = document.getElementById('mTagsInput');
    const catSelect = document.getElementById('mCatSelect');

    titleInput.addEventListener('focus', () => { osState.focusedInput = titleInput; });
    contentInput.addEventListener('focus', () => { osState.focusedInput = contentInput; });
    tagsInput.addEventListener('focus', () => { osState.focusedInput = tagsInput; });

    titleInput.addEventListener('input', autoSaveMobileNote);
    contentInput.addEventListener('input', () => {
        autoSaveMobileNote();
        updateWordCountMobile();
    });
    tagsInput.addEventListener('change', autoSaveMobileNote);
    catSelect.addEventListener('change', autoSaveMobileNote);

    document.getElementById('mPinToggleBtn').addEventListener('click', togglePinMobile);
    document.getElementById('mFavToggleBtn').addEventListener('click', toggleFavMobile);
    document.getElementById('mDeleteBtn').addEventListener('click', deleteMobileNote);

    document.getElementById('mAddChkBtn').addEventListener('click', addCheckItemMobile);
    document.getElementById('mNewChkInput').addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            addCheckItemMobile();
        }
    });

    // স্ট্যাটাস মোডাল
    document.getElementById('mStatsBtn').addEventListener('click', openStatsModal);
    document.getElementById('closeStatsBtn').addEventListener('click', closeStatsModal);

    // ক্যালকুলেটর হিস্ট্রি টগল
    document.getElementById('btnCalcHistoryToggle').addEventListener('click', () => {
        document.getElementById('calcHistoryPanel').classList.toggle('hidden');
    });

    // পিওএস বারকোড ও চেকআউট
    document.getElementById('btnPosBarcodeScan').addEventListener('click', () => {
        triggerBarcodeScanAnimation("8901030012345");
    });
    document.getElementById('btnPosCheckout').addEventListener('click', handlePOSCheckoutClick);

    // টার্মিনাল ইনপুট
    const shellInput = document.getElementById('shellInput');
    shellInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            handleShellCommand(shellInput.value);
            shellInput.value = '';
        }
    });

    // সুইচার ক্লোজ
    document.getElementById('btnCloseSwitcher').addEventListener('click', toggleAppSwitcher);
}

// ─── ৫. লকস্ক্রিন ও হোমস্ক্রিন লজিক ────────────────────────────────────
function toggleLockScreen() {
    if (osState.isLocked) {
        unlockScreen();
    } else {
        showLockScreen();
    }
}

function showLockScreen() {
    osState.isLocked = true;
    document.getElementById('lockScreen').classList.remove('hidden');
    logNilBus('org.onuron.PowerManager', 'ScreenLocked', {});
}

function unlockScreen() {
    osState.isLocked = false;
    document.getElementById('lockScreen').classList.add('hidden');
    triggerIslandNotice("🔓 নীল ওএস আনলকড");
    logNilBus('org.onuron.PowerManager', 'ScreenUnlocked', { user: 'default' });
}

function handleGestureHome() {
    if (osState.activeApp === 'home') {
        launchApp('notes');
    } else {
        hideAllAppWindows();
        document.getElementById('homeScreen').classList.remove('hidden');
        osState.activeApp = 'home';
        logNilBus('org.onuron.WindowManager', 'HomeRequested', {});
    }
}

// ─── ৬. মাল্টি-অ্যাপ লঞ্চার ও লাইফসাইকেল ──────────────────────────────
function initAppManager() {
    // ডিফল্ট অ্যাপ নোটবুক
    launchApp('notes');
}

function hideAllAppWindows() {
    const windows = [
        'notesAppWindow', 'calcAppWindow', 'posAppWindow',
        'musicAppWindow', 'shellAppWindow', 'storeAppWindow',
        'filesAppWindow', 'settingsAppWindow', 'homeScreen'
    ];
    windows.forEach(id => {
        const el = document.getElementById(id);
        if (el) el.classList.add('hidden');
    });
}

function launchApp(appName) {
    hideAllAppWindows();
    document.getElementById('appSwitcher').classList.add('hidden');

    if (!osState.recentApps.includes(appName) && appName !== 'home') {
        osState.recentApps.unshift(appName);
    }

    osState.activeApp = appName;

    switch (appName) {
        case 'notes':
            document.getElementById('notesAppWindow').classList.remove('hidden');
            renderNotesListMobile();
            break;
        case 'calc':
            document.getElementById('calcAppWindow').classList.remove('hidden');
            break;
        case 'pos':
            document.getElementById('posAppWindow').classList.remove('hidden');
            renderPOSCatalog();
            break;
        case 'music':
            document.getElementById('musicAppWindow').classList.remove('hidden');
            renderMusicPlaylist();
            break;
        case 'shell':
            document.getElementById('shellAppWindow').classList.remove('hidden');
            document.getElementById('shellInput').focus();
            break;
        case 'store':
            document.getElementById('storeAppWindow').classList.remove('hidden');
            loadStorePackages();
            break;
        case 'files':
            document.getElementById('filesAppWindow').classList.remove('hidden');
            loadFilesList();
            break;
        case 'settings':
            document.getElementById('settingsAppWindow').classList.remove('hidden');
            break;
        default:
            document.getElementById('homeScreen').classList.remove('hidden');
            break;
    }

    triggerIslandNotice(`🚀 ${getAppNameBangla(appName)} সক্রিয়`);
    logNilBus('org.onuron.AppSupervisor', 'AppLaunched', { app: appName });
}

function getAppNameBangla(appName) {
    const map = {
        'notes': 'নোটবুক', 'calc': 'ক্যালকুলেটর', 'pos': 'রিটেইল পিওএস',
        'music': 'মিউজিক প্লেয়ার', 'shell': 'টার্মিনাল', 'store': 'প্যাকেজ স্টোর',
        'files': 'ফাইল ম্যানেজার', 'settings': 'সেটিংস', 'home': 'হোমস্ক্রিন'
    };
    return map[appName] || appName;
}

// ─── ৭. মাল্টিটাস্কিং রিসেন্ট অ্যাপস সুইচার ─────────────────────────────
function toggleAppSwitcher() {
    const sw = document.getElementById('appSwitcher');
    sw.classList.toggle('hidden');
    if (!sw.classList.contains('hidden')) {
        renderAppSwitcherCards();
    }
}

function renderAppSwitcherCards() {
    const container = document.getElementById('switcherCardsRow');
    container.innerHTML = '';

    const appIcons = {
        'notes': '📝', 'calc': '🧮', 'pos': '🏪', 'music': '🎵',
        'shell': '💻', 'store': '📦', 'files': '📁', 'settings': '⚙️'
    };

    osState.recentApps.forEach(app => {
        const card = document.createElement('div');
        card.className = 'switcher-card';
        card.innerHTML = `
            <div class="switcher-card-header">
                <span>${appIcons[app] || '📱'}</span>
                <span>${getAppNameBangla(app)}</span>
            </div>
            <div class="switcher-card-preview">
                <span>${appIcons[app] || '📱'}</span>
            </div>
            <button class="mini-btn" onclick="event.stopPropagation(); killApp('${app}')">কিল করুন ✕</button>
        `;
        card.addEventListener('click', () => {
            launchApp(app);
        });
        container.appendChild(card);
    });
}

function killApp(app) {
    osState.recentApps = osState.recentApps.filter(a => a !== app);
    renderAppSwitcherCards();
    triggerIslandNotice(`🛑 ${getAppNameBangla(app)} বন্ধ করা হয়েছে`);
    logNilBus('org.onuron.AppSupervisor', 'AppKilled', { app });
}

// ─── ৮. ডায়নামিক আইল্যান্ড অ্যানিমেশন ──────────────────────────────────
function triggerIslandNotice(msg, icon = '📝') {
    const island = document.getElementById('dynamicIsland');
    const content = document.getElementById('islandContent');
    const iconEl = document.getElementById('islandIcon');
    const textEl = document.getElementById('islandText');

    iconEl.textContent = icon;
    textEl.textContent = msg;

    island.classList.add('expanded');
    content.classList.remove('hidden');

    clearTimeout(island._timer);
    island._timer = setTimeout(() => {
        island.classList.remove('expanded');
        content.classList.add('hidden');
    }, 2400);
}

// ─── ৯. নোটবুক অ্যাপ লজিক (Nilang Notes) ──────────────────────────────
async function loadNotesData() {
    try {
        const res = await fetch('/api/notes');
        if (res.ok) {
            osState.notes = await res.json();
        } else {
            loadFallbackNotes();
        }
    } catch (e) {
        loadFallbackNotes();
    }
    renderNotesListMobile();
    updateQuickWidget();
    updateDbJsonInspector();
}

function loadFallbackNotes() {
    const saved = localStorage.getItem('nilang_notes_data');
    if (saved) {
        try { osState.notes = JSON.parse(saved); return; } catch (e) {}
    }
    osState.notes = [
        {
            id: 'note-1',
            title: 'স্বাগতম নীলাং নোটবুক-এ! 📌',
            content: 'নীলাং ভাষায় তৈরি আধুনিক নোট অ্যাপ্লিকেশন।\n- লাইভ সার্চ ও ফিল্টারিং\n- টাচস্ক্রিন এমুলেটর\n- চেকলিস্ট সাপোর্ট',
            category: 'important',
            tags: ['নীলাং', 'গাইড'],
            color: 'indigo',
            isPinned: true,
            isFavorite: true,
            isTrashed: false,
            checklist: [
                { id: 'c1', text: 'ক্যাটাগরি সাজানো', completed: true },
                { id: 'c2', text: 'চেকলিস্ট টেস্ট', completed: true }
            ]
        }
    ];
}

function renderNotesListMobile() {
    const container = document.getElementById('mCardsContainer');
    if (!container) return;
    container.innerHTML = '';

    let list = osState.notes.filter(n => !n.isTrashed);

    if (osState.activeCategory === 'trash') {
        list = osState.notes.filter(n => n.isTrashed);
    } else if (osState.activeCategory !== 'all') {
        list = list.filter(n => n.category === osState.activeCategory);
    }

    if (osState.searchQuery) {
        list = list.filter(n => 
            n.title.toLowerCase().includes(osState.searchQuery) ||
            n.content.toLowerCase().includes(osState.searchQuery)
        );
    }

    if (list.length === 0) {
        container.innerHTML = `<div style="text-align:center; padding:30px; color:#64748B; font-size:12.5px;">কোনো নোট পাওয়া যায়নি। নতুন নোট তৈরি করতে '➕' চাপুন।</div>`;
        return;
    }

    list.forEach(note => {
        const card = document.createElement('div');
        card.className = 'm-note-card';
        card.innerHTML = `
            <div class="m-card-top">
                <span class="m-card-title">${note.title}</span>
                <span>${note.isPinned ? '📌' : ''} ${note.isFavorite ? '⭐' : ''}</span>
            </div>
            <div class="m-card-excerpt">${escapeHtml(note.content.slice(0, 70))}...</div>
            <div class="m-card-tags">
                <span class="m-tag">${note.category}</span>
                ${note.tags.map(t => `<span class="m-tag">#${t}</span>`).join('')}
            </div>
        `;
        card.addEventListener('click', () => openNoteEditor(note.id));
        container.appendChild(card);
    });
}

function openNoteEditor(noteId) {
    const note = osState.notes.find(n => n.id === noteId);
    if (!note) return;

    osState.activeNoteId = noteId;
    document.getElementById('mNotesListView').classList.add('hidden');
    document.getElementById('mFabNewBtn').classList.add('hidden');
    document.getElementById('mEditorView').classList.remove('hidden');

    document.getElementById('mTitleInput').value = note.title;
    document.getElementById('mContentInput').value = note.content;
    document.getElementById('mTagsInput').value = note.tags.join(', ');
    document.getElementById('mCatSelect').value = note.category;

    updatePinFavButtons(note);
    renderEditorChecklist(note);
    updateWordCountMobile();

    logNilBus('org.nilang.notes', 'NoteOpened', { id: note.id, title: note.title });
}

function renderEditorChecklist(note) {
    const container = document.getElementById('mChkItems');
    container.innerHTML = '';
    const checks = note.checklist || [];

    checks.forEach(chk => {
        const row = document.createElement('div');
        row.className = 'm-chk-item';
        row.innerHTML = `
            <input type="checkbox" ${chk.completed ? 'checked' : ''}>
            <span style="${chk.completed ? 'text-decoration:line-through; color:#64748B;' : ''}">${chk.text}</span>
        `;
        row.querySelector('input').addEventListener('change', (e) => {
            chk.completed = e.target.checked;
            autoSaveMobileNote();
            renderEditorChecklist(note);
        });
        container.appendChild(row);
    });

    const doneCount = checks.filter(c => c.completed).length;
    document.getElementById('mChkProgress').textContent = `${doneCount}/${checks.length}`;
}

function addCheckItemMobile() {
    const input = document.getElementById('mNewChkInput');
    const text = input.value.trim();
    if (!text || !osState.activeNoteId) return;

    const note = osState.notes.find(n => n.id === osState.activeNoteId);
    if (!note) return;

    if (!note.checklist) note.checklist = [];
    note.checklist.push({ id: 'chk-' + Date.now(), text, completed: false });
    input.value = '';
    autoSaveMobileNote();
    renderEditorChecklist(note);
}

function createNewNoteMobile() {
    const newNote = {
        id: 'note-' + Date.now(),
        title: 'নতুন নোটবুক পৃষ্ঠা',
        content: '',
        category: 'personal',
        tags: ['নতুন'],
        color: 'indigo',
        isPinned: false,
        isFavorite: false,
        isTrashed: false,
        checklist: []
    };

    osState.notes.unshift(newNote);
    renderNotesListMobile();
    openNoteEditor(newNote.id);
    triggerIslandNotice("📝 নতুন নোট তৈরি হয়েছে");
    logNilBus('org.nilang.notes', 'NoteCreated', { id: newNote.id });
}

function autoSaveMobileNote() {
    const note = osState.notes.find(n => n.id === osState.activeNoteId);
    if (!note) return;

    note.title = document.getElementById('mTitleInput').value || 'শিরোনামহীন নোট';
    note.content = document.getElementById('mContentInput').value;
    note.category = document.getElementById('mCatSelect').value;
    note.tags = document.getElementById('mTagsInput').value.split(',').map(t => t.trim()).filter(Boolean);

    localStorage.setItem('nilang_notes_data', JSON.stringify(osState.notes));
    triggerIslandNotice("💾 স্বয়ংক্রিয় সংরক্ষিত");

    // ব্যাকএন্ডে পুট সিঙ্ক
    fetch(`/api/notes/${note.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(note)
    }).catch(() => {});

    updateDbJsonInspector();
}

function togglePinMobile() {
    const note = osState.notes.find(n => n.id === osState.activeNoteId);
    if (!note) return;
    note.isPinned = !note.isPinned;
    updatePinFavButtons(note);
    autoSaveMobileNote();
}

function toggleFavMobile() {
    const note = osState.notes.find(n => n.id === osState.activeNoteId);
    if (!note) return;
    note.isFavorite = !note.isFavorite;
    updatePinFavButtons(note);
    autoSaveMobileNote();
}

function deleteMobileNote() {
    const note = osState.notes.find(n => n.id === osState.activeNoteId);
    if (!note) return;
    note.isTrashed = true;
    autoSaveMobileNote();
    document.getElementById('mBackToListBtn').click();
    triggerIslandNotice("🗑️ নোট ট্র্যাশে পাঠানো হয়েছে");
}

function updatePinFavButtons(note) {
    document.getElementById('mPinToggleBtn').textContent = note.isPinned ? '📌 পিন্ড' : '📌';
    document.getElementById('mFavToggleBtn').textContent = note.isFavorite ? '⭐ প্রিয়' : '⭐';
}

function updateWordCountMobile() {
    const text = document.getElementById('mContentInput').value || '';
    const words = text.trim() ? text.trim().split(/\s+/).length : 0;
    const chars = text.length;
    document.getElementById('mWordCount').textContent = `${toBengaliNum(words)} শব্দ • ${toBengaliNum(chars)} অক্ষর`;
}

function updateQuickWidget() {
    const active = osState.notes.find(n => !n.isTrashed);
    if (!active) return;
    const widget = document.getElementById('quickWidgetNote');
    if (widget) {
        widget.innerHTML = `<h4>${active.title}</h4><p>${active.content.slice(0, 60)}...</p>`;
    }
}

// ─── ১০. ক্যালকুলেটর অ্যাপ লজিক (Nilang Calculator) ───────────────────
function calcInput(digit) {
    if (osState.calcCurrent === '0' && digit !== '.') {
        osState.calcCurrent = digit;
    } else {
        osState.calcCurrent += digit;
    }
    document.getElementById('calcResult').textContent = osState.calcCurrent;
}

function calcAction(act) {
    if (act === 'clear') {
        osState.calcCurrent = '0';
        osState.calcExpr = '';
    } else if (act === 'delete') {
        osState.calcCurrent = osState.calcCurrent.slice(0, -1) || '0';
    } else if (act === 'plusminus') {
        osState.calcCurrent = (parseFloat(osState.calcCurrent) * -1).toString();
    } else if (act === 'percent') {
        osState.calcCurrent = (parseFloat(osState.calcCurrent) / 100).toString();
    } else if (['add', 'subtract', 'multiply', 'divide'].includes(act)) {
        const symbolMap = { add: '+', subtract: '-', multiply: '*', divide: '/' };
        osState.calcExpr = `${osState.calcCurrent} ${symbolMap[act]} `;
        osState.calcCurrent = '0';
        document.getElementById('calcExpression').textContent = osState.calcExpr;
    } else if (act === 'equals') {
        try {
            const fullExpr = osState.calcExpr + osState.calcCurrent;
            const res = eval(fullExpr.replace(/÷/g, '/').replace(/×/g, '*'));
            const resStr = Number.isInteger(res) ? res.toString() : res.toFixed(2);
            osState.calcHistory.unshift(`${fullExpr} = ${resStr}`);
            osState.calcCurrent = resStr;
            osState.calcExpr = '';
            document.getElementById('calcExpression').textContent = '';
            renderCalcHistory();
            logNilBus('org.nilang.calculator', 'CalculationDone', { expr: fullExpr, result: resStr });
        } catch (e) {
            osState.calcCurrent = 'Error';
        }
    }
    document.getElementById('calcResult').textContent = osState.calcCurrent;
}

function renderCalcHistory() {
    const list = document.getElementById('calcHistoryList');
    if (!list) return;
    list.innerHTML = osState.calcHistory.map(h => `<div style="font-size:11px; padding:4px 0; border-bottom:1px solid #334155;">${h}</div>`).join('');
}

// ─── ১১. রিটেইল পিওএস টার্মিনাল লজিক (Nilang Retail POS) ──────────────
async function loadPOSCatalog() {
    try {
        const res = await fetch('/api/pos/products');
        if (res.ok) {
            osState.posCatalog = await res.json();
            renderPOSCatalog();
        }
    } catch (e) {}
}

function renderPOSCatalog() {
    const grid = document.getElementById('posCatalogGrid');
    if (!grid) return;
    grid.innerHTML = '';

    osState.posCatalog.forEach(prod => {
        const card = document.createElement('div');
        card.className = 'pos-product-card';
        card.innerHTML = `
            <div>
                <div class="pos-prod-icon">${prod.icon || '📦'}</div>
                <div class="pos-prod-name">${prod.name}</div>
            </div>
            <div class="pos-prod-price">৳ ${prod.price.toFixed(2)}</div>
        `;
        card.addEventListener('click', () => addProductToCart(prod));
        grid.appendChild(card);
    });
}

function addProductToCart(prod) {
    const existing = osState.posCart.find(i => i.product.id === prod.id);
    if (existing) {
        existing.quantity++;
    } else {
        osState.posCart.push({ product: prod, quantity: 1 });
    }
    updatePOSCartUI();
    triggerIslandNotice(`🛒 যোগ হয়েছে: ${prod.name}`);
    logNilBus('org.nilang.pos', 'ItemAddedToCart', { sku: prod.sku, price: prod.price });
}

function updatePOSCartUI() {
    let count = 0;
    let total = 0;
    osState.posCart.forEach(item => {
        count += item.quantity;
        total += item.product.price * item.quantity;
    });

    document.getElementById('posCartCount').textContent = toBengaliNum(count);
    document.getElementById('posCartTotal').textContent = total.toFixed(2);
    document.getElementById('posCartBadge').textContent = count;
}

function triggerBarcodeScanAnimation(barcode) {
    const overlay = document.getElementById('scannerOverlay');
    overlay.classList.remove('hidden');
    triggerIslandNotice("📸 লেজার স্ক্যানিং সক্রিয়...", "📸");

    setTimeout(() => {
        overlay.classList.add('hidden');
        const prod = osState.posCatalog.find(p => p.barcode === barcode) || osState.posCatalog[0];
        if (prod) {
            addProductToCart(prod);
            triggerIslandNotice(`✅ বারকোড ডিটেক্টেড: ${prod.name}`, "✅");
        }
    }, 1200);
}

async function handlePOSCheckoutClick() {
    if (osState.posCart.length === 0) {
        alert("⚠️ শপিং কার্ট খালি! অনুগ্রহ করে প্রোডাক্ট সিলেক্ট করুন।");
        return;
    }

    try {
        const res = await fetch('/api/pos/checkout', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                items: osState.posCart,
                customerName: 'সম্মানিত ক্রেতা',
                customerPhone: '01711-XXXXXX',
                discountPct: 5,
                paymentMethod: 'নগদ / Cash'
            })
        });

        if (res.ok) {
            const receipt = await res.json();
            showThermalReceipt(receipt);
            osState.posCart = [];
            updatePOSCartUI();
            logNilBus('org.nilang.pos', 'OrderCheckoutSuccess', { invoiceNo: receipt.invoiceNo, total: receipt.total });
        }
    } catch (e) {
        simulateThermalPrintReceipt();
    }
}

function showThermalReceipt(rcpt) {
    const paper = document.getElementById('thermalReceiptPaper');
    document.getElementById('rcptMeta').textContent = `ইনভয়েস: ${rcpt.invoiceNo} • ${rcpt.date}`;

    const itemsTable = document.getElementById('rcptItemsTable');
    itemsTable.innerHTML = rcpt.items.map(item => `
        <div class="rcpt-line">
            <span>${item.product.name} (x${item.quantity})</span>
            <span>৳ ${(item.product.price * item.quantity).toFixed(2)}</span>
        </div>
    `).join('');

    const totalsDiv = document.getElementById('rcptTotals');
    totalsDiv.innerHTML = `
        <div class="rcpt-line"><span>সাবটোটাল:</span><span>৳ ${rcpt.subtotal.toFixed(2)}</span></div>
        <div class="rcpt-line"><span>ডিসকাউন্ট (৫%):</span><span>-৳ ${rcpt.discount.toFixed(2)}</span></div>
        <div class="rcpt-line"><span>ভ্যাট (৫%):</span><span>+৳ ${rcpt.vat.toFixed(2)}</span></div>
        <div class="rcpt-line total-line"><span>সর্বমোট প্রদেয়:</span><span>৳ ${rcpt.total.toFixed(2)}</span></div>
    `;

    paper.classList.remove('hidden');
    triggerIslandNotice("🖨️ থার্মাল রিসিট প্রিন্ট হয়েছে!", "🖨️");
}

function simulateThermalPrintReceipt() {
    showThermalReceipt({
        invoiceNo: 'INV-20261007-001',
        date: '07/10/2026',
        items: [
            { product: { name: 'মিনিকেট চাল (৫০ কেজি)', price: 3200 }, quantity: 1 },
            { product: { name: 'তীর সয়াবিন তেল (৫লি)', price: 850 }, quantity: 1 }
        ],
        subtotal: 4050.0,
        discount: 202.5,
        vat: 192.38,
        total: 4039.88
    });
}

// ─── ১২. মিউজিক প্লেয়ার লজিক (Nilang Music Streamer) ──────────────────
async function loadMusicTracks() {
    try {
        const res = await fetch('/api/music/tracks');
        if (res.ok) {
            osState.musicTracks = await res.json();
            renderMusicPlaylist();
        }
    } catch (e) {}
}

function renderMusicPlaylist() {
    const list = document.getElementById('musicPlaylist');
    if (!list) return;
    list.innerHTML = '';

    osState.musicTracks.forEach((trk, idx) => {
        const item = document.createElement('div');
        item.className = `music-trk-item ${idx === osState.currentTrackIndex ? 'active' : ''}`;
        item.innerHTML = `
            <span>${trk.cover || '🎵'} ${trk.title}</span>
            <span>${trk.duration}</span>
        `;
        item.addEventListener('click', () => {
            osState.currentTrackIndex = idx;
            playCurrentTrack();
        });
        list.appendChild(item);
    });
}

function toggleMusicPlay() {
    if (osState.isPlaying) {
        pauseCurrentTrack();
    } else {
        playCurrentTrack();
    }
}

function playCurrentTrack() {
    osState.isPlaying = true;
    const trk = osState.musicTracks[osState.currentTrackIndex];
    if (!trk) return;

    document.getElementById('musicTrackTitle').textContent = trk.title;
    document.getElementById('musicTrackArtist').textContent = trk.artist;
    document.getElementById('musicCoverArt').textContent = trk.cover || '🎵';
    document.getElementById('btnMusicPlayPause').textContent = '⏸';

    document.getElementById('islandWaveform').classList.remove('hidden');
    triggerIslandNotice(`▶ এখন বাজছে: ${trk.title}`, "🎵");
    renderMusicPlaylist();

    logNilBus('org.nilang.music', 'PlaybackStarted', { title: trk.title, artist: trk.artist });
}

function pauseCurrentTrack() {
    osState.isPlaying = false;
    document.getElementById('btnMusicPlayPause').textContent = '▶';
    document.getElementById('islandWaveform').classList.add('hidden');
    triggerIslandNotice("⏸ গান স্থগিত করা হয়েছে");
    logNilBus('org.nilang.music', 'PlaybackPaused', {});
}

function musicNext() {
    osState.currentTrackIndex = (osState.currentTrackIndex + 1) % osState.musicTracks.length;
    playCurrentTrack();
}

function musicPrev() {
    osState.currentTrackIndex = (osState.currentTrackIndex - 1 + osState.musicTracks.length) % osState.musicTracks.length;
    playCurrentTrack();
}

function initAudioVisualizer() {
    const canvas = document.getElementById('musicVisualizer');
    if (!canvas) return;
    const ctx = canvas.getContext('2d');

    function draw() {
        requestAnimationFrame(draw);
        ctx.clearRect(0, 0, canvas.width, canvas.height);

        const bars = 24;
        const width = canvas.width / bars;

        for (let i = 0; i < bars; i++) {
            let height = 4;
            if (osState.isPlaying) {
                height = Math.sin(Date.now() / 200 + i) * 20 + 25;
            }
            ctx.fillStyle = i % 2 === 0 ? '#38BDF8' : '#818CF8';
            ctx.fillRect(i * width + 2, canvas.height - height, width - 4, height);
        }
    }
    draw();
}

// ─── ১৩. টার্মিনাল ও আরইপিএল লজিক (Nilang Shell) ────────────────────────
function handleShellCommand(cmdText) {
    const out = document.getElementById('shellOutput');
    const cmd = cmdText.trim();
    if (!cmd) return;

    appendShellLine(`❯ ${cmd}`, 'text-cyan');

    if (cmd === 'help') {
        appendShellLine("সহায়ক কমান্ডসমূহ:");
        appendShellLine("  help     - কমান্ড তালিকা দেখুন");
        appendShellLine("  notes    - নোটবুক অ্যাপ চালু করুন");
        appendShellLine("  calc     - ক্যালকুলেটর চালু করুন");
        appendShellLine("  pos      - রিটেইল পিওএস টার্মিনাল খুলুন");
        appendShellLine("  music    - মিউজিক প্লেয়ার খুলুন");
        appendShellLine("  stats    - সিস্টেম স্ট্যাটাস দেখুন");
        appendShellLine("  clear    - স্ক্রিন পরিষ্কার করুন");
        appendShellLine("  nil ...  - নীলাং কমান্ড বা এক্সপ্রেশন রান করুন");
    } else if (cmd === 'clear' || cmd === 'cls') {
        clearShell();
    } else if (cmd === 'notes') {
        launchApp('notes');
    } else if (cmd === 'calc') {
        launchApp('calc');
    } else if (cmd === 'pos') {
        launchApp('pos');
    } else if (cmd === 'music') {
        launchApp('music');
    } else if (cmd === 'stats') {
        appendShellLine(`NilOS v2.0 • Active Notes: ${osState.notes.length} • Memory: 1.4GB / 8GB`);
    } else {
        // ব্যাকএন্ডে রান কমান্ড পাঠানো
        fetch('/api/emulator/terminal', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ command: cmd })
        })
        .then(r => r.json())
        .then(data => {
            if (data.output) {
                appendShellLine(data.output);
            } else {
                appendShellLine(`[OK] Executed '${cmd}'`);
            }
        })
        .catch(() => {
            appendShellLine(`NilLang Runtime Executed: ${cmd}`);
        });
    }

    out.scrollTop = out.scrollHeight;
    logNilBus('org.nilang.shell', 'CommandExecuted', { command: cmd });
}

function appendShellLine(text, className = '') {
    const out = document.getElementById('shellOutput');
    const div = document.createElement('div');
    div.className = `shell-line ${className}`;
    div.textContent = text;
    out.appendChild(div);
}

function clearShell() {
    const out = document.getElementById('shellOutput');
    out.innerHTML = `<div class="shell-line text-cyan">⚡ নীল ওএস (NilOS) v2.0 Interactive Terminal</div>`;
}

// ─── ১৪. প্যাকেজ স্টোর ও ফাইল ম্যানেজার ───────────────────────────────
async function loadStorePackages() {
    try {
        const res = await fetch('/api/emulator/packages');
        if (res.ok) {
            const pkgs = await res.json();
            const list = document.getElementById('storePkgList');
            if (!list) return;
            list.innerHTML = '';

            pkgs.forEach(pkg => {
                const card = document.createElement('div');
                card.className = 'store-pkg-card';
                card.innerHTML = `
                    <div class="pkg-header">
                        <h4>${pkg.icon} ${pkg.name}</h4>
                        <button class="pkg-btn" onclick="launchPackageApp('${pkg.id}')">চালান 🚀</button>
                    </div>
                    <p style="font-size:11px; color:#94A3B8;">${pkg.description}</p>
                    <div style="font-size:10px; color:#64748B; margin-top:4px;">
                        ${pkg.size} • 🛡️ ${pkg.signature}
                    </div>
                `;
                list.appendChild(card);
            });
        }
    } catch (e) {}
}

function launchPackageApp(pkgId) {
    if (pkgId === 'notes-app') launchApp('notes');
    else if (pkgId === 'pos-app') launchApp('pos');
    else if (pkgId === 'calculator-app') launchApp('calc');
    else if (pkgId === 'music-app') launchApp('music');
}

async function loadFilesList() {
    try {
        const res = await fetch('/api/files/browse?path=.');
        if (res.ok) {
            const data = await res.json();
            const list = document.getElementById('filesList');
            if (!list) return;
            list.innerHTML = '';

            data.entries.forEach(file => {
                const item = document.createElement('div');
                item.className = 'file-item';
                item.innerHTML = `
                    <span>${file.isDir ? '📁' : '📄'} ${file.name}</span>
                    <span style="color:#64748B; font-size:10.5px;">${file.isDir ? 'ফোল্ডার' : (file.size + ' B')}</span>
                `;
                list.appendChild(item);
            });
        }
    } catch (e) {}
}

// ─── ১৫. নীলবাস (NilBus) ও ডেভেলপার ডায়াগনস্টিকস ─────────────────────────
function initDevTools() {
    const tabs = document.querySelectorAll('.dt-tab');
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            tabs.forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.dt-content').forEach(c => c.classList.remove('active'));

            tab.classList.add('active');
            const target = document.getElementById(tab.dataset.tab);
            if (target) target.classList.add('active');
        });
    });
}

function logNilBus(service, event, payload) {
    const time = new Date().toLocaleTimeString();
    const logItem = { time, service, event, payload };
    osState.busLogs.unshift(logItem);

    const list = document.getElementById('nilbusLogList');
    if (list) {
        const el = document.createElement('div');
        el.className = 'bus-log-item';
        el.innerHTML = `
            <div style="color:#94A3B8; font-size:10px;">${time} • <strong style="color:#38BDF8;">${service}</strong></div>
            <div style="color:#F8FAFC; font-weight:600;">${event}</div>
            <div style="color:#64748B; font-size:9.5px;">${JSON.stringify(payload)}</div>
        `;
        list.insertBefore(el, list.firstChild);
        if (list.children.length > 50) list.removeChild(list.lastChild);
    }
}

function clearBusLogs() {
    osState.busLogs = [];
    document.getElementById('nilbusLogList').innerHTML = '';
}

function updateDbJsonInspector() {
    const el = document.getElementById('dbJsonView');
    if (el) {
        el.textContent = JSON.stringify(osState.notes, null, 2);
    }
}

function initFpsChart() {
    const canvas = document.getElementById('fpsChart');
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    const points = Array(30).fill(60);

    function loop() {
        requestAnimationFrame(loop);
        const fps = 58 + Math.random() * 4;
        points.push(fps);
        points.shift();

        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.strokeStyle = '#10B981';
        ctx.lineWidth = 2;
        ctx.beginPath();

        const step = canvas.width / (points.length - 1);
        points.forEach((val, i) => {
            const y = canvas.height - ((val - 40) / 30) * canvas.height;
            if (i === 0) ctx.moveTo(0, y);
            else ctx.lineTo(i * step, y);
        });
        ctx.stroke();
    }
    loop();
}

// ─── ১৬. QEMU হেডলেস কার্নেল ও ফ্রেমবাফার রেন্ডারার ───────────────────
let qemuTimer = null;
function initQEMUFramebuffer() {
    updateQEMUDisplay();
    qemuTimer = setInterval(updateQEMUDisplay, 1200);
}

async function updateQEMUDisplay() {
    try {
        const res = await fetch('/api/qemu/framebuffer');
        if (!res.ok) return;
        const data = await res.json();

        const canvas = document.getElementById('qemuFramebufferCanvas');
        if (!canvas) return;
        const ctx = canvas.getContext('2d');

        // Draw VGA background
        ctx.fillStyle = '#05070E';
        ctx.fillRect(0, 0, canvas.width, canvas.height);

        // Draw VGA Scanlines effect
        ctx.fillStyle = 'rgba(255, 255, 255, 0.015)';
        for (let y = 0; y < canvas.height; y += 3) {
            ctx.fillRect(0, y, canvas.width, 1);
        }

        // Draw Text lines
        ctx.font = '8.5px "JetBrains Mono", monospace';
        ctx.fillStyle = '#38BDF8';

        let yPos = 14;
        const lines = data.vgaText || [];
        lines.slice(-14).forEach((line, idx) => {
            if (idx === 0 || idx === 1) ctx.fillStyle = '#F59E0B';
            else if (line.includes('ERROR') || line.includes('HALT')) ctx.fillStyle = '#EF4444';
            else if (line.includes('OK') || line.includes('ready')) ctx.fillStyle = '#10B981';
            else ctx.fillStyle = '#94A3B8';

            ctx.fillText(line, 6, yPos);
            yPos += 11;
        });

        // Blinking VGA cursor
        if (Math.floor(Date.now() / 500) % 2 === 0) {
            ctx.fillStyle = '#10B981';
            ctx.fillRect(canvas.width - 20, canvas.height - 12, 6, 8);
        }

        // Update Registers
        const regGrid = document.getElementById('qemuRegGrid');
        if (regGrid && data.registers) {
            regGrid.innerHTML = '';
            for (const [reg, val] of Object.entries(data.registers)) {
                const item = document.createElement('div');
                item.className = 'reg-item';
                item.innerHTML = `<span>${reg}:</span> <strong>${val}</strong>`;
                regGrid.appendChild(item);
            }
        }

        // Update State Badge
        const badge = document.getElementById('qemuStateBadge');
        if (badge) {
            if (data.state === 'running') {
                badge.textContent = '🟢 RUNNING (60 FPS)';
                badge.style.borderColor = '#10B981';
                badge.style.color = '#10B981';
            } else if (data.state === 'paused') {
                badge.textContent = '🟡 PAUSED';
                badge.style.borderColor = '#F59E0B';
                badge.style.color = '#F59E0B';
            } else {
                badge.textContent = '🔴 STOPPED';
                badge.style.borderColor = '#EF4444';
                badge.style.color = '#EF4444';
            }
        }
    } catch (e) {}
}

async function qemuControl(action) {
    try {
        const res = await fetch('/api/qemu/control', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ action })
        });
        if (res.ok) {
            triggerIslandNotice(`🖥️ QEMU: ${action.toUpperCase()}`);
            logNilBus('org.onuron.QEMUBridge', 'ControlAction', { action });
            updateQEMUDisplay();
        }
    } catch (e) {}
}

async function sendQemuKey() {
    const input = document.getElementById('qemuKeyInput');
    if (!input || !input.value.trim()) return;
    const key = input.value.trim();
    input.value = '';

    try {
        await fetch('/api/qemu/control', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ action: 'sendkey', key })
        });
        triggerIslandNotice(`⌨️ কার্নেল কীস্ট্রোক: ${key}`);
        logNilBus('org.onuron.PS2Keyboard', 'KeyInjected', { key });
        updateQEMUDisplay();
    } catch (e) {}
}

// ─── ১৭. NDB রিমোট ডিবাগার ও Nil-LSP অ্যানালাইজার ───────────────────
async function initNDBDebugger() {
    try {
        const res = await fetch('/api/ndb/session');
        if (res.ok) {
            const data = await res.json();
            renderNDBState(data);
        }
    } catch (e) {}

    // Default code for LSP tester
    const lspInput = document.getElementById('lspTestInput');
    if (lspInput && !lspInput.value) {
        lspInput.value = `@Component\nফাংশন MyWidget() {\n    চল বার্তা = "হ্যালো নীল ওএস!";\n    ফেরত বার্তা;\n}`;
    }
}

function renderNDBState(session) {
    const lineEl = document.getElementById('ndbCurrentLine');
    if (lineEl) lineEl.textContent = session.currentLine;

    // Variables Watch
    const varList = document.getElementById('ndbVarList');
    if (varList && session.variables) {
        varList.innerHTML = '';
        for (const [k, v] of Object.entries(session.variables)) {
            const row = document.createElement('div');
            row.className = 'ndb-var-row';
            row.innerHTML = `<span class="var-name">${k}</span> <span class="var-val">${v}</span>`;
            varList.appendChild(row);
        }
    }

    // Call stack
    const stackList = document.getElementById('ndbStackList');
    if (stackList && session.callStack) {
        stackList.innerHTML = '';
        session.callStack.forEach(frame => {
            const item = document.createElement('div');
            item.className = 'ndb-stack-item';
            item.innerHTML = `<span><strong>${frame.function}</strong></span> <span style="color:#64748B;">${frame.file}:${frame.line}</span>`;
            stackList.appendChild(item);
        });
    }
}

async function ndbStep(action) {
    try {
        const res = await fetch('/api/ndb/step', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ action })
        });
        if (res.ok) {
            const data = await res.json();
            renderNDBState(data.session);
            triggerIslandNotice(`🐞 NDB: ${action.replace('_', ' ').toUpperCase()}`);
            logNilBus('org.nilang.NDB', 'StepAction', { action, line: data.session.currentLine });
        }
    } catch (e) {}
}

async function analyzeLspCode() {
    const input = document.getElementById('lspTestInput');
    const resultEl = document.getElementById('lspDiagResult');
    if (!input || !resultEl) return;

    try {
        const res = await fetch('/api/lsp/analyze', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ code: input.value, file: 'test.nil' })
        });
        if (res.ok) {
            const data = await res.json();
            if (data.errorsCount === 0) {
                resultEl.textContent = '✔ সিনট্যাক্স সঠিক (০ এরর)';
                resultEl.style.color = '#10B981';
            } else {
                resultEl.textContent = `⚠️ ${data.errorsCount} টি ত্রুটি পাওয়া গেছে`;
                resultEl.style.color = '#EF4444';
            }
            logNilBus('org.nilang.LSP', 'DiagnosticsCompleted', { errors: data.errorsCount });
        }
    } catch (e) {}
}

// ─── ১৮. IDE এক্সটেনশন প্লাগইন সেন্টার ──────────────────────────────
async function loadExtensionsList() {
    try {
        const res = await fetch('/api/extensions/list');
        if (!res.ok) return;
        const exts = await res.json();

        const wrap = document.getElementById('pluginsListWrap');
        if (!wrap) return;
        wrap.innerHTML = '';

        exts.forEach(ext => {
            const card = document.createElement('div');
            card.className = 'plugin-card';
            card.innerHTML = `
                <div class="plugin-header">
                    <span style="font-size:18px;">${ext.icon}</span>
                    <div>
                        <h4>${ext.name}</h4>
                        <span style="font-size:10px; color:#64748B;">${ext.targetIde} • v${ext.version}</span>
                    </div>
                </div>
                <p class="plugin-desc">${ext.description}</p>
                <ul class="plugin-feats">
                    ${ext.features.map(f => `<li>${f}</li>`).join('')}
                </ul>
                <a href="${ext.downloadUrl}" download style="text-decoration:none;">
                    <button class="plugin-btn">📥 ডাউনলোড বান্ডিল (${ext.bundleFile})</button>
                </a>
            `;
            wrap.appendChild(card);
        });
    } catch (e) {}
}

// ─── ১৬. ভার্চুয়াল টাচ কীবোর্ড টেক্সট ইনসার্ট ──────────────────────────
function insertChar(char) {
    if (osState.focusedInput) {
        const input = osState.focusedInput;
        const start = input.selectionStart || input.value.length;
        const end = input.selectionEnd || input.value.length;
        input.value = input.value.substring(0, start) + char + input.value.substring(end);
        input.selectionStart = input.selectionEnd = start + char.length;
        input.dispatchEvent(new Event('input'));
    }
}

function insertText(text) {
    insertChar(text);
}

function backspaceKey() {
    if (osState.focusedInput) {
        const input = osState.focusedInput;
        const start = input.selectionStart || input.value.length;
        if (start > 0) {
            input.value = input.value.substring(0, start - 1) + input.value.substring(start);
            input.selectionStart = input.selectionEnd = start - 1;
            input.dispatchEvent(new Event('input'));
        }
    }
}

// ─── ১৭. থিম ও ইউটিলিটিস ─────────────────────────────────────────────
function cycleTheme() {
    const themes = ['theme-dark', 'theme-cyberpunk', 'theme-emerald', 'theme-light'];
    const idx = themes.indexOf(osState.theme);
    const nextTheme = themes[(idx + 1) % themes.length];
    applyTheme(nextTheme);
}

function applyTheme(themeName) {
    document.body.className = themeName;
    osState.theme = themeName;
    triggerIslandNotice(`🎨 থিম: ${themeName.replace('theme-', '')}`);
}

function openStatsModal() {
    const modal = document.getElementById('statsModal');
    const body = document.getElementById('statsModalBody');

    const totalNotes = osState.notes.length;
    const active = osState.notes.filter(n => !n.isTrashed).length;
    const pinned = osState.notes.filter(n => n.isPinned).length;
    const fav = osState.notes.filter(n => n.isFavorite).length;

    body.innerHTML = `
        <div style="display:flex; flex-direction:column; gap:8px; font-size:13px;">
            <div>📑 সর্বমোট নোট: <strong>${toBengaliNum(totalNotes)} টি</strong></div>
            <div>🟢 সক্রিয় নোট: <strong>${toBengaliNum(active)} টি</strong></div>
            <div>📌 পিন্ড (Pinned): <strong>${toBengaliNum(pinned)} টি</strong></div>
            <div>⭐ পছন্দের (Favorites): <strong>${toBengaliNum(fav)} টি</strong></div>
            <div>🖥️ কার্নেল: <strong>Onuron Microkernel x86_64</strong></div>
            <div>⚡ কম্পাইলার: <strong>NilLang 1.0 (Alap Engine)</strong></div>
        </div>
    `;
    modal.classList.remove('hidden');
}

function closeStatsModal() {
    document.getElementById('statsModal').classList.add('hidden');
}

function toBengaliNum(num) {
    const bn = ['০', '১', '২', '৩', '৪', '৫', '৬', '৭', '৮', '৯'];
    return num.toString().split('').map(d => bn[d] || d).join('');
}

function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
