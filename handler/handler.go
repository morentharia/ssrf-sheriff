package handler

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"path"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/morentharia/ssrf-sheriff/colorjson"
	"github.com/morentharia/ssrf-sheriff/httpserver"
	"github.com/slack-go/slack"

	// "github.com/rs/zerolog/log"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"go.uber.org/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SerializableResponse is a generic type which both can be safely serialized to both XML and JSON
type SerializableResponse struct {
	SecretToken string `json:"token" xml:"token"`
}

// SSRFSheriffRouter is a wrapper around mux.Router to handle HTTP requests to the sheriff, with logging
type SSRFSheriffRouter struct {
	logger      *zap.Logger
	ssrfToken   string
	slackClient *slack.Client
	cfg         config.Provider
}

// NewHTTPServer provides a new HTTP server listener
func NewHTTPServer(
	mux *mux.Router,
	cfg config.Provider,
) *http.Server {
	return &http.Server{
		Addr:    cfg.Get("http.address").String(),
		Handler: mux,
	}
}

// NewSSRFSheriffRouter returns a new SSRFSheriffRouter which is used to route and handle all HTTP requests
func NewSSRFSheriffRouter(
	logger *zap.Logger,
	slackClient *slack.Client,
	cfg config.Provider,
) *SSRFSheriffRouter {
	return &SSRFSheriffRouter{
		logger:      logger,
		ssrfToken:   cfg.Get("ssrf_token").String(),
		slackClient: slackClient,
		cfg:         cfg,
	}
}

func NewSlackClient(cfg config.Provider) (*slack.Client, error) {
	api := slack.New(cfg.Get("slack.token").String())
	return api, nil
}

// StartServer starts the HTTP server
func StartServer(server *http.Server, lc fx.Lifecycle) {
	h := httpserver.NewHandle(server)
	lc.Append(fx.Hook{
		OnStart: h.Start,
		OnStop:  h.Shutdown,
	})
}

// PathHandler is the main handler for all inbound requests
func (s *SSRFSheriffRouter) PathHandler(w http.ResponseWriter, r *http.Request) {
	fileExtension := filepath.Ext(r.URL.Path)
	contentType := mime.TypeByExtension(fileExtension)
	var response string

	switch fileExtension {
	case ".json":
		res, _ := json.Marshal(SerializableResponse{SecretToken: s.ssrfToken})
		response = string(res)
	case ".xml":
		res, _ := xml.Marshal(SerializableResponse{SecretToken: s.ssrfToken})
		response = string(res)
	case ".html":
		tmpl := readTemplateFile("html.html")
		response = fmt.Sprintf(tmpl, s.ssrfToken, s.ssrfToken)
	case ".csv":
		tmpl := readTemplateFile("csv.csv")
		response = fmt.Sprintf(tmpl, s.ssrfToken)
	case ".txt":
		response = fmt.Sprintf("token=%s", s.ssrfToken)

	// TODO: dynamically generate these formats with the secret token rendered in the media
	case ".gif":
		response = readTemplateFile("gif.gif")
	case ".png":
		response = readTemplateFile("png.png")
	case ".jpg", ".jpeg":
		response = readTemplateFile("jpeg.jpg")
	case ".mp3":
		response = readTemplateFile("mp3.mp3")
	case ".mp4":
		response = readTemplateFile("mp4.mp4")
	default:
		response = s.ssrfToken
	}

	if contentType == "" {
		contentType = "text/plain"
	}

	log.Infof("New inbound HTTP request %s", colorjson.Marshal(map[string]interface{}{
		"IP":                    r.RemoteAddr,
		"Path":                  r.URL.Path,
		"Response Content-Type": contentType,
		"Headers":               r.Header,
	}))

	jsonBody, _ := json.Marshal(map[string]interface{}{
		"IP":                    r.RemoteAddr,
		"Path":                  r.URL.Path,
		"Response Content-Type": contentType,
		"Headers":               r.Header,
	})
	channelID, _, err := s.slackClient.PostMessage(
		s.cfg.Get("slack.channel_id").String(),
		slack.MsgOptionText(string(jsonBody), false),
	)
	if err != nil {
		logrus.WithError(err).WithField("channelID", channelID).Error("slack send message")
	}

	responseBytes := []byte(response)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Secret-Token", s.ssrfToken)
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

func readTemplateFile(templateFileName string) string {
	data, err := ioutil.ReadFile(path.Join("templates", path.Clean(templateFileName)))
	if err != nil {
		return ""
	}
	return string(data)
}

// NewServerRouter returns a new mux.Router for handling any HTTP request to /.*
func NewServerRouter(s *SSRFSheriffRouter) *mux.Router {
	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(s.PathHandler)
	return router
}

// NewConfigProvider returns a config.Provider for YAML configuration
func NewConfigProvider() (config.Provider, error) {
	return config.NewYAMLProviderFromFiles("config/base.yaml")
}

// NewLogger returns a new *zap.Logger
func NewLogger() (*zap.Logger, error) {
	// zapConfig := zap.NewProductionConfig()
	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.Encoding = "console"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zapConfig.DisableStacktrace = false

	return zapConfig.Build()
}
