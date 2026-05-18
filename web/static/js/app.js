class FeedPulseApp {
    constructor() {
        this.token = localStorage.getItem('token');
        this.currentView = 'all';
        this.currentFeed = null;
        this.currentArticle = null;
        this.articles = [];
        this.page = 1;
        this.perPage = 20;
        this.loading = false;

        this.init();
    }

    init() {
        if (this.token) {
            this.showMain();
        } else {
            this.showAuth();
        }

        this.bindEvents();
    }

    bindEvents() {
        document.querySelectorAll('.tab').forEach(tab => {
            tab.addEventListener('click', (e) => {
                document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
                document.querySelectorAll('.auth-form').forEach(f => f.classList.remove('active'));
                e.target.classList.add('active');
                document.getElementById(`${e.target.dataset.tab}-form`).classList.add('active');
            });
        });

        document.getElementById('login-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.login();
        });

        document.getElementById('register-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.register();
        });

        document.getElementById('logout-btn').addEventListener('click', () => {
            this.logout();
        });

        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', (e) => {
                e.preventDefault();
                this.switchView(item.dataset.view);
            });
        });

        document.getElementById('add-feed-btn').addEventListener('click', () => {
            this.showAddFeedModal();
        });

        document.getElementById('add-group-btn').addEventListener('click', () => {
            this.showAddGroupModal();
        });

        document.getElementById('close-detail').addEventListener('click', () => {
            this.closeArticleDetail();
        });

        document.getElementById('mark-read-btn').addEventListener('click', () => {
            if (this.currentArticle) {
                this.markArticleRead(this.currentArticle.id, !this.currentArticle.is_read);
            }
        });

        document.getElementById('star-btn').addEventListener('click', () => {
            if (this.currentArticle) {
                this.starArticle(this.currentArticle.id, !this.currentArticle.is_starred);
            }
        });

        document.getElementById('later-btn').addEventListener('click', () => {
            if (this.currentArticle) {
                this.markArticleLater(this.currentArticle.id, !this.currentArticle.is_later);
            }
        });

        document.getElementById('search-input').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.searchArticles(e.target.value);
            }
        });

        document.querySelector('.close-modal').addEventListener('click', () => {
            this.closeModal();
        });

        document.addEventListener('keydown', (e) => {
            if (!this.token) return;
            
            switch(e.key) {
                case 'j':
                    this.nextArticle();
                    break;
                case 'k':
                    this.prevArticle();
                    break;
                case 'm':
                    if (this.currentArticle) {
                        this.markArticleRead(this.currentArticle.id, !this.currentArticle.is_read);
                    }
                    break;
                case 's':
                    if (this.currentArticle) {
                        this.starArticle(this.currentArticle.id, !this.currentArticle.is_starred);
                    }
                    break;
            }
        });
    }

    showAuth() {
        document.getElementById('auth-container').classList.remove('hidden');
        document.getElementById('main-container').classList.add('hidden');
    }

    showMain() {
        document.getElementById('auth-container').classList.add('hidden');
        document.getElementById('main-container').classList.remove('hidden');
        this.loadFeeds();
        this.loadGroups();
        this.loadArticles();
        this.loadStats();
    }

    async api(endpoint, options = {}) {
        const headers = {
            'Content-Type': 'application/json',
            ...(this.token && { 'Authorization': `Bearer ${this.token}` })
        };

        try {
            const response = await fetch(`/api${endpoint}`, {
                headers,
                ...options
            });

            if (response.status === 401) {
                this.token = null;
                localStorage.removeItem('token');
                this.showAuth();
                throw new Error('Unauthorized');
            }

            const result = await response.json();
            if (result.error) {
                throw new Error(result.error);
            }
            return result;
        } catch (error) {
            throw error;
        }
    }

    showToast(message, type = 'error') {
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        toast.textContent = message;
        toast.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 12px 20px;
            border-radius: 8px;
            color: white;
            font-weight: 500;
            z-index: 10000;
            animation: slideIn 0.3s ease;
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        `;
        
        if (type === 'success') {
            toast.style.background = '#4CAF50';
        } else if (type === 'error') {
            toast.style.background = '#f44336';
        } else {
            toast.style.background = '#2196F3';
        }

        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.style.animation = 'slideOut 0.3s ease';
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }

    async login() {
        const email = document.getElementById('login-email').value;
        const password = document.getElementById('login-password').value;

        try {
            const result = await this.api('/login', {
                method: 'POST',
                body: JSON.stringify({ email, password })
            });

            if (result.error) {
                alert(result.error);
                return;
            }

            this.token = result.token;
            localStorage.setItem('token', result.token);
            this.showMain();
        } catch (err) {
            alert('登录失败，请检查邮箱和密码');
        }
    }

    async register() {
        const username = document.getElementById('register-username').value;
        const email = document.getElementById('register-email').value;
        const password = document.getElementById('register-password').value;

        try {
            const result = await this.api('/register', {
                method: 'POST',
                body: JSON.stringify({ username, email, password })
            });

            if (result.error) {
                alert(result.error);
                return;
            }

            this.token = result.token;
            localStorage.setItem('token', result.token);
            this.showMain();
        } catch (err) {
            alert('注册失败，请重试');
        }
    }

    logout() {
        this.token = null;
        localStorage.removeItem('token');
        this.showAuth();
    }

    switchView(view) {
        this.currentView = view;
        this.currentFeed = null;
        this.page = 1;
        
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.toggle('active', item.dataset.view === view);
        });
        document.querySelectorAll('.feed-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelectorAll('.group-item').forEach(item => {
            item.classList.remove('active');
        });

        this.loadArticles();
    }

    async loadArticles() {
        if (this.loading) return;
        this.loading = true;

        let params = new URLSearchParams({
            page: this.page,
            per_page: this.perPage
        });

        if (this.currentFeed) {
            params.append('feed_id', this.currentFeed);
        }

        switch(this.currentView) {
            case 'unread':
                params.append('is_read', 'false');
                break;
            case 'starred':
                params.append('is_starred', 'true');
                break;
            case 'later':
                params.append('is_later', 'true');
                break;
        }

        const result = await this.api(`/articles?${params}`);
        this.articles = result.articles || [];
        this.renderArticles();
        this.loading = false;
    }

    renderArticles() {
        const container = document.getElementById('article-list');
        
        if (this.articles.length === 0) {
            container.innerHTML = '<div class="loading">暂无文章</div>';
            return;
        }

        container.innerHTML = this.articles.map(article => `
            <div class="article-card ${article.is_read ? 'read' : ''} ${article.is_starred ? 'starred' : ''}" 
                 data-id="${article.id}">
                <div class="article-header">
                    <span class="article-source">${article.feed_title || 'Feed'}</span>
                    <span class="article-date">${this.formatDate(article.published_at)}</span>
                </div>
                <div class="article-title">${article.title}</div>
                <div class="article-summary">${article.summary || article.content?.substring(0, 200) || ''}</div>
            </div>
        `).join('');

        container.querySelectorAll('.article-card').forEach(card => {
            card.addEventListener('click', () => {
                const id = parseInt(card.dataset.id);
                this.openArticle(id);
            });
        });
    }

    async openArticle(id) {
        const article = await this.api(`/articles/${id}`);
        this.currentArticle = article;

        if (!article.is_read) {
            this.markArticleRead(id, true);
        }

        document.getElementById('article-detail').classList.remove('hidden');
        
        const content = document.getElementById('detail-content');
        content.innerHTML = `
            <h1>${article.title}</h1>
            <p><strong>作者:</strong> ${article.author || '未知'} | <strong>发布时间:</strong> ${this.formatDate(article.published_at)}</p>
            <hr>
            ${article.content || article.summary}
        `;

        document.getElementById('article-link').href = article.url;
        document.getElementById('star-btn').textContent = article.is_starred ? '★' : '☆';
        document.getElementById('star-btn').classList.toggle('starred', article.is_starred);
        document.getElementById('mark-read-btn').textContent = article.is_read ? '📖' : '⭕';

        const card = document.querySelector(`.article-card[data-id="${id}"]`);
        if (card) {
            card.classList.add('read');
        }
    }

    closeArticleDetail() {
        document.getElementById('article-detail').classList.add('hidden');
        this.currentArticle = null;
    }

    async markArticleRead(id, read) {
        await this.api(`/articles/${id}/read`, {
            method: 'PUT',
            body: JSON.stringify({ read })
        });

        if (this.currentArticle && this.currentArticle.id === id) {
            this.currentArticle.is_read = read;
            document.getElementById('mark-read-btn').textContent = read ? '📖' : '⭕';
        }

        const card = document.querySelector(`.article-card[data-id="${id}"]`);
        if (card) {
            card.classList.toggle('read', read);
        }

        this.loadStats();
    }

    async starArticle(id, starred) {
        await this.api(`/articles/${id}/star`, {
            method: 'PUT',
            body: JSON.stringify({ starred })
        });

        if (this.currentArticle && this.currentArticle.id === id) {
            this.currentArticle.is_starred = starred;
            document.getElementById('star-btn').textContent = starred ? '★' : '☆';
            document.getElementById('star-btn').classList.toggle('starred', starred);
        }

        const card = document.querySelector(`.article-card[data-id="${id}"]`);
        if (card) {
            card.classList.toggle('starred', starred);
        }

        if (this.currentView === 'starred') {
            this.loadArticles();
        }
    }

    async markArticleLater(id, later) {
        await this.api(`/articles/${id}/later`, {
            method: 'PUT',
            body: JSON.stringify({ later })
        });

        if (this.currentArticle && this.currentArticle.id === id) {
            this.currentArticle.is_later = later;
        }

        if (this.currentView === 'later') {
            this.loadArticles();
        }
    }

    async loadFeeds() {
        const feeds = await this.api('/feeds');
        const container = document.getElementById('feeds-list');
        
        container.innerHTML = feeds.map(feed => `
            <div class="feed-item ${feed.health_status === 'unhealthy' ? 'feed-unhealthy' : ''}" 
                 data-id="${feed.id}">
                <span class="feed-title">${feed.title}</span>
            </div>
        `).join('');

        container.querySelectorAll('.feed-item').forEach(item => {
            item.addEventListener('click', () => {
                document.querySelectorAll('.feed-item').forEach(i => i.classList.remove('active'));
                document.querySelectorAll('.group-item').forEach(i => i.classList.remove('active'));
                document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
                item.classList.add('active');
                this.currentFeed = parseInt(item.dataset.id);
                this.currentView = 'feed';
                this.page = 1;
                this.loadArticles();
            });
        });
    }

    async loadGroups() {
        const groups = await this.api('/groups');
        const container = document.getElementById('groups-list');
        
        container.innerHTML = groups.map(group => `
            <div class="group-item" data-id="${group.id}">
                <span>${group.name}</span>
            </div>
        `).join('');
    }

    async loadStats() {
        const stats = await this.api('/stats');
        document.getElementById('unread-count').textContent = stats.unread_count || 0;
    }

    async searchArticles(query) {
        if (!query.trim()) {
            this.loadArticles();
            return;
        }

        const result = await this.api(`/articles/search?q=${encodeURIComponent(query)}`);
        this.articles = result.articles || [];
        this.renderArticles();
    }

    showAddFeedModal() {
        document.getElementById('modal').classList.remove('hidden');
        document.getElementById('modal-body').innerHTML = `
            <h2>添加订阅</h2>
            <div id="error-message" class="error-message hidden" style="color: #f44336; padding: 10px; background: #ffebee; border-radius: 4px; margin-bottom: 15px;"></div>
            <div class="form-group">
                <label>Feed 地址或网站 URL</label>
                <input type="url" id="feed-url-input" placeholder="https://example.com/feed.xml" required>
            </div>
            <div id="discovered-feeds" class="hidden"></div>
            <div class="form-group">
                <button id="discover-btn" class="btn">发现 Feed</button>
            </div>
            <button id="add-feed-confirm" class="btn btn-primary">添加</button>
        `;

        const showError = (message) => {
            const errorDiv = document.getElementById('error-message');
            errorDiv.textContent = message;
            errorDiv.classList.remove('hidden');
        };

        const hideError = () => {
            document.getElementById('error-message').classList.add('hidden');
        };

        document.getElementById('discover-btn').addEventListener('click', async () => {
            const url = document.getElementById('feed-url-input').value;
            if (!url) {
                showError('请输入URL地址');
                return;
            }

            hideError();
            try {
                const result = await this.api(`/feeds/discover?url=${encodeURIComponent(url)}`);
                const feeds = result.feeds || [];
                
                const container = document.getElementById('discovered-feeds');
                container.classList.remove('hidden');
                
                if (feeds.length === 0) {
                    container.innerHTML = '<p style="color: #ff9800;">未发现可用的 Feed，请直接输入 RSS/Atom 订阅地址</p>';
                } else {
                    container.innerHTML = `
                        <p>发现以下 Feed:</p>
                        <ul>
                            ${feeds.map(f => `<li><a href="#" class="feed-link" data-url="${f}">${f}</a></li>`).join('')}
                        </ul>
                    `;
                    
                    container.querySelectorAll('.feed-link').forEach(link => {
                        link.addEventListener('click', (e) => {
                            e.preventDefault();
                            document.getElementById('feed-url-input').value = link.dataset.url;
                        });
                    });
                }
            } catch (error) {
                showError(`发现Feed失败: ${error.message}`);
            }
        });

        document.getElementById('add-feed-confirm').addEventListener('click', async () => {
            const url = document.getElementById('feed-url-input').value;
            if (!url) {
                showError('请输入URL地址');
                return;
            }

            hideError();
            try {
                await this.api('/feeds', {
                    method: 'POST',
                    body: JSON.stringify({ url })
                });

                this.closeModal();
                this.showToast('订阅添加成功！', 'success');
                this.loadFeeds();
                this.loadArticles();
            } catch (error) {
                showError(error.message || '添加订阅失败，请重试');
            }
        });
    }

    showAddGroupModal() {
        document.getElementById('modal').classList.remove('hidden');
        document.getElementById('modal-body').innerHTML = `
            <h2>添加分组</h2>
            <div class="form-group">
                <label>分组名称</label>
                <input type="text" id="group-name-input" required>
            </div>
            <button id="add-group-confirm" class="btn btn-primary">添加</button>
        `;

        document.getElementById('add-group-confirm').addEventListener('click', async () => {
            const name = document.getElementById('group-name-input').value;
            if (!name) return;

            await this.api('/groups', {
                method: 'POST',
                body: JSON.stringify({ name })
            });

            this.closeModal();
            this.loadGroups();
        });
    }

    closeModal() {
        document.getElementById('modal').classList.add('hidden');
    }

    nextArticle() {
        const cards = document.querySelectorAll('.article-card');
        if (cards.length === 0) return;

        let nextIndex = 0;
        if (this.currentArticle) {
            const currentIndex = Array.from(cards).findIndex(c => parseInt(c.dataset.id) === this.currentArticle.id);
            if (currentIndex > -1 && currentIndex < cards.length - 1) {
                nextIndex = currentIndex + 1;
            }
        }

        const nextId = parseInt(cards[nextIndex].dataset.id);
        this.openArticle(nextId);
    }

    prevArticle() {
        const cards = document.querySelectorAll('.article-card');
        if (cards.length === 0) return;

        let prevIndex = cards.length - 1;
        if (this.currentArticle) {
            const currentIndex = Array.from(cards).findIndex(c => parseInt(c.dataset.id) === this.currentArticle.id);
            if (currentIndex > 0) {
                prevIndex = currentIndex - 1;
            }
        }

        const prevId = parseInt(cards[prevIndex].dataset.id);
        this.openArticle(prevId);
    }

    formatDate(dateStr) {
        const date = new Date(dateStr);
        const now = new Date();
        const diff = now - date;

        if (diff < 60000) return '刚刚';
        if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`;
        if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`;
        if (diff < 604800000) return `${Math.floor(diff / 86400000)} 天前`;

        return date.toLocaleDateString('zh-CN');
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new FeedPulseApp();
});
