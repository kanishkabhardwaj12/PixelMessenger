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

func Decode(img image.Image) ([]byte, error) {
	rgba := convertToRGBA(img)
	bounds := rgba.Bounds()

	var sizeBytes [4]byte // 32 bits for the size
	var messageSize uint32

	bitIndex := 7
	dataIndex := 0

	// 1. First, decode the 4-byte message size
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if dataIndex >= 4 {
				goto ReadMessage // Break out of nested loop
			}

			c := rgba.RGBAAt(x, y)
			channels := []uint8{c.R, c.G, c.B}

			for i := 0; i < 3; i++ {
				if dataIndex >= 4 {
					continue
				}

				// Extract the LSB
				bit := channels[i] & 1

				// Set the bit in our size byte
				sizeBytes[dataIndex] = (sizeBytes[dataIndex] << 1) | bit

				bitIndex--
				if bitIndex < 0 {
					bitIndex = 7
					dataIndex++
				}
			}
		}
	}

ReadMessage:
	// Convert the size bytes to a number
	err := binary.Read(bytes.NewReader(sizeBytes[:]), binary.LittleEndian, &messageSize)
	if err != nil {
		return nil, errors.New("failed to decode message size")
	}

	// 2. Now, decode the message itself
	message := make([]byte, messageSize)
	dataIndex = 0
	bitIndex = 7

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Skip the pixels we used for the size
			pixelIndex := (y*bounds.Dx() + x)
			if pixelIndex < (32/3 + 1) { // 32 bits / 3 channels per pixel
				continue
			}

			if dataIndex >= int(messageSize) {
				return message, nil // We are done!
			}

			c := rgba.RGBAAt(x, y)
			channels := []uint8{c.R, c.G, c.B}
			dataByte := message[dataIndex]

			for i := 0; i < 3; i++ {
				if dataIndex >= int(messageSize) {
					continue
				}

				bit := channels[i] & 1
				dataByte = (dataByte << 1) | bit

				bitIndex--
				if bitIndex < 0 {
					bitIndex = 7
					message[dataIndex] = dataByte
					dataIndex++
					if dataIndex < int(messageSize) {
						dataByte = message[dataIndex]
					}
				}
			}
		}
	}
	return nil, errors.New("decoding failed, message may be corrupt or not exist")
}
