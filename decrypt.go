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
	Debug  bool = true
	EnvMap map[string]string
	Mu     sync.Mutex
	Once   sync.Once
)

func loadEnv() {
	EnvMap = make(map[string]string)

	if Debug {
		println("Checking for private key in environment")
	}
	keyHex, fileName := os.Getenv("DOTENV_PRIVATE_KEY_PRODUCTION"), ".env.production"
	if keyHex == "" {
		if Debug {
			println("Not production, trying local key")
		}
		keyHex, fileName = os.Getenv("DOTENV_PRIVATE_KEY"), ".env"
		if keyHex == "" {
			if Debug {
				println("No key found")
			}
			return
		}
	}
	file, err := os.Open(fileName)
	if err != nil {
		if Debug {
			println("Unable to open: " + fileName)
		}
		return
	}
	defer file.Close()

	privateKey, _ := ecies.NewPrivateKeyFromHex(keyHex)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if offset := strings.Index(line, "="); offset > 0 && line[0] != '#' {
			varName := strings.TrimPrefix(line[:offset], "export ")
			value := line[offset+1:]
			
			// Strip surrounding quotes if present
			value = strings.TrimSpace(value)
			if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
			   (strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
				value = value[1:len(value)-1]
			}
			
			// Check if it's an encrypted value
			if strings.HasPrefix(value, "encrypted:") {
				cipherBytes, _ := base64.StdEncoding.DecodeString(value[10:])
				plainBytes, _ := ecies.Decrypt(privateKey, cipherBytes)
				EnvMap[varName] = string(plainBytes)
			} else {
				EnvMap[varName] = value
			}
		}
	}
}

func GetenvMap() map[string]string {
	Once.Do(loadEnv)
	return EnvMap
}

func Getenv(key string) string {
	envMap := GetenvMap()
	val, found := envMap[key]
	if Debug && !found {
		println("Not found in env file: " + key)
	}
	if Debug && len(val) == 0 {
		println("Empty string value for $" + key)
	}
	return val
}

func Environ() []string {
	m := GetenvMap()
	env := make([]string, 0, len(m))
	for k, v := range m {
		env = append(env, k+"="+v)
	}
	return env
}
