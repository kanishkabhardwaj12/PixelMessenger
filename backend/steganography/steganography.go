package steganography

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	mathrand "math/rand"
)

func convertToRGBA(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	newImage := image.NewRGBA(bounds)
	draw.Draw(newImage, bounds, img, bounds.Min, draw.Src)
	return newImage
}

func Encode(img image.Image, message []byte) (*bytes.Buffer, error) {
	newImage := convertToRGBA(img)
	bounds := newImage.Bounds()
	maxSize := (bounds.Dx() * bounds.Dy() * 3) / 8 //each pixel has three values which can hide one bit

	messageSize := uint32(len(message))

	if uint32(maxSize) < messageSize+4 { //+4 because we are also encrypting size of message
		return nil, errors.New("image too small OR message too large for the image")
	}

	payload := new(bytes.Buffer)
	binary.Write(payload, binary.LittleEndian, messageSize)
	payload.Write(message)
	payloadBytes := payload.Bytes()

	dataByteIndex := 0
	bitIndex := 7 // starting from the left-most bit

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if dataByteIndex >= len(payloadBytes) {
				buf := new(bytes.Buffer)
				if err := png.Encode(buf, newImage); err != nil {
					return nil, err
				}
				return buf, nil
			}

			// current byte to encode
			dataByte := payloadBytes[dataByteIndex]

			// rgba colour of the current pixel
			c := newImage.RGBAAt(x, y)
			r, g, b, a := c.R, c.G, c.B, c.A

			channels := []uint8{r, g, b}
			newChannels := make([]uint8, 3)

			for i := 0; i < 3; i++ {
				if dataByteIndex >= len(payloadBytes) {
					newChannels[i] = channels[i] // No more data, just copy
					continue
				}

				// Get the bit we want to hide
				bit := (dataByte >> bitIndex) & 1 // AND-ing with 1 is used to get the same thing back

				// 1. Clear the least significant bit (e.g., 10101101 -> 10101100)
				// 2. Set the bit to our data bit
				newChannels[i] = (channels[i] & 0xFE) | bit // 0xFE = 11111110

				// move to the next bit
				bitIndex--
				if bitIndex < 0 {
					bitIndex = 7
					dataByteIndex++
					if dataByteIndex < len(payloadBytes) {
						dataByte = payloadBytes[dataByteIndex]
					}
				}
				// Set the pixel's new color
				newImage.SetRGBA(x, y, color.RGBA{R: newChannels[0], G: newChannels[1], B: newChannels[2], A: a})
			}
		}
	}
	return nil, errors.New("encoding failed unexpectedly")
}

// Capacity returns how many bytes can be embedded in the provided image
// using the current 1-bit-per-channel LSB scheme (3 channels per pixel).
// This does not account for encryption overhead; if you plan to encrypt
// the message before embedding, subtract 12 (GCM nonce) + 4 (length) from
// the returned capacity to get a safe max message length.
func Capacity(img image.Image) int {
	rgba := convertToRGBA(img)
	bounds := rgba.Bounds()
	maxBytes := (bounds.Dx() * bounds.Dy() * 3) / 8
	return maxBytes
}

// EncodeWithKey will encrypt the message with AES-GCM when a non-empty key
// is provided, then embed the ciphertext using a pseudo-random pixel order
// seeded from the key (to reduce simple LSB detection). If key is nil or
// empty, it falls back to the plain Encode behavior.
func EncodeWithKey(img image.Image, message []byte, key []byte) (*bytes.Buffer, error) {
	if len(key) == 0 {
		return Encode(img, message)
	}

	// Derive a 32-byte key via SHA-256 of the provided key bytes
	k := sha256.Sum256(key)
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Encrypt the message
	ciphertext := gcm.Seal(nil, nonce, message, nil)

	// Build payload: 4-byte little-endian length + ciphertext
	payload := new(bytes.Buffer)
	messageSize := uint32(len(ciphertext))
	binary.Write(payload, binary.LittleEndian, messageSize)
	payload.Write(ciphertext)
	payloadBytes := payload.Bytes()

	// Now embed the payload bits into the image using a pseudo-random
	// ordering based on the key-derived seed.
	newImage := convertToRGBA(img)
	bounds := newImage.Bounds()
	totalChannels := bounds.Dx() * bounds.Dy() * 3
	capacityBits := totalChannels
	if len(payloadBytes)*8 > capacityBits {
		return nil, errors.New("image too small OR message too large for the image (after encryption)")
	}

	// Create channel index permutation
	perm := make([]int, totalChannels)
	for i := 0; i < totalChannels; i++ {
		perm[i] = i
	}
	// Seed math/rand with first 8 bytes of sha256(key) for deterministic shuffle
	seed := int64(binary.LittleEndian.Uint64(k[0:8]))
	r := mathrand.New(mathrand.NewSource(seed))
	r.Shuffle(len(perm), func(i, j int) { perm[i], perm[j] = perm[j], perm[i] })

	// Write bits into channels according to perm
	bitPos := 7
	byteIndex := 0
	for i := 0; i < len(payloadBytes)*8; i++ {
		bidx := perm[i] // channel index
		pixelIdx := bidx / 3
		ch := bidx % 3
		x := bounds.Min.X + (pixelIdx % bounds.Dx())
		y := bounds.Min.Y + (pixelIdx / bounds.Dx())

		dataByte := payloadBytes[byteIndex]
		bit := (dataByte >> bitPos) & 1

		c := newImage.RGBAAt(x, y)
		rch, gch, bch := c.R, c.G, c.B
		switch ch {
		case 0:
			rch = (rch & 0xFE) | bit
		case 1:
			gch = (gch & 0xFE) | bit
		case 2:
			bch = (bch & 0xFE) | bit
		}
		newImage.SetRGBA(x, y, color.RGBA{R: rch, G: gch, B: bch, A: c.A})

		bitPos--
		if bitPos < 0 {
			bitPos = 7
			byteIndex++
		}
	}

	// Prepend the nonce as metadata? For backward-compatibility we keep the
	// stored payload as (length + ciphertext). The nonce must be stored
	// somewhere — easiest is to prefix the PNG with an ancillary chunk, but
	// that's more work. Instead, we encode the nonce by embedding it as the
	// very first bytes of the payload so the decoder can recover it. We'll
	// create a wrapper: nonce(12) + payloadBytes
	wrapped := append(nonce, payloadBytes...)

	// We already embedded payloadBytes; to include nonce we need to rebuild.
	// For simplicity, we will instead embed nonce+payload in a single pass
	// above. To keep this patch small, re-run embedding embedding nonce+payload:
	// Recreate newImage and re-embed with wrapped payload.
	newImage = convertToRGBA(img)
	// Recompute capacity check
	if len(wrapped)*8 > capacityBits {
		return nil, errors.New("image too small OR message+nonce too large for the image (after encryption)")
	}
	// shuffle again deterministically
	r = mathrand.New(mathrand.NewSource(seed))
	r.Shuffle(len(perm), func(i, j int) { perm[i], perm[j] = perm[j], perm[i] })

	bitPos = 7
	byteIndex = 0
	for i := 0; i < len(wrapped)*8; i++ {
		bidx := perm[i]
		pixelIdx := bidx / 3
		ch := bidx % 3
		x := bounds.Min.X + (pixelIdx % bounds.Dx())
		y := bounds.Min.Y + (pixelIdx / bounds.Dx())

		dataByte := wrapped[byteIndex]
		bit := (dataByte >> bitPos) & 1

		c := newImage.RGBAAt(x, y)
		rch, gch, bch := c.R, c.G, c.B
		switch ch {
		case 0:
			rch = (rch & 0xFE) | bit
		case 1:
			gch = (gch & 0xFE) | bit
		case 2:
			bch = (bch & 0xFE) | bit
		}
		newImage.SetRGBA(x, y, color.RGBA{R: rch, G: gch, B: bch, A: c.A})

		bitPos--
		if bitPos < 0 {
			bitPos = 7
			byteIndex++
		}
	}

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, newImage); err != nil {
		return nil, err
	}
	return buf, nil
}

// DecodeWithKey expects the image to contain a wrapped payload that begins
// with a nonce (gcm.NonceSize) followed by a 4-byte length and ciphertext.
// It reconstructs the wrapped payload using the same pseudo-random ordering
// (seeded by key) and then decrypts using AES-GCM.
func DecodeWithKey(img image.Image, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return Decode(img)
	}
	// derive key
	k := sha256.Sum256(key)
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()

	rgba := convertToRGBA(img)
	bounds := rgba.Bounds()
	totalChannels := bounds.Dx() * bounds.Dy() * 3

	// Recreate permutation
	perm := make([]int, totalChannels)
	for i := 0; i < totalChannels; i++ {
		perm[i] = i
	}
	seed := int64(binary.LittleEndian.Uint64(k[0:8]))
	r := mathrand.New(mathrand.NewSource(seed))
	r.Shuffle(len(perm), func(i, j int) { perm[i], perm[j] = perm[j], perm[i] })

	// We'll read enough bits to recover nonce + 4-byte length first
	headerBytesNeeded := nonceSize + 4
	header := make([]byte, headerBytesNeeded)
	bitPos := 7
	byteIndex := 0
	for i := 0; i < headerBytesNeeded*8; i++ {
		bidx := perm[i]
		pixelIdx := bidx / 3
		ch := bidx % 3
		x := bounds.Min.X + (pixelIdx % bounds.Dx())
		y := bounds.Min.Y + (pixelIdx / bounds.Dx())
		c := rgba.RGBAAt(x, y)
		var chv uint8
		switch ch {
		case 0:
			chv = c.R
		case 1:
			chv = c.G
		case 2:
			chv = c.B
		}
		bit := chv & 1
		header[byteIndex] = (header[byteIndex] << 1) | bit
		bitPos--
		if bitPos < 0 {
			bitPos = 7
			byteIndex++
		}
	}

	nonce := header[:nonceSize]
	var cipherLen uint32
	if err := binary.Read(bytes.NewReader(header[nonceSize:]), binary.LittleEndian, &cipherLen); err != nil {
		return nil, errors.New("failed to read ciphertext length")
	}

	// Read ciphertext bytes
	cipherBytes := make([]byte, cipherLen)
	bitStart := headerBytesNeeded * 8
	bitPos = 7
	byteIndex = 0
	for i := 0; i < int(cipherLen)*8; i++ {
		bidx := perm[bitStart+i]
		pixelIdx := bidx / 3
		ch := bidx % 3
		x := bounds.Min.X + (pixelIdx % bounds.Dx())
		y := bounds.Min.Y + (pixelIdx / bounds.Dx())
		c := rgba.RGBAAt(x, y)
		var chv uint8
		switch ch {
		case 0:
			chv = c.R
		case 1:
			chv = c.G
		case 2:
			chv = c.B
		}
		bit := chv & 1
		cipherBytes[byteIndex] = (cipherBytes[byteIndex] << 1) | bit
		bitPos--
		if bitPos < 0 {
			bitPos = 7
			byteIndex++
		}
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func Decode(img image.Image) ([]byte, error) {
	rgba := convertToRGBA(img)
	bounds := rgba.Bounds()
	// We'll perform a single pass over pixels/channels. First collect the
	// 4-byte little-endian size, then continue immediately (without skipping)
	// to read the message bytes. This avoids misalignment when the size does
	// not end exactly on a pixel/channel boundary.

	var sizeBytes [4]byte
	var messageSize uint32

	// state for reading bits
	sizeByteIndex := 0
	sizeBitPos := 7

	// placeholders for message state (initialized once we know messageSize)
	var message []byte
	msgByteIndex := 0
	msgBitPos := 7
	readingMessage := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := rgba.RGBAAt(x, y)
			channels := []uint8{c.R, c.G, c.B}

			for ch := 0; ch < 3; ch++ {
				bit := channels[ch] & 1

				if !readingMessage {
					// fill size bytes (MSB-first per byte)
					sizeBytes[sizeByteIndex] = (sizeBytes[sizeByteIndex] << 1) | bit
					sizeBitPos--
					if sizeBitPos < 0 {
						sizeBitPos = 7
						sizeByteIndex++
						if sizeByteIndex >= 4 {
							// finished reading size; parse it and allocate message
							if err := binary.Read(bytes.NewReader(sizeBytes[:]), binary.LittleEndian, &messageSize); err != nil {
								return nil, errors.New("failed to decode message size")
							}
							// defensive: reject huge sizes
							if messageSize == 0 {
								return nil, nil
							}
							if uint64(messageSize) > uint64(bounds.Dx()*bounds.Dy()*3/8) {
								return nil, errors.New("decoded message size is too large")
							}
							message = make([]byte, messageSize)
							readingMessage = true
							msgByteIndex = 0
							msgBitPos = 7
						}
					}
				} else {
					// reading message bits (MSB-first per byte)
					if msgByteIndex >= int(messageSize) {
						return message, nil // done
					}
					// accumulate into current message byte
					message[msgByteIndex] = (message[msgByteIndex] << 1) | bit
					msgBitPos--
					if msgBitPos < 0 {
						msgBitPos = 7
						msgByteIndex++
					}
				}
			}
		}
	}

	if readingMessage && msgByteIndex >= int(messageSize) {
		return message, nil
	}

	return nil, errors.New("decoding failed, message may be corrupt or not exist")
}
