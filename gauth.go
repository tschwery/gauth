package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"math/big"
	"os/user"
	"path"
	"strings"
	"syscall"
	"time"
)

func TimeStamp() (int64, int) {
	time := time.Now().Unix()
	return time / 30, int(time % 30)
}

func normalizeSecret(sec string) string {
	noPadding := strings.ToUpper(strings.Replace(sec, " ", "", -1))
	padLength := 8 - (len(noPadding) % 8)
	if padLength < 8 {
		return noPadding + strings.Repeat("=", padLength)
	}
	return noPadding
}

func AuthCode(sec string, ts int64, encodeType string) (string, error) {
	key, err := base32.StdEncoding.DecodeString(sec)
	if err != nil {
		return "", err
	}
	enc := hmac.New(sha1.New, key)
	msg := make([]byte, 8, 8)
	msg[0] = (byte)(ts >> (7 * 8) & 0xff)
	msg[1] = (byte)(ts >> (6 * 8) & 0xff)
	msg[2] = (byte)(ts >> (5 * 8) & 0xff)
	msg[3] = (byte)(ts >> (4 * 8) & 0xff)
	msg[4] = (byte)(ts >> (3 * 8) & 0xff)
	msg[5] = (byte)(ts >> (2 * 8) & 0xff)
	msg[6] = (byte)(ts >> (1 * 8) & 0xff)
	msg[7] = (byte)(ts >> (0 * 8) & 0xff)
	if _, err := enc.Write(msg); err != nil {
		return "", err
	}
	hash := enc.Sum(nil)
	offset := hash[19] & 0x0f
	trunc := hash[offset : offset+4]
	trunc[0] &= 0x7F
	if strings.TrimSpace(encodeType) == "Steam" {
		steamChars := "23456789BCDFGHJKMNPQRTVWXY"
		steamCharsLen := int32(len(steamChars))
		var res int32
		if err = binary.Read(bytes.NewReader(trunc), binary.BigEndian, &res); err != nil {
			return "", err
		}
		hotp := make([]byte, 5, 5)
		for i := 0; i < 5;  i++ {
			idx := res % steamCharsLen
			hotp[i] = steamChars[int(idx)]
			res = res / steamCharsLen
		}
		return string(hotp), nil
	} else {
		res := new(big.Int).Mod(new(big.Int).SetBytes(trunc), big.NewInt(1000000))
		return fmt.Sprintf("%06d", res), nil
	}
}

func authCodeOrDie(sec string, ts int64, encodeType string) string {
	str, e := AuthCode(sec, ts, encodeType)
	if e != nil {
		log.Fatal(e)
	}
	return str
}

func main() {
	user, e := user.Current()
	if e != nil {
		log.Fatal(e)
	}
	cfgPath := path.Join(user.HomeDir, ".config/gauth.csv")

	cfgContent, e := ioutil.ReadFile(cfgPath)
	if e != nil {
		log.Fatal(e)
	}

	// Support for 'openssl enc -aes-128-cbc -md sha256 -pass pass:'
	if bytes.Compare(cfgContent[:8], []byte{0x53, 0x61, 0x6c, 0x74, 0x65, 0x64, 0x5f, 0x5f}) == 0 {
		fmt.Printf("Encryption password: ")
		passwd, e := terminal.ReadPassword(syscall.Stdin)
		fmt.Printf("\n")
		if e != nil {
			log.Fatal(e)
		}
		salt := cfgContent[8:16]
		rest := cfgContent[16:]
		salting := sha256.New()
		salting.Write([]byte(passwd))
		salting.Write(salt)
		sum := salting.Sum(nil)
		key := sum[:16]
		iv := sum[16:]
		block, e := aes.NewCipher(key)
		if e != nil {
			log.Fatal(e)
		}

		mode := cipher.NewCBCDecrypter(block, iv)
		mode.CryptBlocks(rest, rest)
		// Remove padding
		i := len(rest) - 1
		for rest[i] < 16 {
			i--
		}
		cfgContent = rest[:i]
	}

	cfgReader := csv.NewReader(bytes.NewReader(cfgContent))
	// Unix-style tabular
	cfgReader.Comma = ':'

	cfg, e := cfgReader.ReadAll()
	if e != nil {
		log.Fatal(e)
	}

	currentTS, progress := TimeStamp()
	prevTS := currentTS - 1
	nextTS := currentTS + 1

	maxWidth := 10
	for _, record := range cfg {
		cWidth := len(record[0])
		if cWidth > maxWidth {
			maxWidth = cWidth
		}
	}
	nameFmt := fmt.Sprintf("%%-%ds", maxWidth)

	fmt.Printf(nameFmt+" prev   curr   next\n", "")
	for _, record := range cfg {
		name := record[0]
		encodeType := "TOTP"
		if len(record) > 2 {
			encodeType = record[2]
		}
		secret := normalizeSecret(record[1])
		prevToken := authCodeOrDie(secret, prevTS, encodeType)
		currentToken := authCodeOrDie(secret, currentTS, encodeType)
		nextToken := authCodeOrDie(secret, nextTS, encodeType)
		fmt.Printf(nameFmt+" % 6s % 6s % 6s\n", name, prevToken, currentToken, nextToken)
	}
	fmt.Printf("[%-29s]\n", strings.Repeat("=", progress))
}
