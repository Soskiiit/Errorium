/**
 * Errorium Single Page Application (Pure Vanilla JS, Zero Frameworks)
 * Implements chat, search, drawer navigation, MCP modal, and responsive interactions
 */

(function () {
    'use strict';

    // =========================================================================
    // Preset Knowledge Base for Default Recent Chats
    // =========================================================================
    const KNOWLEDGE_BASE = {
        'chat-1': {
            title: 'SyntaxError: invalid syntax',
            query: 'SyntaxError: invalid syntax',
            solution: 'This error occurs when Python parser encounters code that violates language grammar rules. The most common causes are missing colons after control flow statements (`if`, `for`, `def`), unclosed parentheses or quotes, or using reserved keywords as identifiers.',
            code: '# ❌ Incorrect:\ndef calculate_total(items)\n    total = 0\n    return total\n\n# ✅ Fixed (add colon and check matching braces):\ndef calculate_total(items):\n    total = 0\n    return total',
            sources: [
                {
                    title: 'SyntaxError: invalid syntax in Python (Top causes & fixes)',
                    score: 4180,
                    accepted: true,
                    tags: ['python', 'syntax-error', 'python-3.x'],
                    url: 'https://stackoverflow.com/questions/27539308'
                },
                {
                    title: 'Why am I getting SyntaxError on the line after my print statement?',
                    score: 1240,
                    accepted: true,
                    tags: ['python', 'print-statement'],
                    url: 'https://stackoverflow.com/questions/15478423'
                }
            ]
        },
        'chat-2': {
            title: 'IndentationError: unexpected indent',
            query: 'IndentationError: unexpected indent',
            solution: 'Python uses indentation to define code blocks. An `unexpected indent` occurs when a line has more leading whitespace or tabs than expected, or when spaces and tabs are mixed within the same file.',
            code: '# ❌ Incorrect:\ndef process_data():\n    step1 = True\n      step2 = False  # Unexpected indentation level\n\n# ✅ Fixed (use consistent 4 spaces per block):\ndef process_data():\n    step1 = True\n    step2 = False',
            sources: [
                {
                    title: 'How to fix Python IndentationError: unexpected indent',
                    score: 2890,
                    accepted: true,
                    tags: ['python', 'indentation', 'pep8'],
                    url: 'https://stackoverflow.com/questions/11256501'
                }
            ]
        },
        'chat-3': {
            title: 'TypeError: ‘list’ object cannot be interpre...',
            query: "TypeError: 'list' object cannot be interpreted as an integer",
            solution: 'This error happens when passing a list to a function or slice index that expects an integer (e.g. `range(my_list)` instead of `range(len(my_list))` or passing an array into `int()` conversion).',
            code: '# ❌ Incorrect:\nitems = [10, 20, 30]\nfor i in range(items):  # Passing list instead of length\n    print(i)\n\n# ✅ Fixed:\nfor i in range(len(items)):\n    print(items[i])\n\n# Or idiomatically:\nfor item in items:\n    print(item)',
            sources: [
                {
                    title: 'TypeError: \'list\' object cannot be interpreted as an integer in range()',
                    score: 1950,
                    accepted: true,
                    tags: ['python', 'list', 'typeerror'],
                    url: 'https://stackoverflow.com/questions/35593121'
                }
            ]
        },
        'chat-4': {
            title: 'NameError: name ‘foobar’ is not defined',
            query: "NameError: name 'foobar' is not defined",
            solution: 'A variable, function, or module name was referenced before it was declared, imported, or within an inaccessible scope.',
            code: '# ❌ Incorrect:\nprint(foobar)\nfoobar = "active"\n\n# ✅ Fixed:\nfoobar = "active"\nprint(foobar)',
            sources: [
                {
                    title: 'NameError: name is not defined in Python',
                    score: 3410,
                    accepted: true,
                    tags: ['python', 'scope', 'name-error'],
                    url: 'https://stackoverflow.com/questions/14872225'
                }
            ]
        },
        'chat-5': {
            title: 'UnboundLocalError: local variable ‘foobar’...',
            query: "UnboundLocalError: local variable 'foobar' referenced before assignment",
            solution: 'Occurs inside a function when assigning to a variable that has the same name as an outer/global variable, making Python treat it as local throughout the function before its assignment statement is reached.',
            code: 'foobar = 10\n\ndef increment():\n    # ❌ foobar += 1  # Raises UnboundLocalError\n    global foobar     # ✅ Explicitly declare global or return new value\n    foobar += 1\n\nincrement()',
            sources: [
                {
                    title: 'UnboundLocalError: local variable referenced before assignment',
                    score: 5120,
                    accepted: true,
                    tags: ['python', 'variable-assignment', 'scope'],
                    url: 'https://stackoverflow.com/questions/10851906'
                }
            ]
        },
        'chat-6': {
            title: 'TypeError: ‘str’ object does not support it...',
            query: "TypeError: 'str' object does not support item assignment",
            solution: 'Strings in Python are immutable objects. You cannot modify an individual character in-place using indexing (`s[0] = "x"`). You must construct a new string or convert to a mutable sequence like a list.',
            code: 'text = "hello"\n\n# ❌ text[0] = "H"  # TypeError\n\n# ✅ Fixed (slice and concatenate):\ntext = "H" + text[1:]\n\n# Or using list conversion:\nchars = list(text)\nchars[0] = "H"\ntext = "".join(chars)',
            sources: [
                {
                    title: 'How to replace a character in a Python string by index?',
                    score: 3740,
                    accepted: true,
                    tags: ['python', 'string', 'immutability'],
                    url: 'https://stackoverflow.com/questions/4175294'
                }
            ]
        },
        'chat-7': {
            title: 'KeyError: ‘spam’',
            query: "KeyError: 'spam'",
            solution: 'Raised when trying to access a dictionary key that does not exist. Use the `.get(key, default)` method or `in` operator to safely verify membership before reading.',
            code: 'data = {"name": "Alice"}\n\n# ❌ value = data["spam"]  # Raises KeyError\n\n# ✅ Fixed:\nvalue = data.get("spam", "default_val")\n\n# Or with membership check:\nif "spam" in data:\n    print(data["spam"])',
            sources: [
                {
                    title: 'How to handle KeyError exceptions when accessing Python dictionaries',
                    score: 2210,
                    accepted: true,
                    tags: ['python', 'dictionary', 'keyerror'],
                    url: 'https://stackoverflow.com/questions/11041405'
                }
            ]
        }
    };

    // =========================================================================
    // DOM Elements Cache
    // =========================================================================
    const elements = {
        app: document.getElementById('app'),
        backdrop: document.getElementById('backdrop'),
        sidebar: document.getElementById('sidebar'),
        sidebarBrand: document.getElementById('sidebar-brand'),
        btnMenu: document.getElementById('btn-menu'),
        btnToggleSidebar: document.getElementById('btn-toggle-sidebar'),
        btnExpandSidebar: document.getElementById('btn-expand-sidebar'),
        btnCloseSidebar: document.getElementById('btn-close-sidebar'),
        headerLogo: document.getElementById('header-logo'),
        btnProfile: document.getElementById('btn-profile'),
        btnLoginRow: document.getElementById('btn-login-row'),
        btnNewChat: document.getElementById('btn-new-chat'),
        btnMcp: document.getElementById('btn-mcp'),
        btnAbout: document.getElementById('btn-about'),
        btnSettings: document.getElementById('btn-settings'),

        searchRecentChats: document.getElementById('search-recent-chats'),
        recentChatsList: document.getElementById('recent-chats-list'),
        recentChatsToggle: document.getElementById('recent-chats-toggle'),
        recentChatsContainer: document.getElementById('recent-chats-container'),

        heroView: document.getElementById('hero-view'),
        chatView: document.getElementById('chat-view'),
        messagesContainer: document.getElementById('messages-container'),

        promptInput: document.getElementById('prompt-input'),
        btnSend: document.getElementById('btn-send'),
        btnAttach: document.getElementById('btn-attach'),
        attachPopover: document.getElementById('attach-popover'),

        // Modals
        modalAuth: document.getElementById('modal-auth'),
        modalMcp: document.getElementById('modal-mcp'),
        modalAbout: document.getElementById('modal-about'),
        modalSettings: document.getElementById('modal-settings'),
        userDisplayName: document.getElementById('user-display-name'),

        btnCopyToken: document.getElementById('btn-copy-token'),
        btnCopyMcpJson: document.getElementById('btn-copy-mcp-json'),
        mcpTokenDisplay: document.getElementById('mcp-token-display'),
        mcpJsonSnippet: document.getElementById('mcp-json-snippet'),

        // Auth Form elements (Auth-screen.png)
        authForm: document.getElementById('auth-form'),
        authEmail: document.getElementById('auth-email'),
        authPassword: document.getElementById('auth-password'),
        authRemember: document.getElementById('auth-remember'),
        authForgotPassword: document.getElementById('auth-forgot-password'),
        btnLoginSubmit: document.getElementById('btn-login-submit'),
        btnLoginYandex: document.getElementById('btn-login-yandex'),
        authSignupLink: document.getElementById('auth-signup-link'),

        toast: document.getElementById('toast')
    };

    // =========================================================================
    // Sidebar Drawer & Desktop Collapse Controls
    // =========================================================================
    const SIDEBAR_STATE_KEY = 'errorium_sidebar_state';

    // Clear legacy boolean key if present to prevent accidental default collapsed state
    try {
        localStorage.removeItem('errorium_sidebar_collapsed');
    } catch (e) {}

    function isWideScreen() {
        return window.innerWidth >= 768;
    }

    function openSidebar() {
        if (isWideScreen()) {
            expandDesktopSidebar(true);
            return;
        }
        elements.sidebar.classList.add('open');
        elements.backdrop.classList.add('active');
        document.body.style.overflow = 'hidden';
    }

    function closeSidebar() {
        elements.sidebar.classList.remove('open');
        elements.backdrop.classList.remove('active');
        document.body.style.overflow = '';
    }

    function collapseDesktopSidebar(saveState = true) {
        document.documentElement.classList.add('sidebar-collapsed');
        elements.app.classList.add('sidebar-collapsed');
        elements.sidebar.classList.add('collapsed');
        if (saveState) {
            try {
                localStorage.setItem(SIDEBAR_STATE_KEY, 'collapsed');
            } catch (e) {}
        }
    }

    function expandDesktopSidebar(saveState = true) {
        document.documentElement.classList.remove('sidebar-collapsed');
        elements.app.classList.remove('sidebar-collapsed');
        elements.sidebar.classList.remove('collapsed');
        if (saveState) {
            try {
                localStorage.setItem(SIDEBAR_STATE_KEY, 'expanded');
            } catch (e) {}
        }
    }

    function toggleDesktopSidebar() {
        if (elements.app.classList.contains('sidebar-collapsed')) {
            expandDesktopSidebar(true);
        } else {
            collapseDesktopSidebar(true);
        }
    }

    // Default state: initially EXPANDED (open by default).
    // Only if user explicitly collapsed it and saved preference, restore collapsed state.
    try {
        const savedState = localStorage.getItem(SIDEBAR_STATE_KEY);
        if (savedState === 'collapsed' && isWideScreen()) {
            collapseDesktopSidebar(false);
        } else {
            expandDesktopSidebar(false);
        }
    } catch (e) {
        expandDesktopSidebar(false);
    }

    // Safely remove preload class after first paint to re-enable smooth interactive transitions
    function removePreload() {
        requestAnimationFrame(() => {
            setTimeout(() => {
                document.body.classList.remove('preload');
            }, 60);
        });
    }

    if (document.readyState === 'loading') {
        window.addEventListener('DOMContentLoaded', removePreload);
    } else {
        removePreload();
    }

    elements.btnToggleSidebar?.addEventListener('click', toggleDesktopSidebar);
    elements.btnExpandSidebar?.addEventListener('click', toggleDesktopSidebar);

    elements.btnMenu?.addEventListener('click', () => {
        if (isWideScreen()) {
            toggleDesktopSidebar();
        } else {
            openSidebar();
        }
    });

    elements.btnCloseSidebar?.addEventListener('click', closeSidebar);
    elements.backdrop?.addEventListener('click', () => {
        closeSidebar();
        closeAllModals();
    });

    // Recent chats search filter
    elements.searchRecentChats?.addEventListener('input', function (e) {
        const query = e.target.value.toLowerCase().trim();
        const items = elements.recentChatsList ? elements.recentChatsList.querySelectorAll('.recent-chat-item') : [];
        items.forEach(item => {
            const title = item.querySelector('.chat-item-title').textContent.toLowerCase();
            if (title.includes(query)) {
                item.style.display = 'flex';
            } else {
                item.style.display = 'none';
            }
        });
    });

    // Recent chats collapse toggle
    elements.recentChatsToggle?.addEventListener('click', function () {
        const isCollapsed = this.classList.toggle('collapsed');
        if (elements.recentChatsContainer) {
            elements.recentChatsContainer.style.display = isCollapsed ? 'none' : '';
        }
    });

    // =========================================================================
    // Modals Controls
    // =========================================================================
    function showModal(modalEl) {
        closeSidebar();
        closeAllModals();
        if (modalEl) {
            modalEl.style.display = 'flex';
        }
    }

    function closeAllModals() {
        document.querySelectorAll('.modal-overlay').forEach(modal => {
            modal.style.display = 'none';
        });
        if (elements.attachPopover) {
            elements.attachPopover.style.display = 'none';
        }
    }

    // Modal Close buttons
    document.querySelectorAll('[data-close-modal]').forEach(btn => {
        btn.addEventListener('click', closeAllModals);
    });

    // Overlay click outside modal card closes modal
    document.querySelectorAll('.modal-overlay').forEach(overlay => {
        overlay.addEventListener('click', function (e) {
            if (e.target === overlay) {
                closeAllModals();
            }
        });
    });

    // Modal Triggers
    elements.btnMcp?.addEventListener('click', () => showModal(elements.modalMcp));
    elements.btnAbout?.addEventListener('click', () => showModal(elements.modalAbout));
    elements.btnSettings?.addEventListener('click', () => showModal(elements.modalSettings));
    elements.btnProfile?.addEventListener('click', () => showModal(elements.modalAuth));
    elements.btnLoginRow?.addEventListener('click', () => showModal(elements.modalAuth));

    // Auth Handlers (Auth-screen.png)
    elements.authForm?.addEventListener('submit', function (e) {
        e.preventDefault();
        const email = elements.authEmail?.value?.trim() || 'dev@errorium.ru';
        handleLogin('Account', email);
    });

    elements.btnLoginSubmit?.addEventListener('click', function (e) {
        if (elements.authForm && !elements.authForm.checkValidity()) {
            elements.authForm.reportValidity();
            return;
        }
        const email = elements.authEmail?.value?.trim() || 'dev@errorium.ru';
        handleLogin('Account', email);
    });

    elements.btnLoginYandex?.addEventListener('click', function () {
        handleLogin('Yandex ID', 'yandex.dev@errorium.ru');
    });

    elements.authForgotPassword?.addEventListener('click', function (e) {
        e.preventDefault();
        showToast('Password reset link sent to your email.');
    });

    elements.authSignupLink?.addEventListener('click', function (e) {
        e.preventDefault();
        showToast('Sign up is open! Enter your email and password to log in.');
    });

    function handleLogin(provider, userEmail = 'dev@errorium.ru') {
        if (elements.userDisplayName) {
            elements.userDisplayName.textContent = userEmail;
        }
        closeAllModals();
        showToast(`Signed in via ${provider}! MCP tokens active.`);
    }

    // Copy to clipboard helpers
    elements.btnCopyToken?.addEventListener('click', () => {
        if (elements.mcpTokenDisplay) {
            navigator.clipboard.writeText(elements.mcpTokenDisplay.value).then(() => {
                showToast('Bearer token copied to clipboard!');
            });
        }
    });

    elements.btnCopyMcpJson?.addEventListener('click', () => {
        if (elements.mcpJsonSnippet) {
            navigator.clipboard.writeText(elements.mcpJsonSnippet.textContent).then(() => {
                showToast('MCP configuration JSON copied!');
            });
        }
    });

    // Toast helper
    let toastTimeout = null;
    function showToast(message) {
        if (toastTimeout) clearTimeout(toastTimeout);
        elements.toast.textContent = message;
        elements.toast.style.display = 'block';
        toastTimeout = setTimeout(() => {
            elements.toast.style.display = 'none';
        }, 2600);
    }

    // =========================================================================
    // Auto-expanding Input & Send Logic
    // =========================================================================
    function adjustTextareaHeight() {
        elements.promptInput.style.height = 'auto';
        elements.promptInput.style.height = Math.min(elements.promptInput.scrollHeight, 140) + 'px';
    }

    elements.promptInput?.addEventListener('input', adjustTextareaHeight);

    elements.promptInput?.addEventListener('keydown', function (e) {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            submitCurrentQuery();
        }
    });

    elements.btnSend?.addEventListener('click', submitCurrentQuery);

    // Attach button popover toggle
    elements.btnAttach?.addEventListener('click', function (e) {
        e.stopPropagation();
        if (elements.attachPopover) {
            const isShown = elements.attachPopover.style.display === 'flex';
            elements.attachPopover.style.display = isShown ? 'none' : 'flex';
        }
    });

    // Popover quick items
    document.querySelectorAll('.popover-item').forEach(item => {
        item.addEventListener('click', function () {
            const action = this.getAttribute('data-action');
            if (action === 'paste-trace') {
                if (elements.promptInput) elements.promptInput.value = 'Traceback (most recent call last):\n  File "main.py", line 42, in <module>\n    process_items(data)\nTypeError: unsupported operand type(s)';
            } else if (action === 'filter-lang') {
                const lang = this.getAttribute('data-lang');
                if (elements.promptInput) elements.promptInput.value = `[${lang}] `;
            }
            if (elements.attachPopover) elements.attachPopover.style.display = 'none';
            adjustTextareaHeight();
            elements.promptInput?.focus();
        });
    });

    document.addEventListener('click', function (e) {
        if (elements.attachPopover && !elements.attachPopover.contains(e.target) && e.target !== elements.btnAttach) {
            elements.attachPopover.style.display = 'none';
        }
    });

    // =========================================================================
    // Core Chat / Query Interaction
    // =========================================================================
    function submitCurrentQuery() {
        if (!elements.promptInput) return;
        const query = elements.promptInput.value.trim();
        if (!query) return;

        elements.promptInput.value = '';
        adjustTextareaHeight();
        loadChatByQuery(query);
    }

    function switchToChatView() {
        if (elements.heroView) elements.heroView.style.display = 'none';
        if (elements.chatView) elements.chatView.style.display = 'flex';
    }

    function switchToHeroView() {
        if (elements.chatView) elements.chatView.style.display = 'none';
        if (elements.heroView) elements.heroView.style.display = 'flex';
        if (elements.messagesContainer) elements.messagesContainer.innerHTML = '';
        if (elements.promptInput) elements.promptInput.value = '';
        adjustTextareaHeight();
        document.querySelectorAll('.recent-chat-item').forEach(i => i.classList.remove('active'));
    }

    elements.btnNewChat?.addEventListener('click', () => {
        switchToHeroView();
        closeSidebar();
    });

    elements.headerLogo?.addEventListener('click', () => {
        switchToHeroView();
    });

    elements.sidebarBrand?.addEventListener('click', () => {
        switchToHeroView();
    });

    // Render Chat Messages
    function loadChatByQuery(query, chatId = null) {
        switchToChatView();
        closeSidebar();

        // Mark active in sidebar if known
        document.querySelectorAll('.recent-chat-item').forEach(item => {
            if (chatId && item.getAttribute('data-chat-id') === chatId) {
                item.classList.add('active');
            } else {
                item.classList.remove('active');
            }
        });

        // Resolve data from KB or synthesize on-the-fly
        let data = chatId ? KNOWLEDGE_BASE[chatId] : null;
        if (!data) {
            // Find match in KB by fuzzy title
            for (const key in KNOWLEDGE_BASE) {
                if (KNOWLEDGE_BASE[key].title.toLowerCase().includes(query.toLowerCase()) ||
                    query.toLowerCase().includes(KNOWLEDGE_BASE[key].title.toLowerCase())) {
                    data = KNOWLEDGE_BASE[key];
                    break;
                }
            }
        }

        if (!data) {
            data = {
                title: query,
                query: query,
                solution: `Analyzed error with BM25 inverted index ranking ($k_1=1.5, b=0.75$). The exception indicates an invalid runtime operation or unresolved reference in your execution environment.`,
                code: `// Diagnostic verification:\ntry {\n    // Execute target operation\n    verifyInputParameters();\n} catch (Exception ex) {\n    System.err.println("Resolved: " + ex.getMessage());\n}`,
                sources: [
                    {
                        title: `Stack Overflow: Solutions and discussions for "${escapeHtml(query)}"`,
                        score: 1840,
                        accepted: true,
                        tags: ['error', 'debugging', 'stack-trace'],
                        url: 'https://stackoverflow.com/search?q=' + encodeURIComponent(query)
                    }
                ]
            };
        }

        renderConversation(query, data);
    }

    function renderConversation(userText, assistantData) {
        elements.messagesContainer.innerHTML = `
            <!-- User Message -->
            <div class="message-row user">
                <div class="user-bubble">${escapeHtml(userText)}</div>
            </div>

            <!-- Assistant Solution Card -->
            <div class="message-row assistant">
                <div class="assistant-card">
                    <div class="card-top-bar">
                        <div class="badge-mascot">
                            <img src="/mascot.png" alt="Mascot" width="22">
                        </div>
                        <span class="card-title">Errorium RAG Synthesis (BM25 Core)</span>
                    </div>

                    <div class="card-solution-text">
                        ${assistantData.solution}
                    </div>

                    ${assistantData.code ? `
                    <div class="code-block-wrapper">
                        <div class="code-block-header">
                            <span>Solution Code</span>
                            <button class="code-copy-btn" onclick="copySnippet(this)">Copy code</button>
                        </div>
                        <pre><code>${escapeHtml(assistantData.code)}</code></pre>
                    </div>
                    ` : ''}

                    <div class="sources-card">
                        <div class="sources-header" onclick="toggleSources(this)">
                            <span>Ranked Stack Overflow Sources (${assistantData.sources.length})</span>
                            <span class="sources-arrow">▾</span>
                        </div>
                        <div class="sources-list">
                            ${assistantData.sources.map(src => `
                                <a href="${src.url}" target="_blank" rel="noopener noreferrer" class="source-item">
                                    <div class="source-left">
                                        <span class="source-score">▲ ${src.score}</span>
                                        <span class="source-title">${src.title}</span>
                                    </div>
                                    <span style="font-size: 0.78rem; color: #64748b;">${src.tags.map(t => '#' + t).join(' ')}</span>
                                </a>
                            `).join('')}
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    // Attach click listeners to default recent chat items
    document.querySelectorAll('.recent-chat-item').forEach(item => {
        item.addEventListener('click', function (e) {
            // Ignore clicking 3 dots button
            if (e.target.closest('.chat-item-more')) {
                e.stopPropagation();
                showToast('Chat options: Rename / Pin / Remove');
                return;
            }
            const chatId = this.getAttribute('data-chat-id');
            const query = this.getAttribute('data-query');
            loadChatByQuery(query, chatId);
        });
    });

    // Escape HTML helper
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Window helpers for dynamically rendered HTML
    window.copySnippet = function (btn) {
        const pre = btn.closest('.code-block-wrapper').querySelector('pre');
        if (pre) {
            navigator.clipboard.writeText(pre.textContent).then(() => {
                const originalText = btn.textContent;
                btn.textContent = 'Copied!';
                setTimeout(() => { btn.textContent = originalText; }, 2000);
            });
        }
    };

    window.toggleSources = function (header) {
        const list = header.nextElementSibling;
        const arrow = header.querySelector('.sources-arrow');
        if (list.style.display === 'none') {
            list.style.display = 'flex';
            arrow.textContent = '▾';
        } else {
            list.style.display = 'none';
            arrow.textContent = '▸';
        }
    };

    // Global keyboard shortcuts (ESC to close modals/sidebar, Ctrl/Cmd+B to toggle sidebar)
    document.addEventListener('keydown', function (e) {
        if (e.key === 'Escape') {
            closeAllModals();
            closeSidebar();
        } else if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'b') {
            e.preventDefault();
            if (isWideScreen()) {
                toggleDesktopSidebar();
            } else {
                elements.sidebar.classList.contains('open') ? closeSidebar() : openSidebar();
            }
        }
    });

})();
