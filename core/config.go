package core

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"codex_core/auth"
	"codex_core/config"
	"codex_core/model"
	"codex_core/network"
	"codex_core/session"
)

type ConfigOptions struct {
	CodexHome       string
	Profile         string
	CWD             string
	Overrides       []string
	EnableFeatures  []string
	DisableFeatures []string

	Values     map[string]any
	APIKey     string
	Auth       *auth.AuthDotJSON
	ProviderID string
	Model      string
	BaseURL    string
	HTTPClient model.HTTPDoer
	Stream     bool

	Client      Options
	SessionRoot string
	Persist     bool
	Tools       *ToolOptions
}

func NewFromConfig(ctx context.Context, options *ConfigOptions) (*Client, error) {
	if options == nil {
		options = &ConfigOptions{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	codexHome := strings.TrimSpace(options.CodexHome)
	if codexHome == "" {
		var err error
		codexHome, err = config.FindCodexHome()
		if err != nil {
			return nil, err
		}
	}
	cfg := &config.Config{Values: cloneAnyMap(options.Values)}
	if options.Values == nil {
		loaded, err := config.LoadEffectiveWithOptions(codexHome, &config.EffectiveOptions{
			Profile: options.Profile, CWD: options.CWD, RawOverrides: options.Overrides,
			EnableFeatures: options.EnableFeatures, DisableFeatures: options.DisableFeatures,
		})
		if err != nil {
			return nil, err
		}
		cfg = loaded
	}
	providerID := firstNonEmpty(options.ProviderID, stringValue(cfg.Values, "model_provider"), model.OpenAIProviderID)
	modelID := firstNonEmpty(options.Model, stringValue(cfg.Values, "model"))
	baseURL := firstNonEmpty(options.BaseURL, stringValue(cfg.Values, "openai_base_url"))
	providerInfo, err := model.ProviderForConfigID(cfg.Values, providerID, baseURL)
	if err != nil {
		return nil, err
	}
	var snapshot *auth.AuthDotJSON
	switch {
	case options.Auth != nil:
		value := *options.Auth
		snapshot = &value
	case strings.TrimSpace(options.APIKey) != "":
		value := auth.FromAPIKey(options.APIKey)
		snapshot = &value
	default:
		resolved, resolveErr := auth.NewStore(codexHome).Resolve()
		if resolveErr != nil {
			return nil, resolveErr
		}
		if resolved != nil {
			value := resolved.Auth
			snapshot = &value
		}
	}
	if snapshot == nil && providerInfo.RequiresOpenAIAuth && strings.TrimSpace(providerInfo.EnvKey) == "" && providerInfo.Auth == nil {
		return nil, errors.New("OpenAI authentication is required: provide APIKey/Auth, set OPENAI_API_KEY, or configure auth.json")
	}
	runtimeProvider := model.CreateRuntimeProviderForID(providerID, *providerInfo, snapshot)
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = network.NewHTTPClient(cfg.RespectSystemProxyEnabled(), 0)
	}
	agent, err := model.NewResponsesAgentRunnerFromRuntimeProviderWithAuth(providerID, runtimeProvider, httpClient, codexHome, snapshot)
	if err != nil {
		return nil, err
	}
	agent.Stream = options.Stream
	manager := runtimeProvider.ModelsManager(nil)
	if strings.TrimSpace(modelID) == "" {
		modelID = manager.GetDefaultModel("", true, model.RefreshOffline)
	}
	clientOptions := options.Client
	if options.Tools != nil {
		router, routerErr := BuildToolRouter(options.Tools)
		if routerErr != nil {
			return nil, routerErr
		}
		clientOptions.Router = router
	}
	clientOptions.Agent = agent
	clientOptions.DefaultModel = modelID
	clientOptions.DefaultProvider = providerID
	if options.Persist {
		root := strings.TrimSpace(options.SessionRoot)
		if root == "" {
			root = filepath.Join(codexHome, "core_sessions")
		}
		clientOptions.SessionStore = session.NewStore(root)
	}
	return New(&clientOptions)
}

func stringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

var _ model.HTTPDoer = (*http.Client)(nil)
