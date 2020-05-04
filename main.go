package main

import (
	"encoding/hex"
	"log"
	"os"
	"os/exec"
	"time"
)

var SPI_ROM_DATA = map[byte][]byte{
	0x60: []byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0x03, 0xa0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x02, 0xff, 0xff, 0xff, 0xff,
		0xf0, 0xff, 0x89, 0x00, 0xf0, 0x01, 0x00, 0x40, 0x00, 0x40, 0x00, 0x40, 0xf9, 0xff, 0x06, 0x00,
		0x09, 0x00, 0xe7, 0x3b, 0xe7, 0x3b, 0xe7, 0x3b, 0xff, 0xff, 0xff, 0xff, 0xff, 0xba, 0x15, 0x62,
		0x11, 0xb8, 0x7f, 0x29, 0x06, 0x5b, 0xff, 0xe7, 0x7e, 0x0e, 0x36, 0x56, 0x9e, 0x85, 0x60, 0xff,
		0x32, 0x32, 0x32, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0x50, 0xfd, 0x00, 0x00, 0xc6, 0x0f, 0x0f, 0x30, 0x61, 0x96, 0x30, 0xf3, 0xd4, 0x14, 0x54, 0x41,
		0x15, 0x54, 0xc7, 0x79, 0x9c, 0x33, 0x36, 0x63, 0x0f, 0x30, 0x61, 0x96, 0x30, 0xf3, 0xd4, 0x14,
		0x54, 0x41, 0x15, 0x54, 0xc7, 0x79, 0x9c, 0x33, 0x36, 0x63, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	},
	0x80: []byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xb2, 0xa1, 0xbe, 0xff, 0x3e, 0x00, 0xf0, 0x01, 0x00, 0x40,
		0x00, 0x40, 0x00, 0x40, 0xfe, 0xff, 0xfe, 0xff, 0x08, 0x00, 0xe7, 0x3b, 0xe7, 0x3b, 0xe7, 0x3b,
	},
}

var count uint8

func incremental(stop chan struct{}) {
	ticker := time.NewTicker(time.Millisecond * 5)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count++
			case <-stop:
				return
			}
		}
	}()
}

var up, left, right, down uint8

func inputResponse(fp *os.File, stop chan struct{}) {
	ticker := time.NewTicker(time.Millisecond * 30)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				dpad := left<<3 | right<<2 | up<<1 | down
				write(fp, []byte{0x30, count, 0x81, 0x00, 0x80, dpad, 0x00, 0x08,
					0x80, 0x00, 0x08, 0x80, 0x00})
			case <-stop:
				return
			}
		}
	}()
}

func uart(fp *os.File, ack byte, subCmd byte, data []byte) {
	write(fp, append([]byte{0x21, count, 0x81, 0x00, 0x80, 0x00, 0x00, 0x08,
		0x80, 0x00, 0x08, 0x80, 0x00, ack, subCmd}, data...))
}

func write(fp *os.File, buf []byte) {
	data := append(buf, make([]byte, 64-len(buf))...)
	fp.Write(data)
	if buf[0] != 0x30 {
		log.Println("write:", hex.EncodeToString(data))
	}
}

func main() {
	target := "/dev/hidg0"

	fp, err := os.OpenFile(target, os.O_RDWR|os.O_SYNC, os.ModeDevice)

	if err != nil {
		panic(err)
	}
	defer fp.Close()

	stopCounter := make(chan struct{})
	stopInput := make(chan struct{})
	incremental(stopCounter)

	go func() {
		buf := make([]byte, 128)

		for {
			n, err := fp.Read(buf)
			log.Println("read:", hex.EncodeToString(buf[:n]), err)
			switch buf[0] {
			case 0x80:
				switch buf[1] {
				case 0x01:
					write(fp, []byte{0x81, buf[1], 0x00, 0x03, 0x00, 0x00, 0x5e, 0x00, 0x53, 0x5e})
				case 0x02, 0x03:
					write(fp, []byte{0x81, buf[1]})
				case 0x04:
					inputResponse(fp, stopInput)
				case 0x05:
					close(stopInput)
					stopInput = make(chan struct{})
				}
			case 0x01:
				switch buf[10] {
				case 0x01: // Bluetooth manual pairing
					uart(fp, 0x81, buf[10], []byte{0x03, 0x01})
				case 0x02: // Request device info
					uart(fp, 0x82, buf[10], []byte{0x03, 0x48, 0x03,
						0x02, 0x5e, 0x53, 0x00, 0x5e, 0x00, 0x00, 0x03, 0x01})
				case 0x03, 0x08, 0x30, 0x38, 0x40, 0x41, 0x48: // Empty response
					uart(fp, 0x80, buf[10], []byte{})
				case 0x04: // Empty response
					uart(fp, 0x80, buf[10], []byte{})
				case 0x10: // Read SPI ROM
					data, ok := SPI_ROM_DATA[buf[12]]
					if ok {
						uart(fp, 0x90, buf[10], append(buf[11:16],
							data[buf[11]:buf[11]+buf[15]]...))
						log.Printf("Read SPI address: %02x%02x[%d] %v\n", buf[12], buf[11], buf[15], data[buf[11]:buf[11]+buf[15]])
					} else {
						log.Printf("Unknown SPI address: %02x[%d]\n", buf[12], buf[15])
					}
				case 0x21:
					uart(fp, 0xa0, buf[10], []byte{0x01, 0x00, 0xff, 0x00, 0x03, 0x00, 0x05, 0x01})
				default:
					log.Println("UART unknown request", buf[10], buf)
				}

			case 0x00:
			case 0x10:
			default:
				log.Println("unknown request", buf[0])
			}
		}
	}()

	buf := make([]byte, 1)

	// Set tty break for read keyboard input directly
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

	for {
		os.Stdin.Read(buf)
		switch buf[0] {
		case 0x61:
			left = 1
		case 0x64:
			right = 1
		case 0x77:
			up = 1
		case 0x73:
			down = 1
		default:
			log.Printf("unknown: %c = 0x%02x\n", buf[0], buf[0])
		}
		time.Sleep(50 * time.Millisecond)
		left = 0
		right = 0
		up = 0
		down = 0
	}
}
