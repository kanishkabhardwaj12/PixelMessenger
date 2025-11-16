import requests
import base64
import cv2
import numpy as np
import os

AI_URL = 'http://localhost:5000'

# create a test image (random noise helps steganography capacity)
img = np.random.randint(0, 256, (120, 160, 3), dtype=np.uint8)
cover_path = 'test_cover.png'
cv2.imwrite(cover_path, img)

# helper to write base64 PNG to file
def save_b64_png(b64str, outpath):
    data = base64.b64decode(b64str)
    with open(outpath, 'wb') as f:
        f.write(data)

# Plain encode
msg_plain = 'Hello from AI-Service plain'
with open(cover_path, 'rb') as f:
    files = {'image': ('cover.png', f, 'image/png')}
    data = {'message': msg_plain}
    r = requests.post(AI_URL + '/encode-image', files=files, data=data, timeout=15)
    r.raise_for_status()
    j = r.json()
    b64 = j.get('encoded_image_base64')
    if not b64:
        print('Plain encode failed:', j)
        raise SystemExit(1)
    save_b64_png(b64, 'stego_plain.png')
    print('Plain encode returned decoded_message:', j.get('decoded_message'))

# Plain decode (server decode endpoint)
with open('stego_plain.png', 'rb') as f:
    files = {'image': ('stego.png', f, 'image/png')}
    r = requests.post(AI_URL + '/decode-image', files=files, timeout=15)
    r.raise_for_status()
    print('Plain decode result:', r.json())

# Keyed encode
msg_keyed = 'Secret message encrypted'
passphrase = 'test-passphrase-123'
with open(cover_path, 'rb') as f:
    files = {'image': ('cover.png', f, 'image/png')}
    data = {'message': msg_keyed, 'passphrase': passphrase}
    r = requests.post(AI_URL + '/encode-image', files=files, data=data, timeout=15)
    r.raise_for_status()
    j = r.json()
    b64 = j.get('encoded_image_base64')
    if not b64:
        print('Keyed encode failed:', j)
        raise SystemExit(1)
    save_b64_png(b64, 'stego_keyed.png')
    print('Keyed encode returned decoded_message (server):', j.get('decoded_message'))

# Keyed decode
with open('stego_keyed.png', 'rb') as f:
    files = {'image': ('stego.png', f, 'image/png')}
    data = {'passphrase': passphrase}
    r = requests.post(AI_URL + '/decode-image', files=files, data=data, timeout=15)
    r.raise_for_status()
    print('Keyed decode result:', r.json())

print('\nFiles written: test_cover.png, stego_plain.png, stego_keyed.png')
