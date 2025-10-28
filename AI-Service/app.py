import numpy as np
import cv2
import requests
from flask import Flask, request, jsonify

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

if __name__ == "__main__":
        # Run the server on port 5000
        app.run(host="0.0.0.0", port=5000, debug=True)
    
