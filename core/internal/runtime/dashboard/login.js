// Blackhole DID Authentication System
class BlackholeDIDAuth {
    constructor() {
        this.currentUser = null;
        this.authMethod = null;
        this.init();
    }

    init() {
        this.bindEventListeners();
        this.checkExistingAuth();
        this.checkServiceStatus();
    }

    bindEventListeners() {
        // Wallet authentication buttons
        document.querySelectorAll('.wallet-btn').forEach(btn => {
            btn.addEventListener('click', (e) => this.handleWalletAuth(e.target.dataset.wallet));
        });

        // OAuth authentication buttons
        document.querySelectorAll('.oauth-btn').forEach(btn => {
            btn.addEventListener('click', (e) => this.handleOAuthAuth(e.target.dataset.provider));
        });

        // Action buttons
        document.getElementById('continueBtn')?.addEventListener('click', () => this.continueToDashboard());
        document.getElementById('logoutBtn')?.addEventListener('click', () => this.logout());
    }

    checkExistingAuth() {
        const savedAuth = localStorage.getItem('blackhole_auth');
        if (savedAuth) {
            try {
                const authData = JSON.parse(savedAuth);
                if (this.isValidAuth(authData)) {
                    this.setAuthenticatedState(authData);
                }
            } catch (e) {
                localStorage.removeItem('blackhole_auth');
            }
        }
    }

    async checkServiceStatus() {
        try {
            const response = await fetch('/api/status');
            if (response.ok) {
                const data = await response.json();
                const identityService = data.services && data.services.identity;
                
                if (identityService && identityService.status !== 'running') {
                    this.showServiceWarning('Identity service is not running. Authentication will use fallback mode.');
                }
            }
        } catch (error) {
            console.warn('Could not check service status:', error);
        }
    }

    showServiceWarning(message) {
        // Create a warning banner at the top of the login card
        const loginCard = document.querySelector('.login-card');
        const existingWarning = document.querySelector('.service-warning');
        
        if (existingWarning) {
            existingWarning.remove();
        }
        
        const warning = document.createElement('div');
        warning.className = 'service-warning';
        warning.innerHTML = `
            <div class="warning-content">
                <span class="warning-icon">⚠️</span>
                <span class="warning-message">${message}</span>
            </div>
        `;
        
        loginCard.insertBefore(warning, loginCard.firstChild);
    }

    isValidAuth(authData) {
        // Check if auth data is recent (within 24 hours)
        const now = Date.now();
        const authTime = authData.timestamp || 0;
        const twentyFourHours = 24 * 60 * 60 * 1000;
        
        return authData.did && authData.method && (now - authTime < twentyFourHours);
    }

    async handleWalletAuth(walletType) {
        this.showLoading('Connecting to wallet...');
        
        try {
            const result = await this.connectWallet(walletType);
            if (result.success) {
                await this.authenticateWithDID(result.address, 'wallet', walletType);
            } else {
                this.showError('walletStatus', result.error);
            }
        } catch (error) {
            this.showError('walletStatus', `Failed to connect to ${walletType}: ${error.message}`);
        } finally {
            this.hideLoading();
        }
    }

    async connectWallet(walletType) {
        switch (walletType) {
            case 'metamask':
                return await this.connectMetaMask();
            case 'walletconnect':
                return await this.connectWalletConnect();
            case 'coinbase':
                return await this.connectCoinbaseWallet();
            default:
                throw new Error('Unsupported wallet type');
        }
    }

    async connectMetaMask() {
        if (typeof window.ethereum === 'undefined') {
            // Show installation prompt
            const install = confirm(
                'MetaMask is not installed.\n\n' +
                'MetaMask is required for wallet authentication.\n' +
                'Would you like to install MetaMask?'
            );
            
            if (install) {
                window.open('https://metamask.io/download/', '_blank');
            }
            
            return { 
                success: false, 
                error: 'MetaMask is not installed. Please install MetaMask extension.' 
            };
        }

        try {
            // Request account access
            const accounts = await window.ethereum.request({ 
                method: 'eth_requestAccounts' 
            });
            
            if (accounts.length === 0) {
                return { 
                    success: false, 
                    error: 'No accounts found. Please unlock MetaMask and try again.' 
                };
            }

            // Get network info
            const chainId = await window.ethereum.request({ method: 'eth_chainId' });
            
            console.log('MetaMask connected:', {
                account: accounts[0],
                chainId: chainId
            });

            return { 
                success: true, 
                address: accounts[0],
                chainId: chainId
            };
        } catch (error) {
            if (error.code === 4001) {
                return { 
                    success: false, 
                    error: 'User rejected the connection request.' 
                };
            }
            
            return { 
                success: false, 
                error: error.message || 'Failed to connect to MetaMask'
            };
        }
    }

    async connectWalletConnect() {
        // Simulate WalletConnect integration
        // In a real implementation, this would use @walletconnect/client
        return new Promise((resolve) => {
            setTimeout(() => {
                // Simulate successful connection
                const mockAddress = '0x' + Math.random().toString(16).substr(2, 40);
                resolve({ 
                    success: true, 
                    address: mockAddress 
                });
            }, 2000);
        });
    }

    async connectCoinbaseWallet() {
        // Simulate Coinbase Wallet integration
        // In a real implementation, this would use @coinbase/wallet-sdk
        return new Promise((resolve) => {
            setTimeout(() => {
                const mockAddress = '0x' + Math.random().toString(16).substr(2, 40);
                resolve({ 
                    success: true, 
                    address: mockAddress 
                });
            }, 1500);
        });
    }

    async handleOAuthAuth(provider) {
        this.showLoading(`Authenticating with ${provider}...`);
        
        try {
            const result = await this.authenticateOAuth(provider);
            if (result.success) {
                await this.authenticateWithDID(result.email, 'oauth', provider, result.userInfo);
            } else {
                this.showError('oauthStatus', result.error);
            }
        } catch (error) {
            this.showError('oauthStatus', `${provider} authentication failed: ${error.message}`);
        } finally {
            this.hideLoading();
        }
    }

    async authenticateOAuth(provider) {
        // Simulate OAuth flow with user confirmation
        // In a real implementation, this would redirect to OAuth provider
        return new Promise((resolve) => {
            // Show OAuth confirmation dialog
            const confirmed = this.showOAuthConfirmation(provider);
            
            if (!confirmed) {
                resolve({ 
                    success: false, 
                    error: 'User cancelled OAuth authentication'
                });
                return;
            }

            // Simulate OAuth provider response after user confirmation
            setTimeout(() => {
                const mockUser = {
                    email: `demo.user@${provider}.com`,
                    name: `Demo ${provider.charAt(0).toUpperCase() + provider.slice(1)} User`,
                    picture: `https://via.placeholder.com/40x40?text=${(provider || 'U').charAt(0).toUpperCase()}`,
                    id: `${provider}_${Math.random().toString(36).substr(2, 9)}`
                };
                
                resolve({ 
                    success: true, 
                    email: mockUser.email,
                    userInfo: mockUser
                });
            }, 1500);
        });
    }

    showOAuthConfirmation(provider) {
        // Show a confirmation dialog for OAuth flow
        const providerNames = {
            'google': 'Google',
            'facebook': 'Facebook', 
            'apple': 'Apple'
        };
        
        const providerName = providerNames[provider] || provider;
        
        return confirm(
            `This will simulate ${providerName} OAuth authentication.\n\n` +
            `In a real implementation, you would be redirected to ${providerName} to sign in.\n\n` +
            `Continue with demo authentication?`
        );
    }

    async authenticateWithDID(identifier, method, provider, userInfo = null) {
        try {
            // Generate or retrieve DID based on identifier
            const did = await this.generateDID(identifier, method);
            
            // Create authentication challenge
            const challenge = await this.createChallenge(did);
            
            // Sign challenge (simplified for demo)
            const signature = await this.signChallenge(challenge, identifier, method);
            
            // Verify signature with identity service
            const verified = await this.verifySignature(did, challenge, signature, method, provider);
            
            if (verified) {
                const authData = {
                    did: did,
                    method: method,
                    provider: provider,
                    identifier: identifier,
                    userInfo: userInfo,
                    timestamp: Date.now()
                };
                
                this.setAuthenticatedState(authData);
                this.saveAuthData(authData);
            } else {
                throw new Error('Signature verification failed');
            }
        } catch (error) {
            throw new Error(`DID authentication failed: ${error.message}`);
        }
    }

    async generateDID(identifier, method) {
        // Simulate DID generation based on identifier
        const hash = await this.hashIdentifier(identifier);
        return `did:blackhole:${method}:${hash}`;
    }

    async hashIdentifier(identifier) {
        // Simple hash simulation - in real implementation use proper crypto
        const encoder = new TextEncoder();
        const data = encoder.encode(identifier);
        const hashBuffer = await crypto.subtle.digest('SHA-256', data);
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        return hashArray.map(b => b.toString(16).padStart(2, '0')).join('').substring(0, 32);
    }

    async createChallenge(did) {
        // Call API to get challenge from identity service
        try {
            const response = await fetch('/api/auth/challenge', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ did: did })
            });
            
            if (!response.ok) {
                console.warn('Identity service unavailable, using fallback challenge generation');
                throw new Error(`Server returned ${response.status}: ${response.statusText}`);
            }
            
            const data = await response.json();
            return data.challenge;
        } catch (error) {
            console.warn('Using fallback challenge generation:', error.message);
            // Fallback to client-side challenge generation
            const array = new Uint8Array(32);
            crypto.getRandomValues(array);
            return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
        }
    }

    async signChallenge(challenge, identifier, method) {
        // Simulate signature creation
        // In real implementation, this would use the wallet or key management
        const message = `${challenge}:${identifier}:${method}`;
        const hash = await this.hashIdentifier(message);
        return `sig_${hash}`;
    }

    async verifySignature(did, challenge, signature, method, provider) {
        // Call API to verify signature with identity service
        try {
            const response = await fetch('/api/auth/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    did: did,
                    challenge: challenge,
                    signature: signature,
                    method: method,
                    provider: provider
                })
            });
            
            if (!response.ok) {
                console.error('Identity service verification failed:', response.status, response.statusText);
                return false;
            }
            
            const data = await response.json();
            return data.success === true;
        } catch (error) {
            console.error('Verification request failed:', error.message);
            return false;
        }
    }


    setAuthenticatedState(authData) {
        this.currentUser = authData;
        this.authMethod = authData.method;
        
        // Update UI
        document.getElementById('authenticatedDID').textContent = authData.did;
        
        let details = `Authenticated via ${authData.method}`;
        if (authData.provider) {
            details += ` (${authData.provider})`;
        }
        if (authData.userInfo && (authData.userInfo.name || authData.userInfo.email)) {
            details += `\nSigned in as: ${authData.userInfo.name || authData.userInfo.email}`;
        }
        
        document.getElementById('authDetails').textContent = details;
        document.getElementById('authStatus').style.display = 'flex';
        document.getElementById('authStatus').classList.add('slide-up');
    }

    saveAuthData(authData) {
        // Save to localStorage (in production, use more secure storage)
        localStorage.setItem('blackhole_auth', JSON.stringify(authData));
    }

    showLoading(message) {
        document.getElementById('loadingOverlay').style.display = 'flex';
        document.querySelector('.loading-message').textContent = message || 'Loading...';
    }

    hideLoading() {
        document.getElementById('loadingOverlay').style.display = 'none';
    }

    showError(containerId, message) {
        const container = document.getElementById(containerId);
        container.style.display = 'block';
        container.innerHTML = `
            <div class="error-message">
                <strong>Error:</strong> ${message}
            </div>
        `;
        container.classList.add('fade-in');
    }

    showSuccess(containerId, did, details) {
        const container = document.getElementById(containerId);
        container.style.display = 'block';
        container.innerHTML = `
            <div class="status-message">
                <strong>✅ Authentication successful!</strong>
            </div>
            <div class="did-display">${did}</div>
            ${details ? `<div class="user-info">${details}</div>` : ''}
        `;
        container.classList.add('fade-in');
    }

    continueToDashboard() {
        // Redirect to main dashboard
        window.location.href = '/';
    }

    logout() {
        // Clear authentication data
        localStorage.removeItem('blackhole_auth');
        this.currentUser = null;
        this.authMethod = null;
        
        // Reset UI
        document.getElementById('authStatus').style.display = 'none';
        document.getElementById('walletStatus').style.display = 'none';
        document.getElementById('oauthStatus').style.display = 'none';
        
        // Reset button states
        document.querySelectorAll('.wallet-btn, .oauth-btn').forEach(btn => {
            btn.classList.remove('connected', 'error');
        });
    }

    // Utility method to get current authentication status
    getAuthStatus() {
        return {
            isAuthenticated: !!this.currentUser,
            user: this.currentUser,
            method: this.authMethod
        };
    }
}

// Initialize the authentication system when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.blackholeAuth = new BlackholeDIDAuth();
    
    // Add some demo functionality for better UX
    addDemoFunctionality();
});

function addDemoFunctionality() {
    // Add hover effects and visual feedback
    document.querySelectorAll('.wallet-btn, .oauth-btn').forEach(btn => {
        btn.addEventListener('mouseenter', () => {
            btn.style.transform = 'translateY(-2px)';
        });
        
        btn.addEventListener('mouseleave', () => {
            btn.style.transform = 'translateY(0)';
        });
    });
    
    // Add keyboard navigation
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            // Close any open status displays
            const authStatus = document.getElementById('authStatus');
            if (authStatus.style.display === 'flex') {
                window.blackholeAuth.logout();
            }
        }
    });
    
    // Add loading state improvements
    const originalShowLoading = window.blackholeAuth.showLoading;
    window.blackholeAuth.showLoading = function(message) {
        // Disable all buttons during loading
        document.querySelectorAll('.wallet-btn, .oauth-btn').forEach(btn => {
            btn.disabled = true;
            btn.style.opacity = '0.6';
        });
        originalShowLoading.call(this, message);
    };
    
    const originalHideLoading = window.blackholeAuth.hideLoading;
    window.blackholeAuth.hideLoading = function() {
        // Re-enable all buttons
        document.querySelectorAll('.wallet-btn, .oauth-btn').forEach(btn => {
            btn.disabled = false;
            btn.style.opacity = '1';
        });
        originalHideLoading.call(this);
    };
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = BlackholeDIDAuth;
}