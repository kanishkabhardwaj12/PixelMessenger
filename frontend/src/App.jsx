import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Lock,
  User,
  Send,
  LogOut,
  Hash,
  Plus,
  Users,
  MessageSquare,
  X,
  Loader2,
  Image as ImageIcon,
  Key,
} from 'lucide-react';

// Base URL for your Go backend's HTTP API
const API_BASE_URL = 'http://localhost:8080';

// Base URL for your Go backend's WebSocket API
const WS_BASE_URL = 'ws://localhost:8080';

// --- Helper Functions ---

/**
 * A simple utility to parse the JWT and extract data
 * This is useful for getting the username without another API call
 */
const parseJwt = (token) => {
  try {
    return JSON.parse(atob(token.split('.')[1]));
  } catch (e) {
    return null;
  }
};

/**
 * A hook to persist the auth token in localStorage
 */
const useAuth = () => {
  const [token, setToken] = useState(null);
  const [username, setUsername] = useState(null);
  const [userId, setUserId] = useState(null);

  useEffect(() => {
    // On initial load, check for an existing token
    const storedToken = localStorage.getItem('pixel-token');
    if (storedToken) {
      const claims = parseJwt(storedToken);
      if (claims && claims.exp * 1000 > Date.now()) {
        setToken(storedToken);
        setUsername(claims.username);
        setUserId(claims.user_id);
      } else {
        localStorage.removeItem('pixel-token');
      }
    }
  }, []);

  const login = (newToken) => {
    const claims = parseJwt(newToken);
    if (claims) {
      localStorage.setItem('pixel-token', newToken);
      setToken(newToken);
      setUsername(claims.username);
      setUserId(claims.user_id);
    }
  };

  const logout = () => {
    localStorage.removeItem('pixel-token');
    setToken(null);
    setUsername(null);
    setUserId(null);
  };

  return { token, username, userId, login, logout };
};

/**
 * A simple API helper to make authenticated requests
 */
const apiFetch = async (endpoint, token, options = {}) => {
  const headers = new Headers(options.headers || {});
  headers.set('Authorization', `Bearer ${token}`);
  
  if (options.body && !(options.body instanceof FormData) && !(options.body instanceof Blob)) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const errorBody = await response.text();
    throw new Error(errorBody || 'API request failed');
  }
  
  // Handle empty responses
  const contentType = response.headers.get("content-type");
  if (contentType && contentType.indexOf("application/json") !== -1) {
    return response.json();
  } else {
    return response.text();
  }
};

// --- Main App Component ---

export default function App() {
  const { token, username, userId, login, logout } = useAuth();
  
  if (!token) {
    return <AuthPage onLogin={login} />;
  }

  return <ChatPage token={token} username={username} userId={userId} onLogout={logout} />;
}

// --- Authentication Page ---

function AuthPage({ onLogin }) {
  const [isLogin, setIsLogin] = useState(true);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState(null);
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    setIsLoading(true);

    const endpoint = isLogin ? '/login' : '/register';
    
    try {
      const response = await fetch(`${API_BASE_URL}${endpoint}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || (typeof data === 'string' ? data : 'An error occurred'));
      }

      if (isLogin) {
        onLogin(data.token);
      } else {
        // Automatically log in after successful registration
        const loginResponse = await fetch(`${API_BASE_URL}/login`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password }),
        });
        const loginData = await loginResponse.json();
        if (!loginResponse.ok) throw new Error('Failed to log in after registration');
        onLogin(loginData.token);
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-900 text-gray-100">
      <div className="w-full max-w-md p-8 space-y-8 bg-gray-800 rounded-2xl shadow-xl">
        <h2 className="text-4xl font-extrabold text-center text-indigo-400">
          PixelMessenger
        </h2>
        <p className="text-center text-gray-400">
          {isLogin ? 'Welcome back! Sign in to continue.' : 'Create an account to start.'}
        </p>
        
        <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
          <div className="relative">
            <input
              id="username"
              name="username"
              type="text"
              autoComplete="username"
              required
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="peer relative block w-full px-4 py-3 pl-12 text-lg bg-gray-700 border border-gray-600 rounded-md placeholder-gray-400 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              placeholder="Username"
            />
            <User className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-gray-400" />
          </div>
          
          <div className="relative">
            <input
              id="password"
              name="password"
              type="password"
              autoComplete={isLogin ? "current-password" : "new-password"}
              required
              value={password}
              onChange={(e) => setPassword(e.targe.value)}
              className="peer relative block w-full px-4 py-3 pl-12 text-lg bg-gray-700 border border-gray-600 rounded-md placeholder-gray-400 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              placeholder="Password"
            />
            <Lock className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-gray-400" />
          </div>
          
          {error && (
            <p className="text-center text-sm text-red-400">{error}</p>
          )}

          <div>
            <button
              type="submit"
              disabled={isLoading}
              className="group relative w-full flex justify-center py-3 px-4 border border-transparent text-lg font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 focus:ring-offset-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? (
                <Loader2 className="h-6 w-6 animate-spin" />
              ) : (
                isLogin ? 'Sign In' : 'Sign Up'
              )}
            </button>
          </div>
        </form>
        
        <div className="text-sm text-center">
          <button
            onClick={() => {
              setIsLogin(!isLogin);
              setError(null);
            }}
            className="font-medium text-indigo-400 hover:text-indigo-300"
          >
            {isLogin ? 'Need an account? Sign up' : 'Already have an account? Sign in'}
          </button>
        </div>
      </div>
    </div>
  );
}

// --- Main Chat Application Page ---

function ChatPage({ token, username, userId, onLogout }) {
  const [rooms, setRooms] = useState([]);
  const [currentRoom, setCurrentRoom] = useState(null);
  const [messages, setMessages] = useState([]);
  const [webSocket, setWebSocket] = useState(null);
  const [error, setError] = useState(null);
  
  // Fetch rooms on load
  const fetchRooms = useCallback(async () => {
    try {
      const userRooms = await apiFetch('/my-rooms', token);
      setRooms(userRooms || []);
    } catch (err) {
      setError(`Failed to fetch rooms: ${err.message}`);
    }
  }, [token]);
  
  useEffect(() => {
    fetchRooms();
  }, [fetchRooms]);

  // Handle WebSocket connection
  useEffect(() => {
    if (!currentRoom || !token) {
      return;
    }

    // CRITICAL: See the note at the top of this file.
    // We pass the token as a query parameter because WebSockets can't send auth headers.
    // Your backend's JwtMiddleware must be modified to read this.
    const wsUrl = `${WS_BASE_URL}/ws?room=${currentRoom.id}&token=${token}`;
    
    const ws = new WebSocket(wsUrl);
    setWebSocket(ws);
    setMessages([]); // Clear messages when joining a new room

    ws.onopen = () => {
      console.log(`WebSocket connected to room: ${currentRoom.id}`);
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected');
    };

    ws.onerror = (err) => {
      console.error('WebSocket error:', err);
      setError('WebSocket connection failed. Check console.');
    };

    // This is the core magic!
    ws.onmessage = async (event) => {
      // The backend sends a binary image (blob)
      if (event.data instanceof Blob) {
        const imageBlob = event.data;
        const imageUrl = URL.createObjectURL(imageBlob);
        
        try {
          // Send the blob to the /decode endpoint
          const response = await apiFetch('/decode', token, {
            method: 'POST',
            body: imageBlob,
          });
          
          // The response is JSON: {"message": "..."}
          const decodedText = response.message;

          // Add a new "decoded" message to our state
          setMessages((prev) => [
            ...prev,
            {
              id: Date.now(),
              type: 'decoded',
              imageUrl: imageUrl,
              text: decodedText,
              sender: '???' // Note: The backend doesn't tell us who sent it, but we could add UserID to the broadcast
            },
          ]);
        } catch (err) {
          console.error('Failed to decode image:', err);
          // Add the image anyway, just without the text
          setMessages((prev) => [
            ...prev,
            {
              id: Date.now(),
              type: 'image',
              imageUrl: imageUrl,
              sender: '???'
            },
          ]);
        }
      } else {
        // This is a normal text message (e.g., "User joined")
        // We don't have this logic, but this is where it would go
        console.log('Received text message:', event.data);
      }
    };

    // Cleanup function
    return () => {
      ws.close();
    };
  }, [currentRoom, token]);

  const handleCreateRoom = async (roomName) => {
    try {
      const newRoom = await apiFetch('/rooms', token, {
        method: 'POST',
        body: JSON.stringify({ name: roomName }),
      });
      setRooms([...rooms, newRoom]);
      setCurrentRoom(newRoom); // Automatically join the new room
    } catch (err) {
      setError(`Failed to create room: ${err.message}`);
    }
  };
  
  const handleInvite = async (inviteUsername) => {
    if (!currentRoom) {
      setError('No room selected');
      return;
    }
    try {
      await apiFetch(`/rooms/${currentRoom.id}/invite`, token, {
        method: 'POST',
        body: JSON.stringify({ username: inviteUsername }),
      });
      // We could add a success message here
    } catch (err) {
      setError(`Failed to invite user: ${err.message}`);
    }
  };

  const handleSendMessage = (text) => {
    if (webSocket && webSocket.readyState === WebSocket.OPEN) {
      // Send the raw text. The backend will handle encoding.
      webSocket.send(text);
      
      // Add a "pending" or "self" message
      // Note: The backend doesn't send the message back to the sender
      // To show our own message, we must add it manually
      setMessages((prev) => [
        ...prev,
        {
          id: Date.now(),
          type: 'self',
          text: text,
          sender: username,
        },
      ]);
      
    } else {
      setError('WebSocket is not connected.');
    }
  };

  return (
    <div className="flex h-screen w-screen bg-gray-900 text-gray-100">
      {/* Sidebar / Room List */}
      <RoomList
        rooms={rooms}
        currentRoom={currentRoom}
        username={username}
        onSelectRoom={setCurrentRoom}
        onCreateRoom={handleCreateRoom}
        onInvite={handleInvite}
        onLogout={onLogout}
      />
      
      {/* Main Chat Area */}
      <div className="flex flex-col flex-1">
        {currentRoom ? (
          <>
            <MessageArea messages={messages} />
            <MessageInput onSend={handleSendMessage} />
          </>
        ) : (
          <div className="flex-1 flex flex-col items-center justify-center text-gray-500">
            <MessageSquare className="h-24 w-24" />
            <p className="text-xl mt-4">Select a room to start chatting</p>
            <p className="text-lg">or create a new room in the sidebar.</p>
          </div>
        )}
      </div>

      {/* Error Modal */}
      {error && <ErrorModal error={error} onClose={() => setError(null)} />}
    </div>
  );
}

// --- Chat Sub-Components ---

function RoomList({ rooms, currentRoom, username, onSelectRoom, onCreateRoom, onInvite, onLogout }) {
  const [newRoomName, setNewRoomName] = useState('');
  const [inviteUsername, setInviteUsername] = useState('');

  const handleCreate = (e) => {
    e.preventDefault();
    if (newRoomName.trim()) {
      onCreateRoom(newRoomName.trim());
      setNewRoomName('');
    }
  };
  
  const handleInvite = (e) => {
    e.preventDefault();
    if (inviteUsername.trim() && currentRoom) {
      onInvite(inviteUsername.trim());
      setInviteUsername('');
    }
  };

  return (
    <div className="w-80 flex-shrink-0 bg-gray-800 flex flex-col p-4 border-r border-gray-700">
      {/* User Info */}
      <div className="flex items-center justify-between p-2 mb-4">
        <div className="flex items-center">
          <div className="w-10 h-10 rounded-full bg-indigo-500 flex items-center justify-center font-bold text-xl">
            {username[0].toUpperCase()}
          </div>
          <span className="ml-3 font-semibold text-lg">{username}</span>
        </div>
        <button onClick={onLogout} title="Logout" className="p-2 rounded-lg text-gray-400 hover:bg-gray-700 hover:text-gray-100">
          <LogOut className="h-5 w-5" />
        </button>
      </div>

      {/* Room Creation */}
      <form onSubmit={handleCreate} className="flex mb-4">
        <input
          type="text"
          value={newRoomName}
          onChange={(e) => setNewRoomName(e.target.value)}
          placeholder="New Room Name"
          className="flex-1 px-3 py-2 bg-gray-700 border border-gray-600 rounded-l-md text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
        <button type="submit" className="p-2 bg-indigo-600 rounded-r-md hover:bg-indigo-700">
          <Plus className="h-5 w-5" />
        </button>
      </form>

      {/* Room List */}
      <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-2">Rooms</h2>
      <div className="flex-1 overflow-y-auto -mr-2 pr-2">
        {rooms.map((room) => (
          <button
            key={room.id}
            onClick={() => onSelectRoom(room)}
            className={`w-full text-left flex items-center p-3 rounded-lg mb-1 ${
              currentRoom?.id === room.id ? 'bg-indigo-500 text-white' : 'hover:bg-gray-700'
            }`}
          >
            <Hash className="h-5 w-5 mr-2" />
            <span className="flex-1 truncate">{room.name}</span>
          </button>
        ))}
      </div>
      
      {/* Invite Area */}
      {currentRoom && (
        <form onSubmit={handleInvite} className="mt-4 p-4 bg-gray-700 rounded-lg">
          <h3 className="text-sm font-semibold mb-2">Invite to '{currentRoom.name}'</h3>
          <div className="flex">
            <input
              type="text"
              value={inviteUsername}
              onChange={(e) => setInviteUsername(e.target.value)}
              placeholder="Username to invite"
              className="flex-1 px-3 py-2 bg-gray-600 border border-gray-500 rounded-l-md text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
            <button type="submit" className="p-2 bg-indigo-600 rounded-r-md hover:bg-indigo-700">
              <Users className="h-5 w-5" />
            </button>
          </div>
        </form>
      )}
    </div>
  );
}

function MessageArea({ messages }) {
  const messagesEndRef = useRef(null);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const renderMessage = (msg) => {
    switch (msg.type) {
      case 'decoded':
        return (
          <div className="p-3 bg-gray-700 rounded-lg max-w-lg">
            <img
              src={msg.imageUrl}
              alt="Hidden message"
              className="max-w-xs rounded-md mb-2 cursor-pointer"
              onClick={() => window.open(msg.imageUrl, '_blank')}
            />
            <p className="text-lg">{msg.text}</p>
            <span className="text-xs text-gray-400">Decoded from image</span>
          </div>
        );
      case 'image':
        // Fallback if decode failed
        return (
          <div className="p-3 bg-gray-700 rounded-lg max-w-lg">
            <img
              src={msg.imageUrl}
              alt="Received"
              className="max-w-xs rounded-md mb-2"
            />
            <span className="text-xs text-gray-400 italic">Could not decode message</span>
          </div>
        );
      case 'self':
        // This is a message we just sent
        return (
          <div className="p-3 bg-indigo-600 rounded-lg max-w-lg self-end">
            <p className="text-lg">{msg.text}</p>
            <span className_Name="text-xs text-indigo-100">You (Sent)</span>
          </div>
        );
      default:
        return null;
    }
  };
  
  return (
    <div className="flex-1 p-6 overflow-y-auto space-y-4 flex flex-col">
      {messages.map((msg) => (
        <div key={msg.id} className="flex">
          {msg.type !== 'self' && (
            <div className="w-8 h-8 rounded-full bg-gray-600 mr-3 flex-shrink-0" />
          )}
          {renderMessage(msg)}
        </div>
      ))}
      <div ref={messagesEndRef} />
    </div>
  );
}

function MessageInput({ onSend }) {
  const [text, setText] = useState('');

  const handleSubmit = (e) => {
    e.preventDefault();
    if (text.trim()) {
      onSend(text.trim());
      setText('');
    }
  };

  return (
    <form onSubmit={handleSubmit} className="p-4 bg-gray-800 border-t border-gray-700">
      <div className="flex items-center bg-gray-700 rounded-lg">
        <input
          type="text"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Type your secret message..."
          className="flex-1 bg-transparent px-5 py-3 text-lg focus:outline-none"
        />
        <button type="submit" className="p-3 text-indigo-400 hover:text-indigo-300 disabled:text-gray-600" disabled={!text.trim()}>
          <Send className="h-6 w-6" />
        </button>
      </div>
    </form>
  );
}

function ErrorModal({ error, onClose }) {
  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center">
      <div className="bg-gray-800 p-6 rounded-lg shadow-xl max-w-sm w-full">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-lg font-bold text-red-400">An Error Occurred</h3>
          <button onClick={onClose} className="p-1 rounded-full hover:bg-gray-700">
            <X className="h-5 w-5" />
          </button>
        </div>
        <p className="text-gray-200">{error}</p>
      </div>
    </div>
  );
}