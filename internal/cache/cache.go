package cache

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

// ErrCacheNotAvailable occurs when cache file is obsolete or does not exist
var ErrCacheNotAvailable = fmt.Errorf("Cache is not available")

func Load(dbPath, password string, updated time.Time) ([]string, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	dirPath := filepath.Join(os.TempDir(), fmt.Sprintf("keepeco.%s", user.Uid))

	_, err = os.Stat(dirPath)
	if os.IsNotExist(err) {
		return nil, ErrCacheNotAvailable
	} else if err != nil {
		return nil, err
	}

	filePath := filepath.Join(dirPath, fmt.Sprintf("%s.%x", filepath.Base(dbPath), md5.Sum([]byte(dbPath))))
	f, err := os.Open(filePath)
	defer f.Close()

	if os.IsNotExist(err) {
		return nil, ErrCacheNotAvailable
	} else if err != nil {
		return nil, err
	}

	cipherText, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	key := buildKey(dbPath, password, updated)
	content, err := decrypt(key, cipherText)
	if err != nil {
		return nil, ErrCacheNotAvailable
	}
	lines := strings.Split(content, "\n")
	return lines, nil
}

func Save(dbPath, password string, updated time.Time, candidates []string) error {
	user, err := user.Current()
	if err != nil {
		return err
	}
	dirPath := filepath.Join(os.TempDir(), fmt.Sprintf("keepeco.%s", user.Uid))

	_, err = os.Stat(dirPath)
	if os.IsNotExist(err) {
		err := os.Mkdir(dirPath, 0700)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	filePath := filepath.Join(dirPath, fmt.Sprintf("%s.%x", filepath.Base(dbPath), md5.Sum([]byte(dbPath))))
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	err = os.Chmod(filePath, 0600)
	if err != nil {
		return err
	}

	key := buildKey(dbPath, password, updated)
	cipherText, err := encrypt(key, strings.Join(candidates, "\n"))
	if err != nil {
		return err
	}
	_, err = f.Write(cipherText)
	if err != nil {
		return err
	}
	return nil
}

func encrypt(key []byte, plainText string) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize()) // Unique nonce is required(NonceSize 12byte)
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}

	cipherText := gcm.Seal(nil, nonce, []byte(plainText), nil)
	cipherText = append(nonce, cipherText...)

	return cipherText, nil
}

func decrypt(key []byte, cipherText []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := cipherText[:gcm.NonceSize()]
	plainByte, err := gcm.Open(nil, nonce, cipherText[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}

	return string(plainByte), nil
}

func buildKey(dbPath, password string, updated time.Time) []byte {
	s := fmt.Sprintf("%s!%s!%d", dbPath, password, updated.UnixNano())
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}
