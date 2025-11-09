document.addEventListener('alpine:init', () => {
  Alpine.data('pluginInstaller', (routes) => ({
    ...routes,
    isLoading: false,
    pollInterval: null,
    progress: 0,
    message: '',

    init() {
      const loadingState = localStorage.getItem('plugin_install_loading');
      if (!loadingState) {
        console.log('[init] No ongoing installation found.');
        return;
      }

      const state = JSON.parse(loadingState);

      // Validate stored data
      const hasValidData =
        (state.github_repo_url && state.github_repo_url.trim() !== '') ||
        (state.file_name && state.file_name.trim() !== '');

      if (!state.isLoading || !hasValidData) {
        localStorage.removeItem('plugin_install_loading');
        return;
      }

      // Restore previous state
      this.github_repo_url = state.github_repo_url || '';
      this.file_name = state.file_name || '';
      this.action_url = state.action || this.plugin_install_github_url;
      this.progress = state.progress || 15;
      this.message = state.message || 'Installing...';
      this.isLoading = true;

      const pluginName =
        this.github_repo_url?.split('/').pop() || this.file_name || '';

      if (!pluginName) {
        this.isLoading = false;
        localStorage.removeItem('plugin_install_loading');
        return;
      }

      const url = `${this.check_install_status_url}?source=${encodeURIComponent(pluginName)}`;

      fetch(url)
        .then((res) => res.json())
        .then((data) => {
          if (data.status === 'success' || data.status === 'failed') {
            this.isLoading = false;
            this.progress = data.progress || 100;
            this.message =
              data.status === 'failed'
                ? 'Installation failed.'
                : 'Installation completed!';
            localStorage.removeItem('plugin_install_loading');
          } else {
            this.startPolling();
          }
        })
        .catch((err) => {
          this.isLoading = false;
          localStorage.removeItem('plugin_install_loading');
        });
    },

    async handleSubmit(event) {
      event.preventDefault();

      // Check for ongoing installation before doing anything
      const existing = localStorage.getItem('plugin_install_loading');
      if (existing) {
        try {
          const state = JSON.parse(existing);
          if (state.isLoading) {
            alert('Installation in progress. Try again later.');
            this.isLoading = true;
            this.progress = state.progress || 15;
            this.message = state.message || 'Resuming...';
            this.startPolling();
            return;
          }
        } catch {
          console.warn('Failed to parse existing installation state.');
        }
      }

      const form = event.target;
      const formData = new FormData(form);

      if (this.action_url === this.plugin_install_github_url) {
        const githubRepoUrl = formData.get('github_repo_url') || '';
        const githubRef = formData.get('github_ref') || '';

        if (!githubRepoUrl || !githubRef) {
          alert(
            'Please enter both a GitHub repository URL and a branch/commit hash.'
          );
          return;
        }

        this.github_repo_url = githubRepoUrl;
        localStorage.setItem(
          'plugin_install_loading',
          JSON.stringify({
            isLoading: true,
            action: this.action_url,
            github_repo_url: githubRepoUrl,
            timestamp: Date.now()
          })
        );
      } else if (this.action_url === this.plugin_install_zip_url) {
        const file = formData.get('plugin_zip_file');
        if (!file || !file.name) {
          alert('Please select a ZIP file first.');
          this.isLoading = false;
          return;
        }

        this.file_name = file.name;
        localStorage.setItem(
          'plugin_install_loading',
          JSON.stringify({
            isLoading: true,
            action: this.action_url,
            file_name: file.name,
            timestamp: Date.now()
          })
        );
      }

      try {
        const res = await fetch(this.action_url, {
          method: 'POST',
          body: formData
        });

        const data = await res.json();

        this.isLoading = true;
        this.progress = 15;
        this.message = 'Initializing...';

        // Other non-OK response status
        if (!res.ok) {
          throw new Error(`HTTP error ${res.status}`);
        }

        if (data.status === 'in-progress') {
          this.startPolling();
        } else {
          alert('Unexpected response from server.');
          this.isLoading = false;
        }
      } catch (err) {
        this.isLoading = false;
        alert('Failed to start installation.');
      }
    },

    startPolling() {
      if (this.pollInterval) return;

      this.pollInterval = setInterval(async () => {
        try {
          const pluginName =
            this.github_repo_url?.split('/').pop() || this.file_name || '';
          const url = `${this.check_install_status_url}?source=${encodeURIComponent(pluginName)}`;

          const res = await fetch(url);
          const data = await res.json();

          console.log('data: ', data);

          this.progress = data.progress || 15;
          this.message = data.message || 'Installing...';

          // persist progress
          const state = JSON.parse(
            localStorage.getItem('plugin_install_loading') || '{}'
          );
          localStorage.setItem(
            'plugin_install_loading',
            JSON.stringify({
              ...state,
              progress: this.progress,
              message: this.message
            })
          );

          if (data.status === 'success') {
            this.progress = 100;
            this.stopPolling();
            this.isLoading = false;
            localStorage.removeItem('plugin_install_loading');
          } else if (data.status === 'failed') {
            this.stopPolling();
            this.isLoading = false;
            localStorage.removeItem('plugin_install_loading');
          }
        } catch (err) {
          console.error('[polling] Error:', err);
        }
      }, 3000);
    },

    stopPolling() {
      if (this.pollInterval) {
        clearInterval(this.pollInterval);
        this.pollInterval = null;
      }
    },

    initFlareEvents() {
      if (typeof $flare === 'undefined' || !$flare.events) {
        console.warn('[pluginInstaller] $flare.events not available.');
        return;
      }

      console.log('[pluginInstaller] Listening for install:progress...');

      $flare.events.on('install:progress', (res) => {
        console.log('[Flare Event] install:progress:', res);

        try {
          const payload =
            typeof res.data === 'string' ? JSON.parse(res.data) : res.data;

          if (payload.success) {
            this.message = payload.success;
            this.progress = 100;
            this.isLoading = false;
            localStorage.removeItem('plugin_install_loading');
            alert(payload.success);
            setTimeout(() => (location.href = this.plugin_index_url), 1000);
          }
        } catch (err) {
          console.error('[pluginInstaller] Failed to process event:', err);
        }
      });
    }
  }));
});
