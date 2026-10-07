// public/app.js — নীলাং নোটবুক ক্লায়েন্ট লজিক

let state = {
    notes: [],
    activeNoteId: null,
    activeCategory: 'all',
    activeTag: null,
    searchQuery: '',
    viewMode: 'grid', // 'grid' or 'list'
    editorTab: 'write', // 'write', 'preview', 'split'
    currentTheme: 'theme-dark'
};

// ─── অ্যাপ ইনিশিয়ালাইজেশন ───────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
    initEventListeners();
    loadNotes();
    loadTheme();
});

// ─── ইভেন্ট হ্যান্ডলার্স ────────────────────────────────────────────────
function initEventListeners() {
    // সার্চ
    const searchInput = document.getElementById('searchInput');
    const clearSearchBtn = document.getElementById('clearSearchBtn');

    searchInput.addEventListener('input', (e) => {
        state.searchQuery = e.target.value.trim().toLowerCase();
        clearSearchBtn.classList.toggle('hidden', state.searchQuery === '');
        renderNotesList();
    });

    clearSearchBtn.addEventListener('click', () => {
        searchInput.value = '';
        state.searchQuery = '';
        clearSearchBtn.classList.add('hidden');
        renderNotesList();
    });

    // কীবোর্ড শর্টকাট (Ctrl + K)
    document.addEventListener('keydown', (e) => {
        if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
            e.preventDefault();
            searchInput.focus();
        }
    });

    // ক্যাটাগরি নেভিগেশন
    document.getElementById('categoryNav').addEventListener('click', (e) => {
        const item = e.target.closest('.nav-item');
        if (!item) return;

        document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('active'));
        item.classList.add('active');

        state.activeCategory = item.dataset.category;
        state.activeTag = null; // রিসেট ট্যাগ
        document.querySelectorAll('.tag-pill').forEach(el => el.classList.remove('active'));

        const label = item.querySelector('.nav-label').textContent;
        document.getElementById('currentCategoryTitle').textContent = label;

        renderNotesList();
    });

    // ভিউ মোড সুইচ (Grid vs List)
    document.getElementById('viewGridBtn').addEventListener('click', () => {
        state.viewMode = 'grid';
        document.getElementById('viewGridBtn').classList.add('active');
        document.getElementById('viewListBtn').classList.remove('active');
        renderNotesList();
    });

    document.getElementById('viewListBtn').addEventListener('click', () => {
        state.viewMode = 'list';
        document.getElementById('viewListBtn').classList.add('active');
        document.getElementById('viewGridBtn').classList.remove('active');
        renderNotesList();
    });

    // নতুন নোট বাটন
    document.getElementById('newNoteHeaderBtn').addEventListener('click', createNewNote);
    document.getElementById('newNoteEmptyBtn').addEventListener('click', createNewNote);

    // এডিটর ইনপুট ফিল্ড লিসেনার্স
    document.getElementById('noteTitleInput').addEventListener('input', autoSaveCurrentNote);
    document.getElementById('noteContentInput').addEventListener('input', () => {
        autoSaveCurrentNote();
        updateMetrics();
        if (state.editorTab !== 'write') {
            renderMarkdown();
        }
    });
    document.getElementById('noteTagsInput').addEventListener('change', autoSaveCurrentNote);
    document.getElementById('noteCategorySelect').addEventListener('change', autoSaveCurrentNote);

    // কালার পিকার
    document.getElementById('colorPicker').addEventListener('click', (e) => {
        const dot = e.target.closest('.color-dot');
        if (!dot) return;
        document.querySelectorAll('.color-dot').forEach(el => el.classList.remove('active'));
        dot.classList.add('active');
        autoSaveCurrentNote();
    });

    // পিন ও ফেভারিট বাটন
    document.getElementById('pinNoteBtn').addEventListener('click', togglePinCurrentNote);
    document.getElementById('favNoteBtn').addEventListener('click', toggleFavCurrentNote);
    document.getElementById('deleteNoteBtn').addEventListener('click', deleteCurrentNote);
    document.getElementById('exportMdBtn').addEventListener('click', exportCurrentNoteAsMarkdown);

    // এডিটর ট্যাব সুইচ
    document.getElementById('tabWriteBtn').addEventListener('click', () => setEditorTab('write'));
    document.getElementById('tabPreviewBtn').addEventListener('click', () => setEditorTab('preview'));
    document.getElementById('tabSplitBtn').addEventListener('click', () => setEditorTab('split'));

    // চেকলিস্ট যোগ
    document.getElementById('addCheckBtn').addEventListener('click', addChecklistItem);
    document.getElementById('newCheckInput').addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            addChecklistItem();
        }
    });

    // থিম সিলেক্টর
    document.getElementById('themeSelector').addEventListener('change', (e) => {
        setTheme(e.target.value);
    });

    // মোডাল ইভেন্টস
    document.getElementById('statsModalBtn').addEventListener('click', openStatsModal);
    document.getElementById('closeStatsModal').addEventListener('click', closeStatsModal);
    document.getElementById('qemuLaunchBtn').addEventListener('click', openQemuModal);
    document.getElementById('closeQemuModal').addEventListener('click', closeQemuModal);

    // QEMU লঞ্চ বাটন
    document.getElementById('startQemuProcessBtn').addEventListener('click', launchQEMU);
    document.getElementById('runNilangDirectBtn').addEventListener('click', runNilangTest);
}

// ─── ডেটা লোড ও সিঙ্ক ────────────────────────────────────────────────
async function loadNotes() {
    try {
        const res = await fetch('/api/notes');
        if (res.ok) {
            state.notes = await res.json();
        } else {
            loadFallbackNotes();
        }
    } catch (err) {
        console.warn("Using local fallback notes:", err);
        loadFallbackNotes();
    }

    updateCountsAndTags();
    renderNotesList();

    if (state.notes.length > 0 && !state.activeNoteId) {
        selectNote(state.notes[0].id);
    }
}

function loadFallbackNotes() {
    const saved = localStorage.getItem('nilang_notes_data');
    if (saved) {
        state.notes = JSON.parse(saved);
    } else {
        state.notes = [
            {
                id: 'note-1',
                title: 'স্বাগতম নীলাং নোটবুক-এ! (Nilang Notes Guide)',
                content: 'নীলাং ভাষায় তৈরি আধুনিক, দ্রুতগতির এবং সুন্দর নোট অ্যাপ্লিকেশন।\n\nপ্রধান সুবিধাসমূহ:\n- সম্পূর্ণ বাংলায় ও ইংরেজিতে নোট সংরক্ষণ\n- লাইভ সার্চ ও ক্যাটাগরি ফিল্টারিং\n- প্রতিটি নোটে কালার ব্যাজ ও প্রায়োরিটি\n- সম্পূর্ণ ক্রস-প্ল্যাটফর্ম (Windows, Linux, Onuron, QEMU)',
                category: 'important',
                tags: ['নীলাং', 'গাইড', 'নোটবুক', 'অনুরণ'],
                color: 'indigo',
                priority: 'high',
                isPinned: true,
                isFavorite: true,
                isArchived: false,
                isTrashed: false,
                checklist: [
                    { id: 'chk-1', text: 'ক্যাটাগরি ও ট্যাগ দিয়ে নোট সাজানো', completed: true },
                    { id: 'chk-2', text: 'ইন্টারেক্টিভ চেকলিস্ট ও টাস্ক তৈরি', completed: true },
                    { id: 'chk-3', text: 'QEMU এমুলেটরে ভার্চুয়াল ওএস টেস্ট করা', completed: true },
                    { id: 'chk-4', text: 'অটো-সেভ ও ডেটা রিকভারি যাচাই করা', completed: false }
                ],
                createdAt: Date.now() / 1000,
                updatedAt: Date.now() / 1000
            },
            {
                id: 'note-2',
                title: 'দৈনিক কাজের পরিকল্পনা ও ডেডলাইন',
                content: 'আজকের সর্বোচ্চ অগ্রাধিকারের কাজগুলো সম্পন্ন করতে হবে। সন্ধ্যে ৬টার মধ্যে বিল্ড টেস্ট সমাপ্ত করা দরকার।',
                category: 'todo',
                tags: ['কাজ', 'প্ল্যানিং', 'টাস্ক'],
                color: 'emerald',
                priority: 'high',
                isPinned: true,
                isFavorite: false,
                isArchived: false,
                isTrashed: false,
                checklist: [
                    { id: 'chk-21', text: 'নীলাং কোডবেজ রিফ্যাক্টরিং', completed: true },
                    { id: 'chk-22', text: 'আলাপ UI ডিক্লারেটিভ কম্পোনেন্ট তৈরি', completed: true },
                    { id: 'chk-23', text: 'QEMU x86_64 কার্নেল বুট টেস্ট', completed: true },
                    { id: 'chk-24', text: 'রিসার্চ পেপার ড্রাফট রিভিশন', completed: false }
                ],
                createdAt: Date.now() / 1000,
                updatedAt: Date.now() / 1000
            }
        ];
    }
}

function saveToLocalStorage() {
    localStorage.setItem('nilang_notes_data', JSON.stringify(state.notes));
}

// ─── নোট তালিকা রেন্ডারিং ─────────────────────────────────────────────
function getFilteredNotes() {
    return state.notes.filter(note => {
        // ক্যাটাগরি ফিল্টার
        if (state.activeCategory === 'trash') {
            if (!note.isTrashed) return false;
        } else if (state.activeCategory === 'archive') {
            if (!note.isArchived || note.isTrashed) return false;
        } else {
            if (note.isTrashed || note.isArchived) return false;

            if (state.activeCategory === 'important') {
                if (!note.isPinned && !note.isFavorite) return false;
            } else if (state.activeCategory !== 'all') {
                if (note.category !== state.activeCategory) return false;
            }
        }

        // ট্যাগ ফিল্টার
        if (state.activeTag) {
            if (!note.tags || !note.tags.includes(state.activeTag)) return false;
        }

        // সার্চ কুয়েরি
        if (state.searchQuery) {
            const inTitle = note.title.toLowerCase().includes(state.searchQuery);
            const inContent = note.content.toLowerCase().includes(state.searchQuery);
            const inTags = note.tags && note.tags.some(t => t.toLowerCase().includes(state.searchQuery));
            if (!inTitle && !inContent && !inTags) return false;
        }

        return true;
    }).sort((a, b) => {
        // পিন্ড নোট সবার আগে
        if (a.isPinned && !b.isPinned) return -1;
        if (!a.isPinned && b.isPinned) return 1;
        return b.updatedAt - a.updatedAt;
    });
}

function renderNotesList() {
    const listEl = document.getElementById('notesList');
    const filtered = getFilteredNotes();

    document.getElementById('categoryNoteCountBadge').textContent = `${filtered.length} টি নোট`;

    if (filtered.length === 0) {
        listEl.innerHTML = `
            <div class="empty-list-msg" style="text-align: center; color: var(--text-dim); padding: 30px 10px;">
                <p>কোনো নোট পাওয়া যায়নি</p>
                <small>নতুন নোট তৈরি করতে '+ নতুন নোট' চাপুন</small>
            </div>
        `;
        return;
    }

    listEl.innerHTML = filtered.map(note => {
        const isActive = note.id === state.activeNoteId ? 'active' : '';
        const isPinned = note.isPinned ? 'pinned' : '';
        const excerpt = note.content ? note.content.substring(0, 75) + '...' : 'খালি নোট';

        // চেকলিস্ট সামারি
        let checkSummary = '';
        if (note.checklist && note.checklist.length > 0) {
            const done = note.checklist.filter(c => c.completed).length;
            checkSummary = `<div class="card-checklist-preview">✔ ${done}/${note.checklist.length} সম্পন্ন</div>`;
        }

        const tagsHtml = (note.tags || []).map(t => `<span class="card-tag">#${t}</span>`).join('');
        const dateStr = formatDate(note.updatedAt);

        return `
            <div class="note-card ${isActive} ${isPinned}" data-id="${note.id}" onclick="selectNote('${note.id}')">
                <div class="card-top">
                    <h4 class="card-title">${escapeHtml(note.title || 'শিরোনামহীন নোট')}</h4>
                    <div class="card-badges">
                        ${note.isPinned ? '<span>📌</span>' : ''}
                        ${note.isFavorite ? '<span>⭐</span>' : ''}
                    </div>
                </div>
                <p class="card-excerpt">${escapeHtml(excerpt)}</p>
                ${checkSummary}
                <div class="card-footer">
                    <div class="card-tags">${tagsHtml}</div>
                    <span class="card-date">${dateStr}</span>
                </div>
            </div>
        `;
    }).join('');
}

// ─── নোট নির্বাচন ও এডিটর লোড ───────────────────────────────────────
function selectNote(noteId) {
    state.activeNoteId = noteId;
    const note = state.notes.find(n => n.id === noteId);

    const editorContainer = document.getElementById('editorContainer');
    const emptyState = document.getElementById('emptyState');

    if (!note) {
        editorContainer.classList.add('hidden');
        emptyState.classList.remove('hidden');
        return;
    }

    editorContainer.classList.remove('hidden');
    emptyState.classList.add('hidden');

    // পপুলেট ফিল্ডস
    document.getElementById('noteTitleInput').value = note.title || '';
    document.getElementById('noteContentInput').value = note.content || '';
    document.getElementById('noteTagsInput').value = (note.tags || []).join(', ');
    document.getElementById('noteCategorySelect').value = note.category || 'personal';

    // কালার সিলেক্ট
    document.querySelectorAll('.color-dot').forEach(d => {
        d.classList.toggle('active', d.dataset.color === (note.color || 'indigo'));
    });

    // পিন ও ফেভারিট স্টেট
    document.getElementById('pinNoteBtn').classList.toggle('active', !!note.isPinned);
    document.getElementById('favNoteBtn').classList.toggle('active', !!note.isFavorite);

    renderChecklist(note.checklist || []);
    updateMetrics();
    renderMarkdown();
    renderNotesList();
}

// ─── নতুন নোট তৈরি ──────────────────────────────────────────────────
async function createNewNote() {
    const newNote = {
        id: 'note-' + (state.notes.length + 1) + '-' + Math.floor(Date.now() / 1000),
        title: 'নতুন নোট',
        content: '',
        category: state.activeCategory === 'all' || state.activeCategory === 'trash' ? 'personal' : state.activeCategory,
        tags: [],
        color: 'indigo',
        priority: 'medium',
        isPinned: false,
        isFavorite: false,
        isArchived: false,
        isTrashed: false,
        checklist: [],
        createdAt: Math.floor(Date.now() / 1000),
        updatedAt: Math.floor(Date.now() / 1000)
    };

    state.notes.unshift(newNote);
    saveToLocalStorage();

    try {
        await fetch('/api/notes', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(newNote)
        });
    } catch (e) {
        console.log('Saved to local storage');
    }

    updateCountsAndTags();
    selectNote(newNote.id);
    document.getElementById('noteTitleInput').focus();
    document.getElementById('noteTitleInput').select();
}

// ─── অটো সেভ ও আপডেট ───────────────────────────────────────────────
let saveTimeout = null;
function autoSaveCurrentNote() {
    if (!state.activeNoteId) return;

    const note = state.notes.find(n => n.id === state.activeNoteId);
    if (!note) return;

    const title = document.getElementById('noteTitleInput').value.trim();
    const content = document.getElementById('noteContentInput').value;
    const tagInput = document.getElementById('noteTagsInput').value;
    const category = document.getElementById('noteCategorySelect').value;
    const activeColorDot = document.querySelector('.color-dot.active');
    const color = activeColorDot ? activeColorDot.dataset.color : 'indigo';

    const tags = tagInput.split(',').map(t => t.trim()).filter(t => t.length > 0);

    note.title = title || 'শিরোনামহীন নোট';
    note.content = content;
    note.tags = tags;
    note.category = category;
    note.color = color;
    note.updatedAt = Math.floor(Date.now() / 1000);

    const saveStatus = document.getElementById('saveStatus');
    saveStatus.textContent = '⏳ সেভ হচ্ছে...';

    clearTimeout(saveTimeout);
    saveTimeout = setTimeout(async () => {
        saveToLocalStorage();
        try {
            await fetch(`/api/notes/${note.id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(note)
            });
        } catch (e) {}

        saveStatus.textContent = '✔ সংরক্ষিত (Saved)';
        updateCountsAndTags();
        renderNotesList();
    }, 400);
}

// ─── পিন, ফেভারিট ও ডিলিট ──────────────────────────────────────────
async function togglePinCurrentNote() {
    const note = state.notes.find(n => n.id === state.activeNoteId);
    if (!note) return;
    note.isPinned = !note.isPinned;
    document.getElementById('pinNoteBtn').classList.toggle('active', note.isPinned);
    autoSaveCurrentNote();
}

async function toggleFavCurrentNote() {
    const note = state.notes.find(n => n.id === state.activeNoteId);
    if (!note) return;
    note.isFavorite = !note.isFavorite;
    document.getElementById('favNoteBtn').classList.toggle('active', note.isFavorite);
    autoSaveCurrentNote();
}

async function deleteCurrentNote() {
    const note = state.notes.find(n => n.id === state.activeNoteId);
    if (!note) return;

    if (note.isTrashed) {
        // স্থায়ী ডিলিট
        if (confirm('নোটটি স্থায়ীভাবে মুছে ফেলতে চান?')) {
            state.notes = state.notes.filter(n => n.id !== note.id);
            try { await fetch(`/api/notes/${note.id}`, { method: 'DELETE' }); } catch (e) {}
        }
    } else {
        note.isTrashed = true;
        note.isPinned = false;
        try {
            await fetch(`/api/notes/${note.id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(note)
            });
        } catch (e) {}
    }

    saveToLocalStorage();
    updateCountsAndTags();
    state.activeNoteId = null;

    const remaining = getFilteredNotes();
    if (remaining.length > 0) {
        selectNote(remaining[0].id);
    } else {
        selectNote(null);
    }
}

// ─── চেকলিস্ট হ্যান্ডলিং ─────────────────────────────────────────────
function renderChecklist(checklist) {
    const listEl = document.getElementById('checklistItems');
    if (!checklist || checklist.length === 0) {
        listEl.innerHTML = `<div style="font-size: 12px; color: var(--text-dim); padding: 4px 0;">কোনো টাস্ক নেই। নিচে নতুন টাস্ক যোগ করুন।</div>`;
        updateChecklistProgress(0, 0);
        return;
    }

    listEl.innerHTML = checklist.map((item, idx) => `
        <div class="check-item-row" data-id="${item.id}">
            <input type="checkbox" ${item.completed ? 'checked' : ''} onchange="toggleCheckItem('${item.id}')">
            <span class="check-item-text ${item.completed ? 'done' : ''}">${escapeHtml(item.text)}</span>
            <button class="remove-check-btn" onclick="removeCheckItem('${item.id}')">✕</button>
        </div>
    `).join('');

    const done = checklist.filter(c => c.completed).length;
    updateChecklistProgress(done, checklist.length);
}

function updateChecklistProgress(done, total) {
    const percent = total > 0 ? Math.round((done * 100) / total) : 0;
    document.getElementById('checklistProgressText').textContent = `${done}/${total} (${percent}%)`;
    document.getElementById('checklistProgressBar').style.width = `${percent}%`;
}

function addChecklistItem() {
    const note = state.notes.find(n => n.id === state.activeNoteId);
    if (!note) return;

    const input = document.getElementById('newCheckInput');
    const text = input.value.trim();
    if (!text) return;

    if (!note.checklist) note.checklist = [];
    note.checklist.push({
        id: 'chk-' + Date.now(),
        text: text,
        completed: false
    });

    input.value = '';
    renderChecklist(note.checklist);
    autoSaveCurrentNote();
}

function toggleCheckItem(checkId) {
    const note = state.notes.find(n => n.id === state.activeNoteId);
    if (!note || !note.checklist) return;

    const item = note.checklist.find(c => c.id === checkId);
    if (item) {
        item.completed = !item.completed;
        renderChecklist(note.checklist);
        autoSaveCurrentNote();
    }
}

function removeCheckItem(checkId) {
    const note = state.notes.find(n => n.id === state.activeNoteId);
    if (!note || !note.checklist) return;

    note.checklist = note.checklist.filter(c => c.id !== checkId);
    renderChecklist(note.checklist);
    autoSaveCurrentNote();
}

// ─── মার্কডাউন ও প্রিভিউ ──────────────────────────────────────────────
function setEditorTab(tab) {
    state.editorTab = tab;
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));

    const writePane = document.getElementById('writePane');
    const previewPane = document.getElementById('previewPane');

    if (tab === 'write') {
        document.getElementById('tabWriteBtn').classList.add('active');
        writePane.classList.remove('hidden');
        previewPane.classList.add('hidden');
    } else if (tab === 'preview') {
        document.getElementById('tabPreviewBtn').classList.add('active');
        writePane.classList.add('hidden');
        previewPane.classList.remove('hidden');
        renderMarkdown();
    } else if (tab === 'split') {
        document.getElementById('tabSplitBtn').classList.add('active');
        writePane.classList.remove('hidden');
        previewPane.classList.remove('hidden');
        renderMarkdown();
    }
}

function renderMarkdown() {
    const raw = document.getElementById('noteContentInput').value;
    const previewEl = document.getElementById('markdownPreview');
    previewEl.innerHTML = parseMarkdownSimple(raw);
}

function parseMarkdownSimple(text) {
    if (!text) return '<em>খালি প্রিভিউ</em>';

    let html = escapeHtml(text);

    // হেডিংস
    html = html.replace(/^### (.*$)/gim, '<h3>$1</h3>');
    html = html.replace(/^## (.*$)/gim, '<h2>$1</h2>');
    html = html.replace(/^# (.*$)/gim, '<h1>$1</h1>');

    // বোল্ড ও ইটালিক
    html = html.replace(/\*\*(.*?)\*\*/gim, '<strong>$1</strong>');
    html = html.replace(/\*(.*?)\*/gim, '<em>$1</em>');

    // কোড ব্লক
    html = html.replace(/```([\s\S]*?)```/gim, '<pre><code>$1</code></pre>');
    html = html.replace(/`([^`]+)`/gim, '<code>$1</code>');

    // লিস্ট
    html = html.replace(/^\- (.*$)/gim, '<ul><li>$1</li></ul>');
    html = html.replace(/<\/ul>\s?<ul>/gim, '');

    // প্যারাগ্রাফ
    html = html.replace(/\n\n/gim, '<p></p>');
    html = html.replace(/\n/gim, '<br>');

    return html;
}

// ─── টেক্সট মেট্রিক্স ────────────────────────────────────────────────
function updateMetrics() {
    const title = document.getElementById('noteTitleInput').value;
    const content = document.getElementById('noteContentInput').value;
    const fullText = (title + ' ' + content).trim();

    const words = fullText.length > 0 ? fullText.split(/\s+/).filter(w => w.length > 0).length : 0;
    const chars = fullText.length;
    const readingTime = words < 50 ? '< ১ মিনিট' : `${Math.ceil(words / 180)} মিনিট`;

    document.getElementById('wordCountBadge').textContent = `${words} শব্দ`;
    document.getElementById('charCountBadge').textContent = `${chars} অক্ষর`;
    document.getElementById('readingTimeBadge').textContent = `পড়ার সময়: ${readingTime}`;
}

// ─── কাউন্টস ও ট্যাগ ক্লাউড ──────────────────────────────────────────
function updateCountsAndTags() {
    const counts = { all: 0, important: 0, work: 0, personal: 0, ideas: 0, study: 0, todo: 0, archive: 0, trash: 0 };
    const tagSet = new Set();

    state.notes.forEach(n => {
        if (n.isTrashed) {
            counts.trash++;
        } else if (n.isArchived) {
            counts.archive++;
        } else {
            counts.all++;
            if (n.isPinned || n.isFavorite) counts.important++;
            if (counts[n.category] !== undefined) counts[n.category]++;

            if (n.tags) {
                n.tags.forEach(t => tagSet.add(t));
            }
        }
    });

    for (const [cat, count] of Object.entries(counts)) {
        const el = document.getElementById(`count-${cat}`);
        if (el) el.textContent = count;
    }

    // ট্যাগ ক্লাউড রেন্ডার
    const tagCloud = document.getElementById('tagCloud');
    if (tagSet.size === 0) {
        tagCloud.innerHTML = `<span style="font-size: 11px; color: var(--text-dim);">কোনো ট্যাগ নেই</span>`;
    } else {
        tagCloud.innerHTML = Array.from(tagSet).map(t => {
            const active = state.activeTag === t ? 'active' : '';
            return `<span class="tag-pill ${active}" onclick="filterByTag('${t}')">#${escapeHtml(t)}</span>`;
        }).join('');
    }
}

function filterByTag(tag) {
    if (state.activeTag === tag) {
        state.activeTag = null;
    } else {
        state.activeTag = tag;
    }
    updateCountsAndTags();
    renderNotesList();
}

// ─── থিম পরিবর্তন ────────────────────────────────────────────────────
function setTheme(themeClass) {
    document.body.className = themeClass;
    state.currentTheme = themeClass;
    localStorage.setItem('nilang_notes_theme', themeClass);
    document.getElementById('themeSelector').value = themeClass;
}

function loadTheme() {
    const saved = localStorage.getItem('nilang_notes_theme') || 'theme-dark';
    setTheme(saved);
}

// ─── এক্সপোর্ট মার্কডাউন ─────────────────────────────────────────────
function exportCurrentNoteAsMarkdown() {
    const note = state.notes.find(n => n.id === state.activeNoteId);
    if (!note) return;

    let md = `# ${note.title}\n\n`;
    md += `*ক্যাটাগরি: ${note.category} | আপডেট: ${new Date(note.updatedAt * 1000).toLocaleString('bn-BD')}*\n\n`;
    if (note.tags && note.tags.length > 0) {
        md += `**ট্যাগসমূহ:** ${note.tags.map(t => '#' + t).join(' ')}\n\n`;
    }
    md += `## বিবরণ\n${note.content}\n\n`;

    if (note.checklist && note.checklist.length > 0) {
        md += `## চেকলিস্ট\n`;
        note.checklist.forEach(c => {
            md += `- [${c.completed ? 'x' : ' '}] ${c.text}\n`;
        });
    }

    const blob = new Blob([md], { type: 'text/markdown;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${note.title.replace(/[^\w\u0980-\u09FF]/g, '_')}.md`;
    a.click();
}

// ─── মোডালস ──────────────────────────────────────────────────────────
function openStatsModal() {
    const modal = document.getElementById('statsModal');
    const body = document.getElementById('statsModalBody');

    let totalNotes = state.notes.length;
    let active = state.notes.filter(n => !n.isTrashed && !n.isArchived).length;
    let pinned = state.notes.filter(n => n.isPinned).length;
    let totalTodos = 0;
    let doneTodos = 0;
    let totalWords = 0;

    state.notes.forEach(n => {
        const text = (n.title + ' ' + n.content).trim();
        totalWords += text.length > 0 ? text.split(/\s+/).length : 0;
        if (n.checklist) {
            totalTodos += n.checklist.length;
            doneTodos += n.checklist.filter(c => c.completed).length;
        }
    });

    const todoPct = totalTodos > 0 ? Math.round((doneTodos * 100) / totalTodos) : 0;

    body.innerHTML = `
        <div class="stats-grid">
            <div class="stat-box">
                <span class="stat-val">${totalNotes}</span>
                <span class="stat-lbl">📑 মোট নোট</span>
            </div>
            <div class="stat-box">
                <span class="stat-val">${active}</span>
                <span class="stat-lbl">🟢 সক্রিয় নোট</span>
            </div>
            <div class="stat-box">
                <span class="stat-val">${pinned}</span>
                <span class="stat-lbl">📌 পিন্ড নোট</span>
            </div>
            <div class="stat-box">
                <span class="stat-val">${totalWords}</span>
                <span class="stat-lbl">✍️ সর্বমোট শব্দ</span>
            </div>
            <div class="stat-box" style="grid-column: span 2;">
                <span class="stat-val">${doneTodos} / ${totalTodos} (${todoPct}%)</span>
                <span class="stat-lbl">✅ চেকলিস্ট টাস্ক সম্পন্ন</span>
            </div>
        </div>
    `;

    modal.classList.remove('hidden');
}

function closeStatsModal() {
    document.getElementById('statsModal').classList.add('hidden');
}

function openQemuModal() {
    document.getElementById('qemuModal').classList.remove('hidden');
}

function closeQemuModal() {
    document.getElementById('qemuModal').classList.add('hidden');
}

// ─── QEMU লঞ্চ হ্যান্ডলার ─────────────────────────────────────────────
async function launchQEMU() {
    const consoleEl = document.getElementById('qemuConsole');
    consoleEl.innerHTML += `<div class="console-line" style="color: #38BDF8;">[LAUNCH] Initializing QEMU Windows Subsystem...</div>`;
    consoleEl.innerHTML += `<div class="console-line" style="color: #FBBF24;">[CMD] qemu-system-x86_64.exe -m 512M -drive format=raw,file=qemu/notes-os.img</div>`;

    try {
        const res = await fetch('/api/qemu/launch', { method: 'POST' });
        const data = await res.json();
        if (data.success) {
            consoleEl.innerHTML += `<div class="console-line" style="color: #4ADE80;">[SUCCESS] QEMU Window Launched (PID: ${data.pid})!</div>`;
            consoleEl.innerHTML += `<div class="console-line" style="color: #E2E8F0;">[TIP] QEMU উইন্ডো ওপেন হয়েছে। ভার্চুয়াল মেশিনে নোট অ্যাপ টেস্ট করুন।</div>`;
        } else {
            consoleEl.innerHTML += `<div class="console-line" style="color: #F87171;">[ERROR] ${data.error || 'QEMU চালু ব্যর্থ'}</div>`;
        }
    } catch (e) {
        consoleEl.innerHTML += `<div class="console-line" style="color: #F87171;">[NET_ERR] লোকাল সার্ভার রেসপন্স দেয়নি। PowerShell দিয়ে ম্যানুয়ালি চালাতে পারেন: .\\qemu\\qemu-run.ps1</div>`;
    }
    consoleEl.scrollTop = consoleEl.scrollHeight;
}

async function runNilangTest() {
    const consoleEl = document.getElementById('qemuConsole');
    consoleEl.innerHTML += `<div class="console-line" style="color: #38BDF8;">[NILANG] Executing 'nil run src/main.nil'...</div>`;

    try {
        const res = await fetch('/api/nilang/run', { method: 'POST' });
        const data = await res.json();
        if (data.success) {
            consoleEl.innerHTML += `<div class="console-line" style="color: #4ADE80;">[OUTPUT] Nilang Interpreter Executed Successfully:</div>`;
            consoleEl.innerHTML += `<pre style="font-size: 11px; color: #CBD5E1; max-height: 120px; overflow-y: auto;">${escapeHtml(data.output)}</pre>`;
        } else {
            consoleEl.innerHTML += `<div class="console-line" style="color: #F87171;">[OUTPUT] ${escapeHtml(data.output || data.error)}</div>`;
        }
    } catch (e) {
        consoleEl.innerHTML += `<div class="console-line" style="color: #F87171;">[ERR] Nilang রান ব্যর্থ।</div>`;
    }
    consoleEl.scrollTop = consoleEl.scrollHeight;
}

// ─── হেল্পার্স ───────────────────────────────────────────────────────
function formatDate(timestamp) {
    if (!timestamp) return 'নতুন';
    const diff = Math.floor(Date.now() / 1000) - timestamp;
    if (diff < 60) return 'এইমাত্র';
    if (diff < 3600) return `${Math.floor(diff / 60)} মি. আগে`;
    if (diff < 86400) return `${Math.floor(diff / 3600)} ঘণ্টা আগে`;
    return `${Math.floor(diff / 86400)} দিন আগে`;
}

function escapeHtml(str) {
    if (!str) return '';
    return str
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}
