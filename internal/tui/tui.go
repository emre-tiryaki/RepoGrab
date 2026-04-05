package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/emre-tiryaki/repograb/internal/download"
	"github.com/emre-tiryaki/repograb/internal/models"
	"github.com/emre-tiryaki/repograb/internal/provider"
)

type sessionState int

const (
	stateInput sessionState = iota
	stateToken
	stateDownloadDir
	stateLoading
	stateBrowser
)

type appConfig struct {
	Tokens      map[string]string `json:"tokens"`
	DownloadDir string            `json:"download_dir"`
}

type repositoryLoadedMsg struct {
	spec  provider.RepositorySpec
	items []models.FileNode
}

type repositoryLoadErrMsg struct {
	spec provider.RepositorySpec
	err  error
}

type downloadCompletedMsg struct {
	targetDir string
	count     int
}

type downloadErrMsg struct {
	err error
}

type MainModel struct {
	state            sessionState
	urlInput         textinput.Model
	tokenInput       textinput.Model
	downloadDirInput textinput.Model
	tokenProvider    string
	pendingSpec      provider.RepositorySpec
	hasPendingSpec   bool
	activeSpec       provider.RepositorySpec
	hasActiveSpec    bool
	browser          BrowserModel
	tokens           map[string]string
	downloadDir      string
	configPath       string
	err              error
	info             string
	loadingText      string
	theme            Theme
	width            int
	height           int
}

func (m MainModel) View() string {
	contentHeight := m.height - 3
	if contentHeight < 6 {
		contentHeight = 6
	}

	var content string

	switch m.state {
	case stateInput:
		body := "Please give repository url:\n\n" + m.urlInput.View()
		if m.err != nil {
			body += "\n\n" + lipgloss.NewStyle().Foreground(m.theme.ErrorColor).Render(m.err.Error())
		}
		if m.info != "" {
			body += "\n\n" + lipgloss.NewStyle().Foreground(m.theme.SecondaryColor).Render(m.info)
		}
		body += "\n\n[t] Access token settings"
		body += "\n[p] Download folder: " + m.downloadDir
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render(body)
	case stateToken:
		helpText := tokenHelpText(m.tokenProvider)
		body := fmt.Sprintf("Access token setup (%s)\n\n%s\n\nToken:\n%s", tokenProviderLabel(m.tokenProvider), helpText, m.tokenInput.View())
		if m.err != nil {
			body += "\n\n" + lipgloss.NewStyle().Foreground(m.theme.ErrorColor).Render(m.err.Error())
		}
		body += "\n\n[Tab] Switch provider • [Enter] Save • [Esc] Cancel"
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render(body)
	case stateDownloadDir:
		body := "Download folder setup\n\nFolder path:\n" + m.downloadDirInput.View()
		if m.err != nil {
			body += "\n\n" + lipgloss.NewStyle().Foreground(m.theme.ErrorColor).Render(m.err.Error())
		}
		body += "\n\n[Enter] Save • [Esc] Cancel"
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render(body)
	case stateLoading:
		loadingText := m.loadingText
		if loadingText == "" {
			loadingText = "Repository is downloading...\n\nPlease wait."
		}
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render(loadingText)
	case stateBrowser:
		body := m.browser.View()
		if m.err != nil {
			body += "\n" + lipgloss.NewStyle().Foreground(m.theme.ErrorColor).Render(m.err.Error())
		}
		if m.info != "" {
			body += "\n" + lipgloss.NewStyle().Foreground(m.theme.SecondaryColor).Render(m.info)
		}

		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render(body)
	}

	footer := m.renderFooter()
	return content + "\n" + footer
}

func (m MainModel) renderFooter() string {
	style := lipgloss.NewStyle().
		Background(m.theme.BackgroundColor).
		Foreground(m.theme.TextColor).
		Width(m.width)

	controls := "[Enter] Confirm • [t] Token Ayarları • [q] Çıkış "
	switch m.state {
	case stateToken:
		controls = "[Tab] Provider değiştir • [Enter] Kaydet • [Esc] İptal • [q] Çıkış "
	case stateDownloadDir:
		controls = "[Enter] Kaydet • [Esc] İptal • [q] Çıkış "
	case stateLoading:
		controls = "[q] Çıkış "
	case stateBrowser:
		controls = "[Space] Seç • [d] İndir • [b] Geri • [q] Çıkış "
	}

	return style.Render(controls)
}

func NewModel() MainModel {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://github.com/user/repo"
	urlInput.Focus()

	tokenInput := textinput.New()
	tokenInput.Placeholder = "paste access token"
	tokenInput.EchoMode = textinput.EchoPassword
	tokenInput.EchoCharacter = '•'

	downloadDirInput := textinput.New()
	downloadDirInput.Placeholder = defaultDownloadDir()

	configPath := configFilePath()
	loadedConfig := loadConfig(configPath)

	downloadDir := loadedConfig.DownloadDir
	if downloadDir == "" {
		downloadDir = defaultDownloadDir()
	}

	tokens := loadedConfig.Tokens
	if tokens == nil {
		tokens = make(map[string]string)
	}

	return MainModel{
		state:            stateInput,
		urlInput:         urlInput,
		tokenInput:       tokenInput,
		downloadDirInput: downloadDirInput,
		theme:            DefaultTheme(),
		tokens:           tokens,
		downloadDir:      downloadDir,
		configPath:       configPath,
		tokenProvider:    provider.ProviderGitHub,
		loadingText:      "Repository is downloading...\n\nPlease wait.",
	}
}

func (m MainModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case repositoryLoadedMsg:
		m.state = stateBrowser
		m.err = nil
		m.info = ""
		m.loadingText = "Repository is downloading...\n\nPlease wait."
		m.activeSpec = msg.spec
		m.hasActiveSpec = true
		m.hasPendingSpec = false
		m.browser = NewBrowserModel(msg.items, m.theme)
		return m, nil
	case downloadCompletedMsg:
		m.state = stateBrowser
		m.err = nil
		m.info = fmt.Sprintf("Downloaded %d file(s) to %s", msg.count, msg.targetDir)
		m.loadingText = "Repository is downloading...\n\nPlease wait."
		return m, nil
	case downloadErrMsg:
		m.state = stateBrowser
		m.info = ""
		m.err = msg.err
		m.loadingText = "Repository is downloading...\n\nPlease wait."
		return m, nil
	case repositoryLoadErrMsg:
		m.err = msg.err
		m.info = ""
		m.loadingText = "Repository is downloading...\n\nPlease wait."
		if errors.Is(msg.err, provider.ErrAuthRequired) {
			m.state = stateToken
			m.tokenProvider = msg.spec.Provider
			m.pendingSpec = msg.spec
			m.hasPendingSpec = true
			m.err = fmt.Errorf("this repository needs access. Enter a %s token and retry.", tokenProviderLabel(msg.spec.Provider))
			m.tokenInput.SetValue(m.tokens[m.tokenProvider])
			m.tokenInput.Focus()
			m.tokenInput.CursorEnd()
			return m, nil
		}

		m.state = stateInput
		m.urlInput.Focus()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			switch m.state {
			case stateInput:
				rawURL := strings.TrimSpace(m.urlInput.Value())
				if rawURL == "" {
					m.err = errors.New("repository url is required")
					m.info = ""
					return m, nil
				}

				selectedProvider, spec, err := m.detectProvider(rawURL)
				if err != nil {
					m.err = err
					m.info = ""
					return m, nil
				}

				m.err = nil
				m.info = ""
				m.state = stateLoading
				m.loadingText = "Repository is downloading...\n\nPlease wait."
				m.pendingSpec = *spec
				m.hasPendingSpec = true
				return m, m.fetchRepositoryCmd(selectedProvider, *spec)
			case stateToken:
				token := strings.TrimSpace(m.tokenInput.Value())
				m.tokens[m.tokenProvider] = token
				m.persistConfig()
				m.err = nil
				m.info = ""
				m.tokenInput.SetValue(token)

				if m.hasPendingSpec && m.pendingSpec.Provider == m.tokenProvider {
					selectedProvider := m.providerForSpec(m.pendingSpec)
					if selectedProvider == nil {
						m.state = stateInput
						m.err = errors.New("unsupported provider")
						return m, nil
					}

					m.state = stateLoading
					m.loadingText = "Repository is downloading...\n\nPlease wait."
					return m, m.fetchRepositoryCmd(selectedProvider, m.pendingSpec)
				}

				m.state = stateInput
				m.urlInput.Focus()
				return m, nil
			case stateDownloadDir:
				rawPath := strings.TrimSpace(m.downloadDirInput.Value())
				resolvedPath, err := resolveDownloadDir(rawPath)
				if err != nil {
					m.err = err
					return m, nil
				}

				m.downloadDir = resolvedPath
				m.persistConfig()
				m.state = stateInput
				m.err = nil
				m.info = "download folder updated"
				m.urlInput.Focus()
				return m, nil
			}
		case "tab":
			if m.state == stateToken {
				m.switchTokenProvider()
				return m, nil
			}
		case "esc":
			if m.state == stateToken {
				m.state = stateInput
				m.err = nil
				m.info = ""
				m.urlInput.Focus()
				return m, nil
			}
			if m.state == stateDownloadDir {
				m.state = stateInput
				m.err = nil
				m.info = ""
				m.urlInput.Focus()
				return m, nil
			}
		case "t":
			if m.state == stateInput {
				if providerName := m.providerNameFromCurrentURL(); providerName != "" {
					m.tokenProvider = providerName
				}
				m.tokenInput.SetValue(m.tokens[m.tokenProvider])
				m.tokenInput.Focus()
				m.tokenInput.CursorEnd()
				m.state = stateToken
				m.err = nil
				m.info = ""
				return m, nil
			}
		case "p":
			if m.state == stateInput {
				m.downloadDirInput.SetValue(m.downloadDir)
				m.downloadDirInput.Focus()
				m.downloadDirInput.CursorEnd()
				m.state = stateDownloadDir
				m.err = nil
				m.info = ""
				return m, nil
			}
		case "up", "k":
			if m.state == stateBrowser && m.browser.Cursor > 0 {
				m.browser.Cursor--
			}
		case "down", "j":
			if m.state == stateBrowser && len(m.browser.Items) > 0 && m.browser.Cursor < len(m.browser.Items)-1 {
				m.browser.Cursor++
			}
		case " ":
			if m.state == stateBrowser && len(m.browser.Items) > 0 {
				if _, ok := m.browser.Selected[m.browser.Cursor]; ok {
					delete(m.browser.Selected, m.browser.Cursor)
				} else {
					m.browser.Selected[m.browser.Cursor] = struct{}{}
				}
			}
		case "b":
			if m.state == stateBrowser {
				m.state = stateInput
				m.info = ""
				m.urlInput.Focus()
			}
		case "d":
			if m.state == stateBrowser {
				selectedItems := m.selectedDownloadItems()
				if len(selectedItems) == 0 {
					m.err = errors.New("select at least one file to download")
					m.info = ""
					return m, nil
				}

				if !m.hasActiveSpec {
					m.err = errors.New("active repository is not set")
					m.info = ""
					return m, nil
				}

				activeProvider := m.providerForSpec(m.activeSpec)
				if activeProvider == nil {
					m.err = errors.New("unsupported provider")
					m.info = ""
					return m, nil
				}

				m.state = stateLoading
				m.err = nil
				m.info = ""
				m.loadingText = "Selected files are downloading...\n\nPlease wait."
				return m, m.downloadSelectedCmd(activeProvider, m.activeSpec, selectedItems)
			}
		}
	}

	switch m.state {
	case stateInput:
		m.urlInput, cmd = m.urlInput.Update(msg)
		m.validateCurrentURLInput()
	case stateToken:
		m.tokenInput, cmd = m.tokenInput.Update(msg)
	case stateDownloadDir:
		m.downloadDirInput, cmd = m.downloadDirInput.Update(msg)
	}

	return m, cmd
}

func (m MainModel) providerForSpec(spec provider.RepositorySpec) provider.GitProvider {
	token := m.tokens[spec.Provider]

	switch spec.Provider {
	case provider.ProviderGitLab:
		return &provider.GitLabProvider{Token: token}
	default:
		return &provider.GithubProvider{Token: token}
	}
}

func (m MainModel) detectProvider(rawURL string) (provider.GitProvider, *provider.RepositorySpec, error) {
	providers := []provider.GitProvider{
		&provider.GithubProvider{Token: m.tokens[provider.ProviderGitHub]},
		&provider.GitLabProvider{Token: m.tokens[provider.ProviderGitLab]},
	}

	for _, selectedProvider := range providers {
		spec, err := selectedProvider.ParseRepositoryURL(rawURL)
		if err == nil {
			if spec.DefaultBranch == "" {
				spec.DefaultBranch = "main"
			}
			return selectedProvider, spec, nil
		}
	}

	return nil, nil, provider.ErrInvalidRepositoryURL
}

func (m MainModel) providerNameFromCurrentURL() string {
	_, spec, err := m.detectProvider(strings.TrimSpace(m.urlInput.Value()))
	if err != nil || spec == nil {
		return ""
	}

	return spec.Provider
}

func (m MainModel) fetchRepositoryCmd(gitProvider provider.GitProvider, spec provider.RepositorySpec) tea.Cmd {
	return func() tea.Msg {
		resolvedSpec := spec
		if err := gitProvider.ResolveRepository(&resolvedSpec); err != nil {
			return repositoryLoadErrMsg{spec: resolvedSpec, err: err}
		}

		items, err := gitProvider.FetchTree(&resolvedSpec)
		if err != nil {
			return repositoryLoadErrMsg{spec: resolvedSpec, err: err}
		}

		return repositoryLoadedMsg{spec: resolvedSpec, items: items}
	}
}

func (m MainModel) selectedDownloadItems() []models.FileNode {
	if len(m.browser.Selected) == 0 {
		return nil
	}

	indices := make([]int, 0, len(m.browser.Selected))
	for index := range m.browser.Selected {
		indices = append(indices, index)
	}
	sort.Ints(indices)

	items := make([]models.FileNode, 0, len(indices))
	for _, index := range indices {
		if index < 0 || index >= len(m.browser.Items) {
			continue
		}

		node := m.browser.Items[index]
		if node.Type == "dir" {
			continue
		}

		items = append(items, node)
	}

	return items
}

func (m MainModel) downloadSelectedCmd(gitProvider provider.GitProvider, spec provider.RepositorySpec, items []models.FileNode) tea.Cmd {
	return func() tea.Msg {
		repoDirName := strings.ReplaceAll(spec.ProjectPath, "/", "_")
		targetBaseDir := m.downloadDir
		if targetBaseDir == "" {
			targetBaseDir = defaultDownloadDir()
		}

		targetDir := filepath.Join(targetBaseDir, repoDirName)

		engine := &download.DownloadEngine{
			Provider:    gitProvider,
			BaseDir:     targetDir,
			MaxParallel: 5,
		}

		if err := engine.DownloadItems(items); err != nil {
			return downloadErrMsg{err: err}
		}

		return downloadCompletedMsg{targetDir: targetDir, count: len(items)}
	}
}

func (m *MainModel) persistConfig() {
	if m.configPath == "" {
		return
	}

	_ = saveConfig(m.configPath, appConfig{
		Tokens:      m.tokens,
		DownloadDir: m.downloadDir,
	})
}

func configFilePath() string {
	configDir, err := os.UserConfigDir()
	if err == nil && configDir != "" {
		return filepath.Join(configDir, "repograb", "config.json")
	}

	homeDir, homeErr := os.UserHomeDir()
	if homeErr == nil && homeDir != "" {
		return filepath.Join(homeDir, ".config", "repograb", "config.json")
	}

	return ""
}

func defaultDownloadDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		return "downloads"
	}

	return filepath.Join(homeDir, "Downloads")
}

func resolveDownloadDir(rawPath string) (string, error) {
	if rawPath == "" {
		return defaultDownloadDir(), nil
	}

	pathValue := rawPath
	if strings.HasPrefix(pathValue, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			pathValue = filepath.Join(homeDir, strings.TrimPrefix(pathValue, "~/"))
		}
	}

	absolutePath, err := filepath.Abs(pathValue)
	if err != nil {
		return "", fmt.Errorf("invalid download folder path")
	}

	if err := os.MkdirAll(absolutePath, 0755); err != nil {
		return "", fmt.Errorf("download folder cannot be created: %w", err)
	}

	return absolutePath, nil
}

func loadConfig(path string) appConfig {
	if path == "" {
		return appConfig{}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return appConfig{}
	}

	var cfg appConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return appConfig{}
	}

	if cfg.Tokens == nil {
		cfg.Tokens = make(map[string]string)
	}

	return cfg
}

func saveConfig(path string, cfg appConfig) error {
	if path == "" {
		return nil
	}

	if cfg.Tokens == nil {
		cfg.Tokens = make(map[string]string)
	}

	if cfg.DownloadDir == "" {
		cfg.DownloadDir = defaultDownloadDir()
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, payload, 0600)
}

func (m *MainModel) validateCurrentURLInput() {
	rawURL := strings.TrimSpace(m.urlInput.Value())
	if rawURL == "" {
		m.err = nil
		return
	}

	_, _, err := m.detectProvider(rawURL)
	if err != nil {
		m.err = errors.New("this url is invalid")
		return
	}

	m.err = nil
}

func (m *MainModel) switchTokenProvider() {
	switch m.tokenProvider {
	case provider.ProviderGitHub:
		m.tokenProvider = provider.ProviderGitLab
	default:
		m.tokenProvider = provider.ProviderGitHub
	}

	m.tokenInput.SetValue(m.tokens[m.tokenProvider])
	m.tokenInput.Focus()
	m.tokenInput.CursorEnd()
}

func tokenProviderLabel(name string) string {
	switch name {
	case provider.ProviderGitLab:
		return "GitLab"
	default:
		return "GitHub"
	}
}

func tokenHelpText(name string) string {
	switch name {
	case provider.ProviderGitLab:
		return "Create a Personal Access Token in GitLab: Preferences > Access Tokens. Grant read_repository access for private repositories."
	default:
		return "Create a Personal Access Token in GitHub: Settings > Developer settings > Personal access tokens. Grant repo read access for private repositories."
	}
}
