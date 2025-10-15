window.addEventListener('alpine:init', () => {
  Alpine.data('pluginInstaller', (routes) => ({
    ...routes,
    isLoading: false,
    pollInterval: null,
    
    init() {
      const loadingState = localStorage.getItem('plugin_install_loading');
      if (loadingState) {
        const state = JSON.parse(loadingState);

        if (state.isLoading) {
          this.github_repo_url = state.github_repo_url;
          this.file_name = state.file_name;
          this.isLoading = true;

          // Check current status before resuming polling
          const pluginName = this.github_repo_url?.split('/').pop() || this.file_name || '';
          const url = `${this.check_install_status_url}?source=${encodeURIComponent(pluginName)}`;

          fetch(url)
            .then(res => res.json())
            .then(data => {
              if (data.status === 'success' || data.status === 'failed') {
                this.isLoading = false;
                localStorage.removeItem('plugin_install_loading');
                if (data.status === 'failed') alert('Installation failed.');
              } else {
                this.startPolling();
              }
            })
            .catch(err => {
              this.isLoading = false;
              localStorage.removeItem('plugin_install_loading');
            });
        }
      }
    },

    handleSubmit(event) {
      const form = event.target;
      const formData = new FormData(form);
      
      if (this.action_url === this.plugin_install_github_url) {
        const githubRepoUrl = formData.get('github_repo_url') || '';
        this.isLoading = true;

        localStorage.setItem('plugin_install_loading', JSON.stringify({
          isLoading: true,
          action: this.action_url,
          github_repo_url: githubRepoUrl,
          timestamp: Date.now()
        }));

        form.submit();
      } 

      else if (this.action_url === this.plugin_install_zip_url) {
        const file = formData.get('plugin_zip_file');
        if (!file || !file.name) {
          alert('Please select a ZIP file first.');
          return;
        }

        this.isLoading = true;

        localStorage.setItem('plugin_install_loading', JSON.stringify({
          isLoading: true,
          type: 'zip',
          file_name: file.name,
          timestamp: Date.now(),
        }));

        form.submit();
      }
    },

    startPolling() {
      if (this.pollInterval) return;

      this.pollInterval = setInterval(async () => {
        try {
          const pluginName = this.github_repo_url?.split('/').pop() || this.file_name || '';
          const url = `${this.check_install_status_url}?source=${encodeURIComponent(pluginName)}`;

          const res = await fetch(url);
          const data = await res.json();

          if (data.status === 'success') {
            this.stopPolling();
            this.isLoading = false;
            localStorage.removeItem('plugin_install_loading');
            location.href = this.plugin_index_url;
          } else if (data.status === 'failed') {
            this.stopPolling();
            this.isLoading = false;
            localStorage.removeItem('plugin_install_loading');
            alert('Installation failed.');
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
        console.log('[polling] Stopped');
      }
    }
  }));
});
