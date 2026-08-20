package dotenvx

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	ecies "github.com/ecies/go/v2"
)

var (
	Debug bool
)

type EnvFile struct {
	Path string
	Key  *ecies.PrivateKey
}

type EnvVar struct {
	Name  string
	Value string
}

// searches for DOTENV_PRIVATE_KEY* environment variables, returns first valid file/key pair
func getEnvFile() (envFile EnvFile, err error) {
	if Debug {
		fmt.Println("Checking for private key in environment")
	}

	// Search for valid key / with corresponding EnvFile
	keysInEnv := 0
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "DOTENV_PRIVATE_KEY") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 || parts[1] == "" {
				continue
			}
			keysInEnv++

			varName, keyHex := parts[0], parts[1]

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
				fmt.Printf("Found key %s and file %s\n", varName, fileName)
				if keysInEnv > 1 {
					fmt.Println("WARNING: Multiple private keys found in environment!")
				}
			}
			if _, err := os.Stat(fileName); err == nil {
				if privateKey, err := ecies.NewPrivateKeyFromHex(keyHex); err == nil {
					return EnvFile{fileName, privateKey}, nil
				} else if Debug {
					fmt.Println("Invalid key format")
				}
			} else if Debug {
				fmt.Printf("Unable to open: %s\n", fileName)
			}
		}
	}

	// No valid file/key combination found
	if keysInEnv == 0 {
		if Debug {
			fmt.Println("No key found")
		}
		err = fmt.Errorf("No key found")
	} else {
		err = fmt.Errorf("No valid file/key combination found")
	}

	return envFile, err
}

const encryptedPrefix = "encrypted:"

func decryptSecret(privateKey *ecies.PrivateKey, base64ciper string) string {
	cipherBytes, _ := base64.StdEncoding.DecodeString(base64ciper)
	plainBytes, _ := ecies.Decrypt(privateKey, cipherBytes)
	return string(plainBytes)
}

// Wrong-key decryption fails the AEAD tag check rather than returning garbage,
// so propagating the error is what separates it from a genuinely empty value.
func decryptSecretStrict(privateKey *ecies.PrivateKey, base64cipher string) (string, error) {
	cipherBytes, err := base64.StdEncoding.DecodeString(base64cipher)
	if err != nil {
		return "", fmt.Errorf("base64: %w", err)
	}
	plainBytes, err := ecies.Decrypt(privateKey, cipherBytes)
	if err != nil {
		return "", err
	}
	if len(plainBytes) == 0 && len(cipherBytes) > 0 {
		return "", fmt.Errorf("decrypted to an empty value")
	}
	return string(plainBytes), nil
}

// ok is false for blank lines, comments, and -- when name is non-empty --
// every variable except that one.
func splitEnvLine(line, name string) (varName, value string, ok bool) {
	offset := strings.Index(line, "=")
	if offset <= 0 || line[0] == '#' {
		return "", "", false
	}
	varName = strings.TrimPrefix(line[:offset], "export ")
	if name != "" && varName != name {
		return "", "", false
	}
	value = strings.TrimSpace(line[offset+1:])
	if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
		(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
		value = value[1 : len(value)-1]
	}
	return varName, value, true
}

func parseEnvVar(line string, privateKey *ecies.PrivateKey, name string) EnvVar {
	varName, value, ok := splitEnvLine(line, name)
	if !ok {
		return EnvVar{}
	}
	if strings.HasPrefix(value, encryptedPrefix) {
		value = decryptSecret(privateKey, value[len(encryptedPrefix):])
	}
	return EnvVar{varName, value}
}

func getEnvVars(envFile *EnvFile, name string) (vars []EnvVar, err error) {
	file, err := os.Open(envFile.Path)
	if err != nil {
		if Debug {
			fmt.Printf("Unable to open %s: %v\n", envFile.Path, err)
		}
		return []EnvVar{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		envVar := parseEnvVar(scanner.Text(), envFile.Key, name)
		if envVar.Name == "" {
			continue
		}
		vars = append(vars, envVar)
	}
	return vars, scanner.Err()
}

// Unlike Getenv and Environ, which pick a file by scanning the environment for
// any usable DOTENV_PRIVATE_KEY* and silently yield "" for whatever they cannot
// decrypt, this names one file and one key and fails on the first value that
// does not decrypt. Callers holding secrets they must not run without want this
// one.
func DecryptFile(path string, privateKeyHex string) ([]EnvVar, error) {
	privateKey, err := ecies.NewPrivateKeyFromHex(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("%s: private key: %w", path, err)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var vars []EnvVar
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		varName, value, ok := splitEnvLine(scanner.Text(), "")
		if !ok {
			continue
		}
		if strings.HasPrefix(value, encryptedPrefix) {
			value, err = decryptSecretStrict(privateKey, value[len(encryptedPrefix):])
			if err != nil {
				return nil, fmt.Errorf("%s: %s: %w", path, varName, err)
			}
		}
		vars = append(vars, EnvVar{varName, value})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return vars, nil
}

func Getenv(key string) string {
	envFile, err := getEnvFile()
	if Debug && (envFile.Key == nil || err != nil) {
		fmt.Printf("Error finding envFile (%+v): %+v\n", envFile, err)
	}
	if err != nil {
		return ""
	}
	vars, err := getEnvVars(&envFile, key)
	if Debug && (len(vars) != 1 || err != nil) {
		fmt.Printf("Error retrieving (%s) (%d values): %+v\n", key, len(vars), err)
	}
	if err != nil || len(vars) == 0 {
		return ""
	}
	return vars[0].Value
}

func Environ() []string {
	envFile, err := getEnvFile()
	if Debug && (envFile.Key == nil || err != nil) {
		fmt.Printf("Error finding envFile (%+v): %+v\n", envFile, err)
	}
	if err != nil {
		return []string{}
	}
	vars, err := getEnvVars(&envFile, "")
	if Debug && (len(vars) == 0 || err != nil) {
		fmt.Printf("Error retrieving all values (%d found): %+v\n", len(vars), err)
	}
	if err != nil {
		return []string{}
	}
	env := make([]string, 0, len(vars))
	for _, v := range vars {
		env = append(env, v.Name+"="+v.Value)
	}
	return env
}
