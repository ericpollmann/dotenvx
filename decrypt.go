package dotenvx

import (
	"bufio"
	"encoding/base64"
	"os"
	"strings"
	"sync"

	ecies "github.com/ecies/go/v2"
)

var (
	Once   sync.Once
	EnvMap map[string]string
	Mu     sync.Mutex
)

func loadEnv() {
	EnvMap = make(map[string]string)

	keyHex, fileName := os.Getenv("DOTENV_PRIVATE_KEY_PRODUCTION"), ".env.production"
	if keyHex == "" {
		keyHex, fileName = os.Getenv("DOTENV_PRIVATE_KEY"), ".env"
		if keyHex == "" {
			return
		}
	}
	file, err := os.Open(fileName)
	if err != nil {
		return
	}
	defer file.Close()

	privateKey, _ := ecies.NewPrivateKeyFromHex(keyHex)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if offset := strings.Index(line, "=encrypted:"); offset >= 0 {
			varName := strings.TrimPrefix(line[:offset], "export ")
			cipherBytes, _ := base64.StdEncoding.DecodeString(line[offset+11:])
			plainBytes, _ := ecies.Decrypt(privateKey, cipherBytes)
			EnvMap[varName] = string(plainBytes)
		} else if offset := strings.Index(line, "="); offset > 0 && line[0] != '#' {
			varName := strings.TrimPrefix(line[:offset], "export ")
			EnvMap[varName] = line[offset+1:]
		}
	}
}

func GetenvMap() map[string]string {
	Once.Do(loadEnv)
	return EnvMap
}

func Getenv(key string) string {
	return GetenvMap()[key]
}

func Environ() []string {
	m := GetenvMap()
	env := make([]string, 0, len(m))
	for k, v := range m {
		env = append(env, k+"="+v)
	}
	return env
}
