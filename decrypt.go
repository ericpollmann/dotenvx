package dotenvx

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"sort"
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

const keyVar = "DOTENV_PRIVATE_KEY"

type keyCandidate struct {
	varName  string
	fileName string
	keyHex   string
}

// Mirrors dotenvx's own convention: DOTENV_PRIVATE_KEY_QA_TEST -> .env.qa.test
func envFileForKeyVar(varName string) string {
	if varName == keyVar {
		return ".env"
	}
	if suffix := strings.TrimPrefix(varName, keyVar+"_"); suffix != varName {
		return ".env." + strings.ReplaceAll(strings.ToLower(suffix), "_", ".")
	}
	return ""
}

// A first-match scan of os.Environ picks by setenv insertion order, so adding or
// renaming an unrelated key silently switches which file gets decrypted.
func chooseCandidate(candidates []keyCandidate) (*keyCandidate, error) {
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].varName < candidates[j].varName })

	for i := range candidates {
		if candidates[i].varName == keyVar {
			if Debug && len(candidates) > 1 {
				fmt.Printf("Multiple private keys match a file; %s wins over the suffixed ones\n", keyVar)
			}
			return &candidates[i], nil
		}
	}
	switch len(candidates) {
	case 0:
		return nil, nil
	case 1:
		return &candidates[0], nil
	}

	names := make([]string, 0, len(candidates))
	for _, c := range candidates {
		names = append(names, c.varName+" -> "+c.fileName)
	}
	return nil, fmt.Errorf("Ambiguous private key: %s, and no %s for .env to break the tie",
		strings.Join(names, ", "), keyVar)
}

func getEnvFile() (envFile EnvFile, err error) {
	if Debug {
		fmt.Println("Checking for private key in environment")
	}

	keysInEnv := 0
	var candidates []keyCandidate
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, keyVar) {
			continue
		}
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 || parts[1] == "" {
			continue
		}
		keysInEnv++

		candidate := keyCandidate{parts[0], envFileForKeyVar(parts[0]), parts[1]}
		if Debug {
			fmt.Printf("Found key %s and file %s\n", candidate.varName, candidate.fileName)
		}
		if _, err := os.Stat(candidate.fileName); err != nil {
			if Debug {
				fmt.Printf("Unable to open: %s\n", candidate.fileName)
			}
			continue
		}
		candidates = append(candidates, candidate)
	}

	chosen, err := chooseCandidate(candidates)
	if err != nil {
		if Debug {
			fmt.Println(err)
		}
		return envFile, err
	}
	if chosen != nil {
		if privateKey, err := ecies.NewPrivateKeyFromHex(chosen.keyHex); err == nil {
			return EnvFile{chosen.fileName, privateKey}, nil
		} else if Debug {
			fmt.Println("Invalid key format")
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
