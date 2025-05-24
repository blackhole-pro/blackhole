import React from 'react';
import { createRoot } from 'react-dom/client';
import { 
  BlackholeProvider, 
  OAuthButtonGroup, 
  OAuthCallback,
  UserProfile
} from '../../client-libs/react/src';
import { BrowserRouter, Routes, Route, Link } from 'react-router-dom';

// Home page component with OAuth login buttons
const HomePage = () => {
  return (
    <div className="container">
      <h1>Blackhole OAuth Example</h1>
      <p>This example demonstrates OAuth authentication with Google, Facebook, and Apple.</p>
      
      <div className="auth-container">
        <h2>Sign in with</h2>
        <OAuthButtonGroup 
          providersToShow={['google', 'facebook', 'apple']}
          vertical={true}
          onSuccess={() => console.log('OAuth initiated')}
          onError={(error) => console.error('OAuth error:', error)}
        />
      </div>
      
      <div className="info-container">
        <p>
          This example demonstrates the OAuth authentication flow using the Blackhole SDK.
          Clicking any of the buttons above will redirect you to the respective provider's
          authentication page. After successful authentication, you will be redirected back
          to the /oauth-callback route, where the OAuthCallback component will process
          the response and authenticate you with the Blackhole network.
        </p>
      </div>
    </div>
  );
};

// OAuth callback handler component
const CallbackPage = () => {
  return (
    <OAuthCallback
      onSuccess={(result) => {
        console.log('Authentication successful:', result);
      }}
      onError={(error) => {
        console.error('Authentication failed:', error);
      }}
      redirectPath="/profile"
      autoRedirect={true}
    />
  );
};

// Profile page that displays user information after successful authentication
const ProfilePage = () => {
  return (
    <div className="container">
      <h1>Your Profile</h1>
      
      <div className="profile-container">
        <UserProfile 
          showDid={true}
          showEmail={true}
          showProfilePicture={true}
          showAuthMethod={true}
          showDisconnect={true}
          onDisconnect={() => {
            console.log('User disconnected');
            window.location.href = '/';
          }}
        />
      </div>
      
      <div className="navigation">
        <Link to="/">Back to Home</Link>
      </div>
    </div>
  );
};

// Main App component with routing
const App = () => {
  return (
    <BrowserRouter>
      <BlackholeProvider
        nodeUrl="https://api.blackhole.example.com"
        defaultDomain="example.com"
        oauthRedirectUri={`${window.location.origin}/oauth-callback`}
        oauthProviders={{
          google: true,
          facebook: true,
          apple: true
        }}
        autoConnect={true}
      >
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/oauth-callback" element={<CallbackPage />} />
          <Route path="/profile" element={<ProfilePage />} />
        </Routes>
      </BlackholeProvider>
    </BrowserRouter>
  );
};

// CSS styles for the example
const styles = `
  body {
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    line-height: 1.6;
    color: #333;
    margin: 0;
    padding: 0;
    background-color: #f8f9fa;
  }
  
  .container {
    max-width: 800px;
    margin: 2rem auto;
    padding: 2rem;
    background-color: #fff;
    border-radius: 8px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  }
  
  h1 {
    color: #2c3e50;
    margin-top: 0;
  }
  
  .auth-container {
    margin: 2rem 0;
    padding: 1.5rem;
    border: 1px solid #eaeaea;
    border-radius: 6px;
    background-color: #fafafa;
  }
  
  .profile-container {
    margin: 2rem 0;
    padding: 1.5rem;
    border: 1px solid #eaeaea;
    border-radius: 6px;
    background-color: #fafafa;
  }
  
  .blackhole-oauth-group {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  
  .blackhole-oauth-button__button {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    padding: 10px 16px;
    border: 1px solid #ddd;
    border-radius: 4px;
    background-color: white;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
    width: 100%;
    max-width: 300px;
  }
  
  .blackhole-oauth-button__button:hover {
    background-color: #f5f5f5;
  }
  
  .blackhole-profile {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
  }
  
  .blackhole-profile__content {
    display: flex;
    align-items: center;
    gap: 1rem;
  }
  
  .blackhole-profile__picture img {
    width: A60px;
    height: 60px;
    border-radius: 50%;
    object-fit: cover;
  }
  
  .blackhole-profile__details {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  
  .blackhole-profile__name {
    font-weight: bold;
    font-size: 1.2rem;
  }
  
  .blackhole-profile__disconnect {
    padding: 8px 16px;
    background-color: #f44336;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
    transition: background-color 0.2s;
  }
  
  .blackhole-profile__disconnect:hover {
    background-color: #d32f2f;
  }
  
  .navigation {
    margin-top: 2rem;
  }
  
  .navigation a {
    color: #3498db;
    text-decoration: none;
  }
  
  .navigation a:hover {
    text-decoration: underline;
  }
  
  .info-container {
    margin-top: 2rem;
    padding: 1rem;
    background-color: #e8f4fc;
    border-radius: 4px;
    border-left: 4px solid #3498db;
  }
`;

// Render the app
const container = document.getElementById('root');
const root = createRoot(container);

// Add styles
const styleElement = document.createElement('style');
styleElement.textContent = styles;
document.head.appendChild(styleElement);

// Render app
root.render(<App />);

// Export the App component for usage in other examples
export default App;