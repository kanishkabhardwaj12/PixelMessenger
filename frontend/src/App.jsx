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

// Base URL for your Go backend's HTTP API and WebSocket API.
// These can be configured via Vite env vars (recommended) or fall back to localhost:8082.
// Set in PowerShell before `npm run dev` like:
// $env:VITE_API_BASE_URL = 'http://localhost:8082'; $env:VITE_WS_BASE_URL = 'ws://localhost:8082'; npm run dev
const API_BASE_URL = typeof import.meta !== 'undefined' && import.meta.env && import.meta.env.VITE_API_BASE_URL
  ? import.meta.env.VITE_API_BASE_URL
  : 'http://localhost:8082';

const WS_BASE_URL = typeof import.meta !== 'undefined' && import.meta.env && import.meta.env.VITE_WS_BASE_URL
  ? import.meta.env.VITE_WS_BASE_URL
  : 'ws://localhost:8082';

// --- IndexedDB helper (minimal) ---
// Stores full image blobs under auto-generated keys like 'img_<timestamp>_<rand>'.
const IDB_DB_NAME = 'pm_images_db';
const IDB_STORE_NAME = 'images';

function openDb() {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(IDB_DB_NAME, 1);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(IDB_STORE_NAME)) {
        db.createObjectStore(IDB_STORE_NAME, { keyPath: 'id' });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

async function saveImageBlob(blob) {
  const db = await openDb();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(IDB_STORE_NAME, 'readwrite');
    const store = tx.objectStore(IDB_STORE_NAME);
    const id = `img_${Date.now()}_${Math.floor(Math.random() * 1e6)}`;
    const entry = { id, blob };
    const req = store.add(entry);
    req.onsuccess = () => resolve(id);
    req.onerror = () => reject(req.error);
  });
}

async function getImageBlob(id) {
  if (!id) return null;
  const db = await openDb();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(IDB_STORE_NAME, 'readonly');
    const store = tx.objectStore(IDB_STORE_NAME);
    const req = store.get(id);
    req.onsuccess = () => {
      const res = req.result;
      resolve(res ? res.blob : null);
    };
    req.onerror = () => reject(req.error);
  });
}

// Convert base64 (string) -> Blob
function base64ToBlob(b64, contentType = 'image/png') {
  const byteCharacters = atob(b64);
  const byteNumbers = new Array(byteCharacters.length);
  for (let i = 0; i < byteCharacters.length; i++) {
    byteNumbers[i] = byteCharacters.charCodeAt(i);
  }
  const byteArray = new Uint8Array(byteNumbers);
  return new Blob([byteArray], { type: contentType });
}

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
    <div className="auth-wrapper flex items-center justify-center min-h-screen bg-gray-900 text-gray-100">
      <div className="auth-box w-full max-w-md p-8 space-y-8 bg-gray-800 rounded-2xl shadow-xl">
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
              onChange={(e) => setPassword(e.target.value)}
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
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [wsStatus, setWsStatus] = useState('disconnected'); // 'connecting' | 'connected' | 'disconnected'
  const reconnectRef = useRef({ attempts: 0, timeoutId: null });
  
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

    let ws;

    const createSocket = () => {
      setWsStatus('connecting');
      ws = new WebSocket(wsUrl);
      setWebSocket(ws);
      setMessages([]); // Clear messages when joining a new room

      ws.onopen = () => {
        console.log(`WebSocket connected to room: ${currentRoom.id}`);
        setWsStatus('connected');
        reconnectRef.current.attempts = 0;
        if (reconnectRef.current.timeoutId) {
          clearTimeout(reconnectRef.current.timeoutId);
          reconnectRef.current.timeoutId = null;
        }
      };

      ws.onclose = () => {
        console.log('WebSocket disconnected');
        setWsStatus('disconnected');
        // Try to reconnect with exponential backoff (capped)
        const attempts = reconnectRef.current.attempts || 0;
        const delay = Math.min(30000, 1000 * Math.pow(2, attempts));
        reconnectRef.current.attempts = attempts + 1;
        reconnectRef.current.timeoutId = setTimeout(() => {
          createSocket();
        }, delay);
      };

      ws.onerror = (err) => {
        console.error('WebSocket error:', err);
        setError('WebSocket connection failed. Check console.');
      };

      // Message handler for this socket instance

      ws.onmessage = async (event) => {
        // The backend now sends a JSON text message containing both the
        // encoded image (base64) and the decoded text. We handle both that
        // case and the legacy Blob case.
        if (typeof event.data === 'string') {
          try {
            const payload = JSON.parse(event.data);
            if (payload && payload.type === 'image' && payload.image_base64) {
              // Convert base64 -> Blob, persist in IndexedDB, and display object URL
              (async () => {
                try {
                  const b64 = payload.image_base64;
                  const blob = base64ToBlob(b64, 'image/png');
                  const imageId = await saveImageBlob(blob).catch(() => null);
                  const objectUrl = blob ? URL.createObjectURL(blob) : null;
                  const decodedText = payload.decoded_text || '';
                  setMessages((prev) => {
                    // Remove any optimistic pending message that matches this image (match by base64 if present)
                    const filtered = prev.filter((m) => !(m.pending && m.imageBase64 === b64));
                    return [
                      ...filtered,
                      {
                        id: Date.now(),
                        type: 'decoded',
                        imageUrl: objectUrl,
                        imageId: imageId || null,
                        text: decodedText,
                        sender: payload.sender_name || payload.sender_id || 'Unknown',
                        timestamp: payload.timestamp || new Date().toISOString(),
                        pending: false,
                      },
                    ];
                  });
                } catch (e) {
                  console.error('Failed to process incoming base64 image', e);
                }
              })();
              return;
            }
          } catch (e) {
            // Not JSON or unexpected format — fall back to treating it as a simple text message
            const payloadText = event.data;
            setMessages((prev) => [
              ...prev,
              {
                id: Date.now(),
                type: 'other',
                text: payloadText,
                sender: 'system',
                timestamp: new Date().toISOString(),
              },
            ]);
            return;
          }
        }

        // Legacy: binary Blob handling (if backend sends raw binary frames)
        if (event.data instanceof Blob) {
          const imageBlob = event.data;

          // Persist blob in IndexedDB and decode via API
          (async () => {
            let objectUrl = null;
            let imageId = null;
            try {
              imageId = await saveImageBlob(imageBlob).catch(() => null);
              objectUrl = imageBlob ? URL.createObjectURL(imageBlob) : null;
            } catch (e) {
              console.warn('Failed to save incoming blob to IDB', e);
            }

            try {
              const response = await apiFetch('/decode', token, {
                method: 'POST',
                body: imageBlob,
              });

              let decodedText = '';
              if (response && response.encoding === 'utf8' && typeof response.message === 'string') {
                decodedText = response.message;
              } else if (response && response.encoding === 'base64' && typeof response.message_base64 === 'string') {
                try {
                  const binStr = atob(response.message_base64);
                  const len = binStr.length;
                  const bytes = new Uint8Array(len);
                  for (let i = 0; i < len; i++) bytes[i] = binStr.charCodeAt(i);
                  const decoded = new TextDecoder().decode(bytes);
                  decodedText = decoded || `[binary message: ${response.message_base64}]`;
                } catch (e) {
                  decodedText = `[binary message: ${response.message_base64}]`;
                }
              } else if (response && typeof response.message === 'string') {
                decodedText = response.message;
              }

              setMessages((prev) => [
                ...prev,
                {
                  id: Date.now(),
                  type: 'decoded',
                  imageUrl: objectUrl || null,
                  imageId: imageId || null,
                  text: decodedText,
                  sender: '???',
                  timestamp: new Date().toISOString(),
                },
              ]);
            } catch (err) {
              console.error('Failed to decode image:', err);
              setMessages((prev) => [
                ...prev,
                {
                  id: Date.now(),
                  type: 'image',
                  imageUrl: objectUrl || null,
                  imageId: imageId || null,
                  sender: '???',
                  timestamp: new Date().toISOString(),
                },
              ]);
            }
          })();
        }
      };
    };

    createSocket();

    

    // Cleanup function
    return () => {
      try {
        if (ws && ws.readyState === WebSocket.OPEN) ws.close();
      } finally {
        if (reconnectRef.current.timeoutId) clearTimeout(reconnectRef.current.timeoutId);
        reconnectRef.current.attempts = 0;
      }
    };
  }, [currentRoom, token]);

  // Persist messages per-room in localStorage so refreshes don't lose chat history.
  const storageKeyFor = (roomId) => `pm_room_${roomId}_messages`;

  // Load cached messages when joining a room
  useEffect(() => {
    if (!currentRoom) {
      setMessages([]);
      return;
    }
    try {
      const key = storageKeyFor(currentRoom.id);
      const raw = localStorage.getItem(key);
      if (raw) {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) {
          // If messages include imageId entries, resolve them from IndexedDB
          (async () => {
            try {
              const resolved = [];
              for (const m of parsed) {
                const copy = { ...m };
                // If there's an imageId, load the blob and create an object URL
                if (copy.imageId) {
                  try {
                    const blob = await getImageBlob(copy.imageId).catch(() => null);
                    if (blob) {
                      copy.imageUrl = URL.createObjectURL(blob);
                    } else if (copy.imageUrl && typeof copy.imageUrl === 'string') {
                      // leave thumbnail if present
                    } else {
                      copy.imageUrl = null;
                    }
                  } catch (e) {
                    copy.imageUrl = copy.imageUrl || null;
                  }
                }
                // Normalize blob: urls (older cache)
                if (copy && copy.imageUrl && typeof copy.imageUrl === 'string' && copy.imageUrl.startsWith('blob:')) {
                  copy.imageUrl = null;
                }
                resolved.push(copy);
              }
              setMessages(resolved);
            } catch (e) {
              console.warn('Failed to resolve cached images', e);
              setMessages(parsed);
            }
          })();
          return;
        }
      }
    } catch (e) {
      console.warn('Failed to load cached messages', e);
    }
    // If nothing cached, start with empty messages and rely on live events
    setMessages([]);
  }, [currentRoom]);

  // Save messages to localStorage whenever they change
  useEffect(() => {
    if (!currentRoom) return;
    try {
      const key = storageKeyFor(currentRoom.id);
      // We'll asynchronously create small thumbnails for large data URLs
      // to avoid hitting localStorage quota while keeping a representative image.
      (async () => {
        const sanitized = [];
        for (const m of messages) {
          try {
            const copy = { ...m };
            if (copy.imageBase64) delete copy.imageBase64;

            // If we already persisted the full image in IndexedDB, keep only the imageId
            if (copy.imageId) {
              // Remove transient object URLs before persisting
              if (copy.imageUrl && typeof copy.imageUrl === 'string' && copy.imageUrl.startsWith('blob:')) {
                delete copy.imageUrl;
              }
            } else {
              // If imageUrl is a large data URL, attempt to make a small thumbnail
              if (typeof copy.imageUrl === 'string' && copy.imageUrl.startsWith('data:') && copy.imageUrl.length > 100000) {
                const thumb = await createThumbnail(copy.imageUrl, 200);
                copy.imageUrl = thumb || null;
              }
              // Normalize object URLs and empty strings to null to avoid rendering issues
              if (copy.imageUrl === '' || (typeof copy.imageUrl === 'string' && copy.imageUrl.startsWith('blob:'))) copy.imageUrl = null;
            }

            sanitized.push(copy);
          } catch (e) {
            // skip problematic entries
          }
        }

        try {
          localStorage.setItem(key, JSON.stringify(sanitized));
        } catch (e) {
          console.warn('Failed to save cached messages', e);
        }
      })();
    } catch (e) {
      console.warn('Failed to save cached messages', e);
    }
  }, [messages, currentRoom]);


  // Helper: create a small PNG thumbnail (data URL) from a large data URL image.
  // Returns data URL string or null on failure.
  const createThumbnail = (dataUrl, maxWidth = 200) => {
    return new Promise((resolve) => {
      try {
        const img = new Image();
        img.onload = () => {
          try {
            const ratio = img.width / img.height || 1;
            const w = Math.min(maxWidth, img.width || maxWidth);
            const h = Math.max(1, Math.round(w / ratio));
            const canvas = document.createElement('canvas');
            canvas.width = w;
            canvas.height = h;
            const ctx = canvas.getContext('2d');
            ctx.drawImage(img, 0, 0, w, h);
            // Use PNG for compatibility; smaller dims keep size low.
            const thumb = canvas.toDataURL('image/png');
            resolve(thumb);
          } catch (e) {
            resolve(null);
          }
        };
        img.onerror = () => resolve(null);
        img.src = dataUrl;
      } catch (e) {
        resolve(null);
      }
    });
  };

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
      // Do not add a plain-text 'self' message here. The backend now broadcasts
      // the encoded image (and the sender will receive it too). We rely on the
      // incoming WebSocket message to render the encoded image + decoded text
      // so the sender and receivers see the exact same thing.
      
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
        wsStatus={wsStatus}
        setMessages={setMessages}
      />
      
      {/* Main Chat Area + Right Panel */}
      <div className="flex flex-1">
        <div className="flex flex-col flex-1">
          {currentRoom ? (
            <>
              <MessageArea messages={messages} username={username} />
              <MessageInput onSend={handleSendMessage} />
            </>
          ) : (
            <div className="flex-1 flex flex-col items-center justify-center text-gray-500">
              <MessageSquare className="h-24 w-27" />
              <p className="text-xl mt-4">Select a room to start chatting</p>
              <p className="text-lg">or create a new room in the sidebar.</p>
            </div>
          )}
        </div>

        {/* Right-side panel removed in favor of a collapsible drawer (toggle below) */}
      </div>

      {/* Floating toggle + collapsible right drawer for the CustomImageForm */}
      {/* Overlay (click to close) */}
      <div className={`${drawerOpen ? 'fixed inset-0 bg-black bg-opacity-40 z-40' : 'hidden'}`} onClick={() => setDrawerOpen(false)} />

      {/* Drawer */}
      <div className={`fixed right-0 top-0 h-full w-80 bg-gray-900 shadow-xl transform transition-transform duration-300 z-50 ${drawerOpen ? 'translate-x-0' : 'translate-x-full'}`}>
        <div className="flex items-center justify-between p-4 border-b border-gray-800">
          <div className="text-sm font-semibold text-gray-100">Send custom image</div>
          <button onClick={() => setDrawerOpen(false)} className="p-2 rounded hover:bg-gray-800">
            <X className="h-5 w-5 text-gray-200" />
          </button>
        </div>
        <div className="p-4 overflow-y-auto h-[calc(100%-64px)]">
          {currentRoom ? (
            <CustomImageForm currentRoom={currentRoom} token={token} username={username} setMessages={setMessages} />
          ) : (
            <div className="text-sm text-gray-400">Select a room to use the encoder</div>
          )}
        </div>
      </div>

      {/* Floating toggle button */}
      <button onClick={() => setDrawerOpen(true)} title="Open image encoder" className="fixed right-6 bottom-24 z-50 p-3 bg-indigo-600 rounded-full shadow-lg hover:bg-indigo-700">
        <ImageIcon className="h-5 w-5 text-white" />
      </button>

      {/* Error Modal */}
      {error && <ErrorModal error={error} onClose={() => setError(null)} />}
    </div>
  );
}

// --- Chat Sub-Components ---

function RoomList({ rooms, currentRoom, username, onSelectRoom, onCreateRoom, onInvite, onLogout, wsStatus, setMessages }) {
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
      {/* Header: app title + connection status */}
      <div className="pm-header p-2 mb-3">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-indigo-500 flex items-center justify-center font-bold text-xl">PM</div>
          <div>
            <div className="pm-title">PixelMessenger</div>
            <div className="text-xs text-gray-400">Secure image-based chat</div>
          </div>
        </div>
        <div>
          <span className={`pm-connection ${wsStatus || 'disconnected'}`}>{wsStatus}</span>
        </div>
      </div>

      {/* User Info */}
      <div className="flex items-center justify-between p-2 mb-4">
        <div className="flex items-center">
          <div className="w-10 h-10 rounded-full bg-indigo-500 flex items-center justify-center font-bold text-xl">
            {(username && username.length > 0) ? username[0].toUpperCase() : '?'}
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

function CustomImageForm({ currentRoom, token, username, setMessages }) {
  const [file, setFile] = useState(null);
  const [message, setMessage] = useState('');
  const [passphrase, setPassphrase] = useState('');
  const [isSending, setIsSending] = useState(false);
  const [error, setError] = useState(null);

  const onFileChange = (e) => {
    setFile(e.target.files && e.target.files[0]);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    if (!file || !message.trim()) {
      setError('Please select a file and enter a secret message');
      return;
    }
    setIsSending(true);
    try {
      // Send multipart/form-data directly to /encode
      const form = new FormData();
      form.append('image', file);
      if (passphrase && passphrase.trim() !== '') {
        form.append('passphrase', passphrase.trim());
      }
      form.append('message', message.trim());
      form.append('room_id', currentRoom.id);

      const resp = await fetch(`${API_BASE_URL}/encode`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` },
        body: form,
      });

      if (!resp.ok) {
        const txt = await resp.text();
        throw new Error(txt || 'Failed to upload');
      }

      const data = await resp.json();

      // Optimistic UI: show the encoded image returned by the server immediately
      if (data && data.encoded_image) {
        const encoded = data.encoded_image;
        // Convert base64 to blob and persist in IndexedDB so the full image
        // is available after refresh; display object URL immediately.
        try {
          const blob = base64ToBlob(encoded, 'image/png');
          const imageId = await saveImageBlob(blob).catch(() => null);
          const objectUrl = blob ? URL.createObjectURL(blob) : null;
          const pendingMessage = {
            id: 'pending-' + Date.now(),
            type: 'decoded',
            imageUrl: objectUrl,
            imageId: imageId || null,
            text: data.decoded_message || message.trim(),
            sender: username || 'You',
            timestamp: new Date().toISOString(),
            pending: true,
          };
          setMessages((prev) => [...prev, pendingMessage]);
        } catch (e) {
          // fallback to inline data URL if something goes wrong
          const dataUrl = `data:image/png;base64,${encoded}`;
          const pendingMessage = {
            id: 'pending-' + Date.now(),
            type: 'decoded',
            imageUrl: dataUrl,
            text: data.decoded_message || message.trim(),
            sender: username || 'You',
            timestamp: new Date().toISOString(),
            pending: true,
          };
          setMessages((prev) => [...prev, pendingMessage]);
        }
      }

      // Clear form
      setMessage('');
  setPassphrase('');
      setFile(null);
      const input = document.getElementById('custom-image-input');
      if (input) input.value = '';
    } catch (err) {
      console.error(err);
      setError(err.message || 'Failed to send');
    } finally {
      setIsSending(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-2">
      <input id="custom-image-input" type="file" accept="image/*" onChange={onFileChange} />
      <div className="flex items-center gap-2">
        <Key className="h-4 w-4 text-gray-400" />
        <input type="password" value={passphrase} onChange={(e) => setPassphrase(e.target.value)} placeholder="Optional passphrase (for keyed embedding)" className="flex-1 px-2 py-1 bg-gray-600 rounded" />
      </div>
      <input type="text" value={message} onChange={(e) => setMessage(e.target.value)} placeholder="Secret message" className="w-full px-2 py-1 bg-gray-600 rounded" />
      {error && <div className="text-xs text-red-400">{error}</div>}
      <div className="flex">
        <button type="submit" disabled={isSending} className="flex-1 p-2 bg-indigo-600 rounded hover:bg-indigo-700">{isSending ? 'Sending...' : 'Send with image'}</button>
      </div>
    </form>
  );
}

function MessageArea({ messages, username }) {
  const messagesEndRef = useRef(null);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const renderMessage = (msg) => {
    switch (msg.type) {
      case 'decoded':
        {
          const displaySender = msg.sender ? (msg.sender === username ? 'You' : msg.sender) : 'Unknown';
          return (
            <div className="msg-bubble other p-3 rounded-lg max-w-lg">
                  {msg.imageUrl ? (
                    <img
                      src={msg.imageUrl}
                      alt="Hidden message"
                      className="max-w-xs rounded-md mb-2 cursor-pointer"
                      onClick={() => window.open(msg.imageUrl, '_blank')}
                    />
                  ) : null}
              <p className="text-lg">{msg.text}  • Decoded from image • </p>
              <div className="msg-meta">{displaySender} {msg.timestamp ? new Date(msg.timestamp).toLocaleTimeString() : ''}</div>
            </div>
          );
        }
      case 'image':
        // Fallback if decode failed
        return (
          <div className="msg-bubble other p-3 rounded-lg max-w-lg">
            {msg.imageUrl ? (
              <img
                src={msg.imageUrl}
                alt="Received"
                className="max-w-xs rounded-md mb-2"
              />
            ) : null}
            <div className="msg-meta">Could not decode message • {msg.timestamp ? new Date(msg.timestamp).toLocaleTimeString() : ''}</div>
          </div>
        );
      case 'self':
        // This is a message we just sent
        return (
          <div className="msg-bubble self p-3 rounded-lg max-w-lg self-end ml-auto text-right">
            <p className="text-lg">{msg.text}</p>
            <div className="msg-meta">You • {msg.timestamp ? new Date(msg.timestamp).toLocaleTimeString() : ''}</div>
          </div>
        );
      case 'other':
        return (
          <div className="msg-bubble other p-2 rounded-lg max-w-lg">
            <p className="text-md">{msg.text}</p>
            <div className="msg-meta">{msg.sender} • {msg.timestamp ? new Date(msg.timestamp).toLocaleTimeString() : ''}</div>
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
    <form id="message-input" onSubmit={handleSubmit} className="p-4 bg-gray-800 border-t border-gray-700">
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