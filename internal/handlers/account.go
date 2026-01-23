package handlers

import (
	"archive/tar"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/imzza/gdrive/internal/auth"
	"github.com/imzza/gdrive/internal/cli"
	drivepkg "github.com/imzza/gdrive/internal/drive"
	"github.com/imzza/gdrive/internal/utils"
)

const AccountConfigFilename = "account.json"

type accountConfig struct {
	Current string `json:"current"`
}

func AccountAddHandler(ctx cli.Context) {
	args := ctx.Args()
	baseDir := getBaseConfigDir(args)
	name := strings.TrimSpace(args.String("name"))

	secret := promptAccountSecret()

	if err := validateAccountName(name); err != nil && name != "" {
		utils.ExitF("Invalid account name: %s", err)
	}

	loginEmail := ""
	if name == "" {
		name = addAccountWithEmail(baseDir, secret, args)
		loginEmail = name
	} else {
		loginEmail = addAccountWithName(baseDir, name, secret, args)
	}

	fmt.Println("")
	fmt.Printf("Saved account credentials in %s\n", baseDir)
	fmt.Println("Keep them safe! If someone gets access to them, they will also be able to access your Google Drive.")

	fmt.Println("")
	fmt.Printf("Logged in as %s\n", loginEmail)
}

func AccountListHandler(ctx cli.Context) {
	args := ctx.Args()
	baseDir := getBaseConfigDir(args)
	accounts, err := listAccounts(baseDir)
	if err != nil {
		utils.ExitF("Failed to list accounts: %s", err)
	}

	if len(accounts) == 0 {
		utils.ExitF("No accounts found. Use `gdrive account add` to add an account.")
	}

	current := ""
	if config, err := loadAccountConfig(baseDir); err == nil {
		current = config.Current
	}

	for _, account := range accounts {
		if account == current && current != "" {
			fmt.Printf("* %s\n", account)
		} else {
			fmt.Printf("  %s\n", account)
		}
	}
}

func AccountCurrentHandler(ctx cli.Context) {
	args := ctx.Args()
	baseDir := getBaseConfigDir(args)
	config, err := loadAccountConfig(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			utils.ExitF("No account selected. Use `gdrive account list` to show accounts.")
		}
		utils.ExitF("Failed to read account config: %s", err)
	}

	if config.Current == "" {
		utils.ExitF("No account selected. Use `gdrive account switch` to select an account.")
	}

	fmt.Println(config.Current)
}

func AccountSwitchHandler(ctx cli.Context) {
	args := ctx.Args()
	baseDir := getBaseConfigDir(args)
	name := strings.TrimSpace(args.String("name"))

	if err := validateAccountName(name); err != nil {
		utils.ExitF("Invalid account name: %s", err)
	}

	if !accountExists(baseDir, name) {
		utils.ExitF("Account '%s' not found", name)
	}

	if err := saveAccountConfig(baseDir, accountConfig{Current: name}); err != nil {
		utils.ExitF("Failed to switch account: %s", err)
	}

	removeLegacyToken(accountDir(baseDir, name))

	fmt.Printf("Switched to account '%s'\n", name)
}

func AccountRemoveHandler(ctx cli.Context) {
	args := ctx.Args()
	baseDir := getBaseConfigDir(args)
	name := strings.TrimSpace(args.String("name"))

	if err := validateAccountName(name); err != nil {
		utils.ExitF("Invalid account name: %s", err)
	}

	accountDir := accountDir(baseDir, name)
	if _, err := os.Stat(accountDir); err != nil {
		if os.IsNotExist(err) {
			utils.ExitF("Account '%s' not found", name)
		}
		utils.ExitF("Failed to access account: %s", err)
	}

	if err := os.RemoveAll(accountDir); err != nil {
		utils.ExitF("Failed to remove account '%s': %s", name, err)
	}

	config, err := loadAccountConfig(baseDir)
	if err == nil && config.Current == name {
		_ = os.Remove(accountConfigPath(baseDir))
	}

	fmt.Printf("Removed account '%s'\n", name)
}

func AccountExportHandler(ctx cli.Context) {
	args := ctx.Args()
	baseDir := getBaseConfigDir(args)
	name := strings.TrimSpace(args.String("name"))

	if err := validateAccountName(name); err != nil {
		utils.ExitF("Invalid account name: %s", err)
	}

	if !accountExists(baseDir, name) {
		utils.ExitF("Account '%s' not found", name)
	}

	accountPath := accountDir(baseDir, name)
	archiveName := fmt.Sprintf("gdrive_export-%s.tar", normalizeArchiveName(name))

	if err := createAccountArchive(accountPath, archiveName); err != nil {
		utils.ExitF("Failed to export account: %s", err)
	}

	fmt.Printf("Exported account '%s' to %s\n", name, archiveName)
}

func AccountImportHandler(ctx cli.Context) {
	args := ctx.Args()
	baseDir := getBaseConfigDir(args)
	archivePath := strings.TrimSpace(args.String("path"))

	if archivePath == "" {
		utils.ExitF("Archive path is required")
	}

	accountName, err := archiveAccountName(archivePath)
	if err != nil {
		utils.ExitF("Failed to read account name from archive: %s", err)
	}

	if err := validateAccountName(accountName); err != nil {
		utils.ExitF("Invalid account name in archive: %s", err)
	}

	if accountExists(baseDir, accountName) {
		utils.ExitF("Account '%s' already exists", accountName)
	}

	if err := unpackAccountArchive(archivePath, baseDir); err != nil {
		utils.ExitF("Failed to import account: %s", err)
	}

	removeLegacyToken(accountDir(baseDir, accountName))

	fmt.Printf("Imported account '%s'\n", accountName)

	if _, err := loadAccountConfig(baseDir); os.IsNotExist(err) {
		if err := saveAccountConfig(baseDir, accountConfig{Current: accountName}); err != nil {
			utils.ExitF("Failed to set current account: %s", err)
		}
		fmt.Printf("Switched to account '%s'\n", accountName)
	}
}

func addAccountWithName(baseDir, name string, secret utils.AccountSecret, args cli.Arguments) string {
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		utils.ExitF("Failed to create config directory: %s", err)
	}

	if accountExists(baseDir, name) {
		utils.ExitF("Account '%s' already exists", name)
	}

	accountPath := accountDir(baseDir, name)
	if err := os.MkdirAll(accountPath, 0700); err != nil {
		utils.ExitF("Failed to create account directory: %s", err)
	}

	if err := utils.SaveAccountSecret(accountPath, secret); err != nil {
		utils.ExitF("Failed to save secret: %s", err)
	}

	client := accountAuthClient(args, accountPath, secret)
	drv, err := drivepkg.New(client)
	if err != nil {
		utils.ExitF("Failed to create drive client: %s", err)
	}

	email, err := drv.UserEmail()
	if err != nil {
		utils.ExitF("Failed to authenticate: %s", err)
	}

	tokenPath := utils.ConfigFilePath(accountPath, TokenFilename)
	if _, err := os.Stat(tokenPath); err != nil {
		utils.ExitF("Failed to create token file: %s", err)
	}

	if err := saveAccountConfig(baseDir, accountConfig{Current: name}); err != nil {
		utils.ExitF("Failed to save account config: %s", err)
	}

	removeLegacyToken(accountPath)

	return email
}

func addAccountWithEmail(baseDir string, secret utils.AccountSecret, args cli.Arguments) string {
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		utils.ExitF("Failed to create config directory: %s", err)
	}

	tmpDir, err := os.MkdirTemp(baseDir, "auth-")
	if err != nil {
		utils.ExitF("Failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	client := accountAuthClient(args, tmpDir, secret)
	drv, err := drivepkg.New(client)
	if err != nil {
		utils.ExitF("Failed to create drive client: %s", err)
	}

	email, err := drv.UserEmail()
	if err != nil {
		utils.ExitF("Failed to authenticate: %s", err)
	}

	if err := validateAccountName(email); err != nil {
		utils.ExitF("Invalid account name: %s", err)
	}

	if accountExists(baseDir, email) {
		utils.ExitF("Account '%s' already exists", email)
	}

	accountPath := accountDir(baseDir, email)
	if err := os.MkdirAll(accountPath, 0700); err != nil {
		utils.ExitF("Failed to create account directory: %s", err)
	}

	if err := utils.SaveAccountSecret(accountPath, secret); err != nil {
		utils.ExitF("Failed to save secret: %s", err)
	}

	tokenPath := utils.ConfigFilePath(tmpDir, TokenFilename)
	if _, err := os.Stat(tokenPath); err != nil {
		utils.ExitF("Failed to create token file: %s", err)
	}

	if err := os.Rename(tokenPath, utils.ConfigFilePath(accountPath, TokenFilename)); err != nil {
		utils.ExitF("Failed to move token file: %s", err)
	}

	if err := saveAccountConfig(baseDir, accountConfig{Current: email}); err != nil {
		utils.ExitF("Failed to save account config: %s", err)
	}

	removeLegacyToken(accountPath)

	return email
}

func accountAuthClient(args cli.Arguments, configDir string, secret utils.AccountSecret) *http.Client {
	accountArgs := copyArgs(args)
	accountArgs["configDir"] = configDir

	if accountArgs.String("refreshToken") != "" && accountArgs.String("accessToken") != "" {
		utils.ExitF("Access token not needed when refresh token is provided")
	}

	if accountArgs.String("refreshToken") != "" {
		return auth.NewRefreshTokenClient(secret.ClientID, secret.ClientSecret, accountArgs.String("refreshToken"))
	}

	if accountArgs.String("accessToken") != "" {
		return auth.NewAccessTokenClient(secret.ClientID, secret.ClientSecret, accountArgs.String("accessToken"))
	}

	if accountArgs.String("serviceAccount") != "" {
		serviceAccountPath := utils.ConfigFilePath(configDir, accountArgs.String("serviceAccount"))
		serviceAccountClient, err := auth.NewServiceAccountClient(serviceAccountPath)
		if err != nil {
			utils.ExitF("Failed to load service account: %s", err)
		}
		return serviceAccountClient
	}

	tokenPath := utils.ConfigFilePath(configDir, TokenFilename)
	client, err := auth.NewFileSourceClient(secret.ClientID, secret.ClientSecret, tokenPath, authCodePrompt)
	if err != nil {
		utils.ExitF("Failed getting oauth client: %s", err.Error())
	}

	return client
}

func resolveActiveConfigDir(baseDir string) (string, error) {
	config, err := loadAccountConfig(baseDir)
	if err == nil {
		if config.Current == "" {
			return "", errors.New("No account selected. Use `gdrive account switch` to select an account.")
		}

		accountPath := accountDir(baseDir, config.Current)
		if _, err := os.Stat(accountPath); err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("Account '%s' not found", config.Current)
			}
			return "", err
		}

		return accountPath, nil
	}

	if os.IsNotExist(err) {
		return baseDir, nil
	}

	return "", err
}

func accountExists(baseDir, name string) bool {
	accountPath := accountDir(baseDir, name)
	if _, err := os.Stat(accountPath); err != nil {
		return false
	}
	if _, err := os.Stat(utils.ConfigFilePath(accountPath, TokenFilename)); err != nil {
		return false
	}
	return true
}

func accountDir(baseDir, name string) string {
	return filepath.Join(baseDir, name)
}

func accountConfigPath(baseDir string) string {
	return filepath.Join(baseDir, AccountConfigFilename)
}

func loadAccountConfig(baseDir string) (accountConfig, error) {
	content, err := os.ReadFile(accountConfigPath(baseDir))
	if err != nil {
		return accountConfig{}, err
	}

	var config accountConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return accountConfig{}, err
	}

	return config, nil
}

func saveAccountConfig(baseDir string, config accountConfig) error {
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return err
	}
	return utils.WriteJSON(accountConfigPath(baseDir), config)
}

func listAccounts(baseDir string) ([]string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	accounts := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if accountExists(baseDir, name) {
			accounts = append(accounts, name)
		}
	}

	sort.Strings(accounts)
	return accounts, nil
}

func validateAccountName(name string) error {
	if name == "" {
		return errors.New("account name cannot be empty")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return errors.New("account name cannot contain path separators")
	}
	return nil
}

func promptAccountSecret() utils.AccountSecret {
	fmt.Println("To add an account you need a Google Client ID and Client Secret.")
	fmt.Println("Instructions for how to create credentials can be found here: https://github.com/glotlabs/gdrive/blob/main/docs/create_google_api_credentials.md")
	fmt.Println("Note that if you are using gdrive on a remote server you should read this first: https://github.com/glotlabs/gdrive#using-gdrive-on-a-remote-server")
	fmt.Println("")

	clientID := promptInput("Client ID")
	clientSecret := promptInput("Client secret")

	if clientID == "" || clientSecret == "" {
		utils.ExitF("Client ID and Client secret are required")
	}

	return utils.AccountSecret{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}

func promptInput(label string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", label)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		utils.ExitF("Failed reading input: %s", err)
	}
	return strings.TrimSpace(line)
}

func normalizeArchiveName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func createAccountArchive(srcDir, archivePath string) error {
	if _, err := os.Stat(archivePath); err == nil {
		return fmt.Errorf("archive '%s' already exists", archivePath)
	}

	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := tar.NewWriter(file)
	defer writer.Close()

	baseName := filepath.Base(srcDir)

	return filepath.WalkDir(srcDir, func(pathname string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Base(pathname) == "token_v2.json" {
			return nil
		}

		rel, err := filepath.Rel(srcDir, pathname)
		if err != nil {
			return err
		}

		name := filepath.ToSlash(filepath.Join(baseName, rel))
		if rel == "." {
			name = filepath.ToSlash(baseName)
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = name

		if err := writer.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		src, err := os.Open(pathname)
		if err != nil {
			return err
		}
		defer src.Close()

		_, err = io.Copy(writer, src)
		return err
	})
}

func archiveAccountName(archivePath string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := tar.NewReader(file)
	roots := map[string]struct{}{}

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		clean := path.Clean(header.Name)
		if clean == "." || clean == "" {
			continue
		}
		parts := strings.Split(clean, "/")
		if len(parts) > 0 && parts[0] != "" {
			roots[parts[0]] = struct{}{}
		}
	}

	if len(roots) == 0 {
		return "", errors.New("archive contains no account directory")
	}
	if len(roots) > 1 {
		return "", errors.New("archive contains multiple account directories")
	}

	for name := range roots {
		return name, nil
	}

	return "", errors.New("failed to read account name from archive")
}

func unpackAccountArchive(archivePath, baseDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := tar.NewReader(file)

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return err
	}

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		clean := path.Clean(header.Name)
		if clean == "." || clean == "" {
			continue
		}

		dest := filepath.Join(baseDir, filepath.FromSlash(clean))
		destAbs, err := filepath.Abs(dest)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(destAbs, baseAbs+string(os.PathSeparator)) && destAbs != baseAbs {
			return fmt.Errorf("invalid archive path: %s", header.Name)
		}

		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0700); err != nil {
				return err
			}
			continue
		}

		if header.Typeflag != tar.TypeReg {
			return fmt.Errorf("unsupported archive entry: %s", header.Name)
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			return err
		}

		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}

		if _, err := io.Copy(out, reader); err != nil {
			out.Close()
			return err
		}
		out.Close()
	}

	return nil
}

func removeLegacyToken(accountPath string) {
	_ = os.Remove(filepath.Join(accountPath, "token_v2.json"))
}

func copyArgs(args cli.Arguments) cli.Arguments {
	copy := cli.Arguments{}
	for key, value := range args {
		copy[key] = value
	}
	return copy
}

func getBaseConfigDir(args cli.Arguments) string {
	if os.Getenv("GDRIVE_CONFIG_DIR") != "" {
		return os.Getenv("GDRIVE_CONFIG_DIR")
	}
	return args.String("configDir")
}
