# 🔐 PixelMessenger - Secure Steganography Chat Application

A real-time messaging application that uses **LSB (Least Significant Bit) steganography** to hide messages inside images, with optional **AES-GCM encryption** for enhanced security.

## 🌟 Features

- **🖼️ Steganography**: Hide messages in image pixels using LSB technique
- **🔒 Encryption**: Optional passphrase-protected AES-GCM encryption
- **💬 Real-time Chat**: WebSocket-based instant messaging
- **🚪 Room System**: Create and join secure chat rooms
- **👥 Multi-user**: JWT-based authentication and authorization
- **📊 Analysis Tools**: Compare original and encoded images
- **🎨 Modern UI**: React + Tailwind CSS responsive interface

## 🏗️ Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Frontend      │────▶│   Backend (Go)   │────▶│  AI-Service     │
│  React + Vite   │     │  WebSocket + JWT │     │  Flask + OpenCV │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌─────────────┐
                        │ PostgreSQL  │
                        └─────────────┘
```

## 📦 Tech Stack

### Frontend
- **React 19** - UI framework
- **Vite** - Build tool
- **Tailwind CSS** - Styling
- **Lucide React** - Icons
- **IndexedDB** - Client-side image storage

### Backend
- **Go** - Server language
- **Gorilla WebSocket** - Real-time communication
- **PostgreSQL** - Database
- **JWT** - Authentication
- **bcrypt** - Password hashing

### AI Service
- **Flask** - Web framework
- **OpenCV** - Image processing
- **NumPy** - Numerical operations
- **cryptography** - AES-GCM encryption

## 🚀 Getting Started

### Prerequisites

- **Go 1.21+**
- **Node.js 18+**
- **Python 3.11+**
- **PostgreSQL 14+**

### 1️⃣ Database Setup

```bash
# Create PostgreSQL database
createdb pixelmessenger

# Or using psql
psql -U postgres
CREATE DATABASE pixelmessenger;
\q
```

### 2️⃣ Backend Setup

```bash
cd backend

# Install dependencies
go mod download

# Set environment variables (Windows PowerShell)
$env:DATABASE_URL="postgresql://username:password@localhost:5432/pixelmessenger"
$env:JWT_SECRET="your-super-secret-jwt-key"
$env:BACKEND_PORT="8082"
$env:AI_SERVICE_URL="http://localhost:5000"

# Run backend
go run main.go
```

**Backend will run on**: `http://localhost:8082`

### 3️⃣ AI Service Setup

```bash
cd AI-Service

# Create virtual environment (recommended)
python -m venv venv
.\venv\Scripts\activate  # Windows
# source venv/bin/activate  # Linux/Mac

# Install dependencies
pip install -r requirements.txt

# Run AI service
python app.py
```

**AI Service will run on**: `http://localhost:5000`

### 4️⃣ Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Run development server
npm run dev
```

**Frontend will run on**: `http://localhost:5173`

## 📖 How It Works

### 1. LSB Steganography

Messages are hidden in the **least significant bit** of each color channel (R, G, B):

```
Original Pixel: RGB(154, 200, 78)
Binary:         10011010, 11001000, 01001110

Hide 'A' (01000001):
Modified Pixel: RGB(154, 201, 78)
Binary:         10011010, 11001001, 01001110
                        ↑         ↑         ↑
                    LSB modified (invisible change!)
```

### 2. Encryption Flow (with passphrase)

```
Message → AES-GCM Encrypt → Nonce + Ciphertext → LSB Embed → Encoded Image
```

### 3. Communication Flow

```
1. User uploads image + message + optional passphrase
2. Frontend → Backend /encode endpoint
3. Backend → AI-Service for steganography encoding
4. AI-Service returns encoded image
5. Backend broadcasts via WebSocket to room members
6. Receivers get both original & encoded images
```

## 🔑 API Endpoints

### Authentication
- `POST /register` - Create new user
- `POST /login` - Login and get JWT token

### Rooms
- `POST /rooms` - Create a room
- `GET /my-rooms` - Get user's rooms
- `POST /rooms/{id}/invite` - Invite user to room
- `GET /rooms/{id}/messages` - Get room message history

### Messaging
- `POST /encode` - Encode message into image
- `POST /decode` - Decode message from image
- `WS /ws?room={id}&token={jwt}` - WebSocket connection

### AI Service
- `POST /encode-image` - Encode message (plain or keyed)
- `POST /decode-image` - Decode message (plain or keyed)
- `POST /best-image` - Select best cover image (AI-powered)

## 🛡️ Security Features

1. **JWT Authentication**: Secure token-based auth
2. **Password Hashing**: bcrypt with salt
3. **Room Authorization**: Users must be room members
4. **AES-GCM Encryption**: Optional passphrase protection
5. **Deterministic Permutation**: Keyed embedding mode
6. **CORS Protection**: Configurable origins

## 📊 Steganography Analysis

The app includes an analysis tool to compare original and encoded images:

- Pixel-level comparison
- LSB change detection
- Capacity calculations
- Binary representation viewer

## 🎯 Usage Example

### Register & Login
```bash
# Register
curl -X POST http://localhost:8082/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"secret123"}'

# Login
curl -X POST http://localhost:8082/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"secret123"}'
```

### Create Room & Send Message
1. Create a room via `/rooms`
2. Connect to WebSocket: `ws://localhost:8082/ws?room={roomId}&token={jwt}`
3. Upload image with message using the UI
4. Receivers see both original and encoded images side-by-side

## 🧪 Testing

### Test Steganography
```bash
cd AI-Service
python test_encode_decode.py
```

### Test Demo
```bash
cd AI-Service
python demo_steganography.py
```

This creates:
- `demo_original.png` - Original image
- `demo_with_hidden_message.png` - Image with hidden message
- `demo_difference_map.png` - Visual diff (amplified)

## 📝 Project Structure

```
PixelMessenger/
├── backend/              # Go backend
│   ├── auth/            # JWT & password hashing
│   ├── handlers/        # HTTP handlers
│   ├── middleware/      # Auth & CORS middleware
│   ├── models/          # Data models
│   ├── storage/         # PostgreSQL queries
│   ├── steganography/   # LSB implementation (Go)
│   ├── websocket/       # WebSocket hub & clients
│   └── main.go          # Entry point
├── frontend/            # React frontend
│   └── src/
│       ├── App.jsx      # Main app component
│       └── StegoAnalysis.jsx  # Analysis tool
├── AI-Service/          # Flask AI service
│   ├── app.py           # Flask endpoints
│   ├── requirements.txt
│   ├── test_encode_decode.py
│   └── demo_steganography.py
└── .env.example         # Environment template
```

## 🐛 Troubleshooting

### Backend won't start
- Check PostgreSQL is running: `psql -U postgres`
- Verify `DATABASE_URL` environment variable
- Ensure `JWT_SECRET` is set

### AI Service connection refused
- Check Flask is running on port 5000
- Verify `AI_SERVICE_URL` in backend env vars
- Check firewall settings

### WebSocket connection fails
- Ensure JWT token is valid
- Check user is member of the room
- Verify WebSocket URL includes `token` query param

## 🚀 Production Deployment

1. **Security**:
   - Use strong `JWT_SECRET`
   - Enable HTTPS/WSS
   - Set proper CORS origins
   - Use production WSGI server (Gunicorn)

2. **Database**:
   - Use managed PostgreSQL (AWS RDS, Google Cloud SQL)
   - Enable SSL connections
   - Regular backups

3. **Scaling**:
   - Use Redis for WebSocket pub/sub (multi-instance)
   - CDN for static assets
   - Load balancer for backend instances

## 📄 License

MIT License - Feel free to use this project for learning and development!

## 👥 Contributors

- **Kanishka Bhardwaj** - [@kanishkabhardwaj12](https://github.com/kanishkabhardwaj12)

## 🙏 Acknowledgments

- Steganography algorithms based on LSB technique
- Inspired by secure messaging applications
- Built as a cybersecurity educational project

---

**⚠️ Disclaimer**: This is an educational project. For production use, conduct proper security audits and follow best practices.
