import React, { useState, useEffect, useRef, useCallback } from 'react';
import { BrowserRouter as Router, Routes, Route, useNavigate } from 'react-router-dom';
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
  Trash2,
} from 'lucide-react';
import axios from 'axios';
import StegoAnalysis from './StegoAnalysis';

// --- Configuration ---
const API_BASE_URL = typeof import.meta !== 'undefined' && import.meta.env && import.meta.env.VITE_API_BASE_URL
  ? import.meta.env.VITE_API_BASE_URL
  : 'http://localhost:8082';

const WS_BASE_URL = typeof import.meta !== 'undefined' && import.meta.env && import.meta.env.VITE_WS_BASE_URL
  ? import.meta.env.VITE_WS_BASE_URL
  : 'ws://localhost:8082';

// --- IndexedDB Helper (Persist Large Images) ---
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

// Helper: Base64 string -> Blob
function base64ToBlob(b64, contentType = 'image/png') {
  try {
    const byteCharacters = atob(b64);
    const byteNumbers = new Array(byteCharacters.length);
    for (let i = 0; i < byteCharacters.length; i++) {
      byteNumbers[i] = byteCharacters.charCodeAt(i);
    }
    const byteArray = new Uint8Array(byteNumbers);
    return new Blob([byteArray], { type: contentType });
  } catch (e) {
    console.error("Failed to convert base64 to blob", e);
    return null;
  }
}

// --- Auth Utilities ---

const parseJwt = (token) => {
  try {
    return JSON.parse(atob(token.split('.')[1]));
  } catch (e) {
    return null;
  }
};

const useAuth = () => {
  const [token, setToken] = useState(null);
  const [username, setUsername] = useState(null);
  const [userId, setUserId] = useState(null);

  useEffect(() => {
    const storedToken = localStorage.getItem('pixel-token');
    if (storedToken) {
      const claims = parseJwt(storedToken);
      if (claims && claims.exp * 1000 > Date.now()) {
        setToken(storedToken);
        setUsername(claims.username);
        setUserId(claims.user_id); // Ensure backend sends "user_id" in claims
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
    localStorage.removeItem('pixel-current-room');
    setToken(null);
    setUsername(null);
    setUserId(null);
  };

  return { token, username, userId, login, logout };
};

const apiFetch = async (endpoint, token, options = {}) => {
  const headers = {
    'Authorization': `Bearer ${token}`,
    ...(options.headers || {})
  };

  if (options.body && !(options.body instanceof FormData) && typeof options.body === 'object') {
    headers['Content-Type'] = 'application/json';
  }

  const config = {
    url: `${API_BASE_URL}${endpoint}`,
    method: options.method || 'GET',
    headers,
    data: options.body,
    validateStatus: null,
  };

  const response = await axios(config);
  if (response.status < 200 || response.status >= 300) {
    const errorBody = typeof response.data === 'string' ? response.data : JSON.stringify(response.data);
    throw new Error(errorBody || 'API request failed');
  }
  return response.data;
};

// --- Main App ---

export default function App() {
  const { token, username, userId, login, logout } = useAuth();

  if (!token) {
    return <AuthPage onLogin={login} />;
  }

  return (
    <Router>
      <Routes>
        <Route 
          path="/" 
          element={<ChatPage token={token} username={username} userId={userId} onLogout={logout} />} 
        />
        <Route 
          path="/analysis" 
          element={<StegoAnalysis token={token} onBack={() => window.history.back()} />} 
        />
      </Routes>
    </Router>
  );
}

// --- Auth Page ---

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
      const response = await axios.post(`${API_BASE_URL}${endpoint}`, {
        username,
        password
      });

      const data = response.data;

      if (isLogin) {
        onLogin(data.token);
      } else {
        // Auto-login after register
        const loginResponse = await axios.post(`${API_BASE_URL}/login`, {
          username,
          password
        });
        onLogin(loginResponse.data.token);
      }
    } catch (err) {
      const errorMsg = err.response?.data?.error || err.response?.data || err.message || 'An error occurred';
      setError(errorMsg);
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
              type="text"
              required
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="peer relative block w-full px-4 py-3 pl-12 text-lg bg-gray-700 border border-gray-600 rounded-md placeholder-gray-400 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
              placeholder="Username"
            />
            <User className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-gray-400" />
          </div>
          
          <div className="relative">
            <input
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="peer relative block w-full px-4 py-3 pl-12 text-lg bg-gray-700 border border-gray-600 rounded-md placeholder-gray-400 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
              placeholder="Password"
            />
            <Lock className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-gray-400" />
          </div>
          
          {error && <p className="text-center text-sm text-red-400">{error}</p>}

          <button
            type="submit"
            disabled={isLoading}
            className="w-full flex justify-center py-3 px-4 border border-transparent text-lg font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50"
          >
            {isLoading ? <Loader2 className="h-6 w-6 animate-spin" /> : (isLogin ? 'Sign In' : 'Sign Up')}
          </button>
        </form>
        
        <div className="text-center">
          <button
            onClick={() => { setIsLogin(!isLogin); setError(null); }}
            className="font-medium text-indigo-400 hover:text-indigo-300"
          >
            {isLogin ? 'Need an account? Sign up' : 'Already have an account? Sign in'}
          </button>
        </div>
      </div>
    </div>
  );
}

// --- Chat Page ---

function ChatPage({ token, username, userId, onLogout }) {
  const navigate = useNavigate();
  const [rooms, setRooms] = useState([]);
  const [currentRoom, setCurrentRoom] = useState(() => {
    const saved = localStorage.getItem('pixel-current-room');
    return saved ? JSON.parse(saved) : null;
  });

  const [messages, setMessages] = useState([]);
  const [error, setError] = useState(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [wsStatus, setWsStatus] = useState('disconnected');
  
  // Refs for stable connection management
  const socketRef = useRef(null);
  const reconnectRef = useRef({ attempts: 0, timeoutId: null });
  const messagesLoadedRef = useRef(false);

  // Persist current room
  useEffect(() => {
    if (currentRoom) {
      localStorage.setItem('pixel-current-room', JSON.stringify(currentRoom));
    } else {
      localStorage.removeItem('pixel-current-room');
    }
  }, [currentRoom]);

  // Fetch Rooms
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

  // Load History
  const fetchRoomMessages = useCallback(async (roomId) => {
    try {
      const msgs = await apiFetch(`/rooms/${roomId}/messages`, token);
      if (Array.isArray(msgs) && msgs.length > 0) {
        const reconstructed = await Promise.all(
          msgs.map(async (m) => {
            let objectUrl = null;
            
            if (m.encoded_image_base64) {
              const blob = base64ToBlob(m.encoded_image_base64);
              if (blob) objectUrl = URL.createObjectURL(blob);
            }
            
            return {
              id: m.id || Date.now() + Math.random(),
              type: 'decoded',
              imageUrl: objectUrl,
              text: '',
              sender: m.sender_name || 'Unknown', // Use sender_name from DB join if available
              senderId: m.sender_id,
              timestamp: m.created_at || new Date().toISOString(),
              pending: false,
            };
          })
        );
        setMessages(reconstructed);
      } else {
        setMessages([]);
      }
    } catch (err) {
      console.warn('Failed to fetch messages:', err);
    }
  }, [token]);

  // Delete Message
  const deleteMessage = useCallback(async (messageId) => {
    if (!currentRoom) return;
    try {
      await apiFetch(`/rooms/${currentRoom.id}/messages/${messageId}`, token, { method: 'DELETE' });
      setMessages((prev) => prev.filter((m) => m.id !== messageId));
      
      if (socketRef.current?.readyState === WebSocket.OPEN) {
        socketRef.current.send(JSON.stringify({ type: 'delete', messageId }));
      }
    } catch (err) {
      alert('Failed to delete message: ' + err.message);
    }
  }, [currentRoom, token]);

  // --- WebSocket Logic (Fixed) ---
  useEffect(() => {
    if (!currentRoom || !token) return;

    // Strict Mode: Check if connection exists
    if (socketRef.current && (socketRef.current.readyState === WebSocket.OPEN || socketRef.current.readyState === WebSocket.CONNECTING)) {
      return; 
    }

    messagesLoadedRef.current = false;
    const wsUrl = `${WS_BASE_URL}/ws?room=${currentRoom.id}&token=${token}`;

    const connect = () => {
      setWsStatus('connecting');
      socketRef.current = new WebSocket(wsUrl);

      // Fetch history immediately before socket opens (or after)
      if (!messagesLoadedRef.current) {
        fetchRoomMessages(currentRoom.id);
        messagesLoadedRef.current = true;
      }

      socketRef.current.onopen = () => {
        console.log(`WebSocket connected to room: ${currentRoom.id}`);
        setWsStatus('connected');
        reconnectRef.current.attempts = 0;
      };

      socketRef.current.onclose = (event) => {
        console.log('WebSocket disconnected', event.code);
        setWsStatus('disconnected');
        socketRef.current = null;

        // STOP RECONNECTING ON AUTH FAILURES
        if (event.code === 1008 || event.code === 401 || reconnectRef.current.attempts > 5) {
          console.error("Connection rejected. Please login again.");
          return;
        }

        const delay = Math.min(10000, 1000 * Math.pow(2, reconnectRef.current.attempts));
        reconnectRef.current.attempts++;
        reconnectRef.current.timeoutId = setTimeout(connect, delay);
      };

      socketRef.current.onmessage = async (event) => {
        try {
          const payload = JSON.parse(event.data);
          
          if (payload.type === 'delete') {
            setMessages(prev => prev.filter(m => m.id !== payload.messageId));
            return;
          }

          if (payload.type === 'image') {
            const b64 = payload.image_base64;
            const blob = base64ToBlob(b64);
            const objectUrl = blob ? URL.createObjectURL(blob) : null;

            setMessages(prev => {
               // De-duplication
               if (prev.some(m => m.id === payload.message_id)) return prev;
               // Remove pending
               const filtered = prev.filter(m => !(m.pending && m.imageBase64 === b64));
               
               return [...filtered, {
                 id: payload.message_id,
                 type: 'decoded',
                 imageUrl: objectUrl,
                 text: '',
                 sender: payload.sender_name || 'User',
                 senderId: payload.sender_id,
                 timestamp: payload.timestamp || new Date().toISOString(),
                 pending: false
               }];
            });
          }
        } catch (e) {
          // Plain text fallback
          setMessages(prev => [...prev, {
            id: Date.now(),
            type: 'other',
            text: event.data,
            sender: 'System',
            timestamp: new Date().toISOString()
          }]);
        }
      };
    };

    connect();

    return () => {
      if (reconnectRef.current.timeoutId) clearTimeout(reconnectRef.current.timeoutId);
      if (socketRef.current) {
        socketRef.current.close();
        socketRef.current = null;
      }
    };
  }, [currentRoom, token, fetchRoomMessages]);

  const handleSendMessage = (text) => {
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      setMessages(prev => [...prev, {
        id: 'pending-' + Date.now(),
        type: 'self',
        text: '',
        sender: username,
        timestamp: new Date().toISOString(),
        pending: true
      }]);
      socketRef.current.send(text); // Backend handles text -> image conversion
    } else {
      setError('WebSocket is not connected.');
    }
  };

  const handleCreateRoom = async (roomName) => {
    try {
      const newRoom = await apiFetch('/rooms', token, {
        method: 'POST',
        body: JSON.stringify({ name: roomName }),
      });
      setRooms([...rooms, newRoom]);
      setCurrentRoom(newRoom);
    } catch (err) {
      setError(`Failed to create room: ${err.message}`);
    }
  };
  
  const handleInvite = async (inviteUsername) => {
    if (!currentRoom) return;
    try {
      await apiFetch(`/rooms/${currentRoom.id}/invite`, token, {
        method: 'POST',
        body: JSON.stringify({ username: inviteUsername }),
      });
      alert(`Invited ${inviteUsername}`);
    } catch (err) {
      setError(`Failed to invite: ${err.message}`);
    }
  };

  return (
    <div className="flex h-screen w-screen bg-gray-900 text-gray-100">
      <RoomList
        rooms={rooms}
        currentRoom={currentRoom}
        username={username}
        onSelectRoom={setCurrentRoom}
        onCreateRoom={handleCreateRoom}
        onInvite={handleInvite}
        onLogout={onLogout}
        wsStatus={wsStatus}
      />
      
      <div className="flex flex-col flex-1">
        {currentRoom ? (
          <>
            <div className="p-4 bg-gray-800 border-b border-gray-700 flex justify-end">
              <button 
                onClick={() => navigate('/analysis')} 
                className="px-4 py-2 bg-indigo-600 rounded hover:bg-indigo-700 text-white"
              >
                Steganalysis
              </button>
            </div>
            <MessageArea 
              messages={messages} 
              username={username} 
              userId={userId} 
              onDeleteMessage={deleteMessage} 
            />
            <MessageInput onSend={handleSendMessage} />
          </>
        ) : (
          <div className="flex-1 flex flex-col items-center justify-center text-gray-500">
            <MessageSquare className="h-24 w-24 mb-4" />
            <p className="text-xl">Select a room to chat</p>
          </div>
        )}
      </div>

      {/* Custom Image Drawer */}
      <div className={`fixed inset-0 bg-black bg-opacity-40 z-40 ${drawerOpen ? '' : 'hidden'}`} onClick={() => setDrawerOpen(false)} />
      <div className={`fixed right-0 top-0 h-full w-80 bg-gray-900 shadow-xl transform transition-transform duration-300 z-50 ${drawerOpen ? 'translate-x-0' : 'translate-x-full'}`}>
        <div className="flex items-center justify-between p-4 border-b border-gray-800">
          <div className="text-sm font-semibold">Send Custom Image</div>
          <button onClick={() => setDrawerOpen(false)}><X className="h-5 w-5" /></button>
        </div>
        <div className="p-4">
          {currentRoom && (
            <CustomImageForm 
              currentRoom={currentRoom} 
              token={token} 
              username={username} 
              setMessages={setMessages} 
            />
          )}
        </div>
      </div>

      <button onClick={() => setDrawerOpen(true)} className="fixed right-6 bottom-24 z-50 p-3 bg-indigo-600 rounded-full shadow-lg hover:bg-indigo-700">
        <ImageIcon className="h-5 w-5 text-white" />
      </button>

      {error && <ErrorModal error={error} onClose={() => setError(null)} />}
    </div>
  );
}

// --- Sub-Components ---

function RoomList({ rooms, currentRoom, username, onSelectRoom, onCreateRoom, onInvite, onLogout, wsStatus }) {
  const [newRoomName, setNewRoomName] = useState('');
  const [inviteUser, setInviteUser] = useState('');

  return (
    <div className="w-80 flex-shrink-0 bg-gray-800 flex flex-col p-4 border-r border-gray-700">
      <div className="mb-6">
        <h1 className="text-xl font-bold text-indigo-400">PixelMessenger</h1>
        <div className={`text-xs mt-1 ${wsStatus === 'connected' ? 'text-green-400' : 'text-red-400'}`}>
          ● {wsStatus}
        </div>
      </div>

      <div className="flex items-center justify-between mb-6 p-2 bg-gray-700 rounded-lg">
        <div className="flex items-center">
          <div className="w-8 h-8 rounded-full bg-indigo-500 flex items-center justify-center font-bold">
            {username?.[0]?.toUpperCase()}
          </div>
          <span className="ml-2 font-medium truncate max-w-[120px]">{username}</span>
        </div>
        <button onClick={onLogout} className="text-gray-400 hover:text-white"><LogOut size={18} /></button>
      </div>

      <form onSubmit={(e) => { e.preventDefault(); if(newRoomName) { onCreateRoom(newRoomName); setNewRoomName(''); } }} className="flex mb-4">
        <input 
          className="flex-1 bg-gray-700 px-3 py-2 rounded-l text-sm focus:outline-none"
          placeholder="New Room" 
          value={newRoomName} 
          onChange={e => setNewRoomName(e.target.value)} 
        />
        <button type="submit" className="bg-indigo-600 px-3 rounded-r"><Plus size={18} /></button>
      </form>

      <div className="flex-1 overflow-y-auto space-y-1">
        {rooms.map(r => (
          <button
            key={r.id}
            onClick={() => onSelectRoom(r)}
            className={`w-full text-left p-3 rounded flex items-center ${currentRoom?.id === r.id ? 'bg-indigo-600' : 'hover:bg-gray-700'}`}
          >
            <Hash size={16} className="mr-2 opacity-70" />
            <span className="truncate">{r.name}</span>
          </button>
        ))}
      </div>

      {currentRoom && (
        <form onSubmit={(e) => { e.preventDefault(); if(inviteUser) { onInvite(inviteUser); setInviteUser(''); } }} className="mt-4 pt-4 border-t border-gray-700">
          <div className="flex">
            <input 
              className="flex-1 bg-gray-700 px-3 py-2 rounded-l text-sm focus:outline-none"
              placeholder="Invite User" 
              value={inviteUser} 
              onChange={e => setInviteUser(e.target.value)} 
            />
            <button type="submit" className="bg-indigo-600 px-3 rounded-r"><Users size={18} /></button>
          </div>
        </form>
      )}
    </div>
  );
}

function MessageArea({ messages, username, userId, onDeleteMessage }) {
  const endRef = useRef(null);
  useEffect(() => endRef.current?.scrollIntoView({ behavior: 'smooth' }), [messages]);

  return (
    <div className="flex-1 p-6 overflow-y-auto space-y-4">
      {messages.map((msg) => {
        const isMe = msg.sender === username || msg.senderId === userId;
        return (
          <div key={msg.id} className={`flex ${isMe ? 'justify-end' : 'justify-start'}`}>
            <div className={`max-w-[70%] rounded-lg p-3 ${isMe ? 'bg-indigo-600' : 'bg-gray-700'} relative group`}>
              
              {isMe && !msg.pending && (
                <button 
                  onClick={() => window.confirm('Delete message?') && onDeleteMessage(msg.id)}
                  className="absolute -left-8 top-0 p-1 text-red-400 opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <Trash2 size={16} />
                </button>
              )}

              {/* Stego Image */}
              {msg.imageUrl && (
                <div className="mb-2">
                  <button type="button" className="block cursor-pointer" onClick={() => window.open(msg.imageUrl)}>
                    <img src={msg.imageUrl} alt="Stego message" className="rounded border border-indigo-500/50 max-h-[360px] object-contain" />
                  </button>
                </div>
              )}

              {/* Text Content */}
              {msg.text && <div className="text-sm font-medium">{msg.text}</div>}
              
              <div className="text-[10px] mt-1 opacity-70 flex justify-between gap-4">
                <span>{msg.sender}</span>
                <span>{msg.pending ? 'Sending...' : new Date(msg.timestamp).toLocaleTimeString()}</span>
              </div>
            </div>
          </div>
        );
      })}
      <div ref={endRef} />
    </div>
  );
}

function MessageInput({ onSend }) {
  const [text, setText] = useState('');
  return (
    <form onSubmit={(e) => { e.preventDefault(); if(text.trim()) { onSend(text); setText(''); } }} className="p-4 bg-gray-800 border-t border-gray-700 flex gap-2">
      <input 
        className="flex-1 bg-gray-700 rounded-full px-4 py-2 focus:outline-none focus:ring-2 focus:ring-indigo-500"
        placeholder="Type a message..."
        value={text}
        onChange={e => setText(e.target.value)}
      />
      <button disabled={!text.trim()} type="submit" className="p-2 bg-indigo-600 rounded-full disabled:opacity-50 hover:bg-indigo-700">
        <Send size={20} />
      </button>
    </form>
  );
}

function CustomImageForm({ currentRoom, token, username, setMessages }) {
  const [file, setFile] = useState(null);
  const [msg, setMsg] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!file || !msg) return;
    
    setLoading(true);
    const form = new FormData();
    form.append('image', file);
    form.append('message', msg);
    form.append('room_id', currentRoom.id);

    try {
      const res = await axios.post(`${API_BASE_URL}/encode`, form, {
        headers: { Authorization: `Bearer ${token}` }
      });

      // Optimistic update
      if (res.data.encoded_image) {
        const blob = base64ToBlob(res.data.encoded_image);
        const url = blob ? URL.createObjectURL(blob) : null;
        setMessages(prev => [...prev, {
          id: 'pending-' + Date.now(),
          type: 'decoded',
          imageUrl: url,
          imageBase64: res.data.encoded_image,
          text: '',
          sender: username,
          timestamp: new Date().toISOString(),
          pending: true
        }]);
      }
      setMsg(''); setFile(null);
    } catch (err) {
      alert('Upload failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="border-2 border-dashed border-gray-700 rounded p-4 text-center cursor-pointer relative hover:border-indigo-500">
        <input type="file" className="absolute inset-0 opacity-0 cursor-pointer" onChange={e => setFile(e.target.files[0])} accept="image/*" />
        {file ? <p className="text-sm text-green-400">{file.name}</p> : <p className="text-sm text-gray-400">Click to upload image</p>}
      </div>
      <input 
        className="w-full bg-gray-700 p-2 rounded text-sm" 
        placeholder="Secret Message" 
        value={msg} 
        onChange={e => setMsg(e.target.value)} 
      />
      <button disabled={loading || !file} type="submit" className="w-full bg-indigo-600 p-2 rounded text-sm hover:bg-indigo-700 disabled:opacity-50">
        {loading ? 'Processing...' : 'Send'}
      </button>
    </form>
  );
}

function ErrorModal({ error, onClose }) {
  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 z-[60] flex items-center justify-center p-4">
      <div className="bg-gray-800 p-6 rounded-lg max-w-sm w-full relative">
        <button onClick={onClose} className="absolute top-2 right-2 text-gray-400"><X size={20}/></button>
        <h3 className="text-red-400 font-bold mb-2">Error</h3>
        <p className="text-gray-300 text-sm">{error}</p>
      </div>
    </div>
  );
}
