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

        // Authorized Sidebar & Header
        sidebarUserGuest: document.getElementById('sidebar-user-guest'),
        sidebarUserAuth: document.getElementById('sidebar-user-auth'),
        btnAuthUserRow: document.getElementById('btn-auth-user-row'),
        sidebarAuthAvatar: document.getElementById('sidebar-auth-avatar'),
        sidebarAuthName: document.getElementById('sidebar-auth-name'),
        sidebarAuthRole: document.getElementById('sidebar-auth-role'),
        btnSettingsAuth: document.getElementById('btn-settings-auth'),
        headerAvatarGuest: document.getElementById('header-avatar-guest'),
        headerAvatarAuth: document.getElementById('header-avatar-auth'),

        // Profile Screen Elements
        profileView: document.getElementById('profile-view'),
        btnProfileBack: document.getElementById('btn-profile-back'),
        profileAvatarLarge: document.getElementById('profile-avatar-large'),
        profileAvatarLetter: document.getElementById('profile-avatar-letter'),
        profileAvatarEditBadge: document.getElementById('profile-avatar-edit-badge'),
        profileUsername: document.getElementById('profile-username'),
        profileInfoUsername: document.getElementById('profile-info-username'),
        profileInfoEmail: document.getElementById('profile-info-email'),
        profileInfoRole: document.getElementById('profile-info-role'),
        profileInfoGender: document.getElementById('profile-info-gender'),
        profileInfoGenderRow: document.getElementById('profile-info-gender-row'),
        profileRowSecurity: document.getElementById('profile-row-security'),
        profileInfoLang: document.getElementById('profile-info-lang'),

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
        tagsPopover: document.getElementById('tags-popover'),

        // Modals
        modalAuth: document.getElementById('modal-auth'),
        modalMcp: document.getElementById('modal-mcp'),
        modalAbout: document.getElementById('modal-about'),
        modalSettings: document.getElementById('modal-settings'),
        userDisplayName: document.getElementById('user-display-name'),

        // Reference Modals (services/frontend/references/)
        modalAuthRef: document.getElementById('modal-auth-reference'),
        refAuthForm: document.getElementById('ref-auth-form'),
        refAuthEmail: document.getElementById('ref-auth-email'),
        refAuthPassword: document.getElementById('ref-auth-password'),
        refAuthRemember: document.getElementById('ref-auth-remember'),
        refAuthForgot: document.getElementById('ref-auth-forgot'),
        refBtnLogin: document.getElementById('ref-btn-login'),
        refBtnYandex: document.getElementById('ref-btn-yandex'),
        refLinkSignup: document.getElementById('ref-link-signup'),

        modalChangePasswordEmail: document.getElementById('modal-change-password-email'),
        refResetEmailForm: document.getElementById('ref-reset-email-form'),
        refResetEmail: document.getElementById('ref-reset-email'),
        refBtnSendReset: document.getElementById('ref-btn-send-reset'),

        modalChangePassword: document.getElementById('modal-change-password'),
        refChangePasswordForm: document.getElementById('ref-change-password-form'),
        refNewPassword: document.getElementById('ref-new-password'),
        refConfirmPassword: document.getElementById('ref-confirm-password'),
        refPasswordError: document.getElementById('ref-password-error'),
        refBtnApprovePassword: document.getElementById('ref-btn-approve-password'),

        modalResetPasswordAlert: document.getElementById('modal-reset-password-alert'),
        refBtnAlertSendReset: document.getElementById('ref-btn-alert-send-reset'),

        btnCopyToken: document.getElementById('btn-copy-token'),
        btnCopyMcpJson: document.getElementById('btn-copy-mcp-json'),
        mcpTokenDisplay: document.getElementById('mcp-token-display'),
        mcpJsonSnippet: document.getElementById('mcp-json-snippet'),

        btnLoginGithub: document.getElementById('btn-login-github'),
        btnLoginGoogle: document.getElementById('btn-login-google'),

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
    // Modals Controls & Authentication State
    // =========================================================================
    const AUTH_USER_KEY = 'errorium_auth_user';

    function getAuthUser() {
        try {
            const data = localStorage.getItem(AUTH_USER_KEY);
            return data ? JSON.parse(data) : null;
        } catch (e) {
            return null;
        }
    }

    function setAuthUser(user) {
        try {
            if (user) {
                localStorage.setItem(AUTH_USER_KEY, JSON.stringify(user));
            } else {
                localStorage.removeItem(AUTH_USER_KEY);
            }
        } catch (e) {}
        updateAuthUI();
    }

    function updateAuthUI() {
        const user = getAuthUser();
        if (user) {
            if (elements.sidebarUserGuest) elements.sidebarUserGuest.style.display = 'none';
            if (elements.sidebarUserAuth) elements.sidebarUserAuth.style.display = 'flex';
            if (elements.headerAvatarGuest) elements.headerAvatarGuest.style.display = 'none';
            if (elements.headerAvatarAuth) {
                elements.headerAvatarAuth.style.display = 'flex';
                elements.headerAvatarAuth.textContent = (user.avatarLetter || user.username[0] || 'T').toUpperCase();
            }

            if (elements.sidebarAuthAvatar) {
                elements.sidebarAuthAvatar.textContent = (user.avatarLetter || user.username[0] || 'T').toUpperCase();
            }
            if (elements.sidebarAuthName) elements.sidebarAuthName.textContent = user.username;
            if (elements.sidebarAuthRole) elements.sidebarAuthRole.textContent = user.role || 'Backend developer';

            if (elements.profileAvatarLetter) {
                elements.profileAvatarLetter.textContent = (user.avatarLetter || user.username[0] || 'T').toUpperCase();
            }
            if (elements.profileUsername) elements.profileUsername.textContent = user.username;
            if (elements.profileInfoUsername) elements.profileInfoUsername.textContent = user.username;
            if (elements.profileInfoEmail) elements.profileInfoEmail.textContent = user.email || 'Test_user@mail.ru';
            if (elements.profileInfoRole) elements.profileInfoRole.textContent = user.role || 'Backend developer';
            if (elements.profileInfoGender) elements.profileInfoGender.textContent = user.gender || 'Male';
            if (elements.profileInfoLang) elements.profileInfoLang.textContent = user.language || 'English';
        } else {
            if (elements.sidebarUserGuest) elements.sidebarUserGuest.style.display = 'flex';
            if (elements.sidebarUserAuth) elements.sidebarUserAuth.style.display = 'none';
            if (elements.headerAvatarGuest) elements.headerAvatarGuest.style.display = 'inline-block';
            if (elements.headerAvatarAuth) elements.headerAvatarAuth.style.display = 'none';
        }
    }

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
        if (elements.tagsPopover) {
            elements.tagsPopover.style.display = 'none';
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

    // Modal & View Triggers
    elements.btnMcp?.addEventListener('click', () => showModal(elements.modalMcp));
    elements.btnAbout?.addEventListener('click', () => showModal(elements.modalAbout));
    elements.btnSettings?.addEventListener('click', () => showModal(elements.modalSettings));
    elements.btnSettingsAuth?.addEventListener('click', () => showModal(elements.modalSettings));

    elements.btnProfile?.addEventListener('click', () => {
        if (getAuthUser()) {
            switchToProfileView();
        } else {
            showModal(elements.modalAuthRef || elements.modalAuth);
        }
    });

    elements.btnLoginRow?.addEventListener('click', () => {
        showModal(elements.modalAuthRef || elements.modalAuth);
    });

    elements.btnAuthUserRow?.addEventListener('click', () => {
        switchToProfileView();
    });

    // Profile View Actions
    elements.btnProfileBack?.addEventListener('click', () => {
        switchToHeroView();
    });

    elements.profileRowSecurity?.addEventListener('click', () => {
        showModal(elements.modalResetPasswordAlert);
    });

    elements.profileInfoGenderRow?.addEventListener('click', () => {
        const user = getAuthUser() || {
            username: 'Test_user',
            email: 'Test_user@mail.ru',
            role: 'Backend developer',
            gender: 'Male',
            language: 'English',
            avatarLetter: 'T'
        };
        user.gender = user.gender === 'Male' ? 'Female' : 'Male';
        setAuthUser(user);
        showToast(`Gender set to ${user.gender}`);
    });

    elements.profileAvatarEditBadge?.addEventListener('click', (e) => {
        e.stopPropagation();
        const user = getAuthUser() || {
            username: 'Test_user',
            email: 'Test_user@mail.ru',
            role: 'Backend developer',
            gender: 'Male',
            language: 'English',
            avatarLetter: 'T'
        };
        const currentName = user.username;
        const newName = currentName === 'Test_user' ? 'Senior_developer' : 'Test_user';
        user.username = newName;
        user.avatarLetter = newName[0].toUpperCase();
        setAuthUser(user);
        showToast(`Username updated to ${newName}`);
    });

    // Reference Modal Handlers
    // 1. Auth Reference Modal (Auth-screen.png)
    elements.refAuthForm?.addEventListener('submit', (e) => {
        e.preventDefault();
        const email = elements.refAuthEmail?.value || 'Test_user@mail.ru';
        const user = {
            username: 'Test_user',
            email: email,
            role: 'Backend developer',
            gender: 'Male',
            language: 'English',
            avatarLetter: 'T'
        };
        setAuthUser(user);
        closeAllModals();
        showToast('Successfully logged in!');
    });

    elements.refBtnYandex?.addEventListener('click', () => {
        const user = {
            username: 'Test_user',
            email: 'Test_user@mail.ru',
            role: 'Backend developer',
            gender: 'Male',
            language: 'English',
            avatarLetter: 'T'
        };
        setAuthUser(user);
        closeAllModals();
        showToast('Logged in via Yandex ID');
    });

    elements.refAuthForgot?.addEventListener('click', () => {
        showModal(elements.modalChangePasswordEmail);
    });

    elements.refLinkSignup?.addEventListener('click', () => {
        showToast('Registration is open. Enter credentials above.');
    });

    // 2. Reset Password Alert Modal (Reset-password-alert.png)
    elements.refBtnAlertSendReset?.addEventListener('click', () => {
        showModal(elements.modalChangePasswordEmail);
    });

    // 3. Change Password Email Modal (Change-password-email-screen.png)
    elements.refResetEmailForm?.addEventListener('submit', (e) => {
        e.preventDefault();
        showToast('Reset email sent! Check your spam folder.');
        setTimeout(() => {
            showModal(elements.modalChangePassword);
        }, 1200);
    });

    // 4. Change Password Modal (Change-password-screen.png)
    elements.refChangePasswordForm?.addEventListener('submit', (e) => {
        e.preventDefault();
        const pwd1 = elements.refNewPassword?.value || '';
        const pwd2 = elements.refConfirmPassword?.value || '';
        if (!pwd1) {
            if (elements.refPasswordError) {
                elements.refPasswordError.textContent = 'Password cannot be empty';
                elements.refPasswordError.style.display = 'block';
            }
            return;
        }
        if (pwd1 !== pwd2) {
            if (elements.refPasswordError) {
                elements.refPasswordError.textContent = 'Passwords do not match';
                elements.refPasswordError.style.display = 'block';
            }
            return;
        }
        if (elements.refPasswordError) {
            elements.refPasswordError.style.display = 'none';
        }
        closeAllModals();
        showToast('Password updated successfully!');
    });

    // Fast Auth Demo
    elements.btnLoginGithub?.addEventListener('click', () => handleLogin('GitHub'));
    elements.btnLoginGoogle?.addEventListener('click', () => handleLogin('Google'));

    function handleLogin(provider) {
        const user = {
            username: 'Test_user',
            email: 'dev@errorium.ru',
            role: 'Backend developer',
            gender: 'Male',
            language: 'English',
            avatarLetter: 'T'
        };
        setAuthUser(user);
        closeAllModals();
        showToast(`Signed in via ${provider} SSO! MCP tokens active.`);
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

    // Attach button popover toggle (Tags Popover - Chat-screen-with-tags.png)
    elements.btnAttach?.addEventListener('click', function (e) {
        e.stopPropagation();
        if (elements.tagsPopover) {
            const isShown = elements.tagsPopover.style.display === 'block' || elements.tagsPopover.style.display === 'flex';
            elements.tagsPopover.style.display = isShown ? 'none' : 'block';
        }
        if (elements.attachPopover) {
            elements.attachPopover.style.display = 'none';
        }
    });

    // Tag pills click listeners
    document.querySelectorAll('.tag-pill').forEach(pill => {
        pill.addEventListener('click', function (e) {
            e.stopPropagation();
            const tag = this.getAttribute('data-tag');
            if (elements.promptInput) {
                const current = elements.promptInput.value;
                elements.promptInput.value = current ? `${current} [${tag}] ` : `[${tag}] `;
            }
            if (elements.tagsPopover) {
                elements.tagsPopover.style.display = 'none';
            }
            adjustTextareaHeight();
            elements.promptInput?.focus();
        });
    });

    // Popover quick items (Preserved for compatibility)
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
        if (elements.tagsPopover && !elements.tagsPopover.contains(e.target) && e.target !== elements.btnAttach && !elements.btnAttach.contains(e.target)) {
            elements.tagsPopover.style.display = 'none';
        }
        if (elements.attachPopover && !elements.attachPopover.contains(e.target) && e.target !== elements.btnAttach && !elements.btnAttach.contains(e.target)) {
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
        if (elements.profileView) elements.profileView.style.display = 'none';
        if (elements.chatView) elements.chatView.style.display = 'flex';
    }

    function switchToProfileView() {
        if (elements.heroView) elements.heroView.style.display = 'none';
        if (elements.chatView) elements.chatView.style.display = 'none';
        if (elements.profileView) elements.profileView.style.display = 'flex';
        closeSidebar();
    }

    function switchToHeroView() {
        if (elements.profileView) elements.profileView.style.display = 'none';
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
        const friendlyGreeting = userText.toLowerCase().includes('syntaxerror')
            ? 'Of course I will be glad to help you with the SyntaxError fixing! Please give me more context, so I will be able to help you!'
            : `Of course I will be glad to help you with the ${escapeHtml(userText)} fixing! Please give me more context, so I will be able to help you!`;

        elements.messagesContainer.innerHTML = `
            <!-- User Message Dialogue Bubble (Chat-screen-dialog.png) -->
            <div class="message-row user" style="justify-content: flex-end;">
                <div class="dialog-user-bubble">
                    <span>${escapeHtml(userText)}</span>
                    <span class="dialog-check-icon" title="Delivered">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round">
                            <polyline points="20 6 9 17 4 12"></polyline>
                            <polyline points="24 10 17 17 14 14"></polyline>
                        </svg>
                    </span>
                </div>
            </div>

            <!-- Assistant Greeting Dialogue Bubble (Chat-screen-dialog.png) -->
            <div class="dialog-assistant-wrapper">
                <div class="dialog-assistant-avatar">
                    <img src="/message.svg" alt="Errorium Mascot">
                </div>
                <div class="dialog-assistant-bubble">
                    ${friendlyGreeting}
                </div>
            </div>

            <!-- Assistant Solution Card (Preserved BM25 RAG) -->
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

    // Initialize authentication UI state from stored session
    updateAuthUI();

})();
