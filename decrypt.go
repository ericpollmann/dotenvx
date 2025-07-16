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
	Debug  bool
	EnvMap map[string]string
	Mu     sync.Mutex
	Once   sync.Once
)

type EnvFile struct {
	Path string
	Key  string
}

// searches for DOTENV_PRIVATE_KEY* environment variables, returns all potential file/key pairs
func findEnvFiles() []EnvFile {
	var envFiles []EnvFile

	if Debug {
		println("Checking for private key in environment")
	}

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "DOTENV_PRIVATE_KEY") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 || parts[1] == "" {
				continue
			}

			varName := parts[0]
			keyHex := parts[1]

			// Map DOTENV_PRIVATE_KEY_SUFFIX to .env.suffix
			fileName := ""
			if varName == "DOTENV_PRIVATE_KEY" {
				fileName = ".env"
			} else if strings.HasPrefix(varName, "DOTENV_PRIVATE_KEY_") {
				suffix := strings.TrimPrefix(varName, "DOTENV_PRIVATE_KEY_")
				// Convert suffix to lowercase, replace _ with .
				suffix = strings.ToLower(suffix)
				suffix = strings.ReplaceAll(suffix, "_", ".")
				fileName = ".env." + suffix
			}

			if Debug {
				println("Trying key " + varName + " with file " + fileName)
			}

			envFiles = append(envFiles, EnvFile{Path: fileName, Key: keyHex})
		}
	}

	if len(envFiles) == 0 && Debug {
		println("No key found")
	}

	return envFiles
}

func processEnvFile(envFile *EnvFile) error {
	file, err := os.Open(envFile.Path)
	if err != nil {
		if Debug {
			if os.IsNotExist(err) {
				println("File not found: " + envFile.Path)
			} else {
				println("Unable to open: " + envFile.Path)
			}
		}
		return err
	}
	defer file.Close()

	privateKey, err := ecies.NewPrivateKeyFromHex(envFile.Key)
	if err != nil {
		if Debug {
			println("Invalid key format")
		}
		return err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if offset := strings.Index(line, "="); offset > 0 && line[0] != '#' {
			varName := strings.TrimPrefix(line[:offset], "export ")
			value := line[offset+1:]

			value = strings.TrimSpace(value)
			if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
				(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
				value = value[1 : len(value)-1]
			}

			if strings.HasPrefix(value, "encrypted:") {
				cipherBytes, _ := base64.StdEncoding.DecodeString(value[10:])
				plainBytes, _ := ecies.Decrypt(privateKey, cipherBytes)
				EnvMap[varName] = string(plainBytes)
			} else {
				EnvMap[varName] = value
			}
		}
	}

	return scanner.Err()
}

func loadEnv() {
	EnvMap = make(map[string]string)

	for _, envFile := range findEnvFiles() {
		err := processEnvFile(&envFile)
		if err == nil {
			return
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
	if Debug {
		if !found {
			println("Not found in env file: " + key)
		}
		if len(val) == 0 {
			println("Empty string value: " + key)
		}
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
