import numpy as np
import cv2
import requests
from flask import Flask, request, jsonify
import base64
import struct
import hashlib
import os
import random
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

app = Flask(__name__)

def analyze_image_from_url(url):
        """
        Downloads an image, analyzes it for complexity, and returns a score.
        A higher score is better for steganography.
        """
        try:
            # Download the image
            resp = requests.get(url, timeout=5)
            resp.raise_for_status() # Raise error for bad responses

            # Convert image to a numpy array
            image_array = np.frombuffer(resp.content, np.uint8)
            # Decode the image
            img = cv2.imdecode(image_array, cv2.IMREAD_COLOR)

            if img is None:
                return 0 # Failed to decode

            # 1. Convert to grayscale
            gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)
            
            # 2. Apply Canny edge detection
            # These thresholds are a good starting point
            edges = cv2.Canny(gray, 100, 200)

            # 3. Calculate the "complexity score"
            # We just count the number of non-zero (white) pixels in the edge map.
            # More edges = higher score = better for steganography.
            score = np.sum(edges > 0)
            
            return score

        except Exception as e:
            print(f"Error analyzing image {url}: {e}")
            return 0 # Return 0 for any image that fails


@app.route("/select-image", methods=["POST"])
def select_image():
        """
        Receives a JSON body with a list of image URLs and optional message.
        Returns the URL of the "best" image.
        Uses message content to add variety to selection.
        """
        data = request.get_json()
        if not data or "image_urls" not in data:
            return jsonify({"error": "Missing 'image_urls' in JSON body"}), 400

        image_urls = data["image_urls"]
        if not image_urls:
            return jsonify({"error": "'image_urls' list is empty"}), 400

        message = data.get("message", "")
        
        # If message is provided, use it to add variety to selection
        if message:
            # Hash the message to get a deterministic but varied selection
            msg_hash = hashlib.sha256(message.encode('utf-8')).digest()
            # Use first 8 bytes as seed for random selection
            seed = int.from_bytes(msg_hash[:8], 'little')
            rng = random.Random(seed)
            
            # Shuffle URLs based on message content
            shuffled_urls = image_urls.copy()
            rng.shuffle(shuffled_urls)
            
            # Take top 5 candidates for analysis
            candidate_urls = shuffled_urls[:min(5, len(shuffled_urls))]
            
            best_image = None
            best_score = -1

            for url in candidate_urls:
                score = analyze_image_from_url(url)
                print(f"Image: {url}, Score: {score}, Message: '{message[:50]}'")
                if score > best_score:
                    best_score = score
                    best_image = url

            if best_image is None:
                # Fallback: return first shuffled image
                best_image = shuffled_urls[0]
        else:
            # No message provided, use original logic
            best_image = None
            best_score = -1

            for url in image_urls:
                score = analyze_image_from_url(url)
                print(f"Image: {url}, Score: {score}")
                if score > best_score:
                    best_score = score
                    best_image = url

            if best_image is None:
                # Fallback: just return the first image if all failed
                best_image = image_urls[0]

        return jsonify({"best_image_url": best_image})

@app.route('/encode-image', methods=['POST'])
def encode_image():
        """
        Accepts multipart/form-data with fields:
          - image: file
          - message: string
        Returns JSON: { "encoded_image_base64": "...", "decoded_message": "..." }
        """
        if 'image' not in request.files:
                return jsonify({'error': 'Missing image file'}), 400
        file = request.files['image']
        message = request.form.get('message', '')
        if message == '':
                return jsonify({'error': 'Missing message'}), 400

        try:
                # Read image bytes
                img_bytes = file.read()
                arr = np.frombuffer(img_bytes, np.uint8)
                img = cv2.imdecode(arr, cv2.IMREAD_UNCHANGED)
                if img is None:
                        return jsonify({'error': 'Failed to decode uploaded image'}), 400
                # Handle optional passphrase for keyed/encrypted embedding
                passphrase = request.form.get('passphrase', None)

                # Ensure image has 3 channels (BGR) and is mutable
                if len(img.shape) == 2:
                        img = cv2.cvtColor(img, cv2.COLOR_GRAY2BGR)
                elif img.shape[2] == 4:
                        # drop alpha for simplicity
                        img = cv2.cvtColor(img, cv2.COLOR_BGRA2BGR)

                h, w, _ = img.shape

                if passphrase:
                        # Keyed/encrypted embedding: AES-GCM + nonce prefix + deterministic permutation
                        key = hashlib.sha256(passphrase.encode('utf-8')).digest()
                        aesgcm = AESGCM(key)
                        # AES-GCM nonce size is 12 bytes
                        nonce = os.urandom(12)

                        # Encrypt message
                        msg_bytes = message.encode('utf-8')
                        ciphertext = aesgcm.encrypt(nonce, msg_bytes, None)

                        # Build wrapped payload: nonce + 4-byte little-endian length + ciphertext
                        cipher_len = len(ciphertext)
                        header = struct.pack('<I', cipher_len)
                        wrapped = nonce + header + ciphertext

                        total_channels = w * h * 3
                        if len(wrapped) * 8 > total_channels:
                                return jsonify({'error': 'Image too small OR message+nonce too large for the image (after encryption)'}), 400

                        # Build deterministic permutation seeded from first 8 bytes of sha256(passphrase)
                        # (match Go: sha256.Sum256(passphrase) first 8 bytes)
                        seed = int.from_bytes(key[:8], 'little')
                        rng = random.Random(seed)
                        perm = list(range(total_channels))
                        rng.shuffle(perm)

                        # Flatten and write bits into permuted channel indices
                        flat = img.flatten()
                        bitPos = 7
                        byteIndex = 0
                        for i in range(len(wrapped) * 8):
                                bidx = perm[i]
                                # write into flattened channel at index bidx
                                dataByte = wrapped[byteIndex]
                                bit = (dataByte >> bitPos) & 1
                                flat[bidx] = (int(flat[bidx]) & 0xFE) | bit
                                bitPos -= 1
                                if bitPos < 0:
                                        bitPos = 7
                                        byteIndex += 1

                        stego = flat.reshape(img.shape)

                        # Encode as PNG
                        success, png = cv2.imencode('.png', stego)
                        if not success:
                                return jsonify({'error': 'Failed to encode stego PNG'}), 500

                        encoded_b64 = base64.b64encode(png.tobytes()).decode('ascii')
                        # Return encoded image and the original message (decoded_message)
                        return jsonify({'encoded_image_base64': encoded_b64, 'decoded_message': message})
                else:
                        # Plain embedding (no key) - existing behavior
                        # Prepare payload: 4-byte little-endian length + message bytes
                        msg_bytes = message.encode('utf-8')
                        msg_len = len(msg_bytes)
                        header = struct.pack('<I', msg_len)
                        payload = header + msg_bytes

                        max_size = (w * h * 3) // 8
                        if msg_len + 4 > max_size:
                                return jsonify({'error': 'Image too small for message'}), 400

                        # Flatten pixel channels for LSB embedding (B, G, R order in OpenCV)
                        flat = img.flatten()
                        total_bits = len(payload) * 8
                        for i in range(total_bits):
                                byte_idx = i // 8
                                bit_in_byte = 7 - (i % 8)
                                bit = (payload[byte_idx] >> bit_in_byte) & 1
                                flat[i] = (int(flat[i]) & 0xFE) | bit

                        # Reshape back to image
                        stego = flat.reshape(img.shape)

                        # Encode as PNG in memory
                        success, png = cv2.imencode('.png', stego)
                        if not success:
                                return jsonify({'error': 'Failed to encode stego PNG'}), 500

                        encoded_b64 = base64.b64encode(png.tobytes()).decode('ascii')

                        return jsonify({'encoded_image_base64': encoded_b64, 'decoded_message': message})
        except Exception as e:
                print('Encode error:', e)
                return jsonify({'error': str(e)}), 500


@app.route('/decode-image', methods=['POST'])
def decode_image():
        """
        Accepts multipart/form-data with:
          - image: file
          - passphrase: optional
        Returns: { "decoded_message": "..." }
        """
        if 'image' not in request.files:
                return jsonify({'error': 'Missing image file'}), 400
        file = request.files['image']
        passphrase = request.form.get('passphrase', None)
        try:
                img_bytes = file.read()
                arr = np.frombuffer(img_bytes, np.uint8)
                img = cv2.imdecode(arr, cv2.IMREAD_UNCHANGED)
                if img is None:
                        return jsonify({'error': 'Failed to decode uploaded image'}), 400

                # Normalize channels
                if len(img.shape) == 2:
                        img = cv2.cvtColor(img, cv2.COLOR_GRAY2BGR)
                elif img.shape[2] == 4:
                        img = cv2.cvtColor(img, cv2.COLOR_BGRA2BGR)

                h, w, _ = img.shape
                flat = img.flatten()
                total_channels = w * h * 3

                if passphrase:
                        key = hashlib.sha256(passphrase.encode('utf-8')).digest()
                        seed = int.from_bytes(key[:8], 'little')
                        rng = random.Random(seed)
                        perm = list(range(total_channels))
                        rng.shuffle(perm)

                        # First read nonce + 4-byte length
                        # AESGCM nonce size is 12
                        nonce_size = 12
                        header_len = nonce_size + 4
                        header_bytes = bytearray(header_len)
                        bitPos = 7
                        byteIndex = 0
                        for i in range(header_len * 8):
                                bidx = perm[i]
                                chv = int(flat[bidx])
                                bit = chv & 1
                                header_bytes[byteIndex] = (header_bytes[byteIndex] << 1) | bit
                                bitPos -= 1
                                if bitPos < 0:
                                        bitPos = 7
                                        byteIndex += 1

                        nonce = bytes(header_bytes[:nonce_size])
                        cipher_len = struct.unpack('<I', bytes(header_bytes[nonce_size:nonce_size+4]))[0]

                        # Read ciphertext
                        cipher_bytes = bytearray(cipher_len)
                        bitStart = header_len * 8
                        bitPos = 7
                        byteIndex = 0
                        for i in range(cipher_len * 8):
                                bidx = perm[bitStart + i]
                                chv = int(flat[bidx])
                                bit = chv & 1
                                cipher_bytes[byteIndex] = (cipher_bytes[byteIndex] << 1) | bit
                                bitPos -= 1
                                if bitPos < 0:
                                        bitPos = 7
                                        byteIndex += 1

                        aesgcm = AESGCM(key)
                        try:
                                plaintext = aesgcm.decrypt(nonce, bytes(cipher_bytes), None)
                        except Exception as e:
                                return jsonify({'error': 'Decryption failed: ' + str(e)}), 400
                        try:
                                decoded = plaintext.decode('utf-8')
                        except:
                                decoded = base64.b64encode(plaintext).decode('ascii')
                        return jsonify({'decoded_message': decoded})
                else:
                        # Plain decode: first 4 bytes little-endian length, then message
                        # Read first 4 bytes
                        size_bytes = bytearray(4)
                        bitPos = 7
                        byteIndex = 0
                        for i in range(4 * 8):
                                chv = int(flat[i])
                                bit = chv & 1
                                size_bytes[byteIndex] = (size_bytes[byteIndex] << 1) | bit
                                bitPos -= 1
                                if bitPos < 0:
                                        bitPos = 7
                                        byteIndex += 1

                        message_size = struct.unpack('<I', bytes(size_bytes))[0]
                        if message_size == 0:
                                return jsonify({'decoded_message': ''})

                        # Defensive: check plausible size
                        if message_size > (w * h * 3 / 8):
                                return jsonify({'error': 'Decoded message size is too large'}), 400

                        msg_bytes = bytearray(message_size)
                        bitStart = 4 * 8
                        bitPos = 7
                        byteIndex = 0
                        for i in range(message_size * 8):
                                idx = bitStart + i
                                chv = int(flat[idx])
                                bit = chv & 1
                                msg_bytes[byteIndex] = (msg_bytes[byteIndex] << 1) | bit
                                bitPos -= 1
                                if bitPos < 0:
                                        bitPos = 7
                                        byteIndex += 1

                        try:
                                decoded = bytes(msg_bytes).decode('utf-8')
                        except:
                                decoded = base64.b64encode(bytes(msg_bytes)).decode('ascii')
                        return jsonify({'decoded_message': decoded})
        except Exception as e:
                print('Decode error:', e)
                return jsonify({'error': str(e)}), 500

if __name__ == "__main__":
        # Run the server on port 5000 (debug disabled for stable test runs)
        app.run(host="0.0.0.0", port=5000, debug=False)
    
