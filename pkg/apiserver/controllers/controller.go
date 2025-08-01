package controllers

import (
	"net"
	"net/http"
	"strings"

	"github.com/alexliesenfeld/health"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	v1 "github.com/crowdsecurity/crowdsec/pkg/apiserver/controllers/v1"
	"github.com/crowdsecurity/crowdsec/pkg/csconfig"
	"github.com/crowdsecurity/crowdsec/pkg/database"
	"github.com/crowdsecurity/crowdsec/pkg/models"
)

type Controller struct {
	DBClient                      *database.Client
	Router                        *gin.Engine
	Profiles                      []*csconfig.ProfileCfg
	AlertsAddChan                 chan []*models.Alert
	DecisionDeleteChan            chan []*models.Decision
	PluginChannel                 chan models.ProfileAlert
	Log                           *log.Logger
	ConsoleConfig                 *csconfig.ConsoleConfig
	TrustedIPs                    []net.IPNet
	HandlerV1                     *v1.Controller
	AutoRegisterCfg               *csconfig.LocalAPIAutoRegisterCfg
	DisableRemoteLapiRegistration bool
}

func (c *Controller) Init() error {
	if err := c.NewV1(); err != nil {
		return err
	}

	/* if we have a V2, just add

	if err := c.NewV2(); err != nil {
		return err
	}

	*/

	return nil
}

// endpoint for health checking
func serveHealth() http.HandlerFunc {
	checker := health.NewChecker(
		// just simple up/down status is enough
		health.WithDisabledDetails(),
		// no caching required
		health.WithDisabledCache(),
	)

	return health.NewHandler(checker)
}

func eitherAuthMiddleware(jwtMiddleware gin.HandlerFunc, apiKeyMiddleware gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch {
		case c.GetHeader("X-Api-Key") != "":
			apiKeyMiddleware(c)
		case c.GetHeader("Authorization") != "":
			jwtMiddleware(c)
		// uh no auth header. is this TLS with mutual authentication?
		case strings.HasPrefix(c.Request.UserAgent(), "crowdsec/"):
			// guess log processors by sniffing user-agent
			jwtMiddleware(c)
		default:
			apiKeyMiddleware(c)
		}
	}
}

func (c *Controller) NewV1() error {
	var err error

	v1Config := v1.ControllerV1Config{
		DbClient:           c.DBClient,
		ProfilesCfg:        c.Profiles,
		DecisionDeleteChan: c.DecisionDeleteChan,
		AlertsAddChan:      c.AlertsAddChan,
		PluginChannel:      c.PluginChannel,
		ConsoleConfig:      *c.ConsoleConfig,
		TrustedIPs:         c.TrustedIPs,
		AutoRegisterCfg:    c.AutoRegisterCfg,
	}

	c.HandlerV1, err = v1.New(&v1Config)
	if err != nil {
		return err
	}

	c.Router.GET("/health", gin.WrapF(serveHealth()))
	c.Router.Use(v1.PrometheusMiddleware())
	// We don't want to compress the response body as it would likely break some existing bouncers
	// But we do want to automatically uncompress incoming requests
	c.Router.Use(gzip.Gzip(gzip.NoCompression, gzip.WithDecompressOnly(), gzip.WithDecompressFn(gzip.DefaultDecompressHandle)))
	c.Router.HandleMethodNotAllowed = true
	c.Router.UnescapePathValues = true
	c.Router.UseRawPath = true
	c.Router.NoRoute(func(ctx *gin.Context) {
		ctx.AbortWithStatus(http.StatusNotFound)
	})
	c.Router.NoMethod(func(ctx *gin.Context) {
		ctx.AbortWithStatus(http.StatusMethodNotAllowed)
	})

	groupV1 := c.Router.Group("/v1")
	groupV1.POST("/watchers", c.HandlerV1.AbortRemoteIf(c.DisableRemoteLapiRegistration), c.HandlerV1.CreateMachine)
	groupV1.POST("/watchers/login", c.HandlerV1.Middlewares.JWT.Middleware.LoginHandler)

	jwtAuth := groupV1.Group("")
	jwtAuth.GET("/refresh_token", c.HandlerV1.Middlewares.JWT.Middleware.RefreshHandler)
	jwtAuth.Use(c.HandlerV1.Middlewares.JWT.Middleware.MiddlewareFunc(), v1.PrometheusMachinesMiddleware())
	{
		jwtAuth.POST("/alerts", c.HandlerV1.CreateAlert)
		jwtAuth.GET("/alerts", c.HandlerV1.FindAlerts)
		jwtAuth.HEAD("/alerts", c.HandlerV1.FindAlerts)
		jwtAuth.GET("/alerts/:alert_id", c.HandlerV1.FindAlertByID)
		jwtAuth.HEAD("/alerts/:alert_id", c.HandlerV1.FindAlertByID)
		jwtAuth.DELETE("/alerts/:alert_id", c.HandlerV1.DeleteAlertByID)
		jwtAuth.DELETE("/alerts", c.HandlerV1.DeleteAlerts)
		jwtAuth.DELETE("/decisions", c.HandlerV1.DeleteDecisions)
		jwtAuth.DELETE("/decisions/:decision_id", c.HandlerV1.DeleteDecisionById)
		jwtAuth.GET("/heartbeat", c.HandlerV1.HeartBeat)
		jwtAuth.GET("/allowlists", c.HandlerV1.GetAllowlists)
		jwtAuth.GET("/allowlists/:allowlist_name", c.HandlerV1.GetAllowlist)
		jwtAuth.GET("/allowlists/check/:ip_or_range", c.HandlerV1.CheckInAllowlist)
		jwtAuth.HEAD("/allowlists/check/:ip_or_range", c.HandlerV1.CheckInAllowlist)
		jwtAuth.POST("/allowlists/check", c.HandlerV1.CheckInAllowlistBulk)
		jwtAuth.DELETE("/watchers/self", c.HandlerV1.DeleteMachine)
	}

	apiKeyAuth := groupV1.Group("")
	apiKeyAuth.Use(c.HandlerV1.Middlewares.APIKey.MiddlewareFunc(), v1.PrometheusBouncersMiddleware())
	{
		apiKeyAuth.GET("/decisions", c.HandlerV1.GetDecision)
		apiKeyAuth.HEAD("/decisions", c.HandlerV1.GetDecision)
		apiKeyAuth.GET("/decisions/stream", c.HandlerV1.StreamDecision)
		apiKeyAuth.HEAD("/decisions/stream", c.HandlerV1.StreamDecision)
	}

	eitherAuth := groupV1.Group("")
	eitherAuth.Use(eitherAuthMiddleware(c.HandlerV1.Middlewares.JWT.Middleware.MiddlewareFunc(), c.HandlerV1.Middlewares.APIKey.MiddlewareFunc()))
	{
		eitherAuth.POST("/usage-metrics", c.HandlerV1.UsageMetrics)
	}

	return nil
}

/*
func (c *Controller) NewV2() error {
	handlerV2, err := v2.New(c.DBClient, c.Ectx)
	if err != nil {
		return err
	}

	v2 := c.Router.Group("/v2")
	v2.POST("/watchers", handlerV2.CreateMachine)
	v2.POST("/watchers/login", handlerV2.Middlewares.JWT.Middleware.LoginHandler)

	jwtAuth := v2.Group("")
	jwtAuth.GET("/refresh_token", handlerV2.Middlewares.JWT.Middleware.RefreshHandler)
	jwtAuth.Use(handlerV2.Middlewares.JWT.Middleware.MiddlewareFunc())
	{
		jwtAuth.POST("/alerts", handlerV2.CreateAlert)
		jwtAuth.GET("/alerts", handlerV2.FindAlerts)
		jwtAuth.DELETE("/alerts", handlerV2.DeleteAlerts)
		jwtAuth.DELETE("/decisions", handlerV2.DeleteDecisions)
		jwtAuth.DELETE("/decisions/:decision_id", handlerV2.DeleteDecisionById)
	}

	apiKeyAuth := v2.Group("")
	apiKeyAuth.Use(handlerV2.Middlewares.APIKey.MiddlewareFuncV2())
	{
		apiKeyAuth.GET("/decisions", handlerV2.GetDecision)
		apiKeyAuth.GET("/decisions/stream", handlerV2.StreamDecision)
	}

	return nil
}

*/
