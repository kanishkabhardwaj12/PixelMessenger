import requests
import os

print("=" * 60)
print("TESTING STEGANOGRAPHY ANALYSIS ENDPOINT")
print("=" * 60)

# Check if test images exist
if not os.path.exists("demo_original.png"):
    print("\n❌ demo_original.png not found!")
    print("   Run demo_steganography.py first to generate test images")
    exit(1)

if not os.path.exists("demo_with_hidden_message.png"):
    print("\n❌ demo_with_hidden_message.png not found!")
    print("   Run demo_steganography.py first to generate test images")
    exit(1)

print("\n✓ Test images found")

# Test the analysis endpoint
print("\n[1] Testing /analyze-stego endpoint...")

try:
    with open("demo_original.png", "rb") as cover_file:
        with open("demo_with_hidden_message.png", "rb") as stego_file:
            files = {
                "cover_image": ("cover.png", cover_file, "image/png"),
                "stego_image": ("stego.png", stego_file, "image/png")
            }
            
            response = requests.post(
                "http://localhost:8082/analyze-stego",
                files=files,
                timeout=10
            )
            
            if response.status_code == 200:
                result = response.json()
                print("\n✅ Analysis successful!")
                print("\n" + "=" * 60)
                print("ANALYSIS RESULTS")
                print("=" * 60)
                print(f"\n📊 Statistics:")
                print(f"  • Total Changes:      {result.get('total_changes', 'N/A')}")
                print(f"  • LSB Changes:        {result.get('lsb_changes', 'N/A')}")
                print(f"  • Max Difference:     {result.get('max_difference', 'N/A')}")
                print(f"  • Image Dimensions:   {result.get('dimensions', 'N/A')}")
                print(f"  • Total Pixels:       {result.get('total_pixels', 'N/A')}")
                print(f"  • Capacity Used:      {result.get('capacity_used', 'N/A')}")
                print(f"  • Steganography:      {'✓ Detected' if result.get('steganography_detected') else '✗ Not Detected'}")
                
                if result.get('sample_pixels'):
                    print(f"\n📷 Sample Pixel Changes:")
                    print(result.get('sample_pixels'))
                
                if result.get('binary_comparison'):
                    print(f"\n💾 Binary LSB Comparison (first 5 changed pixels):")
                    print(result.get('binary_comparison'))
                
                print("=" * 60)
            else:
                print(f"\n❌ Error: HTTP {response.status_code}")
                print(f"   Response: {response.text}")
                
except requests.exceptions.ConnectionError:
    print("\n❌ Connection refused!")
    print("   Make sure the backend is running on http://localhost:8082")
    print("\n   Start backend with:")
    print("   cd backend")
    print("   go run main.go")
except Exception as e:
    print(f"\n❌ Error: {e}")
