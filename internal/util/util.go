package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

func AESGCMEncrypt(nonce []byte, key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return aesgcm.Seal(nil, nonce, plaintext, nil), nil
}

func AESGCMDecrypt(nonce []byte, key []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

func CryptoRandRead(buf []byte) {
	_, err := rand.Read(buf)
	if err == nil {
		return
	}
	waitDur := [10]time.Duration{5 * time.Millisecond, 10 * time.Millisecond, 30 * time.Millisecond, 50 * time.Millisecond,
		100 * time.Millisecond, 300 * time.Millisecond, 500 * time.Millisecond, 1 * time.Second,
		3 * time.Second, 5 * time.Second}
	for i := 0; i < 10; i++ {
		log.Errorf("Failed to get cryptographic random bytes: %v. Retrying...", err)
		_, err = rand.Read(buf)
		if err == nil {
			return
		}
		time.Sleep(time.Millisecond * waitDur[i])
	}
	log.Fatal("Cannot get cryptographic random bytes after 10 retries")
}

// ReadTLS reads TLS data according to its record layer
func ReadTLS(conn net.Conn, buffer []byte) (n int, err error) {
	// TCP is a stream. Multiple TLS messages can arrive at the same time,
	// a single message can also be segmented due to MTU of the IP layer.
	// This function guareentees a single TLS message to be read and everything
	// else is left in the buffer.
	_, err = io.ReadFull(conn, buffer[:5])
	if err != nil {
		return
	}

	dataLength := int(binary.BigEndian.Uint16(buffer[3:5]))
	if dataLength > len(buffer) {
		err = io.ErrShortBuffer
		return
	}
	n, err = io.ReadFull(conn, buffer[5:dataLength+5])
	return n + 5, err
}

func Pipe(dst net.Conn, src net.Conn, srcReadTimeout time.Duration) {
	// The maximum size of TLS message will be 16380+14+16. 14 because of the stream header and 16
	// because of the salt/mac
	// 16408 is the max TLS message size on Firefox
	buf := make([]byte, 16378)
	if srcReadTimeout != 0 {
		src.SetReadDeadline(time.Now().Add(srcReadTimeout))
	}
	for {
		if srcReadTimeout != 0 {
			src.SetReadDeadline(time.Now().Add(srcReadTimeout))
		}
		i, err := io.ReadAtLeast(src, buf, 1)
		if err != nil {
			dst.Close()
			src.Close()
			return
		}
		_, err = dst.Write(buf[:i])
		if err != nil {
			dst.Close()
			src.Close()
			return
		}
	}
}
