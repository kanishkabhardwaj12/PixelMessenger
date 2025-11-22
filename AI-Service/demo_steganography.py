import cv2
import numpy as np
import requests
from PIL import Image
import io

# Create a simple colorful test image
def create_test_image(filename, size=(400, 300)):
    """Create a colorful gradient image"""
    img = np.zeros((size[1], size[0], 3), dtype=np.uint8)
    
    # Create a gradient background
    for y in range(size[1]):
        for x in range(size[0]):
            img[y, x] = [
                int(255 * x / size[0]),           # Red channel
                int(255 * y / size[1]),           # Green channel
                int(128 + 127 * np.sin(x/20))     # Blue channel
            ]
    
    # Add some text to make it more interesting
    cv2.putText(img, 'Original Image', (50, 150), 
                cv2.FONT_HERSHEY_SIMPLEX, 2, (255, 255, 255), 3)
    
    cv2.imwrite(filename, img)
    print(f"Created: {filename}")
    return filename

# Main demo
def main():
    print("=" * 60)
    print("STEGANOGRAPHY DEMONSTRATION")
    print("=" * 60)
    
    # Step 1: Create original image
    print("\n[Step 1] Creating original image...")
    original_img = create_test_image('demo_original.png')
    
    # Step 2: Encode a message into the image
    print("\n[Step 2] Encoding secret message into image...")
    secret_message = "This is a hidden message embedded in the pixels! 🔐"
    
    with open(original_img, 'rb') as f:
        files = {'image': ('image.png', f, 'image/png')}
        data = {'message': secret_message}
        
        try:
            response = requests.post('http://localhost:5000/encode-image', 
                                   files=files, data=data, timeout=10)
            
            if response.status_code == 200:
                result = response.json()
                
                # Save the encoded image
                import base64
                img_data = base64.b64decode(result['encoded_image_base64'])
                with open('demo_with_hidden_message.png', 'wb') as out:
                    out.write(img_data)
                
                print(f"✅ Message encoded successfully!")
                print(f"   Hidden message: '{secret_message}'")
                print(f"   Saved to: demo_with_hidden_message.png")
            else:
                print(f"❌ Error: {response.status_code}")
                return
        except Exception as e:
            print(f"❌ Error connecting to AI-Service: {e}")
            print("   Make sure AI-Service is running on port 5000")
            return
    
    # Step 3: Compare the images
    print("\n[Step 3] Comparing images...")
    original = cv2.imread('demo_original.png')
    encoded = cv2.imread('demo_with_hidden_message.png')
    
    if original is not None and encoded is not None:
        # Ensure same size for comparison
        if original.shape != encoded.shape:
            encoded = cv2.resize(encoded, (original.shape[1], original.shape[0]))
        
        # Calculate difference
        diff = cv2.absdiff(original, encoded)
        total_diff = np.sum(diff)
        max_diff = np.max(diff)
        pixels_changed = np.count_nonzero(diff)
        total_pixels = original.shape[0] * original.shape[1] * 3
        
        print(f"   Total pixel differences: {total_diff}")
        print(f"   Maximum difference per channel: {max_diff}")
        print(f"   Channels changed: {pixels_changed} out of {total_pixels}")
        print(f"   Percentage changed: {100*pixels_changed/total_pixels:.2f}%")
        
        # Create a visual difference map (amplified for visibility)
        diff_amplified = cv2.multiply(diff, np.array([50.0]))
        cv2.imwrite('demo_difference_map.png', diff_amplified)
        print(f"   Difference map saved: demo_difference_map.png")
        print(f"   (Differences amplified 50x for visibility)")
    
    # Step 4: Decode the message
    print("\n[Step 4] Decoding hidden message...")
    with open('demo_with_hidden_message.png', 'rb') as f:
        files = {'image': ('image.png', f, 'image/png')}
        
        try:
            response = requests.post('http://localhost:5000/decode-image', 
                                   files=files, timeout=10)
            
            if response.status_code == 200:
                result = response.json()
                decoded_msg = result.get('decoded_message', '')
                print(f"✅ Message decoded successfully!")
                print(f"   Decoded message: '{decoded_msg}'")
                
                if decoded_msg == secret_message:
                    print(f"   ✅ PERFECT MATCH! Message survived encoding/decoding!")
                else:
                    print(f"   ⚠️ Mismatch detected")
            else:
                print(f"❌ Error: {response.status_code}")
        except Exception as e:
            print(f"❌ Error: {e}")
    
    # Summary
    print("\n" + "=" * 60)
    print("SUMMARY")
    print("=" * 60)
    print("Files created:")
    print("  1. demo_original.png          - Original image (NO hidden message)")
    print("  2. demo_with_hidden_message.png - Image WITH hidden message")
    print("  3. demo_difference_map.png    - Visual difference (amplified)")
    print("\n👁️ The images look identical to the human eye!")
    print("🔐 But one carries a secret message in its pixels!")
    print("=" * 60)

if __name__ == '__main__':
    main()
