package dotenvx

import (
	"os"
	"runtime"
	"strings"
	"testing"

	ecies "github.com/ecies/go/v2"
)

// Helper functions
func inTempDir(t *testing.T) string {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	return originalDir
}

func clearEnvKeys() {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "DOTENV_PRIVATE_KEY") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) > 0 {
				os.Unsetenv(parts[0])
			}
		}
	}
}

// Test getEnvFile function
func TestGetEnvFile_NoKeys(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	_, err := getEnvFile()
	if err == nil {
		t.Error("Expected error when no keys present")
	}
	if err.Error() != "No key found" {
		t.Errorf("Expected 'No key found', got %q", err.Error())
	}
}

func TestGetEnvFile_ValidKeyAndFile(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	// Create .env file
	os.WriteFile(".env", []byte("TEST=value"), 0644)
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	envFile, err := getEnvFile()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if envFile.Path != ".env" {
		t.Errorf("Expected path .env, got %q", envFile.Path)
	}
	if envFile.Key == nil {
		t.Error("Expected valid key, got nil")
	}
}

func TestGetEnvFile_InvalidKeyFormat(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	// Create .env file
	os.WriteFile(".env", []byte("TEST=value"), 0644)
	os.Setenv("DOTENV_PRIVATE_KEY", "invalid-key")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	_, err := getEnvFile()
	if err == nil {
		t.Error("Expected error with invalid key")
	}
	if err.Error() != "No valid file/key combination found" {
		t.Errorf("Expected 'No valid file/key combination found', got %q", err.Error())
	}
}

func TestGetEnvFile_MissingFile(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	// Don't create .env file
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	_, err := getEnvFile()
	if err == nil {
		t.Error("Expected error when file missing")
	}
}

func TestGetEnvFile_CustomEnvironment(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	// Create .env.staging file
	os.WriteFile(".env.staging", []byte("TEST=value"), 0644)
	os.Setenv("DOTENV_PRIVATE_KEY_STAGING", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY_STAGING")

	envFile, err := getEnvFile()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if envFile.Path != ".env.staging" {
		t.Errorf("Expected path .env.staging, got %q", envFile.Path)
	}
}

func TestGetEnvFile_UnderscoreToDot(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	// Create .env.qa.test file
	os.WriteFile(".env.qa.test", []byte("TEST=value"), 0644)
	os.Setenv("DOTENV_PRIVATE_KEY_QA_TEST", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY_QA_TEST")

	envFile, err := getEnvFile()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if envFile.Path != ".env.qa.test" {
		t.Errorf("Expected path .env.qa.test, got %q", envFile.Path)
	}
}

// Test decryptSecret function
func TestDecryptSecret_Valid(t *testing.T) {
	keyHex := "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d"
	privateKey, _ := ecies.NewPrivateKeyFromHex(keyHex)

	encrypted := "BL8cvfR8496FAJV3dbdSZj/D6qlhOc3lAhuAB24AGp4WASPH8BBoe21T+T9jlO/M0GY03RZ94Etk7VPWIP21vh+YLGu0fWe2usFdTFs+/BnlsT8K8+V9Xte/yXA2NhrRxy3T7ygL"

	result := decryptSecret(privateKey, encrypted)
	if result != "hello" {
		t.Errorf("Expected 'hello', got %q", result)
	}
}

// Test parseEnvVar function
func TestParseEnvVar_PlainValue(t *testing.T) {
	line := "TEST=plain value"
	result := parseEnvVar(line, nil, "")

	if result.Name != "TEST" {
		t.Errorf("Expected name 'TEST', got %q", result.Name)
	}
	if result.Value != "plain value" {
		t.Errorf("Expected value 'plain value', got %q", result.Value)
	}
}

func TestParseEnvVar_QuotedValue(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{`TEST="double quoted"`, "double quoted"},
		{`TEST='single quoted'`, "single quoted"},
		{`TEST="with spaces"`, "with spaces"},
	}

	for _, tt := range tests {
		result := parseEnvVar(tt.line, nil, "")
		if result.Value != tt.expected {
			t.Errorf("For %q: expected %q, got %q", tt.line, tt.expected, result.Value)
		}
	}
}

func TestParseEnvVar_ExportPrefix(t *testing.T) {
	line := "export TEST=value"
	result := parseEnvVar(line, nil, "")

	if result.Name != "TEST" {
		t.Errorf("Expected name 'TEST', got %q", result.Name)
	}
	if result.Value != "value" {
		t.Errorf("Expected value 'value', got %q", result.Value)
	}
}

func TestParseEnvVar_Comment(t *testing.T) {
	line := "# TEST=value"
	result := parseEnvVar(line, nil, "")

	if result.Name != "" {
		t.Errorf("Expected empty name for comment, got %q", result.Name)
	}
}

func TestParseEnvVar_EncryptedValue(t *testing.T) {
	keyHex := "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d"
	privateKey, _ := ecies.NewPrivateKeyFromHex(keyHex)

	line := `TEST=encrypted:BL8cvfR8496FAJV3dbdSZj/D6qlhOc3lAhuAB24AGp4WASPH8BBoe21T+T9jlO/M0GY03RZ94Etk7VPWIP21vh+YLGu0fWe2usFdTFs+/BnlsT8K8+V9Xte/yXA2NhrRxy3T7ygL`
	result := parseEnvVar(line, privateKey, "")

	if result.Value != "hello" {
		t.Errorf("Expected decrypted value 'hello', got %q", result.Value)
	}
}

func TestParseEnvVar_SpecificName(t *testing.T) {
	line := "TEST=value"

	// Should return empty when looking for different name
	result := parseEnvVar(line, nil, "OTHER")
	if result.Name != "" {
		t.Error("Expected empty result when name doesn't match")
	}

	// Should return value when name matches
	result = parseEnvVar(line, nil, "TEST")
	if result.Value != "value" {
		t.Errorf("Expected 'value', got %q", result.Value)
	}
}

// Test getEnvVars function
func TestGetEnvVars_FileNotFound(t *testing.T) {
	envFile := &EnvFile{Path: "nonexistent.env", Key: nil}

	vars, err := getEnvVars(envFile, "")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	if len(vars) != 0 {
		t.Error("Expected empty vars for error case")
	}
}

func TestGetEnvVars_AllVars(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	// Create test file
	content := `VAR1=value1
VAR2=value2
# Comment line
VAR3=value3`
	os.WriteFile("test.env", []byte(content), 0644)

	envFile := &EnvFile{Path: "test.env", Key: nil}
	vars, err := getEnvVars(envFile, "")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(vars) != 3 {
		t.Errorf("Expected 3 vars, got %d", len(vars))
	}
}

func TestGetEnvVars_SpecificVar(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	// Create test file
	content := `VAR1=value1
VAR2=value2
VAR3=value3`
	os.WriteFile("test.env", []byte(content), 0644)

	envFile := &EnvFile{Path: "test.env", Key: nil}
	vars, err := getEnvVars(envFile, "VAR2")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(vars) != 1 {
		t.Errorf("Expected 1 var, got %d", len(vars))
	}
	if vars[0].Name != "VAR2" || vars[0].Value != "value2" {
		t.Errorf("Expected VAR2=value2, got %s=%s", vars[0].Name, vars[0].Value)
	}
}

// Test Debug mode
func TestGetEnvFile_DebugMode(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	// Enable debug mode
	Debug = true
	defer func() { Debug = false }()

	// Test with no keys
	_, _ = getEnvFile()

	// Test with invalid key
	os.WriteFile(".env", []byte("TEST=value"), 0644)
	os.Setenv("DOTENV_PRIVATE_KEY", "invalid")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")
	_, _ = getEnvFile()

	// Test with missing file
	os.Remove(".env")
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	_, _ = getEnvFile()
}

func TestGetEnvVars_DebugMode(t *testing.T) {
	Debug = true
	defer func() { Debug = false }()

	envFile := &EnvFile{Path: "nonexistent.env", Key: nil}
	_, _ = getEnvVars(envFile, "")
}

func TestGetenv_DebugMode(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	Debug = true
	defer func() { Debug = false }()

	// Test with no env file
	_ = Getenv("TEST")

	// Test with valid file but var not found
	os.WriteFile(".env", []byte("OTHER=value"), 0644)
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")
	_ = Getenv("TEST")
}

func TestEnviron_DebugMode(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	Debug = true
	defer func() { Debug = false }()

	// Test with no env file
	_ = Environ()

	// Test with empty file
	os.WriteFile(".env", []byte(""), 0644)
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")
	_ = Environ()
}

func TestEnvFileForKeyVar(t *testing.T) {
	tests := map[string]string{
		"DOTENV_PRIVATE_KEY":         ".env",
		"DOTENV_PRIVATE_KEY_STAGING": ".env.staging",
		"DOTENV_PRIVATE_KEY_QA_TEST": ".env.qa.test",
		"DOTENV_PRIVATE_KEYS":        "",
	}
	for varName, expected := range tests {
		if got := envFileForKeyVar(varName); got != expected {
			t.Errorf("For %q: expected %q, got %q", varName, expected, got)
		}
	}
}

// os.Environ lists variables in setenv insertion order, so setting the same two
// keys in either order is what makes this a test of precedence and not of luck.
func setKeys(t *testing.T, names ...string) {
	t.Helper()
	clearEnvKeys()
	for _, name := range names {
		os.Setenv(name, "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
		t.Cleanup(func() { os.Unsetenv(name) })
	}
}

func TestGetEnvFile_UnsuffixedWinsInEitherOrder(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	os.WriteFile(".env", []byte("TEST=default"), 0644)
	os.WriteFile(".env.staging", []byte("TEST=staging"), 0644)
	os.WriteFile(".env.workflowstore", []byte("TEST=workflowstore"), 0644)

	orders := [][]string{
		{"DOTENV_PRIVATE_KEY", "DOTENV_PRIVATE_KEY_STAGING", "DOTENV_PRIVATE_KEY_WORKFLOWSTORE"},
		{"DOTENV_PRIVATE_KEY_WORKFLOWSTORE", "DOTENV_PRIVATE_KEY_STAGING", "DOTENV_PRIVATE_KEY"},
		{"DOTENV_PRIVATE_KEY_STAGING", "DOTENV_PRIVATE_KEY", "DOTENV_PRIVATE_KEY_WORKFLOWSTORE"},
	}
	for _, order := range orders {
		setKeys(t, order...)

		envFile, err := getEnvFile()
		if err != nil {
			t.Fatalf("For order %v: expected no error, got %v", order, err)
		}
		if envFile.Path != ".env" {
			t.Errorf("For order %v: expected .env, got %q", order, envFile.Path)
		}
		if got := Getenv("TEST"); got != "default" {
			t.Errorf("For order %v: expected Getenv 'default', got %q", order, got)
		}
	}
}

func TestGetEnvFile_SingleSuffixedKeyWithOtherKeysPresent(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	// Only .env.staging exists, so neither the unsuffixed nor the prod key applies
	os.WriteFile(".env.staging", []byte("TEST=staging"), 0644)
	setKeys(t, "DOTENV_PRIVATE_KEY_STAGING", "DOTENV_PRIVATE_KEY", "DOTENV_PRIVATE_KEY_PROD")

	envFile, err := getEnvFile()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if envFile.Path != ".env.staging" {
		t.Errorf("Expected .env.staging, got %q", envFile.Path)
	}
}

func TestGetEnvFile_MultipleSuffixedIsAmbiguous(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	os.WriteFile(".env.staging", []byte("TEST=staging"), 0644)
	os.WriteFile(".env.workflowstore", []byte("TEST=workflowstore"), 0644)

	for _, order := range [][]string{
		{"DOTENV_PRIVATE_KEY_STAGING", "DOTENV_PRIVATE_KEY_WORKFLOWSTORE"},
		{"DOTENV_PRIVATE_KEY_WORKFLOWSTORE", "DOTENV_PRIVATE_KEY_STAGING"},
	} {
		setKeys(t, order...)

		_, err := getEnvFile()
		if err == nil {
			t.Fatalf("For order %v: expected an ambiguity error", order)
		}
		expected := "Ambiguous private key: DOTENV_PRIVATE_KEY_STAGING -> .env.staging, " +
			"DOTENV_PRIVATE_KEY_WORKFLOWSTORE -> .env.workflowstore, and no DOTENV_PRIVATE_KEY for .env to break the tie"
		if err.Error() != expected {
			t.Errorf("For order %v: expected %q, got %q", order, expected, err.Error())
		}
		if got := Getenv("TEST"); got != "" {
			t.Errorf("For order %v: expected Getenv to yield \"\", got %q", order, got)
		}
		if got := Environ(); len(got) != 0 {
			t.Errorf("For order %v: expected Environ to yield nothing, got %v", order, got)
		}
	}
}

func TestGetEnvFile_AmbiguityDebugMode(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	Debug = true
	defer func() { Debug = false }()

	os.WriteFile(".env", []byte("TEST=default"), 0644)
	os.WriteFile(".env.staging", []byte("TEST=staging"), 0644)

	setKeys(t, "DOTENV_PRIVATE_KEY_STAGING", "DOTENV_PRIVATE_KEY")
	if envFile, err := getEnvFile(); err != nil || envFile.Path != ".env" {
		t.Errorf("Expected .env with no error, got %q %v", envFile.Path, err)
	}

	// The unsuffixed key stays set but .env is gone, so it cannot break the tie
	os.Remove(".env")
	os.WriteFile(".env.prod", []byte("TEST=prod"), 0644)
	setKeys(t, "DOTENV_PRIVATE_KEY_STAGING", "DOTENV_PRIVATE_KEY_PROD", "DOTENV_PRIVATE_KEY")
	if _, err := getEnvFile(); err == nil {
		t.Error("Expected an ambiguity error")
	}
}

// Test multiple keys warning with debug output
func TestGetEnvFile_MultipleKeysWarning(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	Debug = true
	defer func() { Debug = false }()

	// Create only .env.prod file so DOTENV_PRIVATE_KEY won't find .env
	os.WriteFile(".env.prod", []byte("TEST=prod"), 0644)

	// Set multiple keys - the unsuffixed one has no file, so it must not shadow .env.prod
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	os.Setenv("DOTENV_PRIVATE_KEY_PROD", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY_PROD")

	envFile, err := getEnvFile()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	// Should use .env.prod since .env doesn't exist
	if envFile.Path != ".env.prod" {
		t.Errorf("Expected .env.prod, got %q", envFile.Path)
	}
}

// Test permission denied edge case
func TestGetEnvVars_PermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	defer os.Chdir(inTempDir(t))

	// Create a file with no read permissions
	os.WriteFile("test.env", []byte("TEST=value"), 0000)
	defer os.Chmod("test.env", 0644)

	keyHex := "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d"
	privateKey, _ := ecies.NewPrivateKeyFromHex(keyHex)
	envFile := &EnvFile{Path: "test.env", Key: privateKey}

	vars, err := getEnvVars(envFile, "")
	if err == nil {
		t.Error("Expected error for permission denied")
	}
	if len(vars) != 0 {
		t.Error("Expected empty vars for error case")
	}
}

// Test Environ with getEnvVars error
func TestEnviron_GetEnvVarsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	// Create a file with no read permissions
	os.WriteFile(".env", []byte("TEST=value"), 0000)
	defer os.Chmod(".env", 0644)

	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	result := Environ()
	if len(result) != 0 {
		t.Error("Expected empty result when getEnvVars fails")
	}
}

// Test empty key in environment variable
func TestGetEnvFile_EmptyKeyValue(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	// Set empty key value
	os.Setenv("DOTENV_PRIVATE_KEY", "")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	_, err := getEnvFile()
	if err == nil {
		t.Error("Expected error when key value is empty")
	}
}

// Test DecryptFile
const testKeyHex = "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d"

// "hello" encrypted to testKeyHex's public key
const testCipher = "encrypted:BL8cvfR8496FAJV3dbdSZj/D6qlhOc3lAhuAB24AGp4WASPH8BBoe21T+T9jlO/M0GY03RZ94Etk7VPWIP21vh+YLGu0fWe2usFdTFs+/BnlsT8K8+V9Xte/yXA2NhrRxy3T7ygL"

func TestDecryptFile_PlainAndEncrypted(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	os.WriteFile(".env.staging", []byte("PLAIN=plain value\nSECRET="+testCipher+"\n"), 0644)

	vars, err := DecryptFile(".env.staging", testKeyHex)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(vars) != 2 {
		t.Fatalf("Expected 2 vars, got %d", len(vars))
	}
	if vars[0].Name != "PLAIN" || vars[0].Value != "plain value" {
		t.Errorf("Expected PLAIN=plain value, got %s=%s", vars[0].Name, vars[0].Value)
	}
	if vars[1].Name != "SECRET" || vars[1].Value != "hello" {
		t.Errorf("Expected SECRET=hello, got %s=%s", vars[1].Name, vars[1].Value)
	}
}

// The whole reason this exists: Getenv and Environ return "" here, so a caller
// cannot tell a wrong key from an empty secret.
func TestDecryptFile_WrongKeyErrors(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	wrong, err := ecies.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	os.WriteFile(".env", []byte("SECRET="+testCipher+"\n"), 0644)

	vars, err := DecryptFile(".env", wrong.Hex())
	if err == nil {
		t.Fatalf("Expected error with a valid but wrong key, got vars %+v", vars)
	}
	if !strings.Contains(err.Error(), "SECRET") {
		t.Errorf("Expected the error to name SECRET, got %v", err)
	}
}

func TestDecryptFile_InvalidKeyHex(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	os.WriteFile(".env", []byte("PLAIN=value\n"), 0644)

	if _, err := DecryptFile(".env", "not-hex"); err == nil {
		t.Error("Expected error with invalid key hex")
	}
}

func TestDecryptFile_MissingFile(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	if _, err := DecryptFile(".env.absent", testKeyHex); err == nil {
		t.Error("Expected error for missing file")
	}
}

func TestDecryptFile_BadBase64Errors(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	os.WriteFile(".env", []byte("SECRET=encrypted:!!!not base64!!!\n"), 0644)

	_, err := DecryptFile(".env", testKeyHex)
	if err == nil {
		t.Fatal("Expected error for undecodable base64")
	}
	if !strings.Contains(err.Error(), "SECRET") {
		t.Errorf("Expected the error to name SECRET, got %v", err)
	}
}

func TestDecryptFile_CommentsAndBlankLinesSkipped(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	os.WriteFile(".env", []byte("# a comment\n\nVAR1=one\nVAR2=two\n"), 0644)

	vars, err := DecryptFile(".env", testKeyHex)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(vars) != 2 {
		t.Errorf("Expected 2 vars, got %d", len(vars))
	}
}

// A file with nothing encrypted in it must not need a decryptable key to read.
func TestDecryptFile_PlainFileNeedsNoMatchingKey(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	unrelated, err := ecies.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	os.WriteFile(".env", []byte("VAR1=one\n"), 0644)

	vars, err := DecryptFile(".env", unrelated.Hex())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(vars) != 1 || vars[0].Value != "one" {
		t.Errorf("Expected VAR1=one, got %+v", vars)
	}
}

// Test the integration paths
func TestIntegration_SuccessfulDecryption(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	clearEnvKeys()

	// Create encrypted content
	content := `PLAIN=plain value
ENCRYPTED=encrypted:BL8cvfR8496FAJV3dbdSZj/D6qlhOc3lAhuAB24AGp4WASPH8BBoe21T+T9jlO/M0GY03RZ94Etk7VPWIP21vh+YLGu0fWe2usFdTFs+/BnlsT8K8+V9Xte/yXA2NhrRxy3T7ygL`
	os.WriteFile(".env", []byte(content), 0644)

	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	// Test Getenv
	if got := Getenv("PLAIN"); got != "plain value" {
		t.Errorf("Expected 'plain value', got %q", got)
	}
	if got := Getenv("ENCRYPTED"); got != "hello" {
		t.Errorf("Expected 'hello', got %q", got)
	}

	// Test Environ
	vars := Environ()
	if len(vars) != 2 {
		t.Errorf("Expected 2 vars, got %d", len(vars))
	}
}
