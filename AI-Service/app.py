import numpy as np
import cv2
import requests
from flask import Flask, request, jsonify
import base64
import struct

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
        Receives a JSON body with a list of image URLs.
        Returns the URL of the "best" image.
        """
        data = request.get_json()
        if not data or "image_urls" not in data:
            return jsonify({"error": "Missing 'image_urls' in JSON body"}), 400

        image_urls = data["image_urls"]
        if not image_urls:
            return jsonify({"error": "'image_urls' list is empty"}), 400

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

                # Prepare payload: 4-byte little-endian length + message bytes
                msg_bytes = message.encode('utf-8')
                msg_len = len(msg_bytes)
                header = struct.pack('<I', msg_len)
                payload = header + msg_bytes

                # Ensure image has 3 channels (BGR) and is mutable
                if len(img.shape) == 2:
                        img = cv2.cvtColor(img, cv2.COLOR_GRAY2BGR)
                elif img.shape[2] == 4:
                        # drop alpha for simplicity
                        img = cv2.cvtColor(img, cv2.COLOR_BGRA2BGR)

                h, w, _ = img.shape
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
                        flat[i] = (flat[i] & 0xFE) | bit

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

if __name__ == "__main__":
        # Run the server on port 5000
        app.run(host="0.0.0.0", port=5000, debug=True)
    
